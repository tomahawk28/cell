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
	"flag"
	"log"
	"text/template"
	"time"

	"net/http"

	"github.com/tomahawk28/cell/restful"
)

var (
	httpAddr        = flag.String("http", ":8040", "Listen Address")
	cellAdvisorAddr = flag.String("celladdr", "10.82.26.12", "CellAdvisor Address")
	numsport        = flag.Uint("numsport", 4, "The number of ports ")
	pollPeriod      = flag.Duration("poll", 10*time.Second, "Poll Period")
)

var (
	tmpl = template.Must(template.ParseFiles("template.html"))
)

func main() {

	flag.Parse()

	rtr := restful.BuildCellAdvisorRestfulAPI(int(*numsport), *cellAdvisorAddr, *pollPeriod)

	http.Handle("/api/", rtr)

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
