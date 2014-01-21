package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

const (
	PortVar = "PORT"
)

var router *mux.Router
var port string

func init() {
	log.SetOutput(os.Stdout)

	if port = os.Getenv(PortVar); port == "" {
		port = "8080"
	}

	router = mux.NewRouter()

	router.HandleFunc("/job", hello)
	router.HandleFunc("/start", startjob)
	router.HandleFunc("/stop", stopjob)
	router.HandleFunc("/job/:id", hello)
}

func main() {
	log.Printf("Listening at port %s\n", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		panic(err)
	}
}

func hello(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "hello world\n")
}

var job *Job = NewJob(time.Millisecond*200, func() {
	log.Printf("tick\n")
})

func startjob(w http.ResponseWriter, req *http.Request) {

	job.Start()
}

func stopjob(w http.ResponseWriter, req *http.Request) {
	job.Stop()
}
