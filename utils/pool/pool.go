package pool

import (
	"time"

	"github.com/navidrome/navidrome/log"
)

type Executor func(workload interface{})

type Pool struct {
	name    string
	workers []worker
	exec    Executor
	queue   chan work // receives jobs to send to workers
	done    chan bool // when receives bool stops workers
	working bool
}

// TODO This hardcoded value will go away when the queue is persisted in disk
const bufferSize = 10000

func NewPool(name string, workerCount int, exec Executor) (*Pool, error) {
	p := &Pool{
		name:    name,
		exec:    exec,
		queue:   make(chan work, bufferSize),
		done:    make(chan bool),
		working: false,
	}

	for i := 0; i < workerCount; i++ {
		worker := worker{
			p:  p,
			id: i,
		}
		worker.Start()
		p.workers = append(p.workers, worker)
	}

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if len(p.queue) > 0 {
					log.Debug("Queue status", "poolName", p.name, "items", len(p.queue))
				} else {
					if p.working {
						log.Info("Queue is empty, all items processed", "poolName", p.name)
					}
					p.working = false
				}
			case <-p.done:
				close(p.queue)
				return
			}
		}
	}()

	return p, nil
}

func (p *Pool) Submit(workload interface{}) {
	p.working = true
	p.queue <- work{workload}
}

func (p *Pool) Stop() {
	p.done <- true
}

type work struct {
	workload interface{}
}

type worker struct {
	id int
	p  *Pool
}

// start worker
func (w *worker) Start() {
	go func() {
		for job := range w.p.queue {
			w.p.exec(job.workload) // do work
		}
	}()
}
