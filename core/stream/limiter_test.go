package stream_test

import (
	"context"
	"errors"
	"sync"

	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TranscodeLimiter", func() {
	ctx := log.NewContext(context.TODO())

	Describe("Disabled (MaxConcurrent <= 0)", func() {
		It("never blocks and never returns ErrTooManyTranscodes", func() {
			lim := stream.NewTranscodeLimiter(0, 0)
			for range 100 {
				rel, err := lim.Acquire(ctx, "alice")
				Expect(err).ToNot(HaveOccurred())
				Expect(rel).ToNot(BeNil())
			}
		})
	})

	Describe("Global cap", func() {
		It("rejects requests beyond MaxConcurrent with ErrTooManyTranscodes", func() {
			lim := stream.NewTranscodeLimiter(2, 0)

			rel1, err := lim.Acquire(ctx, "alice")
			Expect(err).ToNot(HaveOccurred())
			rel2, err := lim.Acquire(ctx, "bob")
			Expect(err).ToNot(HaveOccurred())

			_, err = lim.Acquire(ctx, "carol")
			Expect(errors.Is(err, stream.ErrTooManyTranscodes)).To(BeTrue())

			rel1()
			_, err = lim.Acquire(ctx, "carol")
			Expect(err).ToNot(HaveOccurred())

			rel2()
		})

		It("releases a slot only once even if release is called multiple times", func() {
			lim := stream.NewTranscodeLimiter(1, 0)

			rel, err := lim.Acquire(ctx, "alice")
			Expect(err).ToNot(HaveOccurred())

			rel()
			rel()
			rel()

			// After releases, exactly one slot should be available.
			_, err = lim.Acquire(ctx, "alice")
			Expect(err).ToNot(HaveOccurred())
			_, err = lim.Acquire(ctx, "alice")
			Expect(errors.Is(err, stream.ErrTooManyTranscodes)).To(BeTrue())
		})
	})

	Describe("Per-user cap", func() {
		It("rejects a user beyond MaxConcurrentPerUser even if global slots remain", func() {
			lim := stream.NewTranscodeLimiter(10, 2)

			rel1, err := lim.Acquire(ctx, "alice")
			Expect(err).ToNot(HaveOccurred())
			rel2, err := lim.Acquire(ctx, "alice")
			Expect(err).ToNot(HaveOccurred())

			_, err = lim.Acquire(ctx, "alice")
			Expect(errors.Is(err, stream.ErrTooManyTranscodes)).To(BeTrue())

			// A different user is unaffected.
			rel3, err := lim.Acquire(ctx, "bob")
			Expect(err).ToNot(HaveOccurred())

			rel1()
			_, err = lim.Acquire(ctx, "alice")
			Expect(err).ToNot(HaveOccurred())

			rel2()
			rel3()
		})

		It("treats anonymous users as a shared bucket", func() {
			lim := stream.NewTranscodeLimiter(10, 1)

			rel, err := lim.Acquire(ctx, "")
			Expect(err).ToNot(HaveOccurred())

			_, err = lim.Acquire(ctx, "")
			Expect(errors.Is(err, stream.ErrTooManyTranscodes)).To(BeTrue())

			rel()
		})
	})

	Describe("Concurrent safety", func() {
		It("survives parallel Acquire/release with consistent counts", func() {
			lim := stream.NewTranscodeLimiter(5, 0)

			var wg sync.WaitGroup
			var acquired int64
			var rejected int64
			var mu sync.Mutex

			for i := range 50 {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					rel, err := lim.Acquire(ctx, "alice")
					mu.Lock()
					if err == nil {
						acquired++
						mu.Unlock()
						rel()
					} else {
						rejected++
						mu.Unlock()
					}
					_ = i
				}(i)
			}
			wg.Wait()

			Expect(acquired + rejected).To(Equal(int64(50)))
			// After all releases, all 5 slots should be free again.
			for range 5 {
				_, err := lim.Acquire(ctx, "alice")
				Expect(err).ToNot(HaveOccurred())
			}
			_, err := lim.Acquire(ctx, "alice")
			Expect(errors.Is(err, stream.ErrTooManyTranscodes)).To(BeTrue())
		})
	})
})
