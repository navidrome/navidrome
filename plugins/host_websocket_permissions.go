package plugins

import (
	"fmt"
	"net/url"
)

// WebSocketPermissions represents granular WebSocket access permissions for plugins
type WebSocketPermissions struct {
	Reason            string   `json:"reason"`
	AllowedUrls       []string `json:"allowedUrls"`
	AllowLocalNetwork bool     `json:"allowLocalNetwork,omitempty"`
	matcher           *URLMatcher
}

// ParseWebSocketPermissions extracts WebSocket permissions from the raw permission map
func ParseWebSocketPermissions(permissionData any) (*WebSocketPermissions, error) {
	if permissionData == nil {
		return nil, fmt.Errorf("websocket permission data is nil")
	}

	permMap, ok := permissionData.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("websocket permission data is not a map")
	}

	perms := &WebSocketPermissions{
		AllowLocalNetwork: false, // Default to false for security
		matcher:           NewURLMatcher(),
	}

	// Extract reason (required)
	if reason, ok := permMap["reason"].(string); ok && reason != "" {
		perms.Reason = reason
	} else {
		return nil, fmt.Errorf("websocket permission reason is required and must be a non-empty string")
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

	perms.AllowedUrls = make([]string, len(allowedUrlsArray))
	for i, urlRaw := range allowedUrlsArray {
		urlPattern, ok := urlRaw.(string)
		if !ok {
			return nil, fmt.Errorf("URL pattern at index %d must be a string", i)
		}
		perms.AllowedUrls[i] = urlPattern
	}

	// Extract allowLocalNetwork (optional)
	if allowLocalNetwork, exists := permMap["allowLocalNetwork"]; exists {
		if localNet, ok := allowLocalNetwork.(bool); ok {
			perms.AllowLocalNetwork = localNet
		} else {
			return nil, fmt.Errorf("allowLocalNetwork must be a boolean")
		}
	}

	return perms, nil
}

// IsConnectionAllowed checks if a WebSocket connection is allowed
func (w *WebSocketPermissions) IsConnectionAllowed(requestURL string) error {
	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Check local network restrictions
	if !w.AllowLocalNetwork {
		if err := CheckLocalNetwork(parsedURL); err != nil {
			return err
		}
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
