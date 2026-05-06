package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func throttle(limit, backlog int, timeout time.Duration) *RequestThrottle {
	DeferCleanup(configtest.SetupConfig())
	conf.Server.DevArtworkMaxRequests = limit
	conf.Server.DevArtworkThrottleBacklogLimit = backlog
	conf.Server.DevArtworkThrottleBacklogTimeout = timeout
	return NewRequestThrottle()
}

var _ = Describe("RequestThrottle", func() {
	Describe("NewRequestThrottle", func() {
		It("creates a throttle with correct capacity", func() {
			t := throttle(3, 10, time.Second)
			Expect(t).ToNot(BeNil())
			Expect(cap(t.tokens)).To(Equal(3))
			Expect(cap(t.backlogTokens)).To(Equal(13))
		})

		It("returns nil when disabled", func() {
			t := throttle(0, 10, time.Second)
			Expect(t).To(BeNil())
		})
	})

	Describe("acquire", func() {
		It("returns immediately when tokens are available", func() {
			t := throttle(2, 0, time.Second)

			release, err := t.acquire(context.Background())
			Expect(err).ToNot(HaveOccurred())
			Expect(release).ToNot(BeNil())
			release()
		})

		It("allows reuse of token after release", func() {
			t := throttle(1, 0, time.Second)

			release, err := t.acquire(context.Background())
			Expect(err).ToNot(HaveOccurred())
			release()

			release2, err := t.acquire(context.Background())
			Expect(err).ToNot(HaveOccurred())
			Expect(release2).ToNot(BeNil())
			release2()
		})

		It("is safe to call release multiple times", func() {
			t := throttle(1, 0, time.Second)

			release, err := t.acquire(context.Background())
			Expect(err).ToNot(HaveOccurred())

			release()
			Expect(func() { release() }).ToNot(Panic())
		})

		It("blocks in backlog until token is freed", func() {
			t := throttle(1, 1, time.Second)

			release1, err := t.acquire(context.Background())
			Expect(err).ToNot(HaveOccurred())

			done := make(chan struct{})
			go func() {
				defer close(done)
				release2, err := t.acquire(context.Background())
				Expect(err).ToNot(HaveOccurred())
				release2()
			}()

			time.Sleep(20 * time.Millisecond)
			release1()

			Eventually(done, 500*time.Millisecond).Should(BeClosed())
		})

		It("returns ErrThrottleCapacityExceeded when backlog is full", func() {
			t := throttle(1, 0, time.Second)

			release, err := t.acquire(context.Background())
			Expect(err).ToNot(HaveOccurred())
			defer release()

			_, err = t.acquire(context.Background())
			Expect(err).To(MatchError(ErrThrottleCapacityExceeded))
		})

		It("returns ErrThrottleTimeout when backlog times out", func() {
			t := throttle(1, 1, 50*time.Millisecond)

			release, err := t.acquire(context.Background())
			Expect(err).ToNot(HaveOccurred())
			defer release()

			_, err = t.acquire(context.Background())
			Expect(err).To(MatchError(ErrThrottleTimeout))
		})

		It("returns context error when context is cancelled while waiting", func() {
			t := throttle(1, 1, 5*time.Second)
			ctx, cancel := context.WithCancel(context.Background())

			release, err := t.acquire(ctx)
			Expect(err).ToNot(HaveOccurred())
			defer release()

			go func() {
				time.Sleep(50 * time.Millisecond)
				cancel()
			}()

			_, err = t.acquire(ctx)
			Expect(err).To(MatchError(context.Canceled))
		})

		It("never exceeds the concurrency limit", func() {
			const limit = 3
			const goroutines = 20
			t := throttle(limit, goroutines, 5*time.Second)

			var concurrent atomic.Int32
			var maxConcurrent atomic.Int32
			var wg sync.WaitGroup

			for range goroutines {
				wg.Go(func() {
					release, err := t.acquire(context.Background())
					if err != nil {
						return
					}
					cur := concurrent.Add(1)
					for {
						old := maxConcurrent.Load()
						if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
							break
						}
					}
					time.Sleep(5 * time.Millisecond)
					concurrent.Add(-1)
					release()
				})
			}

			wg.Wait()
			Expect(maxConcurrent.Load()).To(BeNumerically("<=", limit))
		})
	})

	Describe("DoBuffered", func() {
		It("works with nil receiver (throttling disabled)", func() {
			data := "image data"
			fetchFn := func() (io.ReadCloser, time.Time, error) {
				return io.NopCloser(bytes.NewReader([]byte(data))), time.Time{}, nil
			}

			var t *RequestThrottle
			buf, _, err := t.DoBuffered(context.Background(), fetchFn)
			Expect(err).ToNot(HaveOccurred())
			Expect(buf.String()).To(Equal(data))
		})

		It("acquires and releases throttle token around fetch", func() {
			t := throttle(1, 0, time.Second)
			fetchFn := func() (io.ReadCloser, time.Time, error) {
				return io.NopCloser(bytes.NewReader([]byte("ok"))), time.Time{}, nil
			}

			buf, _, err := t.DoBuffered(context.Background(), fetchFn)
			Expect(err).ToNot(HaveOccurred())
			Expect(buf.String()).To(Equal("ok"))

			release, err := t.acquire(context.Background())
			Expect(err).ToNot(HaveOccurred())
			release()
		})

		It("releases token on fetch error", func() {
			t := throttle(1, 0, time.Second)
			fetchErr := errors.New("fetch failed")
			fetchFn := func() (io.ReadCloser, time.Time, error) {
				return nil, time.Time{}, fetchErr
			}

			_, _, err := t.DoBuffered(context.Background(), fetchFn)
			Expect(err).To(MatchError(fetchErr))

			release, err := t.acquire(context.Background())
			Expect(err).ToNot(HaveOccurred())
			release()
		})

		It("returns throttle error when capacity exceeded", func() {
			t := throttle(1, 0, time.Second)
			release, err := t.acquire(context.Background())
			Expect(err).ToNot(HaveOccurred())
			defer release()

			_, _, err = t.DoBuffered(context.Background(), nil)
			Expect(err).To(MatchError(ErrThrottleCapacityExceeded))
		})
	})
})
