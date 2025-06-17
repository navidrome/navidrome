package plugins

import (
	"fmt"
	"strings"
)

// Maximum number of HTTP redirects allowed for plugin requests
const httpMaxRedirects = 5

// HTTPPermissions represents granular HTTP access permissions for plugins
type HTTPPermissions struct {
	*NetworkPermissionsBase
	matcher *URLMatcher
}

// ParseHTTPPermissions extracts HTTP permissions from the raw permission map
func ParseHTTPPermissions(permissionData any) (*HTTPPermissions, error) {
	base, err := ParseNetworkPermissionsBase(permissionData)
	if err != nil {
		return nil, fmt.Errorf("http permission error: %w", err)
	}

	// Validate HTTP methods
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true,
		"PATCH": true, "HEAD": true, "OPTIONS": true, "*": true,
	}

	for urlPattern, methods := range base.AllowedUrls {
		for _, method := range methods {
			if !validMethods[strings.ToUpper(method)] {
				return nil, fmt.Errorf("invalid HTTP method '%s' for URL pattern '%s'", method, urlPattern)
			}
		}
	}

	return &HTTPPermissions{
		NetworkPermissionsBase: base,
		matcher:                NewURLMatcher(),
	}, nil
}
