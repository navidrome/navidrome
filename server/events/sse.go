// Package events based on https://thoughtbot.com/blog/writing-a-server-sent-events-server-in-go
package events

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/pl"
	"github.com/navidrome/navidrome/utils/singleton"
)

type Broker interface {
	http.Handler
	SendMessage(ctx context.Context, event Event)
	SendBroadcastMessage(ctx context.Context, event Event)
}

const (
	keepAliveFrequency = 15 * time.Second
	writeTimeOut       = 5 * time.Second
	bufferSize         = 1
)

type (
	message struct {
		id        uint64
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
		displayString  string
		msgC           chan message
	}
)

func (c client) String() string {
	return c.displayString
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

func (b *broker) SendBroadcastMessage(ctx context.Context, evt Event) {
	ctx = broadcastToAll(ctx)
	b.SendMessage(ctx, evt)
}

func (b *broker) SendMessage(ctx context.Context, evt Event) {
	msg := b.prepareMessage(ctx, evt)
	log.Trace("Broker received new event", "type", msg.event, "data", msg.data)
	b.publish <- msg
}

func (b *broker) prepareMessage(ctx context.Context, event Event) message {
	msg := message{}
	msg.data = event.Data(event)
	msg.event = event.Name(event)
	msg.senderCtx = ctx
	return msg
}

// writeEvent writes a message to the given io.Writer, formatted as a Server-Sent Event.
// If the writer is a http.Flusher, it flushes the data immediately instead of buffering it.
func writeEvent(ctx context.Context, w io.Writer, event message, timeout time.Duration) error {
	if err := setWriteTimeout(w, timeout); err != nil {
		log.Debug(ctx, "Error setting write timeout", err)
	}

	_, err := fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", event.id, event.event, event.data)
	if err != nil {
		return err
	}

	// If the writer is a http.Flusher, flush the data immediately.
	if flusher, ok := w.(http.Flusher); ok && flusher != nil {
		flusher.Flush()
	}
	return nil
}

func setWriteTimeout(rw io.Writer, timeout time.Duration) error {
	for {
		switch t := rw.(type) {
		case interface{ SetWriteDeadline(time.Time) error }:
			return t.SetWriteDeadline(time.Now().Add(timeout))
		case interface{ Unwrap() http.ResponseWriter }:
			rw = t.Unwrap()
		default:
			return fmt.Errorf("%T - %w", rw, http.ErrNotSupported)
		}
	}
}

func (b *broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := request.UserFrom(ctx)

	// Make sure that the writer supports flushing.
	_, ok := w.(http.Flusher)
	if !ok {
		log.Error(r, "Streaming unsupported! Events cannot be sent to this client", "address", r.RemoteAddr,
			"userAgent", r.UserAgent(), "user", user.UserName)
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	// Tells Nginx to not buffer this response. See https://stackoverflow.com/a/33414096
	w.Header().Set("X-Accel-Buffering", "no")

	// Each connection registers its own message channel with the Broker's connections registry
	c := b.subscribe(r)
	defer b.unsubscribe(c)
	log.Debug(ctx, "Started new EventStream connection", "client", c.String())

	for event := range pl.ReadOrDone(ctx, c.msgC) {
		log.Trace(ctx, "Sending event to client", "event", event, "client", c.String())
		err := writeEvent(ctx, w, event, writeTimeOut)
		if err != nil {
			log.Debug(ctx, "Error sending event to client. Closing connection", "event", event, "client", c.String(), err)
			return
		}
	}
	log.Trace(ctx, "Client EventStream connection closed", "client", c.String())
}

func (b *broker) subscribe(r *http.Request) client {
	ctx := r.Context()
	user, _ := request.UserFrom(ctx)
	clientUniqueId, _ := request.ClientUniqueIdFrom(ctx)
	c := client{
		id:             id.NewRandom(),
		username:       user.UserName,
		address:        r.RemoteAddr,
		userAgent:      r.UserAgent(),
		clientUniqueId: clientUniqueId,
	}
	if log.IsGreaterOrEqualTo(log.LevelTrace) {
		c.displayString = fmt.Sprintf("%s (%s - %s - %s - %s)", c.id, c.username, c.address, c.clientUniqueId, c.userAgent)
	} else {
		c.displayString = fmt.Sprintf("%s (%s - %s - %s)", c.id, c.username, c.address, c.clientUniqueId)
	}

	c.msgC = make(chan message, bufferSize)

	// Signal the broker that we have a new client
	b.subscribing <- c
	return c
}

func (b *broker) unsubscribe(c client) {
	b.unsubscribing <- c
}

func (b *broker) shouldSend(msg message, c client) bool {
	if broadcastToAll, ok := msg.senderCtx.Value(broadcastToAllKey).(bool); ok && broadcastToAll {
		return true
	}
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
	var eventId uint64

	getNextEventId := func() uint64 {
		eventId++
		return eventId
	}

	for {
		select {
		case c := <-b.subscribing:
			// A new client has connected.
			// Register their message channel
			clients[c] = struct{}{}
			log.Debug("Client added to EventStream broker", "numActiveClients", len(clients), "newClient", c.String())

			// Send a serverStart event to new client
			msg := b.prepareMessage(context.Background(),
				&ServerStart{StartTime: consts.ServerStart, Version: consts.Version})
			sendOrDrop(c, msg)

		case c := <-b.unsubscribing:
			// A client has detached, and we want to
			// stop sending them messages.
			close(c.msgC)
			delete(clients, c)
			log.Debug("Removed client from EventStream broker", "numActiveClients", len(clients), "client", c.String())

		case msg := <-b.publish:
			msg.id = getNextEventId()
			log.Trace("Got new published event", "event", msg)
			// We got a new event from the outside!
			// Send event to all connected clients
			for c := range clients {
				if b.shouldSend(msg, c) {
					log.Trace("Putting event on client's queue", "client", c.String(), "event", msg)
					sendOrDrop(c, msg)
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
				sendOrDrop(c, msg)
			}
		}
	}
}

func sendOrDrop(client client, msg message) {
	select {
	case client.msgC <- msg:
	default:
		if log.IsGreaterOrEqualTo(log.LevelTrace) {
			log.Trace("Event dropped because client's channel is full", "event", msg, "client", client.String())
		}
	}
}

func NoopBroker() Broker {
	return noopBroker{}
}

type noopBroker struct {
	http.Handler
}

func (b noopBroker) SendBroadcastMessage(context.Context, Event) {}

func (noopBroker) SendMessage(context.Context, Event) {}
