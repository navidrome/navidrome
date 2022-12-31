// Based on https://thoughtbot.com/blog/writing-a-server-sent-events-server-in-go
package events

import (
	"context"
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
	"github.com/navidrome/navidrome/utils/singleton"
)

type Broker interface {
	http.Handler
	SendMessage(ctx context.Context, event Event)
}

const (
	keepAliveFrequency = 15 * time.Second
	writeTimeOut       = 5 * time.Second
)

type (
	message struct {
		id        uint32
		event     string
		data      string
		senderCtx context.Context
	}
	messageChan chan message
	clientsChan chan client
	client      struct {
		id             string
		address        string
		username       string
		userAgent      string
		clientUniqueId string
		diode          *diode
	}
)

func (c client) String() string {
	if log.CurrentLevel() >= log.LevelTrace {
		return fmt.Sprintf("%s (%s - %s - %s - %s)", c.id, c.username, c.address, c.clientUniqueId, c.userAgent)
	}
	return fmt.Sprintf("%s (%s - %s - %s)", c.id, c.username, c.address, c.clientUniqueId)
}

type broker struct {
	// Events are pushed to this channel by the main events-gathering routine
	publish messageChan

	// New client connections
	subscribing clientsChan

	// Closed client connections
	unsubscribing clientsChan
}

func GetBroker() Broker {
	return singleton.GetInstance(func() *broker {
		// Instantiate a broker
		broker := &broker{
			publish:       make(messageChan, 2),
			subscribing:   make(clientsChan, 1),
			unsubscribing: make(clientsChan, 1),
		}

		// Set it running - listening and broadcasting events
		go broker.listen()
		return broker
	})
}

func (b *broker) SendMessage(ctx context.Context, evt Event) {
	msg := b.prepareMessage(ctx, evt)
	log.Trace("Broker received new event", "event", msg)
	b.publish <- msg
}

func (b *broker) prepareMessage(ctx context.Context, event Event) message {
	msg := message{}
	msg.data = event.Data(event)
	msg.event = event.Name(event)
	msg.senderCtx = ctx
	return msg
}

var errWriteTimeOut = errors.New("write timeout")

// writeEvent Write to the ResponseWriter, Server Sent Events compatible, and sends it
// right away, by flushing the writer (if it is a Flusher). It waits for the message to be flushed
// or times out after the specified timeout
func writeEvent(w io.Writer, event message, timeout time.Duration) (err error) {
	complete := make(chan struct{}, 1)
	flusher, ok := w.(http.Flusher)
	if ok {
		go func() {
			_, err = fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", event.id, event.event, event.data)
			// Flush the data immediately instead of buffering it for later.
			flusher.Flush()
			complete <- struct{}{}
		}()
	} else {
		complete <- struct{}{}
	}
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
		if err := writeEvent(w, *event, writeTimeOut); errors.Is(err, errWriteTimeOut) {
			log.Debug(ctx, "Timeout sending event to client", "event", *event, "client", c.String())
			return
		}
	}
}

func (b *broker) subscribe(r *http.Request) client {
	ctx := r.Context()
	user, _ := request.UserFrom(ctx)
	clientUniqueId, _ := request.ClientUniqueIdFrom(ctx)
	c := client{
		id:             uuid.NewString(),
		username:       user.UserName,
		address:        r.RemoteAddr,
		userAgent:      r.UserAgent(),
		clientUniqueId: clientUniqueId,
	}
	c.diode = newDiode(ctx, 1024, diodes.AlertFunc(func(missed int) {
		log.Debug("Dropped SSE events", "client", c.String(), "missed", missed)
	}))

	// Signal the broker that we have a new client
	b.subscribing <- c
	return c
}

func (b *broker) unsubscribe(c client) {
	b.unsubscribing <- c
}

func (b *broker) shouldSend(msg message, c client) bool {
	clientUniqueId, originatedFromClient := request.ClientUniqueIdFrom(msg.senderCtx)
	if !originatedFromClient {
		return true
	}
	if c.clientUniqueId == clientUniqueId {
		return false
	}
	if username, ok := request.UsernameFrom(msg.senderCtx); ok {
		return username == c.username
	}
	return true
}

func (b *broker) listen() {
	keepAlive := time.NewTicker(keepAliveFrequency)
	defer keepAlive.Stop()

	clients := map[client]struct{}{}
	var eventId uint32

	getNextEventId := func() uint32 {
		eventId++
		return eventId
	}

	for {
		select {
		case c := <-b.subscribing:
			// A new client has connected.
			// Register their message channel
			clients[c] = struct{}{}
			log.Debug("Client added to event broker", "numClients", len(clients), "newClient", c.String())

			// Send a serverStart event to new client
			msg := b.prepareMessage(context.Background(),
				&ServerStart{StartTime: consts.ServerStart, Version: consts.Version})
			c.diode.put(msg)

		case c := <-b.unsubscribing:
			// A client has detached, and we want to
			// stop sending them messages.
			delete(clients, c)
			log.Debug("Removed client from event broker", "numClients", len(clients), "client", c.String())

		case msg := <-b.publish:
			msg.id = getNextEventId()
			log.Trace("Got new published event", "event", msg)
			// We got a new event from the outside!
			// Send event to all connected clients
			for c := range clients {
				if b.shouldSend(msg, c) {
					log.Trace("Putting event on client's queue", "client", c.String(), "event", msg)
					c.diode.put(msg)
				}
			}

		case ts := <-keepAlive.C:
			// Send a keep alive message every 15 seconds to all connected clients
			if len(clients) == 0 {
				continue
			}
			msg := b.prepareMessage(context.Background(), &KeepAlive{TS: ts.Unix()})
			msg.id = getNextEventId()
			for c := range clients {
				log.Trace("Putting a keepalive event on client's queue", "client", c.String(), "event", msg)
				c.diode.put(msg)
			}
		}
	}
}
