package main

import (
	"net/http"
	"sync"
	"time"
)

type pingJobs struct {
	sync.RWMutex
	m map[string]*pingJob
}

func newPingJobs() *pingJobs {
	return &pingJobs{m: make(map[string]*pingJob)}
}

type pingJob struct {
	ticker *time.Ticker
	URL    string
	Period string
	Status string
}

func newPingJob() *pingJob {
	return &pingJob{Status: "stopped"}
}

type httpResponse struct {
	URL           string
	Timestamp     time.Time
	StatusCode    int
	ContentLength int64
	Duration      time.Duration
	Error         string
}

func ping(url string) *httpResponse {
	response := &httpResponse{URL: url, Timestamp: time.Now()}
	client := &http.Client{}
	resp, err := client.Get(url)
	if err != nil {
		response.Error = err.Error()
		return response
	}

	response.Duration = time.Now().Sub(response.Timestamp)
	response.StatusCode = resp.StatusCode
	response.ContentLength = resp.ContentLength
	return response
}

func (j *pingJob) StartAsync(c chan *httpResponse, period time.Duration, url string) {
	if j.Status != "stopped" {
		j.Stop()
	}
	j.ticker = time.NewTicker(period)
	j.Period = period.String()
	j.URL = url
	go func() {
		for _ = range j.ticker.C {
			c <- ping(url)
		}
	}()
	j.Status = "running"
}

func (j *pingJob) Stop() {
	if j.ticker != nil {
		j.ticker.Stop()
	}
	j.Status = "stopped"
}
