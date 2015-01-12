//JDSU CellAdvisor Web-Live Program
//Copyright (C) 2015 Jihyuk Bok <tomahawk28@gmail.com>
//
//Permission is hereby granted, free of charge, to any person obtaining
//a copy of this software and associated documentation files (the "Software"),
//to deal in the Software without restriction, including without limitation
//the rights to use, copy, modify, merge, publish, distribute, sublicense,
//and/or sell copies of the Software, and to permit persons to whom the
//Software is furnished to do so, subject to the following conditions:
//
//The above copyright notice and this permission notice shall be included
//in all copies or substantial portions of the Software.
//
//THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
//EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
//OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
//IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
//DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
//TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE
//OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package main

import (
	"expvar"
	"flag"
	"fmt"
	"log"
	"sync"
	"text/template"
	"time"

	"net/http"

	"github.com/tomahawk28/cell"
)

var (
	httpAddr        = flag.String("http", ":8040", "Listen Address")
	cellAdvisorAddr = flag.String("celladdr", "10.82.26.12", "CellAdvisor Address")
	numsport        = flag.Uint("numsport", 4, "The number of ports ")
	pollPeriod      = flag.Duration("poll", 30*time.Second, "Poll Period")
)

var (
	screenCache = ScreenCache{time.Now(), []byte{}, sync.RWMutex{}}
	mu          = sync.Mutex{}
	tmpl        = template.Must(template.ParseFiles("template.html"))
)

var (
	sendSuccessCount    = expvar.NewInt("sendSuccessCount")
	receiveSucessCount  = expvar.NewInt("receiveSucessCount")
	sendPendingCount    = expvar.NewInt("sendPendingCount")
	receivePendingCount = expvar.NewInt("receivePendingCount")
)

type Request struct {
	command string
	args    map[string]string
	result  chan []byte
}

type ScreenCache struct {
	last  time.Time
	cache []byte
	mu    sync.RWMutex
}

func Poller(in <-chan *Request, cell *cell.CellAdvisor, thread_number int) {
	done := make(chan struct{})
	defer close(done)
	var err error
	for {
		select {
		case r := <-in:
			log.Println("Thread ", thread_number, ":", r.command)
			switch r.command {
			case "keyp":
				scpicmd := fmt.Sprintf("KEYP:%s", r.args["value"])
				_, err = cell.SendSCPI(scpicmd)
				sendResult(done, r.result, []byte{})
			case "touch":
				scpicmd := fmt.Sprintf("KEYP %s %s", r.args["x"], r.args["y"])
				_, err = cell.SendSCPI(scpicmd)
				sendResult(done, r.result, []byte{})
			case "screen":
				go func() {
					screenCache.mu.Lock()
					defer screenCache.mu.Unlock()
					if time.Now().Sub(screenCache.last).Seconds() > 1 {
						screenCache.last = time.Now()
						screenCache.cache, err = cell.GetScreen()
						if err != nil {
							log.Println(err.Error())
						}
					}
					sendResult(done, r.result, screenCache.cache)
				}()
			case "heartbeat":
				msg, err := cell.GetStatusMessage()
				if err != nil {
					log.Println(err.Error())
				}
				sendResult(done, r.result, msg)
			}
		case <-time.After(time.Second * 15):
			mu.Lock()
			msg, err := cell.GetStatusMessage()
			if err != nil {
				log.Println(err.Error())
			}
			log.Println("Hearbeat:", thread_number, string(msg))
			mu.Unlock()
		}
		//Check Error Status == EOF

		if err != nil && err.Error() == "EOF" {
			return
		}
	}
}

func NewRequest(command string, args map[string]string) *Request {
	return &Request{command, args, make(chan []byte)}
}

func sendResult(done <-chan struct{}, pipe chan<- []byte, result []byte) {
	select {
	case pipe <- result:
		sendSuccessCount.Add(1)
	case <-time.After(time.Second * 3):
		log.Println("Sending Timeout")
		sendPendingCount.Add(1)
	case <-done:
		return
	}
}
func receiveResult(pipe <-chan []byte) []byte {
	select {
	case result := <-pipe:
		receiveSucessCount.Add(1)
		return result
	case <-time.After(time.Second * 5):
		log.Println("Receive Timeout")
		receivePendingCount.Add(1)
	}
	return []byte{}
}

func main() {

	flag.Parse()
	// 4 Ports ready for work
	cell_list := make([]cell.CellAdvisor, *numsport)

	for i, _ := range cell_list {
		cell_list[i] = cell.NewCellAdvisor(*cellAdvisorAddr)
	}

	request_channel := make(chan *Request, len(cell_list))
	for i, _ := range cell_list {
		go Poller(request_channel, &cell_list[i], i)
	}

	http.HandleFunc("/screen", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		request_object := NewRequest("screen", nil)
		request_channel <- request_object

		w.Write(receiveResult(request_object.result))
	})
	http.HandleFunc("/touch", func(w http.ResponseWriter, req *http.Request) {
		query := req.URL.Query()
		x, y := query.Get("x"), query.Get("y")
		if x != "" && y != "" {
			request_object := NewRequest("touch", map[string]string{"x": x, "y": y})
			request_channel <- request_object
			w.Write(receiveResult(request_object.result))
		} else {
			fmt.Fprintf(w, "Coordination not given")
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	http.HandleFunc("/keyp", func(w http.ResponseWriter, req *http.Request) {
		err := req.ParseForm()
		if err != nil {
			fmt.Fprintf(w, "Form Parse error")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		value := req.FormValue("value")

		if value != "" {
			request_object := NewRequest("keyp", map[string]string{"value": value})
			request_channel <- request_object
			w.Write(receiveResult(request_object.result))

		} else {
			fmt.Fprintf(w, "Keypad name not given")
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		err := tmpl.Execute(w, nil)
		if err != nil {
			panic(err)
		}
	})
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}
