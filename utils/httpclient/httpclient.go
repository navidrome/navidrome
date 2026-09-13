// Package httpclient provides a shared http.Client factory that identifies
// Navidrome via the User-Agent header on all outgoing requests.
package httpclient

import (
	"net"
	"net/http"
	"net/netip"
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

// NewExternal is New for URLs from untrusted sources: it refuses to dial private, loopback, link-local
// and unspecified addresses, except those covered by allowed.
func NewExternal(timeout time.Duration, allowed ...netip.Prefix) *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DialContext = (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Control:   netguard.DialControl(allowed...),
	}).DialContext
	return &http.Client{Timeout: timeout, Transport: NewTransport(t)}
}
