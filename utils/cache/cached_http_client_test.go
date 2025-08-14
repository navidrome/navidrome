package cache

import (
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

// Tests for HTTPClient, ensuring caching behavior for GET and POST requests.
var _ = Describe("HTTPClient", func() {
	var chc *HTTPClient
	var ts *httptest.Server
	var requestsReceived int
	var header string

	BeforeEach(func() {
		// Reset counters and initialize test server
		requestsReceived = 0
		header = ""
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestsReceived++
			header = r.Header.Get("head")
			if r.Method == "POST" {
				body, _ := io.ReadAll(r.Body)
				_, _ = fmt.Fprintf(w, "Received: %s", string(body))
			} else {
				_, _ = fmt.Fprintf(w, "Hello, %s", r.URL.Query().Get("name"))
			}
		}))
		chc = NewHTTPClient(http.DefaultClient, consts.DefaultHttpClientTimeOut)
	})

	AfterEach(func() {
		// Clean up test server
		ts.Close()
	})

	Context("GET requests", func() {
		// Tests caching behavior for GET requests
		It("caches repeated GET requests", func() {
			r, err := http.NewRequest("GET", ts.URL+"?name=doe", nil)
			Expect(err).To(BeNil())
			resp, err := chc.Do(r)
			Expect(err).To(BeNil())
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			Expect(err).To(BeNil())
			Expect(string(body)).To(Equal("Hello, doe"))
			Expect(requestsReceived).To(Equal(1))

			// Same request, should hit cache
			r, err = http.NewRequest("GET", ts.URL+"?name=doe", nil)
			Expect(err).To(BeNil())
			resp, err = chc.Do(r)
			Expect(err).To(BeNil())
			defer resp.Body.Close()
			body, err = io.ReadAll(resp.Body)
			Expect(err).To(BeNil())
			Expect(string(body)).To(Equal("Hello, doe"))
			Expect(requestsReceived).To(Equal(1))

			// Different request (no query), should miss cache
			r, err = http.NewRequest("GET", ts.URL, nil)
			Expect(err).To(BeNil())
			resp, err = chc.Do(r)
			Expect(err).To(BeNil())
			defer resp.Body.Close()
			body, err = io.ReadAll(resp.Body)
			Expect(err).To(BeNil())
			Expect(string(body)).To(Equal("Hello, "))
			Expect(requestsReceived).To(Equal(2))

			// Different request (with header), should miss cache
			r, err = http.NewRequest("GET", ts.URL, nil)
			Expect(err).To(BeNil())
			r.Header.Add("head", "this is a header")
			resp, err = chc.Do(r)
			Expect(err).To(BeNil())
			defer resp.Body.Close()
			body, err = io.ReadAll(resp.Body)
			Expect(err).To(BeNil())
			Expect(string(body)).To(Equal("Hello, "))
			Expect(header).To(Equal("this is a header"))
			Expect(requestsReceived).To(Equal(3))
		})

		// Tests cache expiration after TTL
		It("expires GET responses after TTL", func() {
			chc = NewHTTPClient(http.DefaultClient, 10*time.Millisecond)

			r, err := http.NewRequest("GET", ts.URL+"?name=doe", nil)
			Expect(err).To(BeNil())
			resp, err := chc.Do(r)
			Expect(err).To(BeNil())
			defer resp.Body.Close()
			Expect(requestsReceived).To(Equal(1))

			// Wait just beyond TTL (consider mock clock for determinism)
			time.Sleep(15 * time.Millisecond)

			// Same request, should miss cache due to expiration
			r, err = http.NewRequest("GET", ts.URL+"?name=doe", nil)
			Expect(err).To(BeNil())
			resp, err = chc.Do(r)
			Expect(err).To(BeNil())
			defer resp.Body.Close()
			Expect(requestsReceived).To(Equal(2))
		})

		// Tests error handling for invalid URLs
		It("handles unreachable URLs", func() {
			r, err := http.NewRequest("GET", "http://invalid.example.com", nil)
			Expect(err).To(BeNil())
			resp, err := chc.Do(r)
			Expect(err).NotTo(BeNil())
			Expect(resp).To(BeNil())
			Expect(requestsReceived).To(Equal(0)) // No requests should reach the test server
		})
	})

	Context("POST requests", func() {
		// Tests caching behavior for POST requests with bodies
		It("caches repeated POST requests with bodies", func() {
			body := strings.NewReader("testdata")
			r, err := http.NewRequest("POST", ts.URL, body)
			Expect(err).To(BeNil())
			resp, err := chc.Do(r)
			Expect(err).To(BeNil())
			defer resp.Body.Close()
			respBody, err := io.ReadAll(resp.Body)
			Expect(err).To(BeNil())
			Expect(string(respBody)).To(Equal("Received: testdata"))
			Expect(requestsReceived).To(Equal(1))

			// Same request, should hit cache
			body = strings.NewReader("testdata")
			r, err = http.NewRequest("POST", ts.URL, body)
			Expect(err).To(BeNil())
			resp, err = chc.Do(r)
			Expect(err).To(BeNil())
			defer resp.Body.Close()
			respBody, err = io.ReadAll(resp.Body)
			Expect(err).To(BeNil())
			Expect(string(respBody)).To(Equal("Received: testdata"))
			Expect(requestsReceived).To(Equal(1))

			// Different body, should miss cache
			body = strings.NewReader("different")
			r, err = http.NewRequest("POST", ts.URL, body)
			Expect(err).To(BeNil())
			resp, err = chc.Do(r)
			Expect(err).To(BeNil())
			defer resp.Body.Close()
			respBody, err = io.ReadAll(resp.Body)
			Expect(err).To(BeNil())
			Expect(string(respBody)).To(Equal("Received: different"))
			Expect(requestsReceived).To(Equal(2))
		})
	})
})
