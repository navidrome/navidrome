// Based on https://thoughtbot.com/blog/writing-a-server-sent-events-server-in-go
package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

var errWriteTimeOut = errors.New("write timeout")

const (
	keepAliveFrequency = 15 * time.Second
	writeTimeOut       = 5 * time.Second
)

var eventId uint32

type (
	message struct {
		ID    uint32
		Event string
		Data  string
	}
	messageChan chan message
	clientsChan chan client
	client      struct {
		id        string
		address   string
		username  string
		userAgent string
		channel   messageChan
		done      chan struct{}
	}
)

func (c client) String() string {
	return fmt.Sprintf("%s (%s - %s - %s)", c.id, c.username, c.address, c.userAgent)
}

type broker struct {
	// Events are pushed to this channel by the main events-gathering routine
	publish messageChan

	// New client connections
	subscribing clientsChan

	// Closed client connections
	unsubscribing clientsChan
}

func NewBroker() Broker {
	// Instantiate a broker
	broker := &broker{
		publish:       make(messageChan, 100),
		subscribing:   make(clientsChan, 1),
		unsubscribing: make(clientsChan, 1),
	}

	// Set it running - listening and broadcasting events
	go broker.listen()

	return broker
}

func (b *broker) SendMessage(evt Event) {
	msg := b.preparePackage(evt)
	log.Trace("Broker received new event", "event", msg)
	b.publish <- msg
}

func (b *broker) newEventID() uint32 {
	return atomic.AddUint32(&eventId, 1)
}

func (b *broker) preparePackage(event Event) message {
	pkg := message{}
	pkg.ID = b.newEventID()
	pkg.Event = event.EventName()
	data, _ := json.Marshal(event)
	pkg.Data = string(data)
	return pkg
}

// writeEvent Write to the ResponseWriter, Server Sent Events compatible
func writeEvent(w io.Writer, event message, timeout time.Duration) (n int, err error) {
	flusher, _ := w.(http.Flusher)
	complete := make(chan struct{}, 1)
	go func() {
		n, err = fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", event.ID, event.Event, event.Data)
		// Flush the data immediately instead of buffering it for later.
		flusher.Flush()
		complete <- struct{}{}
	}()
	select {
	case <-complete:
		return
	case <-time.After(timeout):
		return 0, errWriteTimeOut
	}
}

func (b *broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := request.UserFrom(ctx)

	// Make sure that the writer supports flushing.
	_, ok := w.(http.Flusher)
	if !ok {
		log.Error(w, "Streaming unsupported! Events cannot be sent to this client", "address", r.RemoteAddr,
			"userAgent", r.UserAgent(), "user", user.UserName)
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Each connection registers its own message channel with the Broker's connections registry
	c := b.subscribe(r)
	defer b.unsubscribe(c)
	log.Debug(ctx, "New broker client", "client", c.String())

	for {
		select {
		case event := <-c.channel:
			log.Trace(ctx, "Sending event to client", "event", event, "client", c.String())
			_, err := writeEvent(w, event, writeTimeOut)
			if err == errWriteTimeOut {
				return
			}
		case <-c.done:
			log.Trace(ctx, "Closing event stream connection", "client", c.String())
			return
		case <-ctx.Done():
			log.Trace(ctx, "Client closed the connection", "client", c.String())
			return
		}
	}
}

func (b *broker) subscribe(r *http.Request) client {
	user, _ := request.UserFrom(r.Context())
	id, _ := uuid.NewRandom()
	c := client{
		id:        id.String(),
		username:  user.UserName,
		address:   r.RemoteAddr,
		userAgent: r.UserAgent(),
		channel:   make(messageChan, 5),
		done:      make(chan struct{}, 1),
	}

	// Signal the broker that we have a new client
	b.subscribing <- c
	return c
}

func (b *broker) unsubscribe(c client) {
	b.unsubscribing <- c
}

func (b *broker) listen() {
	keepAlive := time.NewTicker(keepAliveFrequency)
	defer keepAlive.Stop()

	clients := map[client]struct{}{}

	for {
		select {
		case c := <-b.subscribing:
			// A new client has connected.
			// Register their message channel
			clients[c] = struct{}{}
			log.Debug("Client added to event broker", "numClients", len(clients), "newClient", c.String())

			// Send a serverStart event to new client
			c.channel <- b.preparePackage(&ServerStart{serverStart})

		case c := <-b.unsubscribing:
			// A client has detached and we want to
			// stop sending them messages.
			close(c.channel)
			delete(clients, c)
			log.Debug("Removed client from event broker", "numClients", len(clients), "client", c.String())

		case event := <-b.publish:
			// We got a new event from the outside!
			// Send event to all connected clients
			for c := range clients {
				log.Trace("Putting event on client's queue", "client", c.String(), "event", event)
				// Use non-blocking send. If cannot send, terminate the client's connection
				select {
				case c.channel <- event:
				default:
					log.Warn("Could not send event to client", "client", c.String(), "event", event)
				}
			}

		case ts := <-keepAlive.C:
			// Send a keep alive message every 15 seconds
			b.SendMessage(&KeepAlive{TS: ts.Unix()})
		}
	}
}

var serverStart time.Time

func init() {
	serverStart = time.Now()
}
