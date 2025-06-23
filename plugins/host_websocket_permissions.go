package plugins

import (
	"fmt"

	"github.com/navidrome/navidrome/plugins/schema"
)

// WebSocketPermissions represents granular WebSocket access permissions for plugins
type webSocketPermissions struct {
	*networkPermissionsBase
	AllowedUrls []string `json:"allowedUrls"`
	matcher     *urlMatcher
}

// parseWebSocketPermissions extracts WebSocket permissions from the schema
func parseWebSocketPermissions(permData *schema.PluginManifestPermissionsWebsocket) (*webSocketPermissions, error) {
	if len(permData.AllowedUrls) == 0 {
		return nil, fmt.Errorf("allowedUrls must contain at least one URL pattern")
	}

	return &webSocketPermissions{
		networkPermissionsBase: &networkPermissionsBase{
			AllowLocalNetwork: permData.AllowLocalNetwork,
		},
		AllowedUrls: permData.AllowedUrls,
		matcher:     newURLMatcher(),
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
