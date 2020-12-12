package events

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model/request"
	"github.com/google/uuid"
)

type Broker interface {
	http.Handler
	SendMessage(event Event)
}

const keepAliveFrequency = 15 * time.Second

var eventId uint32

type (
	message struct {
		ID    uint32
		Event string
		Data  string
	}
	messageChan chan message
	consumer    struct {
		id        string
		address   string
		username  string
		userAgent string
		channel   messageChan
	}
)

func (c consumer) String() string {
	return fmt.Sprintf("%s (%s, %s - %s)", c.id, c.username, c.address, c.userAgent)
}

func newEventID() uint32 {
	return atomic.AddUint32(&eventId, 1)
}

// Broker manages all SSE event transactions and contains a consumer pool
type broker struct {
	// consumers subscriber pool using which assigns a Distributed ID for each consumer
	consumers map[messageChan]consumer
	mtx       *sync.Mutex
	done      chan bool
}

// NewBroker returns a valid SSE broker
func NewBroker() Broker {
	b := &broker{
		consumers: make(map[messageChan]consumer),
		mtx:       new(sync.Mutex),
		done:      make(chan bool),
	}
	go b.keepAlive()
	return b
}

// subscribe returns a new broker consumer; listens to broker's events and generates a
// valid consumer ID
func (b *broker) subscribe(r *http.Request) messageChan {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	// Generate unique id for each consumer
	id, _ := uuid.NewRandom()
	user, _ := request.UserFrom(r.Context())
	c := consumer{
		id:        id.String(),
		username:  user.UserName,
		address:   r.RemoteAddr,
		userAgent: r.UserAgent(),
		channel:   make(messageChan),
	}

	cm := make(messageChan)
	b.consumers[cm] = c

	log.Debug("New broker consumer", "consumer", c.String(), "numConsumers", len(b.consumers))
	return cm
}

// unsubscribe removes a consumer from broker's pool
func (b *broker) unsubscribe(cm messageChan) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	c := b.consumers[cm]
	close(cm)
	delete(b.consumers, cm)
	log.Debug("Consumer removed", "consumer", c.String(), "numConsumers", len(b.consumers))
}

func (b *broker) preparePackage(event Event) message {
	pkg := message{}
	pkg.ID = newEventID()
	pkg.Event = event.EventName()
	data, _ := json.Marshal(event)
	pkg.Data = string(data)
	return pkg
}

// SendMessage issues a new event to either one or many consumers
func (b *broker) SendMessage(event Event) {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	msg := b.preparePackage(event)

	pubMsg := 0
	for s := range b.consumers {
		// Push to specific consumer
		s <- msg
		pubMsg++
		break
	}

	log.Trace("Sent message to all subscribers", "count", pubMsg)
}

// Close removes any channels leftovers from broker's pool
func (b *broker) Close() {
	b.done <- true
	for k := range b.consumers {
		close(k)
		delete(b.consumers, k)
	}
}

// ServeHTTP receives and attach a new broker subscription to every HTTP request
// using required streaming HTTP headers
func (b *broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := request.UserFrom(ctx)
	f, ok := w.(http.Flusher)
	if !ok {
		log.Error(w, "Streaming unsupported! Events cannot be sent to this consumer", "address", r.RemoteAddr,
			"userAgent", r.UserAgent(), "user", user.UserName)
		http.Error(w, "Streaming is not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Send initial message, with server startTime
	startMsg := b.preparePackage(&ServerStart{serverStart})
	_, _ = fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", startMsg.ID, startMsg.Event, startMsg.Data)
	f.Flush()

	// Create new consumer channel for stream events
	c := b.subscribe(r)
	defer b.unsubscribe(c)

	for {
		select {
		case msg := <-c:
			// Write to the ResponseWriter
			_, _ = fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", msg.ID, msg.Event, msg.Data)

			// Flush the data immediately instead of buffering it for later.
			f.Flush()
		case <-ctx.Done():
			return
		}
	}
}

// Send a keep alive message every 15 seconds
func (b *broker) keepAlive() {
	keepAlive := time.NewTicker(keepAliveFrequency)
	defer keepAlive.Stop()

	for {
		select {
		case ts := <-keepAlive.C:
			b.SendMessage(&KeepAlive{TS: ts.Unix()})
		case <-b.done:
			return
		}
	}
}

var serverStart time.Time

func init() {
	serverStart = time.Now()
}
