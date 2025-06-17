package plugins

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

// Maximum number of HTTP redirects allowed for plugin requests
const httpMaxRedirects = 5

// HttpPermissions represents granular HTTP access permissions for plugins
type HttpPermissions struct {
	Reason string `json:"reason"`
	// AllowedUrls maps URL patterns to allowed HTTP methods
	// Redirect destinations must also be included in this list
	AllowedUrls       map[string][]string `json:"allowedUrls"`
	AllowLocalNetwork bool                `json:"allowLocalNetwork,omitempty"`
}

// ParseHttpPermissions extracts HTTP permissions from the raw permission map
func ParseHttpPermissions(permissionData any) (*HttpPermissions, error) {
	if permissionData == nil {
		return nil, fmt.Errorf("http permission data is nil")
	}

	permMap, ok := permissionData.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("http permission data is not a map")
	}

	perms := &HttpPermissions{
		AllowLocalNetwork: false, // Default to false for security
	}

	// Extract reason (required)
	if reason, ok := permMap["reason"].(string); ok && reason != "" {
		perms.Reason = reason
	} else {
		return nil, fmt.Errorf("http permission reason is required and must be a non-empty string")
	}

	// Extract allowedUrls
	allowedUrlsRaw, exists := permMap["allowedUrls"]
	if !exists {
		return nil, fmt.Errorf("allowedUrls field is required")
	}

	allowedUrlsMap, ok := allowedUrlsRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("allowedUrls must be a map")
	}

	if len(allowedUrlsMap) == 0 {
		return nil, fmt.Errorf("allowedUrls must contain at least one URL pattern")
	}

	perms.AllowedUrls = make(map[string][]string)
	for urlPattern, methodsRaw := range allowedUrlsMap {
		methodsArray, ok := methodsRaw.([]any)
		if !ok {
			return nil, fmt.Errorf("methods for URL pattern %s must be an array", urlPattern)
		}

		var methods []string
		for _, methodRaw := range methodsArray {
			method, ok := methodRaw.(string)
			if !ok {
				return nil, fmt.Errorf("HTTP method must be a string")
			}
			methods = append(methods, strings.ToUpper(method))
		}
		perms.AllowedUrls[urlPattern] = methods
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

// IsRequestAllowed checks if a specific HTTP request is allowed by the permissions
func (h *HttpPermissions) IsRequestAllowed(requestURL, method string) error {
	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Check local network restrictions
	if !h.AllowLocalNetwork {
		if err := h.checkLocalNetwork(parsedURL); err != nil {
			return err
		}
	}

	// allowedUrls is now required - no fallback to allow all URLs
	if h.AllowedUrls == nil || len(h.AllowedUrls) == 0 {
		return fmt.Errorf("no allowed URLs configured for plugin")
	}

	// Check URL patterns and methods
	// First try exact matches, then wildcard matches
	method = strings.ToUpper(method)

	// Phase 1: Check for exact matches first
	for urlPattern, allowedMethods := range h.AllowedUrls {
		if !strings.Contains(urlPattern, "*") && h.matchesURLPattern(requestURL, urlPattern) {
			// Check if method is allowed
			for _, allowedMethod := range allowedMethods {
				if allowedMethod == "*" || allowedMethod == method {
					return nil
				}
			}
			return fmt.Errorf("HTTP method %s not allowed for URL pattern %s", method, urlPattern)
		}
	}

	// Phase 2: Check wildcard patterns
	for urlPattern, allowedMethods := range h.AllowedUrls {
		if strings.Contains(urlPattern, "*") && h.matchesURLPattern(requestURL, urlPattern) {
			// Check if method is allowed
			for _, allowedMethod := range allowedMethods {
				if allowedMethod == "*" || allowedMethod == method {
					return nil
				}
			}
			return fmt.Errorf("HTTP method %s not allowed for URL pattern %s", method, urlPattern)
		}
	}

	return fmt.Errorf("URL %s does not match any allowed URL patterns", requestURL)
}

// matchesURLPattern checks if a URL matches a given pattern
func (h *HttpPermissions) matchesURLPattern(requestURL, pattern string) bool {
	// Handle wildcard pattern
	if pattern == "*" {
		return true
	}

	// Parse both URLs to handle path matching correctly
	reqURL, err := url.Parse(requestURL)
	if err != nil {
		return false
	}

	patternURL, err := url.Parse(pattern)
	if err != nil {
		// If pattern is not a valid URL, treat it as a simple string pattern
		regexPattern := h.urlPatternToRegex(pattern)
		matched, err := regexp.MatchString(regexPattern, requestURL)
		if err != nil {
			return false
		}
		return matched
	}

	// Match scheme
	if patternURL.Scheme != "" && patternURL.Scheme != reqURL.Scheme {
		return false
	}

	// Match host with wildcard support
	if !h.matchesHost(reqURL.Host, patternURL.Host) {
		return false
	}

	// Match path with wildcard support
	// Special case: if pattern URL has empty path and contains wildcards, allow any path (domain-only wildcard matching)
	if (patternURL.Path == "" || patternURL.Path == "/") && strings.Contains(pattern, "*") {
		// This is a domain-only wildcard pattern, allow any path
		return true
	}
	if !h.matchesPath(reqURL.Path, patternURL.Path) {
		return false
	}

	return true
}

// urlPatternToRegex converts a URL pattern with wildcards to a regex pattern
func (h *HttpPermissions) urlPatternToRegex(pattern string) string {
	// Escape special regex characters except *
	escaped := regexp.QuoteMeta(pattern)

	// Replace escaped \* with regex pattern for wildcard matching
	// For subdomain: *.example.com -> [^.]*\.example\.com
	// For path: /api/* -> /api/.*
	escaped = strings.ReplaceAll(escaped, "\\*", ".*")

	// Anchor the pattern to match the full URL
	return "^" + escaped + "$"
}

// matchesHost checks if a host matches a pattern with wildcard support
func (h *HttpPermissions) matchesHost(host, pattern string) bool {
	if pattern == "" {
		return true
	}

	if pattern == "*" {
		return true
	}

	// Handle wildcard patterns anywhere in the host
	if strings.Contains(pattern, "*") {
		patterns := []string{
			strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\*", "[0-9.]+"), // IP pattern
			strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\*", "[^.]*"),   // Domain pattern
		}

		for _, regexPattern := range patterns {
			fullPattern := "^" + regexPattern + "$"
			if matched, err := regexp.MatchString(fullPattern, host); err == nil && matched {
				return true
			}
		}
		return false
	}

	return host == pattern
}

// matchesPath checks if a path matches a pattern with wildcard support
func (h *HttpPermissions) matchesPath(path, pattern string) bool {
	// Normalize empty paths to "/"
	if path == "" {
		path = "/"
	}
	if pattern == "" {
		pattern = "/"
	}

	if pattern == "*" {
		return true
	}

	// Handle wildcard paths
	if strings.HasSuffix(pattern, "/*") {
		prefix := pattern[:len(pattern)-2] // Remove "/*"
		if prefix == "" {
			prefix = "/"
		}
		return strings.HasPrefix(path, prefix) || path == prefix
	}

	return path == pattern
}

// checkLocalNetwork checks if the URL points to a local/private network address
func (h *HttpPermissions) checkLocalNetwork(parsedURL *url.URL) error {
	host := parsedURL.Hostname()

	// Check for localhost
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return fmt.Errorf("requests to localhost are not allowed")
	}

	// Check for private IP ranges
	ip := net.ParseIP(host)
	if ip != nil {
		if h.isPrivateIP(ip) {
			return fmt.Errorf("requests to private IP addresses are not allowed")
		}
	}

	return nil
}

// isPrivateIP checks if an IP address is in a private range
func (h *HttpPermissions) isPrivateIP(ip net.IP) bool {
	// Private IPv4 ranges
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16", // Link-local
	}

	for _, rangeStr := range privateRanges {
		_, privateCIDR, err := net.ParseCIDR(rangeStr)
		if err != nil {
			continue
		}
		if privateCIDR.Contains(ip) {
			return true
		}
	}

	// Check for IPv6 private ranges
	if ip.To4() == nil { // IPv6
		// Link-local IPv6 (fe80::/10)
		if ip.IsLinkLocalUnicast() {
			return true
		}
		// Loopback IPv6 (::1)
		if ip.IsLoopback() {
			return true
		}
		// Unique local IPv6 (fc00::/7)
		if len(ip) == 16 && (ip[0]&0xfe) == 0xfc {
			return true
		}
	}

	return false
}
