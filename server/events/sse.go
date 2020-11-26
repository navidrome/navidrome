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
		address   string
		username  string
		userAgent string
		channel   messageChan
	}
)

func (c client) String() string {
	return fmt.Sprintf("%s (%s - %s)", c.username, c.address, c.userAgent)
}

type broker struct {
	// Events are pushed to this channel by the main events-gathering routine
	notifier messageChan

	// New client connections
	newClients clientsChan

	// Closed client connections
	closingClients clientsChan

	// Client connections registry
	clients map[client]bool
}

func NewBroker() Broker {
	// Instantiate a broker
	broker := &broker{
		notifier:       make(messageChan, 100),
		newClients:     make(clientsChan),
		closingClients: make(clientsChan),
		clients:        make(map[client]bool),
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

func (broker *broker) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	// Make sure that the writer supports flushing.
	flusher, ok := rw.(http.Flusher)
	user, _ := request.UserFrom(ctx)
	if !ok {
		log.Error(rw, "Streaming unsupported! Events cannot be sent to this client", "address", req.RemoteAddr,
			"userAgent", req.UserAgent(), "user", user.UserName)
		http.Error(rw, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache, no-transform")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	// Each connection registers its own message channel with the Broker's connections registry
	client := client{
		username:  user.UserName,
		address:   req.RemoteAddr,
		userAgent: req.UserAgent(),
		channel:   make(messageChan),
	}

	// Signal the broker that we have a new client
	broker.newClients <- client

	log.Debug(ctx, "New broker client", "client", client.String())

	// Remove this client from the map of connected clients
	// when this handler exits.
	defer func() {
		broker.closingClients <- client
	}()

	// Listen to client close and un-register messageChan
	notify := ctx.Done()
	go func() {
		<-notify
		broker.closingClients <- client
	}()

	for {
		// Write to the ResponseWriter
		// Server Sent Events compatible
		event := <-client.channel
		log.Trace(ctx, "Sending event to client", "event", event, "client", client.String())
		_, _ = fmt.Fprintf(rw, "id: %d\nevent: %s\ndata: %s\n\n", event.ID, event.Event, event.Data)

		// Flush the data immediately instead of buffering it for later.
		flusher.Flush()
	}
}

func (broker *broker) listen() {
	keepAlive := time.NewTicker(keepAliveFrequency)
	defer keepAlive.Stop()

	for {
		select {
		case s := <-broker.newClients:
			// A new client has connected.
			// Register their message channel
			broker.clients[s] = true
			log.Debug("Client added to event broker", "numClients", len(broker.clients), "newClient", s.String())

			// Send a serverStart event to new client
			s.channel <- broker.preparePackage(&ServerStart{serverStart})

		case s := <-broker.closingClients:
			// A client has detached and we want to
			// stop sending them messages.
			delete(broker.clients, s)
			log.Debug("Removed client from event broker", "numClients", len(broker.clients), "client", s.String())

		case event := <-broker.notifier:
			// We got a new event from the outside!
			// Send event to all connected clients
			for client := range broker.clients {
				log.Trace("Putting event on client's queue", "client", client.String(), "event", event)
				client.channel <- event
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
