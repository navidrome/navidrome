// Test plugin for native auth (JWT) HTTP endpoint integration tests.
// Build with: tinygo build -o ../test-http-endpoint-native.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"encoding/json"

	"github.com/navidrome/navidrome/plugins/pdk/go/httpendpoint"
)

func init() {
	httpendpoint.Register(&testNativeEndpoint{})
}

type testNativeEndpoint struct{}

func (t *testNativeEndpoint) HandleRequest(req httpendpoint.HTTPHandleRequest) (httpendpoint.HTTPHandleResponse, error) {
	switch req.Path {
	case "/hello":
		return httpendpoint.HTTPHandleResponse{
			Status: 200,
			Headers: map[string][]string{
				"Content-Type": {"text/plain"},
			},
			Body: "Hello from native auth plugin!",
		}, nil

	case "/echo":
		// Echo back the request as JSON
		data, _ := json.Marshal(map[string]any{
			"method":   req.Method,
			"path":     req.Path,
			"query":    req.Query,
			"body":     req.Body,
			"hasUser":  req.User != nil,
			"username": userName(req.User),
		})
		return httpendpoint.HTTPHandleResponse{
			Status: 200,
			Headers: map[string][]string{
				"Content-Type": {"application/json"},
			},
			Body: string(data),
		}, nil

	default:
		return httpendpoint.HTTPHandleResponse{
			Status: 404,
			Body:   "Not found: " + req.Path,
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
