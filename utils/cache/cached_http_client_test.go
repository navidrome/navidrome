package cache

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
			r, _ := http.NewRequest("GET", ts.URL+"?name=doe", nil)
			resp, err := chc.Do(r)
			Expect(err).To(BeNil())
			body, err := io.ReadAll(resp.Body)
			Expect(err).To(BeNil())
			Expect(string(body)).To(Equal("Hello, [doe]"))
			Expect(requestsReceived).To(Equal(1))

			// Same request
			r, _ = http.NewRequest("GET", ts.URL+"?name=doe", nil)
			resp, err = chc.Do(r)
			Expect(err).To(BeNil())
			body, err = io.ReadAll(resp.Body)
			Expect(err).To(BeNil())
			Expect(string(body)).To(Equal("Hello, [doe]"))
			Expect(requestsReceived).To(Equal(1))

			// Different request
			r, _ = http.NewRequest("GET", ts.URL, nil)
			resp, err = chc.Do(r)
			Expect(err).To(BeNil())
			body, err = io.ReadAll(resp.Body)
			Expect(err).To(BeNil())
			Expect(string(body)).To(Equal("Hello, []"))
			Expect(requestsReceived).To(Equal(2))

			// Different again (same as before, but with header)
			r, _ = http.NewRequest("GET", ts.URL, nil)
			r.Header.Add("head", "this is a header")
			resp, err = chc.Do(r)
			Expect(err).To(BeNil())
			body, err = io.ReadAll(resp.Body)
			Expect(err).To(BeNil())
			Expect(string(body)).To(Equal("Hello, []"))
			Expect(header).To(Equal("this is a header"))
			Expect(requestsReceived).To(Equal(3))
		})

		It("expires responses after TTL", func() {
			requestsReceived = 0
			chc = NewHTTPClient(http.DefaultClient, 10*time.Millisecond)

			r, _ := http.NewRequest("GET", ts.URL+"?name=doe", nil)
			_, err := chc.Do(r)
			Expect(err).To(BeNil())
			Expect(requestsReceived).To(Equal(1))

			// Wait more than the TTL
			time.Sleep(50 * time.Millisecond)

			// Same request
			r, _ = http.NewRequest("GET", ts.URL+"?name=doe", nil)
			_, err = chc.Do(r)
			Expect(err).To(BeNil())
			Expect(requestsReceived).To(Equal(2))
		})
	})
})
