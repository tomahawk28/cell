// Package restful provides RESTful API features to CellAdvisor
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

var (
	okb = []byte("OK")
)

type pollRequest struct {
	command string
	args    url.Values
	result  chan pollResult
}

type pollResult struct {
	resultByte  []byte
	requestType string
	requestErr  error
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
	requestChannel chan *pollRequest
	pollPeriod     time.Duration
}

// NewCellHTTPServer retuning CellAdviosr http server object
func createCellAdvisorHTTPServer(threadNumber int, cellAddr string, pollPeriod time.Duration) cellServer {
	screenCache := pollScreenCache{time.Now(), []byte{}, sync.RWMutex{}}

	rc := make(chan *pollRequest, threadNumber)

	server := cellServer{screenCache, rc, pollPeriod}

	for i := 0; i < threadNumber; i++ {
		element := cell.NewCellAdvisor(cellAddr)
		go server.poller(&element, i)
	}

	return server
}

// BuildCellAdvisorRestfulAPI returning automatic RESTful API server set
// user could access directly api/screen/*, and api/scpi/*
// after deploy retuning object to sever
func BuildCellAdvisorRestfulAPI(prefix string, threadNumber int, cellAddr string, pollPeriod time.Duration) *mux.Router {

	s := createCellAdvisorHTTPServer(threadNumber, cellAddr, pollPeriod)
	rtr := mux.NewRouter()
	rtr.Handle("/"+prefix+"api/{command}.json", s)
	rtr.Handle("/"+prefix+"api/screen/{command}", s)
	rtr.Handle("/"+prefix+"api/scpi/{command}", s).Methods("POST")

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
	request := createRequest(command, values)
	server.requestChannel <- request
	result := receiveResult(request.result)
	if result == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Poller thread respond timeout"))
		return
	}
	w.Header().Set("Content-Type", result.requestType)
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
	for {
		msg := []byte{}
		responseType := "text/plain"
		num := 0
		err = nil
		select {
		case request := <-server.requestChannel:
			log.Println("Thread ", threadNumber, ":", request.command)
			switch request.command {
			case "keyp":
				if value := request.args.Get("value"); value == "" {
					err = errors.New("keyp value missing")
				} else {
					scpicmd := fmt.Sprintf("KEYP:%s", request.args.Get("value"))
					num, err = cell.SendSCPI(scpicmd)
					msg = []byte(fmt.Sprintf("keypad: %d byte sent", num))
				}
			case "touch":
				if x, y := request.args.Get("x"), request.args.Get("y"); x == "" || y == "" {
					err = errors.New("x,y value missing")
				} else {
					scpicmd := fmt.Sprintf("KEYP %s %s", request.args.Get("x"), request.args.Get("y"))
					num, err = cell.SendSCPI(scpicmd)
					msg = []byte(fmt.Sprintf("touch: %d byte sent", num))
				}
			case "screen":
				responseType = "application/jpeg"
				server.screenCache.mu.RLock()
				msg = server.screenCache.cache
				server.screenCache.mu.RUnlock()
			case "refresh_screen":
				func() {
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
				msg = []byte("refresh_screen : done")
			case "interference_power":
				responseType = "application/json"
				msg, err = cell.GetInterferencePower()
			case "heartbeat":
				msg, err = cell.GetStatusMessage()
			default:
				err = errors.New("Unknown command")
			}
			sendResult(done, request.result, createResult(msg, responseType, err))
		case <-time.After(server.pollPeriod):
			r := createRequest("heartbeat", nil)
			server.requestChannel <- r
			<-r.result
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

func createRequest(command string, args url.Values) *pollRequest {
	return &pollRequest{command, args, make(chan pollResult)}
}

func createResult(result []byte, resultType string, err error) pollResult {
	if resultType == "" {
		resultType = "text/plain"
	}
	return pollResult{result, resultType, err}
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
