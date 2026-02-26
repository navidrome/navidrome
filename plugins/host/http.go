package host

import "context"

// HTTPRequest represents an outbound HTTP request from a plugin.
type HTTPRequest struct {
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      []byte            `json:"body,omitempty"`
	TimeoutMs int32             `json:"timeoutMs,omitempty"`
}

// HTTPResponse represents the response from an outbound HTTP request.
type HTTPResponse struct {
	StatusCode int32             `json:"statusCode"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       []byte            `json:"body,omitempty"`
}

// HTTPService provides outbound HTTP request capabilities for plugins.
//
// This service allows plugins to make HTTP requests to external services.
// Requests are validated against the plugin's declared requiredHosts patterns
// from the http permission in the manifest. Redirects are followed but each
// redirect destination is also validated against the allowed hosts.
//
//nd:hostservice name=HTTP permission=http
type HTTPService interface {
	// Send executes an HTTP request and returns the response.
	//
	// Parameters:
	//   - request: The HTTP request to execute, including method, URL, headers, body, and timeout
	//
	// Returns the HTTP response with status code, headers, and body.
	// Network errors, timeouts, and permission failures are returned as Go errors.
	// Successful HTTP calls (including 4xx/5xx status codes) return a non-nil response with nil error.
	//nd:hostfunc
	Send(ctx context.Context, request HTTPRequest) (*HTTPResponse, error)
}
