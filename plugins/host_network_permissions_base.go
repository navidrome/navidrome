package plugins

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

// NetworkPermissionsBase contains common functionality for network-based permissions
type networkPermissionsBase struct {
	Reason            string `json:"reason"`
	AllowLocalNetwork bool   `json:"allowLocalNetwork,omitempty"`
}

// URLMatcher provides URL pattern matching functionality
type urlMatcher struct{}

// newURLMatcher creates a new URL matcher instance
func newURLMatcher() *urlMatcher {
	return &urlMatcher{}
}

// checkURLPolicy performs common checks for a URL against network policies.
func checkURLPolicy(requestURL string, allowLocalNetwork bool) (*url.URL, error) {
	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Check local network restrictions
	if !allowLocalNetwork {
		if err := checkLocalNetwork(parsedURL); err != nil {
			return nil, err
		}
	}
	return parsedURL, nil
}

// MatchesURLPattern checks if a URL matches a given pattern
func (m *urlMatcher) MatchesURLPattern(requestURL, pattern string) bool {
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
		regexPattern := m.urlPatternToRegex(pattern)
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
	if !m.matchesHost(reqURL.Host, patternURL.Host) {
		return false
	}

	// Match path with wildcard support
	// Special case: if pattern URL has empty path and contains wildcards, allow any path (domain-only wildcard matching)
	if (patternURL.Path == "" || patternURL.Path == "/") && strings.Contains(pattern, "*") {
		// This is a domain-only wildcard pattern, allow any path
		return true
	}
	if !m.matchesPath(reqURL.Path, patternURL.Path) {
		return false
	}

	return true
}

// urlPatternToRegex converts a URL pattern with wildcards to a regex pattern
func (m *urlMatcher) urlPatternToRegex(pattern string) string {
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
func (m *urlMatcher) matchesHost(host, pattern string) bool {
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
func (m *urlMatcher) matchesPath(path, pattern string) bool {
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
		return strings.HasPrefix(path, prefix)
	}

	return path == pattern
}

// CheckLocalNetwork checks if the URL is accessing local network resources
func checkLocalNetwork(parsedURL *url.URL) error {
	host := parsedURL.Hostname()

	// Check for localhost variants
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return fmt.Errorf("requests to localhost are not allowed")
	}

	// Try to parse as IP address
	ip := net.ParseIP(host)
	if ip != nil && isPrivateIP(ip) {
		return fmt.Errorf("requests to private IP addresses are not allowed")
	}

	return nil
}

// IsPrivateIP checks if an IP is loopback, private, or link-local (IPv4/IPv6).
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() {
		return true
	}
	// IPv4 link-local: 169.254.0.0/16
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 169 && ip4[1] == 254
	}
	// IPv6 link-local: fe80::/10
	if ip16 := ip.To16(); ip16 != nil && ip.To4() == nil {
		return ip16[0] == 0xfe && (ip16[1]&0xc0) == 0x80
	}
	return false
}
