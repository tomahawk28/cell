package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
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

func main() {
	flag.Parse()
	cell := cell.NewCellAdvisor(*cellAdvisorAddr)

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
		if query["x"] != nil && query["y"] != nil {
			x, err := strconv.ParseFloat(query["x"][0], 32)
			if err != nil {
				fmt.Fprintf(w, "X is not float unit")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			y, err := strconv.ParseFloat(query["y"][0], 32)
			if err != nil {
				fmt.Fprintf(w, "Y is not float unit")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			scpicmd := fmt.Sprintf("KEYP %.0f %.0f", x, y)
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
	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}

var (
	tmpl = template.Must(template.ParseFiles("template.html"))
	mu   = sync.RWMutex{}
)
