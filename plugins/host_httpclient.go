package plugins

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
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
}

// newHttpClientService creates a new HttpClientService for a plugin.
func newHttpClientService(pluginName string, permission *HTTPPermission) *httpClientServiceImpl {
	var requiredHosts []string
	if permission != nil {
		requiredHosts = permission.RequiredHosts
	}
	return &httpClientServiceImpl{
		pluginName:    pluginName,
		requiredHosts: requiredHosts,
	}
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

	// Validate host against allowed hosts
	if len(s.requiredHosts) > 0 {
		if !s.isHostAllowed(parsedURL.Host) {
			return nil, fmt.Errorf("host %q is not allowed", parsedURL.Host)
		}
	}

	// Build HTTP client with timeout
	timeout := cmp.Or(time.Duration(request.TimeoutMs)*time.Millisecond, httpClientDefaultTimeout)
	client := &http.Client{
		Timeout: timeout,
	}

	// Configure redirect policy to validate hosts
	if len(s.requiredHosts) > 0 {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= httpClientMaxRedirects {
				log.Warn(ctx, "HTTP redirect limit exceeded", "plugin", s.pluginName, "url", req.URL.String(), "redirectCount", len(via))
				return http.ErrUseLastResponse
			}
			if !s.isHostAllowed(req.URL.Host) {
				log.Warn(ctx, "HTTP redirect blocked by permissions", "plugin", s.pluginName, "url", req.URL.String())
				return fmt.Errorf("host %q is not allowed", req.URL.Host)
			}
			return nil
		}
	}

	// Build request body
	method := strings.ToUpper(request.Method)
	var body io.Reader
	if len(request.Body) > 0 && (method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch) {
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
	resp, err := client.Do(httpReq) //nolint:gosec // URL is validated against requiredHosts
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

func (s *httpClientServiceImpl) isHostAllowed(hostStr string) bool {
	// Strip port from host if present
	hostWithoutPort := hostStr
	if idx := strings.LastIndex(hostStr, ":"); idx != -1 {
		hostWithoutPort = hostStr[:idx]
	}

	for _, pattern := range s.requiredHosts {
		if matchHostPattern(pattern, hostWithoutPort) {
			return true
		}
	}
	return false
}

// Verify interface implementation
var _ host.HttpClientService = (*httpClientServiceImpl)(nil)
