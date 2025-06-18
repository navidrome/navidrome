package plugins

import (
	"fmt"
)

// WebSocketPermissions represents granular WebSocket access permissions for plugins
type webSocketPermissions struct {
	*networkPermissionsBase
	AllowedUrls []string `json:"allowedUrls"`
	matcher     *urlMatcher
}

// ParseWebSocketPermissions extracts WebSocket permissions from the raw permission map
func parseWebSocketPermissions(permissionData any) (*webSocketPermissions, error) {
	if permissionData == nil {
		return nil, fmt.Errorf("websocket permission data is nil")
	}

	permMap, ok := permissionData.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("websocket permission data is not a map")
	}

	base, err := parseNetworkPermissionsBase(permMap)
	if err != nil {
		return nil, err
	}

	// Extract allowedUrls (array format)
	allowedUrlsRaw, exists := permMap["allowedUrls"]
	if !exists {
		return nil, fmt.Errorf("allowedUrls field is required")
	}

	allowedUrlsArray, ok := allowedUrlsRaw.([]any)
	if !ok {
		return nil, fmt.Errorf("allowedUrls must be an array")
	}

	if len(allowedUrlsArray) == 0 {
		return nil, fmt.Errorf("allowedUrls must contain at least one URL pattern")
	}

	allowedUrls := make([]string, len(allowedUrlsArray))
	for i, urlRaw := range allowedUrlsArray {
		urlPattern, ok := urlRaw.(string)
		if !ok {
			return nil, fmt.Errorf("URL pattern at index %d must be a string", i)
		}
		allowedUrls[i] = urlPattern
	}

	return &webSocketPermissions{
		networkPermissionsBase: base,
		AllowedUrls:            allowedUrls,
		matcher:                newURLMatcher(),
	}, nil
}

// IsConnectionAllowed checks if a WebSocket connection is allowed
func (w *webSocketPermissions) IsConnectionAllowed(requestURL string) error {
	if _, err := checkURLPolicy(requestURL, w.AllowLocalNetwork); err != nil {
		return err
	}

	// allowedUrls is required - no fallback to allow all URLs
	if len(w.AllowedUrls) == 0 {
		return fmt.Errorf("no allowed URLs configured for plugin")
	}

	// Check URL patterns
	// First try exact matches, then wildcard matches

	// Phase 1: Check for exact matches first
	for _, urlPattern := range w.AllowedUrls {
		if urlPattern == "*" || (!containsWildcard(urlPattern) && w.matcher.MatchesURLPattern(requestURL, urlPattern)) {
			return nil
		}
	}

	// Phase 2: Check wildcard patterns
	for _, urlPattern := range w.AllowedUrls {
		if containsWildcard(urlPattern) && w.matcher.MatchesURLPattern(requestURL, urlPattern) {
			return nil
		}
	}

	return fmt.Errorf("URL %s does not match any allowed URL patterns", requestURL)
}

// containsWildcard checks if a URL pattern contains wildcard characters
func containsWildcard(pattern string) bool {
	if pattern == "*" {
		return true
	}

	// Check for wildcards anywhere in the pattern
	for _, char := range pattern {
		if char == '*' {
			return true
		}
	}

	return false
}
