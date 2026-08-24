// Package httpclient provides a shared http.Client factory that identifies
// Navidrome via the User-Agent header on all outgoing requests.
package httpclient

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/navidrome/navidrome/consts"
)

// maxRetryAfterSeconds only keeps the conversion to nanoseconds below from wrapping;
// callers apply their own policy cap.
const maxRetryAfterSeconds int64 = math.MaxInt64 / int64(time.Second)

// RetryAfter reads the wait a rate-limited server asked for, in the delta-seconds form:
// X-RateLimit-Reset-In takes precedence, then Retry-After. Absent or unparseable means 0.
func RetryAfter(h http.Header) time.Duration {
	for _, name := range []string{"X-RateLimit-Reset-In", "Retry-After"} {
		if secs, err := strconv.Atoi(h.Get(name)); err == nil && secs > 0 {
			return time.Duration(min(int64(secs), maxRetryAfterSeconds)) * time.Second
		}
	}
	return 0
}

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
