package pool

import (
	"time"

	"github.com/deluan/navidrome/log"
)

type Executor func(workload interface{})

type Pool struct {
	name          string
	item          interface{}
	workers       []worker
	exec          Executor
	logTicker     *time.Ticker
	workerChannel chan chan work
	queue         chan work // receives jobs to send to workers
	end           chan bool // when receives bool stops workers
	//queue *dque.DQue
}

// TODO This hardcoded value will go away when the queue is persisted in disk
const bufferSize = 10000

func NewPool(name string, workerCount int, item interface{}, exec Executor) (*Pool, error) {
	p := &Pool{
		name:  name,
		item:  item,
		exec:  exec,
		queue: make(chan work, bufferSize),
		end:   make(chan bool),
	}

	//q, err := dque.NewOrOpen(name, filepath.Join(conf.Server.DataFolder, "queues", name), 50, p.itemBuilder)
	//if err != nil {
	//	return nil, err
	//}
	//p.queue = q

	p.workerChannel = make(chan chan work)
	for i := 0; i < workerCount; i++ {
		worker := worker{
			p:             p,
			id:            i,
			channel:       make(chan work),
			workerChannel: p.workerChannel,
			end:           make(chan bool)}
		worker.Start()
		p.workers = append(p.workers, worker)
	}

	// start pool
	go func() {
		p.logTicker = time.NewTicker(10 * time.Second)
		running := false
		for {
			select {
			case <-p.logTicker.C:
				if len(p.queue) > 0 {
					log.Debug("Queue status", "pool", p.name, "items", len(p.queue))
				} else {
					if running {
						log.Info("Finished draining queue", "pool", p.name)
					}
					running = false
				}
			case <-p.end:
				for _, w := range p.workers {
					w.Stop() // stop worker
				}
				return
			case work := <-p.queue:
				running = true
				worker := <-p.workerChannel // wait for available channel
				worker <- work              // dispatch work to worker
			}
		}
	}()
	return p, nil
}

func (p *Pool) Submit(workload interface{}) {
	p.queue <- work{workload}
}

//func (p *Pool) itemBuilder() interface{} {
//	t := reflect.TypeOf(p.item)
//	return reflect.New(t).Interface()
//}
//
type work struct {
	workload interface{}
}

type worker struct {
	id            int
	p             *Pool
	workerChannel chan chan work // used to communicate between dispatcher and workers
	channel       chan work
	end           chan bool
}

// start worker
func (w *worker) Start() {
	go func() {
		for {
			w.workerChannel <- w.channel // when the worker is available place channel in queue
			select {
			case job := <-w.channel: // worker has received job
				w.p.exec(job.workload) // do work
			case <-w.end:
				return
			}
		}
	}()
}

// end worker
func (w *worker) Stop() {
	w.end <- true
}
