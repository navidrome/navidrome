// Package httpclient provides a shared http.Client factory that identifies
// Navidrome via the User-Agent header on all outgoing requests.
package httpclient

import (
	"net/http"
	"time"

	"github.com/navidrome/navidrome/consts"
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
