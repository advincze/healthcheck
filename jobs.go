package main

import (
	"sync"

	"time"
)

type Job struct {
	*time.Ticker
	Id     string
	period time.Duration
	f      func()
}

func NewJob(id string, period time.Duration, f func()) *Job {
	return &Job{
		Id:     id,
		period: period,
		f:      f,
	}
}

func (j *Job) Start() {
	j.Ticker = time.NewTicker(j.period)
	go func() {
		for _ = range j.Ticker.C {
			j.f()
		}
	}()
}

func (j *Job) Stop() {
	if j.Ticker != nil {
		j.Ticker.Stop()
	}
}

type jobList struct {
	sync.RWMutex
	m map[string]*Job
}

var jobs = &jobList{m: make(map[string]*Job, 100)}

func (j *jobList) Add(job *Job) {
	j.Lock()
	defer j.Unlock()

	j.m[job.Id] = job
}

func (j *jobList) Remove(id string) {
	j.Lock()
	defer j.Unlock()

	delete(j.m, id)
}

func (j *jobList) List() []*Job {
	j.RLock()
	defer j.RUnlock()

	jj := make([]*Job, 0, len(j.m))
	for _, job := range j.m {
		jj = append(jj, job)
	}
	return jj
}
