package subsonic

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("authLimiter", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	acquire := func(l *authLimiter, key string) (*authSlot, bool) {
		GinkgoHelper()
		return l.acquire(ctx, key)
	}

	It("blocks a key after the configured number of failures", func() {
		l := newAuthLimiter(2, time.Minute)
		for range 2 {
			slot, ok := acquire(l, "k")
			Expect(ok).To(BeTrue())
			slot.release(true)
		}

		_, ok := acquire(l, "k")
		Expect(ok).To(BeFalse())
	})

	It("never counts successful checks", func() {
		l := newAuthLimiter(2, time.Minute)
		for range 50 {
			slot, ok := acquire(l, "k")
			Expect(ok).To(BeTrue())
			slot.release(false)
		}
	})

	It("keeps keys independent", func() {
		l := newAuthLimiter(1, time.Minute)
		slot, _ := acquire(l, "a")
		slot.release(true)
		_, ok := acquire(l, "a")
		Expect(ok).To(BeFalse())

		_, ok = acquire(l, "b")
		Expect(ok).To(BeTrue())
	})

	It("waits for an in-flight check instead of failing the request", func() {
		l := newAuthLimiter(1, time.Minute)
		held, ok := acquire(l, "k")
		Expect(ok).To(BeTrue())

		waiting := make(chan bool, 1)
		go func() {
			slot, ok := l.acquire(ctx, "k")
			slot.release(false)
			waiting <- ok
		}()
		Consistently(waiting, 50*time.Millisecond).ShouldNot(Receive())

		held.release(false)
		Eventually(waiting).Should(Receive(BeTrue()))
	})

	It("stops waiting when the request is canceled", func() {
		l := newAuthLimiter(1, time.Minute)
		held, _ := acquire(l, "k")
		DeferCleanup(func() { held.release(false) })

		canceled, cancel := context.WithCancel(context.Background())
		cancel()
		_, ok := l.acquire(canceled, "k")
		Expect(ok).To(BeFalse())
	})

	It("does not let concurrent guesses overshoot the limit", func() {
		l := newAuthLimiter(5, time.Minute)
		hold := make(chan struct{})
		var checks atomic.Int32
		var wg sync.WaitGroup
		for range 50 {
			wg.Go(func() {
				slot, ok := l.acquire(ctx, "k")
				if !ok {
					return
				}
				checks.Add(1)
				<-hold
				slot.release(true)
			})
		}

		Eventually(checks.Load).Should(Equal(int32(5)))
		Consistently(checks.Load, 100*time.Millisecond).Should(Equal(int32(5)))
		close(hold)
		wg.Wait()

		_, ok := acquire(l, "k")
		Expect(ok).To(BeFalse())
	})

	It("allows everything when the limit is disabled", func() {
		l := newAuthLimiter(0, time.Minute)
		for range 10 {
			slot, ok := acquire(l, "k")
			Expect(ok).To(BeTrue())
			slot.release(true)
		}
	})
})

// testing/synctest's fake clock needs a *testing.T, which Ginkgo doesn't give.
func TestAuthLimiterWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := NewWithT(t)
		ctx := context.Background()
		l := newAuthLimiter(1, 20*time.Second)
		for _, key := range []string{"a", "b"} {
			slot, ok := l.acquire(ctx, key)
			g.Expect(ok).To(BeTrue())
			slot.release(true)
		}
		_, ok := l.acquire(ctx, "a")
		g.Expect(ok).To(BeFalse())

		time.Sleep(20 * time.Second)
		slot, ok := l.acquire(ctx, "a")
		g.Expect(ok).To(BeTrue())
		slot.release(false)
		g.Expect(l.keys).To(HaveLen(1), "expired keys must be dropped")
	})
}
