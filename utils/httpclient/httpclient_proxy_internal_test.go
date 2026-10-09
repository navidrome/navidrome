package httpclient

import (
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
	var proxy *httptest.Server
	var proxied atomic.Int32
	var proxyURL *url.URL

	BeforeEach(func() {
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

		resp, err := NewExternal(time.Second).Get("http://navidrome.example.com/cover.jpg")
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

		_, err := NewExternal(time.Second).Get(proxy.URL + "/secret")
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

		_, err := NewExternal(time.Second).Get(target.URL)
		Expect(err).To(MatchError(netguard.ErrPrivateAddress))
	})
})
