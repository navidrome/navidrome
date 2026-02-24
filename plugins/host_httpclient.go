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
	"strings"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/host"
)

const (
	httpClientDefaultTimeout     = 10 * time.Second
	httpClientMaxRedirects       = 5
	httpClientMaxResponseBodyLen = 10 * 1024 * 1024 // 10 MB
)

// httpClientServiceImpl implements host.HttpClientService.
type httpClientServiceImpl struct {
	pluginName    string
	requiredHosts []string
	client        *http.Client
}

// newHttpClientService creates a new HttpClientService for a plugin.
func newHttpClientService(pluginName string, permission *HTTPPermission) *httpClientServiceImpl {
	var requiredHosts []string
	if permission != nil {
		requiredHosts = permission.RequiredHosts
	}
	svc := &httpClientServiceImpl{
		pluginName:    pluginName,
		requiredHosts: requiredHosts,
	}
	svc.client = &http.Client{
		Transport: http.DefaultTransport,
		// Timeout is set per-request via context deadline, not here.
		// CheckRedirect validates hosts and enforces redirect limits.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= httpClientMaxRedirects {
				log.Warn(req.Context(), "HTTP redirect limit exceeded", "plugin", svc.pluginName, "url", req.URL.String(), "redirectCount", len(via))
				return http.ErrUseLastResponse
			}
			if err := svc.validateHost(req.Context(), req.URL.Host); err != nil {
				log.Warn(req.Context(), "HTTP redirect blocked", "plugin", svc.pluginName, "url", req.URL.String(), "err", err)
				return err
			}
			return nil
		},
	}
	return svc
}

func (s *httpClientServiceImpl) Do(ctx context.Context, request host.HttpRequest) (*host.HttpResponse, error) {
	// Parse and validate URL
	parsedURL, err := url.Parse(request.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Validate URL scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("invalid URL scheme %q: must be http or https", parsedURL.Scheme)
	}

	// Validate host against allowed hosts and private IP restrictions
	if err := s.validateHost(ctx, parsedURL.Host); err != nil {
		return nil, err
	}

	// Apply per-request timeout via context deadline
	timeout := cmp.Or(time.Duration(request.TimeoutMs)*time.Millisecond, httpClientDefaultTimeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

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
	resp, err := s.client.Do(httpReq) //nolint:gosec // URL is validated against requiredHosts
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	log.Trace(ctx, "HttpClient request", "plugin", s.pluginName, "method", method, "url", request.URL, "status", resp.StatusCode)

	// Read response body (with size limit to prevent memory exhaustion)
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, httpClientMaxResponseBodyLen))
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	// Flatten response headers (first value only)
	headers := make(map[string]string, len(resp.Header))
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	return &host.HttpResponse{
		StatusCode: int32(resp.StatusCode),
		Headers:    headers,
		Body:       respBody,
	}, nil
}

// validateHost checks whether a request to the given host is permitted.
// When requiredHosts is set, it checks against the allowlist.
// When requiredHosts is empty, it blocks private/loopback IPs to prevent SSRF.
func (s *httpClientServiceImpl) validateHost(ctx context.Context, hostStr string) error {
	hostname := extractHostname(hostStr)

	if len(s.requiredHosts) > 0 {
		if !s.isHostAllowed(hostname) {
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

func (s *httpClientServiceImpl) isHostAllowed(hostname string) bool {
	for _, pattern := range s.requiredHosts {
		if matchHostPattern(pattern, hostname) {
			return true
		}
	}
	return false
}

// extractHostname returns the hostname portion of a host string, stripping
// any port number. It handles IPv6 addresses correctly (e.g. "[::1]:8080").
func extractHostname(hostStr string) string {
	if h, _, err := net.SplitHostPort(hostStr); err == nil {
		return h
	}
	return hostStr
}

// isPrivateOrLoopback returns true if the given hostname resolves to or is
// a private, loopback, or link-local IP address. This includes:
// IPv4: 127.0.0.0/8, 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 169.254.0.0/16
// IPv6: ::1, fc00::/7, fe80::/10
// It also blocks "localhost" by name.
func isPrivateOrLoopback(hostname string) bool {
	if strings.EqualFold(hostname, "localhost") {
		return true
	}
	ip := net.ParseIP(hostname)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

// Verify interface implementation
var _ host.HttpClientService = (*httpClientServiceImpl)(nil)
