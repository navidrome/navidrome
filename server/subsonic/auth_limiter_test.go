package subsonic

import (
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("authLimiter", func() {
	It("gives a slot back on release, never going below zero", func() {
		l := newAuthLimiter(2, time.Minute)
		l.release("k")
		l.release("k")
		Expect(l.acquire("k")).To(BeTrue())
		Expect(l.acquire("k")).To(BeTrue())
		Expect(l.acquire("k")).To(BeFalse())

		l.release("k")
		Expect(l.acquire("k")).To(BeTrue())
	})

	It("never lets concurrent attempts overshoot the limit", func() {
		l := newAuthLimiter(5, time.Minute)
		var allowed atomic.Int32
		var wg sync.WaitGroup
		for range 100 {
			wg.Go(func() {
				if l.acquire("k") {
					allowed.Add(1)
				}
			})
		}
		wg.Wait()
		Expect(allowed.Load()).To(Equal(int32(5)))
	})
})

// testing/synctest's fake clock needs a *testing.T, which Ginkgo doesn't give.
func TestAuthLimiterWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := NewWithT(t)
		l := newAuthLimiter(1, 20*time.Second)
		g.Expect(l.acquire("a")).To(BeTrue())
		g.Expect(l.acquire("b")).To(BeTrue())
		g.Expect(l.acquire("a")).To(BeFalse())

		time.Sleep(20 * time.Second)
		g.Expect(l.acquire("a")).To(BeTrue())
		g.Expect(l.keys).To(HaveLen(1), "expired keys must be dropped")
	})
}
