package plugins

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/plugins/host"
)

// subsonicAPIVersion is the Subsonic API version used for plugin calls.
// This is defined locally to avoid import cycle with server/subsonic.
const subsonicAPIVersion = "1.16.1"

// subsonicAPIServiceImpl implements host.SubsonicAPIService.
// It provides plugins with access to Navidrome's Subsonic API.
//
// Authentication: The plugin must provide a valid 'u' (username) parameter in the URL.
// URL Format: Only the path and query parameters are used - host/protocol are ignored.
// Automatic Parameters: The service adds 'c' (client), 'v' (version), 'f' (format).
type subsonicAPIServiceImpl struct {
	pluginID       string
	router         SubsonicRouter
	ds             model.DataStore
	allowedUserIDs []string // User IDs this plugin can access (from DB configuration)
	allUsers       bool     // If true, plugin can access all users
	userIDMap      map[string]struct{}
}

// newSubsonicAPIService creates a new SubsonicAPIService for a plugin.
func newSubsonicAPIService(pluginID string, router SubsonicRouter, ds model.DataStore, allowedUserIDs []string, allUsers bool) host.SubsonicAPIService {
	userIDMap := make(map[string]struct{})
	for _, id := range allowedUserIDs {
		userIDMap[id] = struct{}{}
	}
	return &subsonicAPIServiceImpl{
		pluginID:       pluginID,
		router:         router,
		ds:             ds,
		allowedUserIDs: allowedUserIDs,
		allUsers:       allUsers,
		userIDMap:      userIDMap,
	}
}

func (s *subsonicAPIServiceImpl) Call(ctx context.Context, uri string) (string, error) {
	if s.router == nil {
		return "", fmt.Errorf("SubsonicAPI router not available")
	}

	// Parse the input URL
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %w", err)
	}

	// Extract query parameters
	query := parsedURL.Query()

	// Validate that 'u' (username) parameter is present
	username := query.Get("u")
	if username == "" {
		return "", fmt.Errorf("missing required parameter 'u' (username)")
	}

	if err := s.checkPermissions(ctx, username); err != nil {
		log.Warn(ctx, "SubsonicAPI call blocked by permissions", "plugin", s.pluginID, "user", username, err)
		return "", err
	}

	// Add required Subsonic API parameters
	query.Set("c", s.pluginID)         // Client name (plugin ID)
	query.Set("f", "json")             // Response format
	query.Set("v", subsonicAPIVersion) // API version

	// Extract the endpoint from the path
	endpoint := path.Base(parsedURL.Path)

	// Build the final URL with processed path and modified query parameters
	finalURL := &url.URL{
		Path:     "/" + endpoint,
		RawQuery: query.Encode(),
	}

	// Create HTTP request with a fresh context to avoid Chi RouteContext pollution.
	// Using http.NewRequest (instead of http.NewRequestWithContext) ensures the internal
	// SubsonicAPI call doesn't inherit routing information from the parent handler,
	// which would cause Chi to invoke the wrong handler. Authentication context is
	// explicitly added in the next step via request.WithInternalAuth.
	httpReq, err := http.NewRequest("GET", finalURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set internal authentication context using the username from the 'u' parameter
	authCtx := request.WithInternalAuth(httpReq.Context(), username)
	httpReq = httpReq.WithContext(authCtx)

	// Use ResponseRecorder to capture the response
	recorder := httptest.NewRecorder()

	// Call the subsonic router
	s.router.ServeHTTP(recorder, httpReq)

	// Return the response body as JSON
	return recorder.Body.String(), nil
}

func (s *subsonicAPIServiceImpl) checkPermissions(ctx context.Context, username string) error {
	// If allUsers is true, allow any user
	if s.allUsers {
		return nil
	}

	// Must have at least one allowed user ID configured
	if len(s.allowedUserIDs) == 0 {
		return fmt.Errorf("no users configured for plugin %s", s.pluginID)
	}

	// Look up the user by username to get their ID
	usr, err := s.ds.User(ctx).FindByUsername(username)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return fmt.Errorf("username %s not found", username)
		}
		return err
	}

	// Check if the user's ID is in the allowed list
	if _, ok := s.userIDMap[usr.ID]; !ok {
		return fmt.Errorf("user %s is not authorized for this plugin", username)
	}

	return nil
}
