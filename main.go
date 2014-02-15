package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"github.com/advincze/auth"
	"github.com/gorilla/mux"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

	"time"
)

var router = mux.NewRouter()
var jobs = newPingJobs()
var pingCh chan *httpResponse

var session *mgo.Session
var db *mgo.Database

func init() {
	session, err := mgo.Dial(mgoURL)
	if err != nil {
		panic(err)
	}
	db = session.DB("healthcheck")

	pingCh = make(chan *httpResponse)

	auth.SetConstantAuth("foo", "bar")

	router.HandleFunc("/jobs", listJobs).Methods("GET")
	router.HandleFunc("/jobs/{id}", showJob).Methods("GET")
	router.HandleFunc("/jobs/{id}/_stop", stopJob).Methods("POST")
	router.HandleFunc("/jobs/{id}/_restart", restartJob).Methods("POST")
	router.HandleFunc("/jobs", createJob).Methods("POST")
	router.HandleFunc("/pings", searchPings).Methods("GET")
}

func main() {

	go startJobsPersistor()

	if err := http.ListenAndServe(":"+port, auth.Basic(router)); err != nil {
		panic(err)
	}
}

func startJobsPersistor() {
	for resp := range pingCh {
		err := db.C("ping").Insert(resp)
		if err != nil {
			panic(err)
		}
	}
}

func listJobs(w http.ResponseWriter, req *http.Request) {
	jobs.RLock()
	defer jobs.RUnlock()
	bytes, err := json.MarshalIndent(jobs.m, "", " ")
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
		URL    string
	}{}
	err = json.Unmarshal(body, &conf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	period, err := time.ParseDuration(conf.Period)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	job := newPingJob()
	job.StartAsync(pingCh, period, conf.URL)

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
	job.StartAsync(pingCh, period, job.URL)

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

	var pings []*httpResponse

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
