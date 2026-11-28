package host

import (
	"context"
	"io"
)

// HTTPRequest represents an outbound HTTP request from a plugin.
type HTTPRequest struct {
	Method            string            `json:"method"`
	URL               string            `json:"url"`
	Headers           map[string]string `json:"headers,omitempty"`
	NoFollowRedirects bool              `json:"noFollowRedirects,omitempty"`
	Body              []byte            `json:"body,omitempty"`
	TimeoutMs         int32             `json:"timeoutMs,omitempty"`
}

// HTTPResponse represents the response from an outbound HTTP request.
type HTTPResponse struct {
	StatusCode int32             `json:"statusCode"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       []byte            `json:"body,omitempty"`
}

// HTTPStreamRequest is an outbound HTTP request whose body, if any, is a stream. The body is consumed:
// its handle stops working for the plugin once SendStream starts.
type HTTPStreamRequest struct {
	Method            string            `json:"method"`
	URL               string            `json:"url"`
	Headers           map[string]string `json:"headers,omitempty"`
	NoFollowRedirects bool              `json:"noFollowRedirects,omitempty"`
	Body              io.ReadCloser     `json:"body,omitempty"`
	TimeoutMs         int32             `json:"timeoutMs,omitempty"`
}

// HTTPStreamResponse is the response to an HTTPStreamRequest, with its body as a stream.
type HTTPStreamResponse struct {
	StatusCode int32             `json:"statusCode"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       io.ReadCloser     `json:"body,omitempty"`
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

	// SendStream executes an HTTP request with streamed request and response bodies.
	//
	// It applies the same host checks as Send, but the response body has no size limit, and
	// TimeoutMs only bounds the wait for the response headers. A request with a body does not
	// follow 307/308 redirects, since the body cannot be sent again.
	// Closing the response body ends the request; the call's end closes it if the plugin
	// still holds it.
	//nd:hostfunc
	SendStream(ctx context.Context, request HTTPStreamRequest) (*HTTPStreamResponse, error)
}
