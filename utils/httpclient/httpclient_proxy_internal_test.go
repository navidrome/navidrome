package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/navidrome/navidrome/utils/netguard"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NewExternal with a proxy", func() {
	var ctx context.Context

	var proxy *httptest.Server
	var proxied atomic.Int32
	var proxyURL *url.URL

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		proxied.Store(0)
		proxy = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			proxied.Add(1)
			_, _ = w.Write([]byte("via proxy"))
		}))
		DeferCleanup(proxy.Close)
		proxyURL, _ = url.Parse(proxy.URL)
		prev := proxyFunc
		DeferCleanup(func() { proxyFunc = prev })
	})

	It("dials the configured proxy even though it listens on a private address", func() {
		proxyFunc = func(*http.Request) (*url.URL, error) { return proxyURL, nil }

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://navidrome.example.com/cover.jpg", nil)
		Expect(err).ToNot(HaveOccurred())
		resp, err := NewExternal(time.Second).Do(req)
		Expect(err).ToNot(HaveOccurred())
		defer resp.Body.Close()
		Expect(proxied.Load()).To(Equal(int32(1)))
	})

	// net/http never proxies loopback targets, so a URL aimed at a loopback proxy is dialed directly.
	It("refuses a direct dial to the proxy's own address when the request is not proxied", func() {
		proxyFunc = func(r *http.Request) (*url.URL, error) {
			if r.URL.Hostname() == "127.0.0.1" {
				return nil, nil
			}
			return proxyURL, nil
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, proxy.URL+"/secret", nil)
		Expect(err).ToNot(HaveOccurred())
		_, err = NewExternal(time.Second).Do(req)
		Expect(err).To(MatchError(netguard.ErrPrivateAddress))
		Expect(proxied.Load()).To(BeZero())
	})

	It("still refuses a direct dial to a private address", func() {
		proxyFunc = func(r *http.Request) (*url.URL, error) {
			if r.URL.Scheme == "https" {
				return proxyURL, nil
			}
			return nil, nil
		}
		target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		DeferCleanup(target.Close)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.URL, nil)
		Expect(err).ToNot(HaveOccurred())
		_, err = NewExternal(time.Second).Do(req)
		Expect(err).To(MatchError(netguard.ErrPrivateAddress))
	})
})
