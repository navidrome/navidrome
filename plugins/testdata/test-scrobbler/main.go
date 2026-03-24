// Test scrobbler plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../test-scrobbler.wasm -target wasip1 -buildmode=c-shared ./main.go
package main

import (
	"fmt"
	"strconv"

	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/scrobbler"
)

func init() {
	scrobbler.Register(&testScrobbler{})
}

type testScrobbler struct{}

// IsAuthorized checks if a user is authorized.
func (t *testScrobbler) IsAuthorized(scrobbler.IsAuthorizedRequest) (bool, error) {
	return checkAuthConfig(), nil
}

// NowPlaying sends a now playing notification.
func (t *testScrobbler) NowPlaying(input scrobbler.NowPlayingRequest) error {
	// Check for configured error
	if err := checkConfigError(); err != nil {
		return err
	}

	// Log the now playing (for potential debugging)
	artistName := ""
	if len(input.Track.Artists) > 0 {
		artistName = input.Track.Artists[0].Name
	}
	pdk.Log(pdk.LogInfo, "NowPlaying: "+input.Track.Title+" by "+artistName)
	return nil
}

// Scrobble submits a scrobble.
func (t *testScrobbler) Scrobble(input scrobbler.ScrobbleRequest) error {
	// Check for configured error
	if err := checkConfigError(); err != nil {
		return err
	}

	// Log the scrobble (for potential debugging)
	artistName := ""
	if len(input.Track.Artists) > 0 {
		artistName = input.Track.Artists[0].Name
	}
	pdk.Log(pdk.LogInfo, "Scrobble: "+input.Track.Title+" by "+artistName)
	return nil
}

// checkConfigError checks if the plugin is configured to return an error.
// If "error" config is set, it returns the appropriate ScrobblerError.
// Error types: "not_authorized", "retry_later", "unrecoverable"
func checkConfigError() error {
	errMsg, hasErr := pdk.GetConfig("error")
	if !hasErr || errMsg == "" {
		return nil
	}
	errType, _ := pdk.GetConfig("error_type")
	switch errType {
	case scrobbler.ScrobblerErrorNotAuthorized.Error():
		return fmt.Errorf("%w: %s", scrobbler.ScrobblerErrorNotAuthorized, errMsg)
	case scrobbler.ScrobblerErrorRetryLater.Error():
		return fmt.Errorf("%w: %s", scrobbler.ScrobblerErrorRetryLater, errMsg)
	default:
		return fmt.Errorf("%w: %s", scrobbler.ScrobblerErrorUnrecoverable, errMsg)
	}
}

// checkAuthConfig returns whether the plugin is configured to authorize users.
// If "authorized" config is set to "false", users are not authorized.
// Default is true (authorized).
func checkAuthConfig() bool {
	authStr, hasAuth := pdk.GetConfig("authorized")
	if !hasAuth {
		return true // Default: authorized
	}
	auth, err := strconv.ParseBool(authStr)
	if err != nil {
		return true // Default on parse error
	}
	return auth
}

func main() {}
