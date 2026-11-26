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
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus/hooks/test"
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
			fake := &tests.FakeHttpClient{Res: http.Response{StatusCode: http.StatusOK, ProtoMajor: 1, ProtoMinor: 1, Body: io.NopCloser(strings.NewReader("ok"))}}
			chc = NewHTTPClient(fake, consts.DefaultHttpClientTimeOut)

			ctx, cancel := context.WithCancel(context.WithValue(GinkgoT().Context(), ctxKey{}, "value"))
			cancel()
			r, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
			_, err := chc.Do(r)
			Expect(err).ToNot(HaveOccurred())
			sent := fake.SavedRequest.Context()
			Expect(sent.Value(ctxKey{})).To(Equal("value"))
			Expect(sent.Err()).To(Succeed())
		})

		It("does not log request headers", func() {
			l, hook := test.NewNullLogger()
			prevLogger, prevLevel := log.SetDefaultLogger(l), log.CurrentLevel()
			log.SetLevel(log.LevelTrace)
			DeferCleanup(func() {
				log.SetDefaultLogger(prevLogger)
				log.SetLevel(prevLevel)
			})

			r, _ := http.NewRequest(http.MethodGet, ts.URL, nil)
			r.Header.Set("Authorization", "Bearer secret-token")
			_, err := chc.Do(r)
			Expect(err).ToNot(HaveOccurred())

			Expect(hook.AllEntries()).ToNot(BeEmpty())
			for _, e := range hook.AllEntries() {
				line, _ := e.String()
				Expect(line).ToNot(ContainSubstring("secret-token"))
			}
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
