package stream

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TranscodingThrottle", func() {
	Describe("Acquire/Release", func() {
		It("allows up to maxConcurrent acquires", func() {
			t := newTranscodingThrottle(2, 10, time.Second)
			Expect(t.Acquire(context.Background())).To(Succeed())
			Expect(t.Acquire(context.Background())).To(Succeed())
			// Third should block, so test it doesn't return immediately
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			err := t.Acquire(ctx)
			Expect(err).To(MatchError(ErrTranscodingBusy))
		})

		It("releases a slot and allows new acquire", func() {
			t := newTranscodingThrottle(1, 10, time.Second)
			Expect(t.Acquire(context.Background())).To(Succeed())
			t.Release()
			Expect(t.Acquire(context.Background())).To(Succeed())
		})

		It("returns ErrTranscodingBusy when backlog limit is reached", func() {
			t := newTranscodingThrottle(1, 2, 5*time.Second)
			// Fill the slot
			Expect(t.Acquire(context.Background())).To(Succeed())

			// Fill the backlog (2 waiters) — they block in goroutines
			var wg sync.WaitGroup
			for i := 0; i < 2; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_ = t.Acquire(context.Background())
				}()
			}
			// Give goroutines time to enter backlog
			time.Sleep(50 * time.Millisecond)

			// Third waiter should be rejected immediately (backlog full)
			err := t.Acquire(context.Background())
			Expect(err).To(MatchError(ErrTranscodingBusy))

			// Clean up: release all
			t.Release()
			t.Release()
			t.Release()
			wg.Wait()
		})

		It("returns ErrTranscodingBusy when timeout expires", func() {
			t := newTranscodingThrottle(1, 10, 50*time.Millisecond)
			Expect(t.Acquire(context.Background())).To(Succeed())
			err := t.Acquire(context.Background())
			Expect(err).To(MatchError(ErrTranscodingBusy))
		})

		It("respects context cancellation", func() {
			t := newTranscodingThrottle(1, 10, 5*time.Second)
			Expect(t.Acquire(context.Background())).To(Succeed())
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			err := t.Acquire(ctx)
			Expect(err).To(MatchError(ErrTranscodingBusy))
		})

		It("is disabled when maxConcurrent is 0", func() {
			t := newTranscodingThrottle(0, 10, time.Second)
			for i := 0; i < 100; i++ {
				Expect(t.Acquire(context.Background())).To(Succeed())
			}
		})
	})
})

var _ = Describe("releaseOnClose", func() {
	It("calls release exactly once on Close", func() {
		var count int
		rc := &releaseOnClose{
			ReadCloser: io.NopCloser(strings.NewReader("data")),
			release:    func() { count++ },
		}
		Expect(rc.Close()).To(Succeed())
		Expect(rc.Close()).To(Succeed()) // double close
		Expect(count).To(Equal(1))
	})

	It("propagates close error from underlying ReadCloser", func() {
		rc := &releaseOnClose{
			ReadCloser: &failCloser{},
			release:    func() {},
		}
		err := rc.Close()
		Expect(err).To(MatchError("close failed"))
	})
})

// failCloser is a ReadCloser whose Close always returns an error
type failCloser struct{ io.Reader }

func (f *failCloser) Read(p []byte) (int, error) { return 0, io.EOF }
func (f *failCloser) Close() error               { return errors.New("close failed") }
