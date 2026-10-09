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

type proxyAddrKey struct{}

// NewExternal is New for URLs from untrusted sources: it refuses to dial private, loopback,
// link-local and unspecified addresses, except those covered by allowed.
func NewExternal(timeout time.Duration, allowed ...netip.Prefix) *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = proxyFunc
	direct := net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	guarded := direct
	guarded.Control = netguard.DialControl(allowed...)
	t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		// Exempt the hop to the proxy, which relays the request and is operator config. A URL
		// aimed at the proxy's own address is not proxied, so it stays guarded.
		if proxy, ok := ctx.Value(proxyAddrKey{}).(string); ok && proxy == addr {
			return direct.DialContext(ctx, network, addr)
		}
		return guarded.DialContext(ctx, network, addr)
	}
	return &http.Client{Timeout: timeout, Transport: NewTransport(&proxyTagger{base: t})}
}

// proxyTagger records the proxy each request resolves to, so the dialer can tell a hop to the
// proxy from a dial to the URL's own host.
type proxyTagger struct{ base http.RoundTripper }

func (p *proxyTagger) RoundTrip(req *http.Request) (*http.Response, error) {
	if u, err := proxyFunc(req); err == nil && u != nil {
		req = req.WithContext(context.WithValue(req.Context(), proxyAddrKey{}, proxyAddr(u)))
	}
	return p.base.RoundTrip(req)
}

// proxyAddr mirrors how net/http addresses a proxy connection.
func proxyAddr(u *url.URL) string {
	port := u.Port()
	if port == "" {
		port = map[string]string{"http": "80", "https": "443", "socks5": "1080", "socks5h": "1080"}[u.Scheme]
	}
	return net.JoinHostPort(u.Hostname(), port)
}
