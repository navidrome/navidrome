// Based on https://thoughtbot.com/blog/writing-a-server-sent-events-server-in-go
package events

import (
	"encoding/json"
	"fmt"
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

const keepAliveFrequency = 15 * time.Second

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
	}
)

func (c client) String() string {
	return fmt.Sprintf("%s (%s - %s - %s)", c.id, c.username, c.address, c.userAgent)
}

type broker struct {
	// Events are pushed to this channel by the main events-gathering routine
	notifier messageChan

	// New client connections
	newClients clientsChan

	// Closed client connections
	closingClients clientsChan

	// Client connections registry
	clients map[client]struct{}
}

func NewBroker() Broker {
	// Instantiate a broker
	broker := &broker{
		notifier:       make(messageChan, 100),
		newClients:     make(clientsChan, 1),
		closingClients: make(clientsChan, 1),
		clients:        make(map[client]struct{}),
	}

	// Set it running - listening and broadcasting events
	go broker.listen()

	return broker
}

func (broker *broker) SendMessage(evt Event) {
	msg := broker.preparePackage(evt)
	log.Trace("Broker received new event", "event", msg)
	broker.notifier <- msg
}

func (broker *broker) preparePackage(event Event) message {
	pkg := message{}
	pkg.ID = atomic.AddUint32(&eventId, 1)
	pkg.Event = event.EventName()
	data, _ := json.Marshal(event)
	pkg.Data = string(data)
	return pkg
}

func (broker *broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Make sure that the writer supports flushing.
	flusher, ok := w.(http.Flusher)
	user, _ := request.UserFrom(ctx)
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
	client := broker.subscribe(r)
	defer broker.unsubscribe(client)
	log.Debug(ctx, "New broker client", "client", client.String())

	for {
		select {
		case event := <-client.channel:
			// Write to the ResponseWriter
			// Server Sent Events compatible
			log.Trace(ctx, "Sending event to client", "event", event, "client", client.String())
			_, _ = fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", event.ID, event.Event, event.Data)

			// Flush the data immediately instead of buffering it for later.
			flusher.Flush()
		case <-ctx.Done():
			log.Trace(ctx, "Closing event stream connection", "client", client.String())
			return
		}
	}
}

func (broker *broker) subscribe(r *http.Request) client {
	user, _ := request.UserFrom(r.Context())
	id, _ := uuid.NewRandom()
	client := client{
		id:        id.String(),
		username:  user.UserName,
		address:   r.RemoteAddr,
		userAgent: r.UserAgent(),
		channel:   make(messageChan, 5),
	}

	// Signal the broker that we have a new client
	broker.newClients <- client
	return client
}

func (broker *broker) unsubscribe(c client) {
	broker.closingClients <- c
}

func (broker *broker) listen() {
	keepAlive := time.NewTicker(keepAliveFrequency)
	defer keepAlive.Stop()

	for {
		select {
		case s := <-broker.newClients:
			// A new client has connected.
			// Register their message channel
			broker.clients[s] = struct{}{}
			log.Debug("Client added to event broker", "numClients", len(broker.clients), "newClient", s.String())

			// Send a serverStart event to new client
			s.channel <- broker.preparePackage(&ServerStart{serverStart})

		case s := <-broker.closingClients:
			// A client has detached and we want to
			// stop sending them messages.
			close(s.channel)
			delete(broker.clients, s)
			log.Debug("Removed client from event broker", "numClients", len(broker.clients), "client", s.String())

		case event := <-broker.notifier:
			// We got a new event from the outside!
			// Send event to all connected clients
			for client := range broker.clients {
				log.Trace("Putting event on client's queue", "client", client.String(), "event", event)
				// Use non-blocking send
				select {
				case client.channel <- event:
				default:
					log.Warn("Could not send message to client", "client", client.String(), "event", event)
				}
			}

		case ts := <-keepAlive.C:
			// Send a keep alive message every 15 seconds
			broker.SendMessage(&KeepAlive{TS: ts.Unix()})
		}
	}
}

var serverStart time.Time

func init() {
	serverStart = time.Now()
}
