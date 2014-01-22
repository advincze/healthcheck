package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync"
	"github.com/gorilla/mux"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

	"time"
)

const portVar = "PORT"
const servicesVar = "VCAP_SERVICES"

var port string
var mgoUrl string
var router *mux.Router
var jobs = struct {
	sync.RWMutex
	m map[string]*PingJob
}{m: make(map[string]*PingJob)}
var pingCh chan *HttpResponse

func loadMongoDBUrlFromEnv() (string, error) {
	if s := os.Getenv(servicesVar); s != "" {

		var srv struct {
			UserProvied []struct {
				Credentials struct {
					Uri string
				}
			} `json:"user-provided"`
		}

		json.Unmarshal([]byte(s), &srv)

		return srv.UserProvied[0].Credentials.Uri, nil
	}
	return "", errors.New("MongoDB Config not found")
}

var session *mgo.Session
var db *mgo.Database

func init() {
	if port = os.Getenv(portVar); port == "" {
		port = "8080"
	}
	mgoUrl, err := loadMongoDBUrlFromEnv()
	if err != nil {
		mgoUrl = "localhost"
	}

	session, err := mgo.Dial(mgoUrl)
	if err != nil {
		panic(err)
	}
	db = session.DB("healthcheck")

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

	go startJobsPersistor()

	if err := http.ListenAndServe(":"+port, router); err != nil {
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
