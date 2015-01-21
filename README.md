# JDSU CellAdvisor RESTful API  
JDSU CellAdvisor API by Go Language

Usage 
------
```go
func main(){
 // BuildCellAdvisorRestfulAPI got the number of TCP connections for API, 
 // CellAdvisor IP and, Heartbeat cheking period
 rtr := restful.BuildCellAdvisorRestfulAPI("4", "192.168.0.1", time.Second*10)
 http.Handle("/api/", rtr)
 log.Fatal(http.ListenAndServe(":80", nil))
//now you could access 
// SCPI command: http://{celladvisorIP}:{port}/api/scpi/{keyp|youch}
// Screen capture http://{celladvisorIP}:{port}/api/screen/{refresh_screen|screen}

}
