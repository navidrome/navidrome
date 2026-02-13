// Test plugin for public (unauthenticated) HTTP endpoint integration tests.
// Build with: tinygo build -o ../test-http-endpoint-public.wasm -target wasip1 -buildmode=c-shared .
package main

import (
	"github.com/navidrome/navidrome/plugins/pdk/go/httpendpoint"
)

func init() {
	httpendpoint.Register(&testPublicEndpoint{})
}

type testPublicEndpoint struct{}

func (t *testPublicEndpoint) HandleRequest(req httpendpoint.HTTPHandleRequest) (httpendpoint.HTTPHandleResponse, error) {
	switch req.Path {
	case "/webhook":
		return httpendpoint.HTTPHandleResponse{
			Status: 200,
			Headers: map[string][]string{
				"Content-Type": {"text/plain"},
			},
			Body: []byte("webhook received"),
		}, nil

	case "/check-no-user":
		// Verify that no user info is provided for public endpoints
		hasUser := "false"
		if req.User != nil {
			hasUser = "true"
		}
		return httpendpoint.HTTPHandleResponse{
			Status: 200,
			Body:   []byte("hasUser=" + hasUser),
		}, nil

	default:
		return httpendpoint.HTTPHandleResponse{
			Status: 404,
			Body:   []byte("Not found: " + req.Path),
		}, nil
	}
}

func main() {}
