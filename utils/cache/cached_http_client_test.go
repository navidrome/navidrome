package cache

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/navidrome/navidrome/consts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTTPClient", func() {
	Context("GET", func() {
		var chc *HTTPClient
		var ts *httptest.Server
		var requestsReceived int
		var header string

		BeforeEach(func() {
			requestsReceived = 0
			header = ""
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestsReceived++
				header = r.Header.Get("head")
				_, _ = fmt.Fprintf(w, "Hello, %s", r.URL.Query()["name"])
			}))
			chc = NewHTTPClient(http.DefaultClient, consts.DefaultHttpClientTimeOut)
		})

		AfterEach(func() {
			defer ts.Close()
		})

		It("caches repeated requests", func() {
			r, _ := http.NewRequest(http.MethodGet, ts.URL+"?name=doe", nil)
			resp, err := chc.Do(r)
			Expect(err).ToNot(HaveOccurred())
			body, err := io.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(body)).To(Equal("Hello, [doe]"))
			Expect(requestsReceived).To(Equal(1))

			// Same request
			r, _ = http.NewRequest(http.MethodGet, ts.URL+"?name=doe", nil)
			resp, err = chc.Do(r)
			Expect(err).ToNot(HaveOccurred())
			body, err = io.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(body)).To(Equal("Hello, [doe]"))
			Expect(requestsReceived).To(Equal(1))

			// Different request
			r, _ = http.NewRequest(http.MethodGet, ts.URL, nil)
			resp, err = chc.Do(r)
			Expect(err).ToNot(HaveOccurred())
			body, err = io.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(body)).To(Equal("Hello, []"))
			Expect(requestsReceived).To(Equal(2))

			// Different again (same as before, but with header)
			r, _ = http.NewRequest(http.MethodGet, ts.URL, nil)
			r.Header.Add("head", "this is a header")
			resp, err = chc.Do(r)
			Expect(err).ToNot(HaveOccurred())
			body, err = io.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(body)).To(Equal("Hello, []"))
			Expect(header).To(Equal("this is a header"))
			Expect(requestsReceived).To(Equal(3))
		})

		It("expires responses after TTL", func() {
			requestsReceived = 0
			chc = NewHTTPClient(http.DefaultClient, 10*time.Millisecond)

			r, _ := http.NewRequest(http.MethodGet, ts.URL+"?name=doe", nil)
			_, err := chc.Do(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(requestsReceived).To(Equal(1))

			// Wait more than the TTL
			time.Sleep(50 * time.Millisecond)

			// Same request
			r, _ = http.NewRequest(http.MethodGet, ts.URL+"?name=doe", nil)
			_, err = chc.Do(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(requestsReceived).To(Equal(2))
		})

		It("passes the caller's context values, but not its cancellation", func() {
			type ctxKey struct{}
			var sent context.Context
			doer := doerFunc(func(req *http.Request) (*http.Response, error) {
				sent = req.Context()
				return http.DefaultClient.Do(req)
			})
			chc = NewHTTPClient(doer, consts.DefaultHttpClientTimeOut)

			ctx, cancel := context.WithCancel(context.WithValue(GinkgoT().Context(), ctxKey{}, "value"))
			cancel()
			r, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
			resp, err := chc.Do(r)
			Expect(err).ToNot(HaveOccurred())
			_ = resp.Body.Close()
			Expect(sent.Value(ctxKey{})).To(Equal("value"))
			Expect(sent.Err()).To(Succeed())
		})
	})

	Context("non-GET", func() {
		var chc *HTTPClient
		var ts *httptest.Server
		var requestsReceived int

		BeforeEach(func() {
			requestsReceived = 0
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestsReceived++
				if requestsReceived == 1 {
					w.WriteHeader(http.StatusBadGateway)
					return
				}
				_, _ = fmt.Fprint(w, "ok")
			}))
			chc = NewHTTPClient(http.DefaultClient, consts.DefaultHttpClientTimeOut)
		})

		AfterEach(func() {
			ts.Close()
		})

		It("sends every request to the server, so an error reply is not replayed", func() {
			r, _ := http.NewRequest(http.MethodPost, ts.URL, strings.NewReader("listen"))
			resp, err := chc.Do(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusBadGateway))
			_ = resp.Body.Close()

			// Same request
			r, _ = http.NewRequest(http.MethodPost, ts.URL, strings.NewReader("listen"))
			resp, err = chc.Do(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body, err := io.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(body)).To(Equal("ok"))
			Expect(requestsReceived).To(Equal(2))
		})

		It("keeps the caller's context", func() {
			ctx, cancel := context.WithCancel(GinkgoT().Context())
			cancel()
			r, _ := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL, strings.NewReader("listen"))
			_, err := chc.Do(r)
			Expect(err).To(MatchError(context.Canceled))
			Expect(requestsReceived).To(Equal(0))
		})
	})
})

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }
