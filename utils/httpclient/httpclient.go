package httpclient

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/conf"
)

// New returns an HTTP client suitable for external service calls.
// When EnablePostQuantumTLS is false (the default), it disables post-quantum
// key exchange (Kyber/ML-KEM) to avoid "connection reset by peer" errors with
// servers that can't handle the larger TLS ClientHello.
// See https://github.com/golang/go/issues/70139
func New(timeout time.Duration) *http.Client {
	if conf.Server.EnablePostQuantumTLS {
		return &http.Client{Timeout: timeout}
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		MinVersion:       tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256, tls.CurveP384},
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}
