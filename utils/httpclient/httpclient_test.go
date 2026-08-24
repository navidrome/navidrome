package httpclient_test

import (
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/utils/httpclient"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("httpclient", func() {
	var server *httptest.Server
	var receivedUA string

	BeforeEach(func() {
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedUA = r.Header.Get("User-Agent")
		}))
		DeferCleanup(server.Close)
	})

	Describe("New", func() {
		It("sets the Navidrome User-Agent when the request has none", func() {
			c := httpclient.New(time.Second)
			resp, err := c.Get(server.URL)
			Expect(err).ToNot(HaveOccurred())
			resp.Body.Close()
			Expect(receivedUA).To(Equal(consts.HTTPUserAgent))
		})

		It("keeps a User-Agent already set by the caller", func() {
			c := httpclient.New(time.Second)
			req, err := http.NewRequest(http.MethodGet, server.URL, nil)
			Expect(err).ToNot(HaveOccurred())
			req.Header.Set("User-Agent", "CustomAgent/1.0")
			resp, err := c.Do(req)
			Expect(err).ToNot(HaveOccurred())
			resp.Body.Close()
			Expect(receivedUA).To(Equal("CustomAgent/1.0"))
		})

		It("applies the given timeout", func() {
			c := httpclient.New(5 * time.Second)
			Expect(c.Timeout).To(Equal(5 * time.Second))
		})
	})

	Describe("NewTransport", func() {
		It("uses the default transport when base is nil", func() {
			c := &http.Client{Transport: httpclient.NewTransport(nil)}
			resp, err := c.Get(server.URL)
			Expect(err).ToNot(HaveOccurred())
			resp.Body.Close()
			Expect(receivedUA).To(Equal(consts.HTTPUserAgent))
		})

		It("does not modify the original request", func() {
			c := &http.Client{Transport: httpclient.NewTransport(nil)}
			req, err := http.NewRequest(http.MethodGet, server.URL, nil)
			Expect(err).ToNot(HaveOccurred())
			resp, err := c.Do(req)
			Expect(err).ToNot(HaveOccurred())
			resp.Body.Close()
			Expect(req.Header).ToNot(HaveKey("User-Agent"))
		})
	})

	Describe("HTTPUserAgent", func() {
		It("identifies Navidrome with version and project URL", func() {
			Expect(consts.HTTPUserAgent).To(Equal("Navidrome/" + consts.Version + " - https://github.com/navidrome"))
		})
	})

	Describe("RetryAfter", func() {
		DescribeTable("reads the delta-seconds the server asked for",
			func(header http.Header, expected time.Duration) {
				Expect(httpclient.RetryAfter(header)).To(Equal(expected))
			},
			Entry("no headers", http.Header{}, time.Duration(0)),
			Entry("X-RateLimit-Reset-In", http.Header{"X-Ratelimit-Reset-In": []string{"3"}}, 3*time.Second),
			Entry("Retry-After", http.Header{"Retry-After": []string{"12"}}, 12*time.Second),
			Entry("X-RateLimit-Reset-In wins",
				http.Header{"X-Ratelimit-Reset-In": []string{"3"}, "Retry-After": []string{"12"}}, 3*time.Second),
			Entry("falls through an unparseable first header",
				http.Header{"X-Ratelimit-Reset-In": []string{"soon"}, "Retry-After": []string{"12"}}, 12*time.Second),
			Entry("HTTP-date form is not supported",
				http.Header{"Retry-After": []string{"Wed, 21 Oct 2015 07:28:00 GMT"}}, time.Duration(0)),
			Entry("zero", http.Header{"Retry-After": []string{"0"}}, time.Duration(0)),
			Entry("negative", http.Header{"Retry-After": []string{"-5"}}, time.Duration(0)),
			// Scaling before clamping would wrap past int64 nanoseconds, landing on a tiny delay.
			Entry("a value that would overflow int64 nanoseconds",
				http.Header{"Retry-After": []string{"18446744074"}}, 9223372036*time.Second),
		)
	})
})
