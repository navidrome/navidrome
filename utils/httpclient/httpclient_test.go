package httpclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/utils/httpclient"
	"github.com/navidrome/navidrome/utils/netguard"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("httpclient", func() {
	var ctx context.Context

	var server *httptest.Server
	var receivedUA string

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		receivedUA = ""
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedUA = r.Header.Get("User-Agent")
		}))
		DeferCleanup(server.Close)
	})

	Describe("New", func() {
		It("sets the Navidrome User-Agent when the request has none", func() {
			c := httpclient.New(time.Second)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			Expect(err).ToNot(HaveOccurred())
			resp, err := c.Do(req)
			Expect(err).ToNot(HaveOccurred())
			resp.Body.Close()
			Expect(receivedUA).To(Equal(consts.HTTPUserAgent))
		})

		It("keeps a User-Agent already set by the caller", func() {
			c := httpclient.New(time.Second)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
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

	Describe("NewExternal", func() {
		It("refuses to connect to a loopback server", func() {
			c := httpclient.NewExternal(time.Second)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			Expect(err).ToNot(HaveOccurred())
			_, err = c.Do(req)
			Expect(err).To(MatchError(netguard.ErrPrivateAddress))
			Expect(receivedUA).To(BeEmpty())
		})

		It("refuses a redirect from an allowed host to a private address", func() {
			redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "http://169.254.169.254/latest/meta-data/", http.StatusFound)
			}))
			DeferCleanup(redirector.Close)

			c := httpclient.NewExternal(time.Second, netip.MustParsePrefix("127.0.0.1/32"))
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, redirector.URL, nil)
			Expect(err).ToNot(HaveOccurred())
			_, err = c.Do(req)
			Expect(err).To(MatchError(netguard.ErrPrivateAddress))
		})

		It("connects to addresses covered by an allowed prefix and sets the User-Agent", func() {
			c := httpclient.NewExternal(time.Second, netip.MustParsePrefix("127.0.0.0/8"))
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			Expect(err).ToNot(HaveOccurred())
			resp, err := c.Do(req)
			Expect(err).ToNot(HaveOccurred())
			resp.Body.Close()
			Expect(receivedUA).To(Equal(consts.HTTPUserAgent))
		})
	})

	Describe("NewTransport", func() {
		It("uses the default transport when base is nil", func() {
			c := &http.Client{Transport: httpclient.NewTransport(nil)}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			Expect(err).ToNot(HaveOccurred())
			resp, err := c.Do(req)
			Expect(err).ToNot(HaveOccurred())
			resp.Body.Close()
			Expect(receivedUA).To(Equal(consts.HTTPUserAgent))
		})

		It("does not modify the original request", func() {
			c := &http.Client{Transport: httpclient.NewTransport(nil)}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
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
})
