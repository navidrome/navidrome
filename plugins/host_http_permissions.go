package plugins

import (
	"fmt"
	"strings"

	"github.com/navidrome/navidrome/plugins/schema"
)

// Maximum number of HTTP redirects allowed for plugin requests
const httpMaxRedirects = 5

// HTTPPermissions represents granular HTTP access permissions for plugins
type httpPermissions struct {
	*networkPermissionsBase
	AllowedUrls map[string][]string `json:"allowedUrls"`
	matcher     *urlMatcher
}

// parseHTTPPermissionsTyped extracts HTTP permissions from the typed permission struct
func parseHTTPPermissionsTyped(permData *schema.PluginManifestPermissionsHttp) (*httpPermissions, error) {
	base := &networkPermissionsBase{
		AllowLocalNetwork: permData.AllowLocalNetwork,
	}

	if len(permData.AllowedUrls) == 0 {
		return nil, fmt.Errorf("allowedUrls must contain at least one URL pattern")
	}

	allowedUrls := make(map[string][]string)
	for urlPattern, methodsRaw := range permData.AllowedUrls {
		methodsArray, ok := methodsRaw.([]interface{})
		if !ok {
			return nil, fmt.Errorf("operations for URL pattern %s must be an array", urlPattern)
		}

		var methods []string
		for _, methodRaw := range methodsArray {
			method, ok := methodRaw.(string)
			if !ok {
				return nil, fmt.Errorf("operation must be a string")
			}
			methods = append(methods, strings.ToUpper(method))
		}
		allowedUrls[urlPattern] = methods
	}

	// Validate HTTP methods
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true,
		"PATCH": true, "HEAD": true, "OPTIONS": true, "*": true,
	}

	for urlPattern, methods := range allowedUrls {
		for _, method := range methods {
			if !validMethods[strings.ToUpper(method)] {
				return nil, fmt.Errorf("invalid HTTP method '%s' for URL pattern '%s'", method, urlPattern)
			}
		}
	}

	return &httpPermissions{
		networkPermissionsBase: base,
		AllowedUrls:            allowedUrls,
		matcher:                newURLMatcher(),
	}, nil
}

// IsRequestAllowed checks if a specific network request is allowed by the permissions
func (p *httpPermissions) IsRequestAllowed(requestURL, operation string) error {
	if _, err := checkURLPolicy(requestURL, p.AllowLocalNetwork); err != nil {
		return err
	}

	// allowedUrls is now required - no fallback to allow all URLs
	if p.AllowedUrls == nil || len(p.AllowedUrls) == 0 {
		return fmt.Errorf("no allowed URLs configured for plugin")
	}

	matcher := newURLMatcher()

	// Check URL patterns and operations
	// First try exact matches, then wildcard matches
	operation = strings.ToUpper(operation)

	// Phase 1: Check for exact matches first
	for urlPattern, allowedOperations := range p.AllowedUrls {
		if !strings.Contains(urlPattern, "*") && matcher.MatchesURLPattern(requestURL, urlPattern) {
			// Check if operation is allowed
			for _, allowedOperation := range allowedOperations {
				if allowedOperation == "*" || allowedOperation == operation {
					return nil
				}
			}
			return fmt.Errorf("operation %s not allowed for URL pattern %s", operation, urlPattern)
		}
	}

	// Phase 2: Check wildcard patterns
	for urlPattern, allowedOperations := range p.AllowedUrls {
		if strings.Contains(urlPattern, "*") && matcher.MatchesURLPattern(requestURL, urlPattern) {
			// Check if operation is allowed
			for _, allowedOperation := range allowedOperations {
				if allowedOperation == "*" || allowedOperation == operation {
					return nil
				}
			}
			return fmt.Errorf("operation %s not allowed for URL pattern %s", operation, urlPattern)
		}
	}

	return fmt.Errorf("URL %s does not match any allowed URL patterns", requestURL)
}
