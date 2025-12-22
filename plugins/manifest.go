package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Capability represents a plugin capability type
type Capability string

const (
	CapabilityMetadataAgent Capability = "MetadataAgent"
	// Future capabilities:
	// CapabilityScrobbler     Capability = "Scrobbler"
)

// Manifest represents the plugin manifest exported by the nd_manifest function.
// The manifest describes the plugin's metadata, capabilities, and permissions.
type Manifest struct {
	Name         string       `json:"name"`
	Author       string       `json:"author"`
	Version      string       `json:"version"`
	Description  string       `json:"description,omitempty"`
	Website      string       `json:"website,omitempty"`
	Capabilities []Capability `json:"capabilities"`
	Permissions  Permissions  `json:"permissions,omitempty"`
}

// Permissions defines the plugin's required permissions
type Permissions struct {
	HTTP   *HTTPPermission   `json:"http,omitempty"`
	Config *ConfigPermission `json:"config,omitempty"`
}

// HTTPPermission defines HTTP access permissions for a plugin
type HTTPPermission struct {
	Reason      string              `json:"reason,omitempty"`
	AllowedURLs map[string][]string `json:"allowedUrls,omitempty"`
}

// ConfigPermission defines config access permissions for a plugin
type ConfigPermission struct {
	Reason string `json:"reason,omitempty"`
}

// Validate checks if the manifest is valid
func (m *Manifest) Validate() error {
	if m.Name == "" {
		return errors.New("plugin manifest: name is required")
	}
	if m.Author == "" {
		return errors.New("plugin manifest: author is required")
	}
	if m.Version == "" {
		return errors.New("plugin manifest: version is required")
	}
	if len(m.Capabilities) == 0 {
		return errors.New("plugin manifest: at least one capability is required")
	}

	// Validate capabilities
	for _, cap := range m.Capabilities {
		if !isValidCapability(cap) {
			return fmt.Errorf("plugin manifest: unknown capability %q", cap)
		}
	}

	// Validate HTTP permissions if present
	if m.Permissions.HTTP != nil {
		if err := m.validateHTTPPermissions(); err != nil {
			return err
		}
	}

	return nil
}

// HasCapability checks if the plugin has a specific capability
func (m *Manifest) HasCapability(cap Capability) bool {
	for _, c := range m.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// AllowedHosts returns a list of allowed hosts for HTTP requests.
// This extracts hostnames from the AllowedURLs patterns.
func (m *Manifest) AllowedHosts() []string {
	if m.Permissions.HTTP == nil || len(m.Permissions.HTTP.AllowedURLs) == 0 {
		return nil
	}

	hosts := make([]string, 0, len(m.Permissions.HTTP.AllowedURLs))
	for urlPattern := range m.Permissions.HTTP.AllowedURLs {
		host := extractHost(urlPattern)
		if host != "" {
			hosts = append(hosts, host)
		}
	}
	return hosts
}

// ParseManifest parses JSON data into a Manifest struct
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("plugin manifest: invalid JSON: %w", err)
	}
	return &m, nil
}

// validateHTTPPermissions validates the HTTP permission configuration
func (m *Manifest) validateHTTPPermissions() error {
	for urlPattern := range m.Permissions.HTTP.AllowedURLs {
		if !isValidURLPattern(urlPattern) {
			return fmt.Errorf("plugin manifest: invalid URL pattern %q", urlPattern)
		}
	}
	return nil
}

// isValidCapability checks if a capability is known
func isValidCapability(cap Capability) bool {
	switch cap {
	case CapabilityMetadataAgent:
		return true
	default:
		return false
	}
}

// isValidURLPattern checks if a URL pattern is valid.
// Valid patterns are URLs that may contain wildcards (*).
// Examples:
//   - https://api.example.com/*
//   - https://*.example.com/api/*
//   - https://example.com/v1/endpoint
func isValidURLPattern(pattern string) bool {
	// Remove wildcards temporarily to validate the base URL
	testURL := strings.ReplaceAll(pattern, "*", "wildcard")

	u, err := url.Parse(testURL)
	if err != nil {
		return false
	}

	// Must have a scheme (http or https)
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	// Must have a host
	if u.Host == "" {
		return false
	}

	return true
}

// extractHost extracts the hostname from a URL pattern.
// For patterns with wildcards in the host, it returns the pattern as-is for glob matching.
// Examples:
//   - https://api.example.com/* -> api.example.com
//   - https://*.example.com/api/* -> *.example.com
func extractHost(pattern string) string {
	// Remove wildcards temporarily to parse the URL
	testURL := strings.ReplaceAll(pattern, "*", "wildcard")

	u, err := url.Parse(testURL)
	if err != nil {
		return ""
	}

	// Restore wildcards in the host
	host := strings.ReplaceAll(u.Hostname(), "wildcard", "*")
	return host
}
