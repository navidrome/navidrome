package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/lifecycle"
	"github.com/navidrome/navidrome/core/rustworker"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils/httpclient"
	"github.com/navidrome/navidrome/utils/singleton"
)

var (
	errCircuitOpen       = errors.New("integration circuit open")
	errWorkerUnavailable = errors.New("integration gRPC worker unavailable")
)

// Gateway is the hub for all outbound HTTP. Adapters no longer own
// point-to-point http.Client instances: they share this RoundTripper, which
// prefers the Rust gRPC worker (async I/O, pooling, signing) and falls back
// to Go net/http when the worker is unavailable. DestArtwork uses a separate
// SSRF-safe fallback so image CDNs never skip the private-IP checks.
//
// When the gRPC worker is active, circuit breaking is owned by the worker.
// The Go-side breaker only protects the plain-HTTP fallback path. Production
// does not fall back to Go HTTP when an active worker call fails.
type Gateway struct {
	fallback        http.RoundTripper
	artworkFallback http.RoundTripper
	breakers        sync.Map // Destination -> *circuitBreaker
	grpcMu          sync.Mutex
	grpc            *grpcClient
	workerExpected  bool
	shutdown        bool
	restarts        int
}

func Get() *Gateway {
	return singleton.GetInstance(func() *Gateway {
		return newGateway()
	})
}

func newGateway() *Gateway {
	g := &Gateway{
		fallback:        httpclient.NewTransport(nil),
		artworkFallback: &boundedRoundTripper{inner: httpclient.NewTransport(newSSRFTransport()), limit: maxArtworkBodyBytes},
		workerExpected:  conf.Server.Integration.Enabled,
	}
	if conf.Server.Integration.Enabled {
		if client, err := startGRPCClient(context.Background()); err != nil {
			if errors.Is(err, rustworker.ErrSkippedInTests) {
				g.workerExpected = false
			} else {
				log.Error("Rust integration gRPC worker unavailable", err)
			}
		} else {
			g.grpc = client
			g.attachWorker(client)
			log.Info("Outbound HTTP routed through Rust gRPC integration worker")
		}
	}
	lifecycle.Register(g)
	return g
}

func (g *Gateway) HTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = consts.DefaultHttpClientTimeOut
	}
	return &http.Client{Timeout: timeout, Transport: httpclient.NewTransport(g)}
}

func HTTPClient(timeout time.Duration) *http.Client {
	return Get().HTTPClient(timeout)
}

// ArtworkHTTPClient fetches attacker-influenced image URLs through DestArtwork.
// Production prefers the Rust worker (SSRF in the worker); tests and worker
// outages use the Go SSRF transport.
func (g *Gateway) ArtworkHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &http.Client{
		Timeout:       timeout,
		Transport:     destTransport{g: g, dest: DestArtwork},
		CheckRedirect: artworkRedirectCheck,
	}
}

func ArtworkHTTPClient(timeout time.Duration) *http.Client {
	return Get().ArtworkHTTPClient(timeout)
}

type destTransport struct {
	g    *Gateway
	dest Destination
}

func (t destTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.g.roundTripDest(req, t.dest)
}

func (g *Gateway) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil || req.URL == nil {
		return nil, errors.New("nil request")
	}
	return g.roundTripDest(req, DestinationFromHost(req.URL.Host))
}

func allowHTTPFallback() bool {
	return false
}

func (g *Gateway) roundTripDest(req *http.Request, dest Destination) (*http.Response, error) {
	if req == nil {
		return nil, errors.New("nil request")
	}
	if dest == "" {
		dest = DestUnknown
	}

	if dest == DestUnknown && g.workerExpected && !allowHTTPFallback() {
		host := ""
		if req.URL != nil {
			host = req.URL.Host
		}
		return nil, fmt.Errorf("%w: unknown outbound host %q", errWorkerUnavailable, host)
	}

	fallback := g.fallback
	if dest == DestArtwork && g.artworkFallback != nil {
		fallback = g.artworkFallback
	}

	g.grpcMu.Lock()
	client := g.grpc
	g.grpcMu.Unlock()

	useGRPC := client != nil && dest != DestUnknown
	if useGRPC {
		resp, err := client.roundTrip(req.Context(), dest, req)
		if err == nil {
			return resp, nil
		}
		if isWorkerCircuitOpen(err) {
			return nil, err
		}
		if isWorkerTransportFailure(err) {
			g.tryRestartWorker(err)
		}
		if !allowHTTPFallback() {
			return nil, err
		}
		log.Warn(req.Context(), "gRPC outbound failed, falling back to Go HTTP", "dest", dest, err)
	} else if g.workerExpected && dest != DestUnknown && !allowHTTPFallback() {
		return nil, fmt.Errorf("%w: %s", errWorkerUnavailable, dest)
	}

	breaker := g.breaker(dest)
	if !breaker.allow() {
		return nil, fmt.Errorf("%w: %s", errCircuitOpen, dest)
	}

	rt := fallback
	if dest != DestArtwork {
		rt = &boundedRoundTripper{inner: fallback, limit: maxResponseBody(dest)}
	}
	resp, err := rt.RoundTrip(req)
	if err != nil {
		breaker.failure()
		return nil, err
	}
	if resp.StatusCode >= 500 {
		breaker.failure()
	} else {
		breaker.success()
	}
	return resp, nil
}

func (g *Gateway) WorkerReady() bool {
	g.grpcMu.Lock()
	defer g.grpcMu.Unlock()
	return g.grpc != nil
}

func (g *Gateway) Close() {
	g.grpcMu.Lock()
	defer g.grpcMu.Unlock()
	g.shutdown = true
	if g.grpc != nil {
		g.grpc.close()
		g.grpc = nil
	}
}

func (g *Gateway) Sign(ctx context.Context, params map[string]string, secret string) string {
	g.grpcMu.Lock()
	client := g.grpc
	g.grpcMu.Unlock()
	if client != nil {
		sig, err := client.sign(ctx, params, secret)
		if err == nil && sig != "" {
			return sig
		}
		if g.workerExpected && !allowHTTPFallback() {
			log.Error(ctx, "Integration worker sign failed; refusing Go fallback in production", err)
			return ""
		}
		if err != nil {
			log.Warn(ctx, "Integration worker sign failed; using Go fallback", err)
		}
	} else if g.workerExpected && !allowHTTPFallback() {
		log.Error(ctx, "Integration worker sign unavailable in production", nil)
		return ""
	}
	return signAudioscrobbler(params, secret)
}

func Sign(ctx context.Context, params map[string]string, secret string) string {
	return Get().Sign(ctx, params, secret)
}

func (g *Gateway) breaker(dest Destination) *circuitBreaker {
	actual, _ := g.breakers.LoadOrStore(dest, &circuitBreaker{})
	return actual.(*circuitBreaker)
}

// boundedRoundTripper caps fallback response bodies.
type boundedRoundTripper struct {
	inner http.RoundTripper
	limit int64
}

func (t *boundedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.inner.RoundTrip(req)
	if err != nil || resp == nil || resp.Body == nil {
		return resp, err
	}
	limit := t.limit
	if limit <= 0 {
		limit = maxArtworkBodyBytes
	}
	if resp.ContentLength > limit {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("integration response exceeds %d bytes", limit)
	}
	resp.Body = &boundedBody{reader: resp.Body, remaining: limit, limit: limit}
	return resp, nil
}

type boundedBody struct {
	reader    io.ReadCloser
	remaining int64
	limit     int64
}

func (b *boundedBody) Read(p []byte) (int, error) {
	if b.remaining <= 0 {
		return 0, fmt.Errorf("integration response exceeds %d bytes", b.limit)
	}
	if int64(len(p)) > b.remaining {
		p = p[:b.remaining]
	}
	n, err := b.reader.Read(p)
	b.remaining -= int64(n)
	return n, err
}

func (b *boundedBody) Close() error { return b.reader.Close() }
