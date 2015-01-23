// Package restful provides RESTful API features to CellAdvisor
package restful

import (
	"encoding/json"
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
	code     int
	isbinary bool
	data     interface{}
}

func (result pollResult) String() string {
	if result.isbinary {
		if v, ok := result.data.([]byte); ok {
			return string(v)
		} else {
			fmt.Println("FUCKNO")
			return ""
		}
	}
	middledata := map[string]interface{}{
		"success": result.code == http.StatusOK,
		"data":    result.data,
	}

	m, err := json.Marshal(middledata)
	if err != nil {
		log.Println("json parsing error")
		return ""
	}
	return string(m)

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
func BuildCellAdvisorRestfulAPI(threadNumber int, cellAddr string, pollPeriod time.Duration) *mux.Router {

	s := createCellAdvisorHTTPServer(threadNumber, cellAddr, pollPeriod)
	rtr := mux.NewRouter()
	rtr.Handle("/api/{command}.json", s)
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
	request := createRequest(command, values)
	server.requestChannel <- request
	result := receiveResult(request.result)
	if result == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Poller thread respond timeout"))
		return
	}
	if result.isbinary {
		w.Header().Set("Content-Type", "application/jpeg")
	} else {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(result.code)
	fmt.Fprintf(w, "%s", result)
	//w.Write([]byte(result.requestErr.Error()))
	return
}

func (server *cellServer) poller(cell *cell.CellAdvisor, threadNumber int) {
	done := make(chan struct{})
	defer close(done)
	var err error
	var data interface{}
	var isbinary bool
	for {
		data, isbinary = "", false
		numsent, code := 0, http.StatusOK
		err = nil

		select {
		case request := <-server.requestChannel:
			log.Printf("Thread(%d) get_request:%s", threadNumber, request.command)
			switch request.command {
			case "keyp":
				if value := request.args.Get("value"); value == "" {
					code = http.StatusBadRequest
					data = "keyp value missing"
				} else {
					scpicmd := fmt.Sprintf("KEYP:%s", request.args.Get("value"))
					numsent, err = cell.SendSCPI(scpicmd)
					if err != nil {
						code = http.StatusInternalServerError
						data = err.Error()
					} else {
						data = fmt.Sprintf("keypad: %d byte sent", numsent)
					}
				}
			case "touch":
				if x, y := request.args.Get("x"), request.args.Get("y"); x == "" || y == "" {
					code = http.StatusBadRequest
					data = "x,y value missing"
				} else {
					scpicmd := fmt.Sprintf("KEYP %s %s", request.args.Get("x"), request.args.Get("y"))
					numsent, err = cell.SendSCPI(scpicmd)
					if err != nil {
						code = http.StatusInternalServerError
						data = err.Error()
					} else {
						data = fmt.Sprintf("touch: %d byte sent", numsent)
					}
				}
			case "screen":
				server.screenCache.mu.RLock()
				isbinary = true
				data = server.screenCache.cache
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
				data = "refresh_screen : cashe done"
			case "interference_power":
				data, err = cell.GetInterferencePower()
				if err != nil {
					code = http.StatusInternalServerError
					data = err.Error()
				}
			case "heartbeat":
				data, err = cell.GetStatusMessage()
				if err != nil {
					code = http.StatusInternalServerError
					data = err.Error()
				}

			default:
				code = http.StatusBadRequest
				data = fmt.Sprintf("unknown command name : %s", request.command)
			}
			sendResult(done, request.result, pollResult{code, isbinary, data})
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
