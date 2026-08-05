package events

import (
	"context"
	"net/http"
	"time"

	coreradio "github.com/navidrome/navidrome/core/radio"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Events", func() {
	Describe("Event", func() {
		type TestEvent struct {
			baseEvent
			Test string
		}

		It("marshals Event to JSON", func() {
			testEvent := TestEvent{Test: "some data"}
			data := testEvent.Data(&testEvent)
			Expect(data).To(Equal(`{"Test":"some data"}`))
			name := testEvent.Name(&testEvent)
			Expect(name).To(Equal("testEvent"))
		})
	})

	Describe("RadioNowPlaying", func() {
		It("serialises with a stable event name and payload", func() {
			updatedAt := time.Date(2026, time.June, 30, 12, 34, 56, 0, time.UTC)
			event := &RadioNowPlaying{
				RadioID:   "rd-1",
				Title:     "Artist - Title",
				UpdatedAt: updatedAt,
			}

			Expect(event.Name(event)).To(Equal("radioNowPlaying"))
			Expect(event.Data(event)).To(Equal(`{"radioId":"rd-1","title":"Artist - Title","updatedAt":"2026-06-30T12:34:56Z"}`))
		})

		It("publishes metadata updates through the broker", func() {
			ctx := request.WithClientUniqueId(request.WithUsername(context.Background(), "janedoe"), "browser-1")
			updatedAt := time.Date(2026, time.June, 30, 12, 34, 56, 0, time.UTC)
			capture := &capturingBroker{}

			publish := NewRadioMetadataPublisher(capture)
			publish(ctx, coreradio.TitleUpdate{
				RadioID:   "rd-1",
				Title:     "Artist - Title",
				UpdatedAt: updatedAt,
			})

			username, hasUsername := request.UsernameFrom(capture.ctx)
			clientID, hasClientID := request.ClientUniqueIdFrom(capture.ctx)
			Expect(hasUsername).To(BeTrue())
			Expect(username).To(Equal("janedoe"))
			Expect(hasClientID).To(BeTrue())
			Expect(clientID).To(Equal("browser-1"))
			Expect(capture.broadcast).To(BeNil())
			Expect(capture.event).To(Equal(&RadioNowPlaying{
				RadioID:   "rd-1",
				Title:     "Artist - Title",
				UpdatedAt: updatedAt,
			}))
			routingBroker := broker{}
			Expect(routingBroker.shouldSend(message{senderCtx: capture.ctx}, client{username: "janedoe", clientUniqueId: "browser-1"})).To(BeTrue())
			Expect(routingBroker.shouldSend(message{senderCtx: capture.ctx}, client{username: "johndoe", clientUniqueId: "browser-2"})).To(BeFalse())
		})
	})

	Describe("RefreshResource", func() {
		var rr *RefreshResource
		BeforeEach(func() {
			rr = &RefreshResource{}
		})

		It("should render to full refresh if event is empty", func() {
			data := rr.Data(rr)
			Expect(data).To(Equal(`{"*":"*"}`))
		})
		It("should group resources based on name", func() {
			rr.With("album", "al-1").With("song", "sg-1").With("artist", "ar-1")
			rr.With("album", "al-2", "al-3").With("song", "sg-2").With("artist", "ar-2")
			data := rr.Data(rr)
			Expect(data).To(Equal(`{"album":["al-1","al-2","al-3"],"artist":["ar-1","ar-2"],"song":["sg-1","sg-2"]}`))
		})
		It("should send a * when no ids are specified", func() {
			rr.With("album")
			data := rr.Data(rr)
			Expect(data).To(Equal(`{"album":["*"]}`))
		})
	})
})

type capturingBroker struct {
	ctx       context.Context
	event     Event
	broadcast Event
}

func (b *capturingBroker) ServeHTTP(http.ResponseWriter, *http.Request) {}

func (b *capturingBroker) SendMessage(ctx context.Context, event Event) {
	b.ctx = ctx
	b.event = event
}

func (b *capturingBroker) SendBroadcastMessage(_ context.Context, event Event) {
	b.broadcast = event
}
