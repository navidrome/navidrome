// Package httpclient provides a shared http.Client factory that identifies
// Navidrome via the User-Agent header on all outgoing requests.
package httpclient

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/utils/netguard"
)

type uaTransport struct {
	base http.RoundTripper
}

func (t *uaTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if _, ok := req.Header["User-Agent"]; !ok {
		req = req.Clone(req.Context())
		req.Header.Set("User-Agent", consts.HTTPUserAgent)
	}
	return t.base.RoundTrip(req)
}

// NewTransport wraps base (or http.DefaultTransport if nil) to set the
// Navidrome User-Agent on requests that don't have one.
func NewTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &uaTransport{base: base}
}

func New(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, Transport: NewTransport(nil)}
}

// proxyFunc resolves the proxy for a request; tests replace it.
var proxyFunc = http.ProxyFromEnvironment

// NewExternal is New for URLs from untrusted sources: it refuses to dial private, loopback,
// link-local and unspecified addresses, except those covered by allowed.
func NewExternal(timeout time.Duration, allowed ...netip.Prefix) *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = proxyFunc
	direct := net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	guarded := direct
	guarded.Control = netguard.DialControl(allowed...)
	proxies := proxyEndpoints()
	t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		// A proxy relays the request itself, so the guard cannot see the target through it;
		// refusing the operator's own proxy would only break every fetch.
		if proxies[addr] {
			return direct.DialContext(ctx, network, addr)
		}
		return guarded.DialContext(ctx, network, addr)
	}
	return &http.Client{Timeout: timeout, Transport: NewTransport(t)}
}

// proxyEndpoints lists the addresses the configured proxies are reached at.
func proxyEndpoints() map[string]bool {
	endpoints := map[string]bool{}
	for _, scheme := range []string{"http", "https"} {
		u, err := proxyFunc(&http.Request{URL: &url.URL{Scheme: scheme, Host: "navidrome.example.com"}})
		if err != nil || u == nil {
			continue
		}
		endpoints[proxyAddr(u)] = true
	}
	return endpoints
}

// proxyAddr mirrors how net/http addresses a proxy connection.
func proxyAddr(u *url.URL) string {
	port := u.Port()
	if port == "" {
		port = map[string]string{"http": "80", "https": "443", "socks5": "1080", "socks5h": "1080"}[u.Scheme]
	}
	return net.JoinHostPort(u.Hostname(), port)
}
