package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ThrottleBacklog", func() {
	It("is a passthrough when limit is 0", func() {
		m := ThrottleBacklog(0, 10, time.Second)
		r := chi.NewRouter()
		r.Use(m)
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.String()).To(Equal("ok"))
	})

	It("returns 429 when capacity is exceeded", func() {
		_, secondStatus := runTwoRequests(ThrottleBacklog(1, 0, time.Second))
		Expect(secondStatus).To(Equal(http.StatusTooManyRequests))
	})

	It("returns 429 when backlog times out", func() {
		_, secondStatus := runTwoRequests(ThrottleBacklog(1, 1, 50*time.Millisecond))
		Expect(secondStatus).To(Equal(http.StatusTooManyRequests))
	})

	It("releases capacity when the handler panics", func() {
		m := ThrottleBacklog(1, 0, time.Second)
		r := chi.NewRouter()
		r.Use(middleware.Recoverer)
		r.Use(m)
		r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
			panic("boom")
		})
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/panic", nil)
		r.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusInternalServerError))

		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.String()).To(Equal("ok"))
	})

	It("preserves response headers and status code", func() {
		m := ThrottleBacklog(2, 0, time.Second)
		r := chi.NewRouter()
		r.Use(m)
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/jpeg")
			w.Header().Set("Cache-Control", "public")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("body"))
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusCreated))
		Expect(w.Header().Get("Content-Type")).To(Equal("image/jpeg"))
		Expect(w.Header().Get("Cache-Control")).To(Equal("public"))
		Expect(w.Body.String()).To(Equal("body"))
	})

	It("uses the first response status code", func() {
		m := ThrottleBacklog(2, 0, time.Second)
		r := chi.NewRouter()
		r.Use(m)
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("body"))
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusCreated))
		Expect(w.Body.String()).To(Equal("body"))
	})

	It("never exceeds the concurrency limit", func() {
		const limit = 3
		const goroutines = 20
		m := ThrottleBacklog(limit, goroutines, 5*time.Second)

		var concurrent atomic.Int32
		var maxConcurrent atomic.Int32

		r := chi.NewRouter()
		r.Use(m)
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			cur := concurrent.Add(1)
			for {
				old := maxConcurrent.Load()
				if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			concurrent.Add(-1)
			_, _ = w.Write([]byte("ok"))
		})

		var wg sync.WaitGroup
		for range goroutines {
			wg.Go(func() {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("GET", "/test", nil)
				r.ServeHTTP(w, req)
			})
		}

		wg.Wait()
		Expect(maxConcurrent.Load()).To(BeNumerically("<=", limit))
	})

	// Regression: with only 1 token, a slow client blocking during response
	// writing must NOT prevent other requests from being served. Chi's original
	// ThrottleBacklog holds the token for the entire handler lifecycle including
	// io.Copy, causing starvation. The buffered implementation releases it first.
	Context("when a client is slow to read the response", func() {
		slowClientTest := func(m func(http.Handler) http.Handler) (*chi.Mux, chan struct{}, chan struct{}) {
			handlerReached := make(chan struct{}, 1)
			router := chi.NewRouter()
			router.Use(m)
			router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
				select {
				case handlerReached <- struct{}{}:
				default:
				}
				_, _ = io.Copy(w, strings.NewReader("image data"))
			})

			unblocked := make(chan struct{})
			slow := newSlowTestWriter(unblocked)

			reqDone := make(chan struct{})
			go func() {
				defer close(reqDone)
				req, _ := http.NewRequest("GET", "/test", nil)
				router.ServeHTTP(slow, req)
			}()
			<-handlerReached

			return router, unblocked, reqDone
		}

		It("does not starve concurrent requests with buffered middleware", func() {
			router, unblocked, reqDone := slowClientTest(ThrottleBacklog(1, 1, 500*time.Millisecond))

			Eventually(func() int {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("GET", "/test", nil)
				router.ServeHTTP(w, req)
				return w.Code
			}, 2*time.Second, 10*time.Millisecond).Should(Equal(http.StatusOK))

			close(unblocked)
			Eventually(reqDone, 2*time.Second).Should(BeClosed())
		})

		It("starves concurrent requests with Chi's original middleware", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.DevArtworkThrottleBuffered = false

			router, unblocked, reqDone := slowClientTest(ThrottleBacklog(1, 1, 500*time.Millisecond))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusTooManyRequests))

			close(unblocked)
			Eventually(reqDone, 2*time.Second).Should(BeClosed())
		})
	})
})

// runTwoRequests sends two concurrent requests through a throttled router. The
// first request holds the token until the second has been dispatched.
func runTwoRequests(m func(http.Handler) http.Handler) (firstStatus, secondStatus int) {
	held := make(chan struct{})
	release := make(chan struct{})
	r := chi.NewRouter()
	r.Use(m)
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		select {
		case held <- struct{}{}:
		default:
		}
		<-release
		_, _ = w.Write([]byte("ok"))
	})

	done := make(chan int)
	go func() {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)
		done <- w.Code
	}()
	<-held

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
	secondStatus = w.Code

	close(release)
	firstStatus = <-done
	return firstStatus, secondStatus
}

// slowTestWriter implements http.ResponseWriter without embedding
// httptest.ResponseRecorder. This is necessary because ResponseRecorder
// promotes io.ReaderFrom, which io.Copy prefers over Write — bypassing
// our blocking Write and defeating the slow-client simulation.
type slowTestWriter struct {
	header    http.Header
	body      bytes.Buffer
	code      int
	unblocked chan struct{}
}

func newSlowTestWriter(unblocked chan struct{}) *slowTestWriter {
	return &slowTestWriter{header: make(http.Header), unblocked: unblocked}
}

func (w *slowTestWriter) Header() http.Header { return w.header }

func (w *slowTestWriter) WriteHeader(code int) { w.code = code }

func (w *slowTestWriter) Write(p []byte) (int, error) {
	<-w.unblocked
	return w.body.Write(p)
}
