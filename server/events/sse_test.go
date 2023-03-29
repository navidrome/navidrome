package events

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Broker", func() {
	var b broker

	BeforeEach(func() {
		b = broker{}
	})

	Describe("shouldSend", func() {
		var c client
		var ctx context.Context
		BeforeEach(func() {
			ctx = context.Background()
			c = client{
				clientUniqueId: "1111",
				username:       "janedoe",
			}
		})
		Context("request has clientUniqueId", func() {
			It("sends message for same username, different clientUniqueId", func() {
				ctx = request.WithClientUniqueId(ctx, "2222")
				ctx = request.WithUsername(ctx, "janedoe")
				m := message{senderCtx: ctx}
				Expect(b.shouldSend(m, c)).To(BeTrue())
			})
			It("does not send message for same username, same clientUniqueId", func() {
				ctx = request.WithClientUniqueId(ctx, "1111")
				ctx = request.WithUsername(ctx, "janedoe")
				m := message{senderCtx: ctx}
				Expect(b.shouldSend(m, c)).To(BeFalse())
			})
			It("does not send message for different username", func() {
				ctx = request.WithClientUniqueId(ctx, "3333")
				ctx = request.WithUsername(ctx, "johndoe")
				m := message{senderCtx: ctx}
				Expect(b.shouldSend(m, c)).To(BeFalse())
			})
		})
		Context("request does not have clientUniqueId", func() {
			It("sends message for same username", func() {
				ctx = request.WithUsername(ctx, "janedoe")
				m := message{senderCtx: ctx}
				Expect(b.shouldSend(m, c)).To(BeTrue())
			})
			It("sends message for different username", func() {
				ctx = request.WithUsername(ctx, "johndoe")
				m := message{senderCtx: ctx}
				Expect(b.shouldSend(m, c)).To(BeTrue())
			})
		})
	})

	Describe("writeEvent", func() {
		var (
			timeout   time.Duration
			buffer    *bytes.Buffer
			event     message
			senderCtx context.Context
			cancel    context.CancelFunc
		)

		BeforeEach(func() {
			buffer = &bytes.Buffer{}
			senderCtx, cancel = context.WithCancel(context.Background())
			DeferCleanup(cancel)
		})

		Context("with an HTTP flusher", func() {
			var flusher *fakeFlusher

			BeforeEach(func() {
				flusher = &fakeFlusher{Writer: buffer}
				event = message{
					senderCtx: senderCtx,
					id:        1,
					event:     "test",
					data:      "testdata",
				}
			})

			Context("when the write completes before the timeout", func() {
				BeforeEach(func() {
					timeout = 1 * time.Second
				})
				It("should successfully write the event", func() {
					err := writeEvent(flusher, event, timeout)
					Expect(err).NotTo(HaveOccurred())
					Expect(buffer.String()).To(Equal(fmt.Sprintf("id: %d\nevent: %s\ndata: %s\n\n", event.id, event.event, event.data)))
					Expect(flusher.flushed.Load()).To(BeTrue())
				})
			})

			Context("when the write does not complete before the timeout", func() {
				BeforeEach(func() {
					timeout = 1 * time.Millisecond
					flusher.delay = 10 * time.Millisecond
				})

				It("should return an errWriteTimeOut error", func() {
					err := writeEvent(flusher, event, timeout)
					Expect(err).To(MatchError(errWriteTimeOut))
					Expect(flusher.flushed.Load()).To(BeFalse())
				})
			})

			Context("without an HTTP flusher", func() {
				var writer *fakeWriter

				BeforeEach(func() {
					writer = &fakeWriter{Writer: buffer}
					event = message{
						senderCtx: senderCtx,
						id:        1,
						event:     "test",
						data:      "testdata",
					}
				})

				Context("when the write completes before the timeout", func() {
					BeforeEach(func() {
						timeout = 1 * time.Second
					})

					It("should successfully write the event", func() {
						err := writeEvent(writer, event, timeout)
						Expect(err).NotTo(HaveOccurred())
						Eventually(writer.done.Load).Should(BeTrue())
						Expect(buffer.String()).To(Equal(fmt.Sprintf("id: %d\nevent: %s\ndata: %s\n\n", event.id, event.event, event.data)))
					})
				})

				Context("when the write does not complete before the timeout", func() {
					BeforeEach(func() {
						timeout = 1 * time.Millisecond
						writer.delay = 10 * time.Millisecond
					})

					It("should return an errWriteTimeOut error", func() {
						err := writeEvent(writer, event, timeout)
						Expect(err).To(MatchError(errWriteTimeOut))
						Expect(writer.done.Load()).To(BeFalse())
					})
				})
			})
		})
	})
})

type fakeWriter struct {
	io.Writer
	delay time.Duration
	done  atomic.Bool
}

func (f *fakeWriter) Write(p []byte) (n int, err error) {
	time.Sleep(f.delay)
	f.done.Store(true)
	return f.Writer.Write(p)
}

type fakeFlusher struct {
	io.Writer
	delay   time.Duration
	flushed atomic.Bool
}

func (f *fakeFlusher) Write(p []byte) (n int, err error) {
	time.Sleep(f.delay)
	return f.Writer.Write(p)
}

func (f *fakeFlusher) Flush() {
	f.flushed.Store(true)
}
