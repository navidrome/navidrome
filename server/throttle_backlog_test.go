package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ThrottleBacklog", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
	})

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

	It("falls back to Chi's ThrottleBacklog when buffered mode is disabled", func() {
		conf.Server.DevArtworkThrottleBuffered = false

		m := ThrottleBacklog(1, 1, 500*time.Millisecond)
		held := make(chan struct{})
		r := chi.NewRouter()
		r.Use(m)
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Handler", "reached")
			close(held)
			time.Sleep(2 * time.Second)
			_, _ = w.Write([]byte("ok"))
		})

		// With Chi's ThrottleBacklog (non-buffered), a slow client holds the
		// token for the entire handler duration, so the second request should
		// get 429 after the backlog timeout.
		done := make(chan struct{})
		go func() {
			defer close(done)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			r.ServeHTTP(w, req)
		}()
		<-held

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusTooManyRequests))
		Eventually(done, 3*time.Second).Should(BeClosed())
	})

	It("passes requests through when capacity is available", func() {
		m := ThrottleBacklog(2, 0, time.Second)
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
		m := ThrottleBacklog(1, 0, time.Second)
		held := make(chan struct{})
		r := chi.NewRouter()
		r.Use(m)
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			close(held)
			time.Sleep(2 * time.Second)
			_, _ = w.Write([]byte("ok"))
		})

		done := make(chan struct{})
		go func() {
			defer close(done)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			r.ServeHTTP(w, req)
		}()
		<-held

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusTooManyRequests))
		Eventually(done, 3*time.Second).Should(BeClosed())
	})

	It("returns 429 when backlog times out", func() {
		m := ThrottleBacklog(1, 1, 50*time.Millisecond)
		held := make(chan struct{})
		r := chi.NewRouter()
		r.Use(m)
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			close(held)
			time.Sleep(2 * time.Second)
			_, _ = w.Write([]byte("ok"))
		})

		done := make(chan struct{})
		go func() {
			defer close(done)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			r.ServeHTTP(w, req)
		}()
		<-held

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		Expect(w.Code).To(Equal(http.StatusTooManyRequests))
		Eventually(done, 3*time.Second).Should(BeClosed())
	})

	It("buffers the response and releases the token before writing to client", func() {
		m := ThrottleBacklog(1, 1, 5*time.Second)
		artworkData := "fake image data"
		r := chi.NewRouter()
		r.Use(m)
		r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom", "value")
			_, _ = io.Copy(w, strings.NewReader(artworkData))
		})

		unblocked := make(chan struct{})
		slow := &slowTestWriter{
			ResponseRecorder: httptest.NewRecorder(),
			unblocked:        unblocked,
		}

		reqDone := make(chan struct{})
		go func() {
			defer close(reqDone)
			req, _ := http.NewRequest("GET", "/test", nil)
			r.ServeHTTP(slow, req)
		}()

		// A concurrent request should succeed while the first is stalled on writing
		Eventually(func() int {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			r.ServeHTTP(w, req)
			return w.Code
		}, 2*time.Second, 10*time.Millisecond).Should(Equal(http.StatusOK))

		close(unblocked)
		Eventually(reqDone, 2*time.Second).Should(BeClosed())
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
})
var _ = Describe("SetWriteTimeout", func() {
	It("sets the write deadline on a writer that supports SetWriteDeadline", func() {
		w := &mockDeadlineWriter{}
		err := setWriteTimeout(w, 30*time.Second)
		Expect(err).ToNot(HaveOccurred())
		Expect(w.deadline).To(BeTemporally("~", time.Now().Add(30*time.Second), time.Second))
	})

	It("unwraps wrapped writers to find SetWriteDeadline", func() {
		inner := &mockDeadlineResponseWriter{}
		wrapped := &mockUnwrappingWriter{inner: inner}
		err := setWriteTimeout(wrapped, 15*time.Second)
		Expect(err).ToNot(HaveOccurred())
		Expect(inner.deadline).To(BeTemporally("~", time.Now().Add(15*time.Second), time.Second))
	})

	It("returns ErrNotSupported for writers without SetWriteDeadline", func() {
		w := httptest.NewRecorder()
		err := setWriteTimeout(w, 10*time.Second)
		Expect(err).To(MatchError(http.ErrNotSupported))
	})
})

// mockDeadlineWriter implements io.Writer + SetWriteDeadline
type mockDeadlineWriter struct {
	deadline time.Time
}

func (w *mockDeadlineWriter) SetWriteDeadline(t time.Time) error {
	w.deadline = t
	return nil
}

func (w *mockDeadlineWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

// mockDeadlineResponseWriter implements http.ResponseWriter + SetWriteDeadline
type mockDeadlineResponseWriter struct {
	httptest.ResponseRecorder
	deadline time.Time
}

func (w *mockDeadlineResponseWriter) SetWriteDeadline(t time.Time) error {
	w.deadline = t
	return nil
}

// mockUnwrappingWriter wraps a ResponseWriter via Unwrap (no SetWriteDeadline itself)
type mockUnwrappingWriter struct {
	inner http.ResponseWriter
}

func (w *mockUnwrappingWriter) Write(p []byte) (int, error) {
	return w.inner.Write(p)
}

func (w *mockUnwrappingWriter) Unwrap() http.ResponseWriter {
	return w.inner
}

type slowTestWriter struct {
	*httptest.ResponseRecorder
	unblocked chan struct{}
}

func (w *slowTestWriter) Write(p []byte) (int, error) {
	<-w.unblocked
	return w.ResponseRecorder.Write(p)
}
