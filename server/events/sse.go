// Based on https://thoughtbot.com/blog/writing-a-server-sent-events-server-in-go
package events

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model/request"
)

type Broker interface {
	http.Handler
	SendMessage(event Event)
}

var serverStart time.Time

type broker struct {
	// Events are pushed to this channel by the main events-gathering routine
	notifier chan []byte

	// New client connections
	newClients chan chan []byte

	// Closed client connections
	closingClients chan chan []byte

	// Client connections registry
	clients map[chan []byte]bool
}

func NewBroker() Broker {
	// Instantiate a broker
	broker := &broker{
		notifier:       make(chan []byte, 100),
		newClients:     make(chan chan []byte),
		closingClients: make(chan chan []byte),
		clients:        make(map[chan []byte]bool),
	}

	// Set it running - listening and broadcasting events
	go broker.listen()

	return broker
}

func (broker *broker) SendMessage(event Event) {
	data := broker.formatEvent(event)

	log.Trace("Broker received new event", "name", event.EventName(), "payload", string(data))
	broker.notifier <- data
}

func (broker *broker) formatEvent(event Event) []byte {
	pkg := struct {
		Event `json:"data"`
		Name  string `json:"name"`
	}{}
	pkg.Name = event.EventName()
	pkg.Event = event
	data, _ := json.Marshal(pkg)
	return data
}

func (broker *broker) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Make sure that the writer supports flushing.
	flusher, ok := rw.(http.Flusher)

	username, _ := request.UsernameFrom(req.Context())
	if !ok {
		log.Error(rw, "Streaming unsupported! Events cannot be sent to this client", "address", req.RemoteAddr,
			"userAgent", req.UserAgent(), "user", username)
		http.Error(rw, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache, no-transform")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	// Each connection registers its own message channel with the Broker's connections registry
	messageChan := make(chan []byte)

	// Signal the broker that we have a new connection
	broker.newClients <- messageChan

	// Remove this client from the map of connected clients
	// when this handler exits.
	defer func() {
		broker.closingClients <- messageChan
	}()

	// Listen to connection close and un-register messageChan
	notify := req.Context().Done()
	go func() {
		<-notify
		broker.closingClients <- messageChan
	}()

	for {
		// Write to the ResponseWriter
		// Server Sent Events compatible
		_, _ = fmt.Fprintf(rw, "data: %s\n\n", <-messageChan)

		// Flush the data immediately instead of buffering it for later.
		flusher.Flush()
	}
}

func (broker *broker) listen() {
	keepAlive := time.NewTicker(15 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case s := <-broker.newClients:
			// A new client has connected.
			// Register their message channel
			broker.clients[s] = true
			log.Debug("Client added to event broker", "numClients", len(broker.clients))

			// Send a serverStart event to new client
			s <- broker.formatEvent(&ServerStart{serverStart})

		case s := <-broker.closingClients:
			// A client has dettached and we want to
			// stop sending them messages.
			delete(broker.clients, s)
			log.Debug("Removed client from event broker", "numClients", len(broker.clients))

		case event := <-broker.notifier:
			// We got a new event from the outside!
			// Send event to all connected clients
			for clientMessageChan := range broker.clients {
				clientMessageChan <- event
			}

		case ts := <-keepAlive.C:
			// Send a keep alive packet every 15 seconds
			broker.SendMessage(&KeepAlive{TS: ts.Unix()})
		}
	}
}

func init() {
	serverStart = time.Now()
}
