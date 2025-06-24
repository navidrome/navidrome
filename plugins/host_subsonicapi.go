package plugins

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"

	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/plugins/host/subsonicapi"
	"github.com/navidrome/navidrome/server/subsonic"
)

// SubsonicAPIService is the interface for the Subsonic API service
//
// Authentication: The plugin must provide valid authentication parameters in the URL:
//   - Required: `u` (username) - The service validates this parameter is present
//   - Example: `"/rest/ping?u=admin"`
//
// URL Format: Only the path and query parameters from the URL are used - host, protocol, and method are ignored
//
// Automatic Parameters: The service automatically adds:
//   - `c`: Plugin name (client identifier)
//   - `v`: Subsonic API version (1.16.1)
//   - `f`: Response format (json)
//
// See example usage in the `plugins/examples/subsonicapi-demo` plugin
type subsonicAPIServiceImpl struct {
	pluginID string
	router   SubsonicRouter
}

func (s *subsonicAPIServiceImpl) Call(ctx context.Context, req *subsonicapi.CallRequest) (*subsonicapi.CallResponse, error) {
	if s.router == nil {
		return &subsonicapi.CallResponse{
			Error: "SubsonicAPI router not available",
		}, nil
	}

	// Parse the input URL
	parsedURL, err := url.Parse(req.Url)
	if err != nil {
		return &subsonicapi.CallResponse{
			Error: fmt.Sprintf("invalid URL format: %v", err),
		}, nil
	}

	// Extract query parameters
	query := parsedURL.Query()

	// Validate that 'u' (username) parameter is present
	username := query.Get("u")
	if username == "" {
		return &subsonicapi.CallResponse{
			Error: "missing required parameter 'u' (username)",
		}, nil
	}

	// Add required Subsonic API parameters
	query.Set("c", s.pluginID)       // Client name (plugin ID)
	query.Set("f", "json")           // Response format
	query.Set("v", subsonic.Version) // API version

	// Extract the endpoint from the path
	endpoint := path.Base(parsedURL.Path)

	// Build the final URL with processed path and modified query parameters
	finalURL := &url.URL{
		Path:     "/" + endpoint,
		RawQuery: query.Encode(),
	}

	// Create HTTP request with internal authentication
	httpReq, err := http.NewRequestWithContext(ctx, "GET", finalURL.String(), nil)
	if err != nil {
		return &subsonicapi.CallResponse{
			Error: fmt.Sprintf("failed to create HTTP request: %v", err),
		}, nil
	}

	// Set internal authentication context using the username from the 'u' parameter
	authCtx := request.WithInternalAuth(httpReq.Context(), username)
	httpReq = httpReq.WithContext(authCtx)

	// Use ResponseRecorder to capture the response
	recorder := httptest.NewRecorder()

	// Call the subsonic router
	s.router.ServeHTTP(recorder, httpReq)

	// Return the response body as JSON
	return &subsonicapi.CallResponse{
		Json: recorder.Body.String(),
	}, nil
}
