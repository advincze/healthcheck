package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"github.com/gorilla/mux"
	"labix.org/v2/mgo/bson"

	"time"
)

var router *mux.Router
var jobs = struct {
	sync.RWMutex
	m map[string]*PingJob
}{m: make(map[string]*PingJob)}
var pingCh chan *HttpResponse

func init() {
	pingCh = make(chan *HttpResponse)

	router = mux.NewRouter()
	router.HandleFunc("/jobs", basicAuth(listJobs, auth)).Methods("GET")
	router.HandleFunc("/jobs/{id}", basicAuth(showJob, auth)).Methods("GET")
	router.HandleFunc("/jobs/{id}/_stop", basicAuth(stopJob, auth)).Methods("POST")
	router.HandleFunc("/jobs/{id}/_stop", basicAuth(restartJob, auth)).Methods("POST")
	router.HandleFunc("/jobs", basicAuth(createJob, auth)).Methods("POST")
	router.HandleFunc("/pings", basicAuth(searchPings, auth)).Methods("GET")
}

func main() {

	go startJobs()

	if err := http.ListenAndServe(":8080", router); err != nil {
		panic(err)
	}
}

func startJobs() {
	for resp := range pingCh {
		err := db.C("ping").Insert(resp)
		if err != nil {
			panic(err)
		}
	}
}

func listJobs(w http.ResponseWriter, req *http.Request) {
	bytes, err := json.MarshalIndent(jobs, "", " ")
	if err != nil {
		panic(err)
	}

	fmt.Fprintf(w, "%s", bytes)
}

func createJob(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	var conf = struct {
		Period string
		Url    string
	}{}
	err = json.Unmarshal(body, &conf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	period, err := time.ParseDuration(conf.Period)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	job := NewPingJob()
	job.StartAsync(pingCh, period, conf.Url)

	key := fmt.Sprintf("%d", time.Now().UnixNano())
	jobs.Lock()
	jobs.m[key] = job
	jobs.Unlock()

	bytes, err := json.MarshalIndent(job, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(w, "%s", bytes)
}

func showJob(w http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]
	if id == "" {
		http.Error(w, "jobid expected", http.StatusBadRequest)
	}

	jobs.RLock()
	job, ok := jobs.m[id]
	jobs.RUnlock()

	if !ok {
		http.Error(w, "could not find jobid", http.StatusBadRequest)
	}

	bytes, err := json.MarshalIndent(job, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(w, "%s", bytes)
}

func stopJob(w http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]
	if id == "" {
		http.Error(w, "jobid expected", http.StatusBadRequest)
	}

	jobs.RLock()
	job, ok := jobs.m[id]
	jobs.RUnlock()

	if !ok {
		http.Error(w, "could not find jobid", http.StatusBadRequest)
	}

	job.Stop()

	bytes, err := json.MarshalIndent(job, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(w, "%s", bytes)
}

func restartJob(w http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]
	if id == "" {
		http.Error(w, "jobid expected", http.StatusBadRequest)
	}

	jobs.RLock()
	job, ok := jobs.m[id]
	jobs.RUnlock()

	if !ok {
		http.Error(w, "could not find jobid", http.StatusBadRequest)
	}

	period, _ := time.ParseDuration(job.Period)
	job.StartAsync(pingCh, period, job.Url)

	bytes, err := json.MarshalIndent(job, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(w, "%s", bytes)
}

func killJob(w http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]
	if id == "" {
		http.Error(w, "jobid expected", http.StatusBadRequest)
	}

	jobs.RLock()
	job, ok := jobs.m[id]
	jobs.RUnlock()

	if !ok {
		http.Error(w, "could not find jobid", http.StatusBadRequest)
	}

	job.Stop()
	jobs.Lock()
	delete(jobs.m, id)
	jobs.Unlock()

	bytes, err := json.MarshalIndent(job, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(w, "%s", bytes)
}

func searchPings(w http.ResponseWriter, req *http.Request) {

	query := bson.M{}

	url := req.URL.Query().Get("url")
	if url != "" {
		query["url"] = url
	}

	last := req.URL.Query().Get("last")
	if last != "" {
		dur, err := time.ParseDuration(last)
		if err == nil {
			query["timestamp"] = bson.M{"$gte": time.Now().Add(-dur)}
		}
	}

	statuscodeStr := req.URL.Query().Get("statuscode")
	if statuscodeStr != "" {
		sc, err := strconv.Atoi(statuscodeStr)
		if err == nil {
			query["statuscode"] = sc
		}
	}

	var pings []*HttpResponse

	err := db.C("ping").Find(query).All(&pings)
	if err != nil {
		panic(err)
	}

	bytes, err := json.MarshalIndent(pings, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(w, "%s", bytes)
}
