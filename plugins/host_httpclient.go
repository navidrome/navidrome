package plugins

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/plugins/streams"
	"github.com/navidrome/navidrome/utils/httpclient"
	"github.com/navidrome/navidrome/utils/netguard"
)

const (
	httpClientDefaultTimeout     = 10 * time.Second
	httpClientMaxRedirects       = 5
	httpClientMaxResponseBodyLen = 10 * 1024 * 1024 // 10 MB
)

// contextKey is used for per-request redirect control via context.
type contextKey struct{}

// noFollowRedirectsKey signals the CheckRedirect callback to stop following redirects.
var noFollowRedirectsKey = contextKey{}

// httpServiceImpl implements host.HTTPService.
type httpServiceImpl struct {
	pluginName    string
	requiredHosts []string
	client        *http.Client
	transport     *http.Transport
}

// newHTTPService creates a new HTTPService for a plugin.
func newHTTPService(pluginName string, permission *HTTPPermission) *httpServiceImpl {
	var requiredHosts []string
	if permission != nil {
		requiredHosts = permission.RequiredHosts
	}
	svc := &httpServiceImpl{
		pluginName:    pluginName,
		requiredHosts: requiredHosts,
	}
	svc.transport = http.DefaultTransport.(*http.Transport).Clone()
	svc.transport.DialContext = (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Control:   svc.dialControl,
		Resolver:  dialResolver,
	}).DialContext
	// No client timeout: it is set per-request via context deadline.
	svc.client = &http.Client{Transport: httpclient.NewTransport(svc.transport)}
	svc.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.Context().Value(noFollowRedirectsKey) != nil {
			return http.ErrUseLastResponse
		}
		if len(via) >= httpClientMaxRedirects {
			log.Warn(req.Context(), "HTTP redirect limit exceeded", "plugin", svc.pluginName, "url", req.URL.String(), "redirectCount", len(via))
			return http.ErrUseLastResponse
		}
		if err := svc.validateHost(req.Context(), req.URL.Host); err != nil {
			log.Warn(req.Context(), "HTTP redirect blocked", "plugin", svc.pluginName, "url", req.URL.String(), "err", err)
			return err
		}
		return nil
	}
	return svc
}

// Close releases the plugin's pooled connections when the plugin is unloaded.
func (s *httpServiceImpl) Close() error {
	s.transport.CloseIdleConnections()
	return nil
}

func (s *httpServiceImpl) Send(ctx context.Context, request host.HTTPRequest) (*host.HTTPResponse, error) {
	if err := s.validateURL(ctx, request.URL); err != nil {
		return nil, err
	}

	// Apply per-request timeout via context deadline
	timeout := cmp.Or(time.Duration(request.TimeoutMs)*time.Millisecond, httpClientDefaultTimeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Signal CheckRedirect to not follow redirects for this request
	if request.NoFollowRedirects {
		ctx = context.WithValue(ctx, noFollowRedirectsKey, true)
	}

	// Build request body
	method := strings.ToUpper(request.Method)
	var body io.Reader
	if len(request.Body) > 0 {
		body = bytes.NewReader(request.Body)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, method, request.URL, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	for k, v := range request.Headers {
		httpReq.Header.Set(k, v)
	}

	// Execute request
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	log.Trace(ctx, "HTTP request", "plugin", s.pluginName, "method", method, "url", request.URL, "status", resp.StatusCode)

	// Read response body (with size limit to prevent memory exhaustion)
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, httpClientMaxResponseBodyLen))
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return &host.HTTPResponse{
		StatusCode: int32(resp.StatusCode),
		Headers:    flattenHeaders(resp.Header),
		Body:       respBody,
	}, nil
}

var errHeaderTimeout = fmt.Errorf("timed out waiting for the response headers: %w", context.DeadlineExceeded)

// SendStream ties the request to the call through a pending entry in its scope until the headers arrive.
// The request ctx is detached, so a response body handed to host code does not keep the plugin instance alive.
func (s *httpServiceImpl) SendStream(ctx context.Context, request host.HTTPStreamRequest) (*host.HTTPStreamResponse, error) {
	reqCtx, cancel := context.WithCancel(streams.Detached(ctx))
	pending := &pendingRequest{cancel: cancel, body: request.Body}
	release, err := streams.Hold(ctx, pending)
	if err != nil {
		return nil, err
	}
	defer release()

	httpReq, err := s.newStreamRequest(ctx, reqCtx, request)
	if err != nil {
		_ = pending.Close()
		return nil, err
	}

	timeout := cmp.Or(time.Duration(request.TimeoutMs)*time.Millisecond, httpClientDefaultTimeout)
	timer := time.AfterFunc(timeout, func() { pending.abort(errHeaderTimeout) })
	defer timer.Stop()

	resp, err := s.client.Do(httpReq)
	if reason := pending.promote(); reason != nil {
		if resp != nil {
			_ = resp.Body.Close()
		}
		return nil, reason
	}
	if err != nil {
		cancel()
		return nil, err
	}
	log.Trace(ctx, "HTTP stream request", "plugin", s.pluginName, "method", httpReq.Method, "url", request.URL, "status", resp.StatusCode)
	return &host.HTTPStreamResponse{
		StatusCode: int32(resp.StatusCode),
		Headers:    flattenHeaders(resp.Header),
		Body:       &responseBody{ReadCloser: resp.Body, cancel: cancel},
	}, nil
}

func (s *httpServiceImpl) newStreamRequest(ctx, reqCtx context.Context, request host.HTTPStreamRequest) (*http.Request, error) {
	if err := s.validateURL(ctx, request.URL); err != nil {
		return nil, err
	}
	if request.NoFollowRedirects {
		reqCtx = context.WithValue(reqCtx, noFollowRedirectsKey, true)
	}
	httpReq, err := http.NewRequestWithContext(reqCtx, strings.ToUpper(request.Method), request.URL, request.Body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	for k, v := range request.Headers {
		httpReq.Header.Set(k, v)
	}
	// net/http sends ContentLength, not the header; without it a body goes chunked, which S3-style PUTs refuse.
	if cl := httpReq.Header.Get("Content-Length"); cl != "" && request.Body != nil {
		n, err := strconv.ParseInt(cl, 10, 64)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("invalid Content-Length %q", cl)
		}
		httpReq.ContentLength = n
	}
	return httpReq, nil
}

type pendingState uint8

const (
	pendingWaiting pendingState = iota
	pendingPromoted
	pendingCancelled
)

// pendingRequest is a SendStream request waiting for its response headers. One switch decides between promote
// (headers arrived) and abort (header timeout or the call ended); the loser's action is a no-op.
type pendingRequest struct {
	mu     sync.Mutex
	state  pendingState
	reason error
	cancel context.CancelFunc
	body   io.Closer
}

// abort cancels the request and closes its body: cancelling alone does not unblock a writer stuck reading the body.
func (p *pendingRequest) abort(reason error) {
	p.mu.Lock()
	if p.state != pendingWaiting {
		p.mu.Unlock()
		return
	}
	p.state, p.reason = pendingCancelled, reason
	p.mu.Unlock()
	p.cancel()
	if p.body != nil {
		_ = p.body.Close()
	}
}

// promote returns the abort reason if abort won.
func (p *pendingRequest) promote() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state == pendingCancelled {
		return p.reason
	}
	p.state = pendingPromoted
	return nil
}

func (p *pendingRequest) Close() error {
	p.abort(errCallEnded)
	return nil
}

// responseBody ends the request when closed, since the request ctx outlives the call.
type responseBody struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b *responseBody) Close() error {
	err := b.ReadCloser.Close()
	b.cancel()
	return err
}

func (s *httpServiceImpl) validateURL(ctx context.Context, rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme %q: must be http or https", parsedURL.Scheme)
	}
	return s.validateHost(ctx, parsedURL.Host)
}

// flattenHeaders keeps the first value of each header.
func flattenHeaders(h http.Header) map[string]string {
	headers := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	return headers
}

// validateHost checks whether a request to the given host is permitted.
// When requiredHosts is set, it checks against the allowlist.
// When requiredHosts is empty, it blocks private/loopback IPs to prevent SSRF.
func (s *httpServiceImpl) validateHost(ctx context.Context, hostStr string) error {
	hostname := extractHostname(hostStr)

	if len(s.requiredHosts) > 0 {
		if !isHostInAllowlist(s.requiredHosts, hostname) {
			return fmt.Errorf("host %q is not allowed", hostStr)
		}
		return nil
	}

	// No explicit allowlist: block private/loopback IPs
	if isPrivateOrLoopback(hostname) {
		log.Warn(ctx, "HTTP request to private/loopback address blocked", "plugin", s.pluginName, "host", hostStr)
		return fmt.Errorf("host %q is not allowed: private/loopback addresses require explicit requiredHosts in manifest", hostStr)
	}
	return nil
}

func (s *httpServiceImpl) dialControl(_, address string, _ syscall.RawConn) error {
	return checkPrivateDial(s.requiredHosts, address)
}

// extractHostname returns the hostname portion of a host string, stripping
// any port number and IPv6 brackets. It handles IPv6 addresses correctly
// (e.g. "[::1]:8080" → "::1", "[::1]" → "::1").
func extractHostname(hostStr string) string {
	if h, _, err := net.SplitHostPort(hostStr); err == nil {
		return h
	}
	// Strip IPv6 brackets when no port is present (e.g. "[::1]" → "::1")
	if strings.HasPrefix(hostStr, "[") && strings.HasSuffix(hostStr, "]") {
		return hostStr[1 : len(hostStr)-1]
	}
	return hostStr
}

// isPrivateOrLoopback is a pre-flight check on the literal host (IP or "localhost"); it does not
// resolve names, so dialControl remains the real guard.
func isPrivateOrLoopback(hostname string) bool {
	if strings.EqualFold(hostname, "localhost") {
		return true
	}
	ip := net.ParseIP(hostname)
	if ip == nil {
		return false
	}
	return netguard.IsPrivateIP(ip)
}

// Verify interface implementation
var _ host.HTTPService = (*httpServiceImpl)(nil)
