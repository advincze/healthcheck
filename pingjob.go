package main

import (
	"net/http"
	"time"
)

type PingJob struct {
	ticker *time.Ticker
	Url    string
	Period string
	Status string
}

func NewPingJob() *PingJob {
	return &PingJob{Status: "stopped"}
}

type HttpResponse struct {
	Url           string
	Timestamp     time.Time
	StatusCode    int
	ContentLength int64
	Duration      time.Duration
	Error         string
}

func ping(url string) *HttpResponse {
	response := &HttpResponse{Url: url, Timestamp: time.Now()}
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

func (j *PingJob) StartAsync(c chan *HttpResponse, period time.Duration, url string) {
	if j.Status != "stopped" {
		j.Stop()
	}
	j.ticker = time.NewTicker(period)
	j.Period = period.String()
	j.Url = url
	go func() {
		for _ = range j.ticker.C {
			c <- ping(url)
		}
	}()
	j.Status = "running"
}

func (j *PingJob) Stop() {
	if j.ticker != nil {
		j.ticker.Stop()
	}
	j.Status = "stopped"
}
