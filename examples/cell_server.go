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

type Request struct {
	command string
	args    map[string]string
	result  chan []byte
}

func Poller(in <-chan Request, cell *CellAdvisor) {
	for r := range in {
		switch r.command {
		case "scpi":
		case "touch":
		case "screen":
		case "heartbeat":
		}
	}
}

func main() {
	flag.Parse()
	cell_list := []CellAdvisor{cell.NewCellAdvisor(*cellAdvisorAddr), cell.NewCellAdvisor(*cellAdvisorAddr)}
	//cell := cell.NewCellAdvisor(*cellAdvisorAddr)

	//Prevent socket closing after web page leaves
	ticker := time.NewTicker(*pollPeriod)
	go func() {
		for _ = range ticker.C {
			mu.Lock()
			cell.SendMessage(0x50, "")
			log.Println("Hearbeat msg: ", string(cell.GetMessage()))
			mu.Unlock()
		}
	}()

	http.HandleFunc("/screen", func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(cell.GetScreen())
		mu.Unlock()
	})
	http.HandleFunc("/touch", func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		query := req.URL.Query()
		x, y := query.Get("x"), query.Get("y")
		if x != "" && y != "" {
			scpicmd := fmt.Sprintf("KEYP %s %s", x, y)
			log.Print(scpicmd)
			cell.SendSCPI(scpicmd)
			w.WriteHeader(200)
		} else {
			fmt.Fprintf(w, "Coordination not given")
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	http.HandleFunc("/keyp", func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		err := req.ParseForm()
		if err != nil {
			fmt.Fprintf(w, "Form Parse error")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		value := req.FormValue("value")

		if value != "" {
			scpicmd := fmt.Sprintf("KEYP:%s", value)
			log.Print(scpicmd)
			cell.SendSCPI(scpicmd)

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

var (
	tmpl = template.Must(template.ParseFiles("template.html"))
	mu   = sync.RWMutex{}
)
