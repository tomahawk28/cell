package main

import (
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
	pollPeriod      = flag.Duration("poll", 30*time.Second, "Poll Period")
)

var (
	screenCache = ScreenCache{time.Now(), []byte{}, sync.RWMutex{}}
	mu          = sync.Mutex{}
	tmpl        = template.Must(template.ParseFiles("template.html"))
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
	for {
		select {
		case r := <-in:
			log.Println("Thread ", thread_number, ":", r.command)
			switch r.command {
			case "keyp":
				scpicmd := fmt.Sprintf("KEYP:%s", r.args["value"])
				cell.SendSCPI(scpicmd)
				r.result <- []byte("")
			case "touch":
				scpicmd := fmt.Sprintf("KEYP %s %s", r.args["x"], r.args["y"])
				cell.SendSCPI(scpicmd)
				r.result <- []byte("")
			case "screen":
				go func() {
					screenCache.mu.Lock()
					defer screenCache.mu.Unlock()
					if time.Now().Sub(screenCache.last).Seconds() > 1 {
						screenCache.last = time.Now()
						screenCache.cache = cell.GetScreen()
					}
					r.result <- screenCache.cache
				}()
			case "heartbeat":
				cell.SendMessage(0x50, "")
				r.result <- cell.GetMessage()
			}
		case <-time.After(time.Second * 20):
			mu.Lock()
			cell.SendMessage(0x50, "")
			log.Println("Hearbeat:", thread_number, string(cell.GetMessage()))
			mu.Unlock()
		}
	}
}

func NewRequest(command string, args map[string]string) *Request {
	return &Request{command, args, make(chan []byte, 1)}
}

func main() {
	flag.Parse()
	cell_list := []cell.CellAdvisor{cell.NewCellAdvisor(*cellAdvisorAddr),
		cell.NewCellAdvisor(*cellAdvisorAddr),
		cell.NewCellAdvisor(*cellAdvisorAddr)}
	//cell := cell.NewCellAdvisor(*cellAdvisorAddr)

	request_channel := make(chan *Request, 20)
	for i, _ := range cell_list {
		go Poller(request_channel, &cell_list[i], i)
	}

	http.HandleFunc("/screen", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		request_object := NewRequest("screen", nil)
		request_channel <- request_object

		//For supporting AJAX sending JPEG, must encoding data through base64
		w.Write(<-request_object.result)
	})
	http.HandleFunc("/touch", func(w http.ResponseWriter, req *http.Request) {
		query := req.URL.Query()
		x, y := query.Get("x"), query.Get("y")
		if x != "" && y != "" {
			request_object := NewRequest("touch", map[string]string{"x": x, "y": y})
			request_channel <- request_object
			w.Write(<-request_object.result)
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
			w.Write(<-request_object.result)

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