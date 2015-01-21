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

package restful

import (
	"errors"
	"expvar"
	"fmt"
	"log"
	"sync"
	"time"

	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/tomahawk28/cell"
)

var (
	sendSuccessCount    = expvar.NewInt("sendSuccessCount")
	receiveSucessCount  = expvar.NewInt("receiveSucessCount")
	sendPendingCount    = expvar.NewInt("sendPendingCount")
	receivePendingCount = expvar.NewInt("receivePendingCount")
)

type pollRequest struct {
	command string
	args    url.Values
	result  chan pollResult
}

type pollResult struct {
	resultByte []byte
	requestErr error
}

type pollScreenCache struct {
	last  time.Time
	cache []byte
	mu    sync.RWMutex
}

// cellServer implements http API method to cellAdvisor,
type cellServer struct {
	//ScreenCache implements web cache, for any cases clients order new screen image,
	//API fetch screen only if its 1 minutes later after last capture
	screenCache    pollScreenCache
	mu             sync.RWMutex
	requestChannel chan *pollRequest
	pollPeriod     time.Duration
}

func NewCellHttpServer(threadNumber int, cellAddr string, pollPeriod time.Duration) cellServer {
	screenCache := pollScreenCache{time.Now(), []byte{}, sync.RWMutex{}}
	mu := sync.RWMutex{}

	rc := make(chan *pollRequest, threadNumber)

	server := cellServer{screenCache, mu, rc, pollPeriod}

	for i := 0; i < threadNumber; i++ {
		element := cell.NewCellAdvisor(cellAddr)
		go server.poller(&element, i)
	}

	return server
}

func BuildCellAdvisorRestfulAPI(threadNumber int, cellAddr string, pollPeriod time.Duration) *mux.Router {
	s := NewCellHttpServer(threadNumber, cellAddr, pollPeriod)

	rtr := mux.NewRouter()
	rtr.Handle("/api/screen/{command}", s)
	rtr.Handle("/api/scpi/{command}", s).Methods("POST")

	return rtr
}

func (server cellServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var values url.Values
	params := mux.Vars(r)
	command := params["command"]
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		values = r.Form
	}
	request := NewRequest(command, values)
	server.requestChannel <- request
	result := receiveResult(request.result)
	if result == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Poller thread respond timeout"))
		return
	}
	if result.requestErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(result.requestErr.Error()))
		return
	}

	w.Write(result.resultByte)
}

func (server *cellServer) poller(cell *cell.CellAdvisor, threadNumber int) {
	done := make(chan struct{})
	defer close(done)
	var err error
	var msg []byte
	for {
		log.Println("waiting..", threadNumber)
		select {
		case request := <-server.requestChannel:
			log.Println("Thread ", threadNumber, ":", request.command)
			switch request.command {
			case "keyp":
				if value := request.args.Get("value"); value == "" {
					sendResult(done, request.result, pollResult{nil, errors.New("keyp value missing")})
				} else {
					scpicmd := fmt.Sprintf("KEYP:%s", request.args.Get("value"))
					_, err = cell.SendSCPI(scpicmd)
					sendResult(done, request.result, pollResult{[]byte("OK"), nil})
				}
			case "touch":
				if x, y := request.args.Get("x"), request.args.Get("y"); x == "" || y == "" {
					sendResult(done, request.result, pollResult{nil, errors.New("x,y value missing")})
				} else {
					scpicmd := fmt.Sprintf("KEYP %s %s", request.args.Get("x"), request.args.Get("y"))
					_, err = cell.SendSCPI(scpicmd)
					sendResult(done, request.result, pollResult{[]byte("OK"), nil})
				}
			case "screen":
				server.screenCache.mu.RLock()
				sendResult(done, request.result, pollResult{server.screenCache.cache, nil})
				server.screenCache.mu.RUnlock()
			case "refresh_screen":
				go func() {
					if len(server.screenCache.cache) == 0 || time.Now().Sub(server.screenCache.last).Seconds() > 1 {
						server.screenCache.mu.Lock()
						defer server.screenCache.mu.Unlock()
						if len(server.screenCache.cache) == 0 || time.Now().Sub(server.screenCache.last).Seconds() > 1 {
							server.screenCache.last = time.Now()
							server.screenCache.cache, err = cell.GetScreen()
							if err != nil {
								log.Println(err.Error())
							}
						}
					}
				}()
				sendResult(done, request.result, pollResult{[]byte("OK"), nil})
			case "heartbeat":
				msg, err = cell.GetStatusMessage()
				sendResult(done, request.result, pollResult{[]byte(msg), nil})
			default:
				sendResult(done, request.result, pollResult{nil, errors.New("unknown command")})
			}
		case <-time.After(server.pollPeriod):
			server.mu.Lock()
			msg, err = cell.GetStatusMessage()
			log.Println("Hearbeat:", threadNumber, string(msg))
			server.mu.Unlock()
		}
		//Check Error Status == EOF
		if err != nil {
			switch err.Error() {
			case "EOF":
				log.Println("Connection loses on ", threadNumber, ", Poller exited")
				return
			default:
				log.Println("Thread ", threadNumber, " got error: ", err.Error())
			}
		}
	}
}

func NewRequest(command string, args url.Values) *pollRequest {
	return &pollRequest{command, args, make(chan pollResult)}
}

func sendResult(done <-chan struct{}, pipe chan<- pollResult, result pollResult) {
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
func receiveResult(pipe <-chan pollResult) *pollResult {
	select {
	case result := <-pipe:
		receiveSucessCount.Add(1)
		return &result
	case <-time.After(time.Second * 5):
		log.Println("Receive Timeout")
		receivePendingCount.Add(1)
	}
	return nil
}
