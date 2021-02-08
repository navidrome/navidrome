package agents

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/navidrome/navidrome/consts"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CachedHttpClient", func() {
	Context("Default TTL", func() {
		var chc *CachedHTTPClient
		var ts *httptest.Server
		var requestsReceived int

		BeforeEach(func() {
			ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestsReceived++
				_, _ = fmt.Fprintf(w, "Hello, %s", r.URL.Query()["name"])
			}))
			chc = NewCachedHTTPClient(http.DefaultClient, consts.DefaultCachedHttpClientTTL)
		})

		AfterEach(func() {
			defer ts.Close()
		})

		It("caches repeated requests", func() {
			r, _ := http.NewRequest("GET", ts.URL+"?name=doe", nil)
			resp, err := chc.Do(r)
			Expect(err).To(BeNil())
			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).To(BeNil())
			Expect(string(body)).To(Equal("Hello, [doe]"))
			Expect(requestsReceived).To(Equal(1))

			// Same request
			r, _ = http.NewRequest("GET", ts.URL+"?name=doe", nil)
			resp, err = chc.Do(r)
			Expect(err).To(BeNil())
			body, err = ioutil.ReadAll(resp.Body)
			Expect(err).To(BeNil())
			Expect(string(body)).To(Equal("Hello, [doe]"))
			Expect(requestsReceived).To(Equal(1))

			// Different request
			r, _ = http.NewRequest("GET", ts.URL, nil)
			resp, err = chc.Do(r)
			Expect(err).To(BeNil())
			body, err = ioutil.ReadAll(resp.Body)
			Expect(err).To(BeNil())
			Expect(string(body)).To(Equal("Hello, []"))
			Expect(requestsReceived).To(Equal(2))
		})

		It("expires responses after TTL", func() {
			requestsReceived = 0
			chc = NewCachedHTTPClient(http.DefaultClient, 10*time.Millisecond)

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
