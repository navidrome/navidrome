// Based on https://thoughtbot.com/blog/writing-a-server-sent-events-server-in-go
package events

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"code.cloudfoundry.org/go-diodes"
	"github.com/google/uuid"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/request"
)

type Broker interface {
	http.Handler
	SendMessage(event Event)
}

const (
	keepAliveFrequency = 15 * time.Second
	writeTimeOut       = 5 * time.Second
)

var (
	errWriteTimeOut = errors.New("write timeout")
)

type (
	message struct {
		Data string
	}
	messageChan chan message
	clientsChan chan client
	client      struct {
		id        string
		address   string
		username  string
		userAgent string
		diode     *diode
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
	msg := b.prepareMessage(evt)
	log.Trace("Broker received new event", "event", msg)
	b.publish <- msg
}

func (b *broker) prepareMessage(event Event) message {
	msg := message{}
	msg.Data = event.Prepare(event)
	return msg
}

// writeEvent Write to the ResponseWriter, Server Sent Events compatible
func writeEvent(w io.Writer, event message, timeout time.Duration) (err error) {
	flusher, _ := w.(http.Flusher)
	complete := make(chan struct{}, 1)
	go func() {
		_, err = fmt.Fprintf(w, "data: %s\n\n", event.Data)
		// Flush the data immediately instead of buffering it for later.
		flusher.Flush()
		complete <- struct{}{}
	}()
	select {
	case <-complete:
		return
	case <-time.After(timeout):
		return errWriteTimeOut
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
	// Tells Nginx to not buffer this response. See https://stackoverflow.com/a/33414096
	w.Header().Set("X-Accel-Buffering", "no")

	// Each connection registers its own message channel with the Broker's connections registry
	c := b.subscribe(r)
	defer b.unsubscribe(c)
	log.Debug(ctx, "New broker client", "client", c.String())

	for {
		event := c.diode.next()
		if event == nil {
			log.Trace(ctx, "Client closed the EventStream connection", "client", c.String())
			return
		}
		log.Trace(ctx, "Sending event to client", "event", *event, "client", c.String())
		if err := writeEvent(w, *event, writeTimeOut); err == errWriteTimeOut {
			log.Debug(ctx, "Timeout sending event to client", "event", *event, "client", c.String())
			return
		}
	}
}

func (b *broker) subscribe(r *http.Request) client {
	user, _ := request.UserFrom(r.Context())
	c := client{
		id:        uuid.NewString(),
		username:  user.UserName,
		address:   r.RemoteAddr,
		userAgent: r.UserAgent(),
	}
	c.diode = newDiode(r.Context(), 1024, diodes.AlertFunc(func(missed int) {
		log.Trace("Dropped SSE events", "client", c.String(), "missed", missed)
	}))

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
			c.diode.set(b.prepareMessage(&ServerStart{StartTime: consts.ServerStart}))

		case c := <-b.unsubscribing:
			// A client has detached and we want to
			// stop sending them messages.
			delete(clients, c)
			log.Debug("Removed client from event broker", "numClients", len(clients), "client", c.String())

		case event := <-b.publish:
			// We got a new event from the outside!
			// Send event to all connected clients
			for c := range clients {
				log.Trace("Putting event on client's queue", "client", c.String(), "event", event)
				c.diode.set(event)
			}

		case ts := <-keepAlive.C:
			// Send a keep alive message every 15 seconds
			b.SendMessage(&KeepAlive{TS: ts.Unix()})
		}
	}
}
