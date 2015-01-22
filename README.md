# JDSU CellAdvisor RESTful API  
JDSU CellAdvisor API by Go Language

[![GoDoc](https://godoc.org/github.com/tomahawk28/cell/restful?status.svg)](https://godoc.org/github.com/tomahawk28/cell/restful)

Usage 
------
```go
func main(){

 // BuildCellAdvisorRestfulAPI functions get argumets 
 // 1. RESTFul prefix URL string 
 // 2. The number of TCP connections for API
 // 3. CellAdvisor IP 
 // 4. Heartbeat cheking period
 // for example, 

 rtr := restful.BuildCellAdvisorRestfulAPI("", 4, "192.168.0.1", time.Second*10)
 http.Handle("/api/", rtr)
 log.Fatal(http.ListenAndServe(":80", nil))

// Now you could access 
// SCPI command: http://{celladvisorIP}:{port}/api/scpi/{keyp|youch}
// Screen capture http://{celladvisorIP}:{port}/api/screen/{refresh_screen|screen}

}
```

Maintainer
------
Ji-hyuk.Bok@jdsu.com

License
-----
MIT
