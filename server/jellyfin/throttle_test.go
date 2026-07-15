package jellyfin

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("throttleStreams", func() {
	// serve fires n concurrent requests through the middleware and reports the highest number that
	// were ever inside the handler at once.
	serve := func(limit, n int) int32 {
		var inFlight, peak int32
		release := make(chan struct{})
		h := throttleStreams(limit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cur := atomic.AddInt32(&inFlight, 1)
			for {
				old := atomic.LoadInt32(&peak)
				if cur <= old || atomic.CompareAndSwapInt32(&peak, old, cur) {
					break
				}
			}
			<-release // hold the slot until every request has had a chance to enter
			atomic.AddInt32(&inFlight, -1)
		}))

		var wg sync.WaitGroup
		for range n {
			wg.Add(1)
			go func() {
				defer wg.Done()
				h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/Items", nil))
			}()
		}
		// Give the admitted requests time to pile up before letting them finish.
		time.Sleep(100 * time.Millisecond)
		close(release)
		wg.Wait()
		return atomic.LoadInt32(&peak)
	}

	It("admits no more than the limit at once", func() {
		Expect(serve(2, 8)).To(Equal(int32(2)))
	})

	It("queues the excess rather than rejecting it", func() {
		// All 8 still complete — they wait for a slot instead of getting a 429.
		var served int32
		release := make(chan struct{})
		h := throttleStreams(2)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-release
			atomic.AddInt32(&served, 1)
		}))
		var wg sync.WaitGroup
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/Items", nil))
			}()
		}
		close(release)
		wg.Wait()
		Expect(served).To(Equal(int32(8)))
	})

	// chi's ThrottleBacklog panics on a non-positive limit, so a user disabling the cap must not
	// crash the server at startup.
	It("is disabled, not panicking, when the limit is zero", func() {
		Expect(func() { serve(0, 4) }).ToNot(Panic())
		Expect(serve(0, 4)).To(BeNumerically(">", int32(1)))
	})
})
