// Test plugin for HTTP endpoint integration tests.
// Build with: tinygo build -o ../test-http-endpoint.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"encoding/json"

	"github.com/navidrome/navidrome/plugins/pdk/go/httpendpoint"
)

func init() {
	httpendpoint.Register(&testEndpoint{})
}

type testEndpoint struct{}

func (t *testEndpoint) HandleRequest(req httpendpoint.HTTPHandleRequest) (httpendpoint.HTTPHandleResponse, error) {
	switch req.Path {
	case "/hello":
		return httpendpoint.HTTPHandleResponse{
			Status: 200,
			Headers: map[string][]string{
				"Content-Type": {"text/plain"},
			},
			Body: []byte("Hello from plugin!"),
		}, nil

	case "/echo":
		// Echo back the request as JSON
		data, _ := json.Marshal(map[string]any{
			"method":   req.Method,
			"path":     req.Path,
			"query":    req.Query,
			"body":     string(req.Body),
			"hasUser":  req.User != nil,
			"username": userName(req.User),
		})
		return httpendpoint.HTTPHandleResponse{
			Status: 200,
			Headers: map[string][]string{
				"Content-Type": {"application/json"},
			},
			Body: data,
		}, nil

	case "/binary":
		// Return raw binary data (PNG header)
		return httpendpoint.HTTPHandleResponse{
			Status: 200,
			Headers: map[string][]string{
				"Content-Type": {"image/png"},
			},
			Body: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
		}, nil

	case "/error":
		return httpendpoint.HTTPHandleResponse{
			Status: 500,
			Body:   []byte("Something went wrong"),
		}, nil

	default:
		return httpendpoint.HTTPHandleResponse{
			Status: 404,
			Body:   []byte("Not found: " + req.Path),
		}, nil
	}
}

func userName(u *httpendpoint.HTTPUser) string {
	if u == nil {
		return ""
	}
	return u.Username
}

func main() {}
