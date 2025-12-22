// Fake scrobbler plugin for Navidrome plugin system integration tests.
// Build with: tinygo build -o ../fake-scrobbler.wasm -target wasip1 -buildmode=c-shared ./main.go
package main

import (
	"encoding/json"
	"strconv"

	"github.com/extism/go-pdk"
)

// Manifest types
type Manifest struct {
	Name        string `json:"name"`
	Author      string `json:"author"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// Scrobbler input/output types

type AuthInput struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type AuthOutput struct {
	Authorized bool `json:"authorized"`
}

type TrackInfo struct {
	ID                string  `json:"id"`
	Title             string  `json:"title"`
	Album             string  `json:"album"`
	Artist            string  `json:"artist"`
	AlbumArtist       string  `json:"album_artist"`
	Duration          float32 `json:"duration"`
	TrackNumber       int     `json:"track_number"`
	DiscNumber        int     `json:"disc_number"`
	MbzRecordingID    string  `json:"mbz_recording_id,omitempty"`
	MbzAlbumID        string  `json:"mbz_album_id,omitempty"`
	MbzArtistID       string  `json:"mbz_artist_id,omitempty"`
	MbzReleaseGroupID string  `json:"mbz_release_group_id,omitempty"`
}

type NowPlayingInput struct {
	UserID   string    `json:"user_id"`
	Username string    `json:"username"`
	Track    TrackInfo `json:"track"`
	Position int       `json:"position"`
}

type ScrobbleInput struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Track     TrackInfo `json:"track"`
	Timestamp int64     `json:"timestamp"`
}

type ScrobblerOutput struct {
	Error     string `json:"error,omitempty"`
	ErrorType string `json:"error_type,omitempty"`
}

// checkConfigError checks if the plugin is configured to return an error.
// If "error" config is set, it returns the error message and error type.
func checkConfigError() (bool, string, string) {
	errMsg, hasErr := pdk.GetConfig("error")
	if !hasErr || errMsg == "" {
		return false, "", ""
	}
	errType, _ := pdk.GetConfig("error_type")
	if errType == "" {
		errType = "unrecoverable"
	}
	return true, errMsg, errType
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

//go:wasmexport nd_manifest
func ndManifest() int32 {
	manifest := Manifest{
		Name:        "Fake Scrobbler",
		Author:      "Navidrome Test",
		Version:     "1.0.0",
		Description: "A fake scrobbler plugin for integration testing",
	}
	out, err := json.Marshal(manifest)
	if err != nil {
		pdk.SetError(err)
		return 1
	}
	pdk.Output(out)
	return 0
}

//go:wasmexport nd_scrobbler_is_authorized
func ndScrobblerIsAuthorized() int32 {
	var input AuthInput
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}

	output := AuthOutput{
		Authorized: checkAuthConfig(),
	}

	if err := pdk.OutputJSON(output); err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

//go:wasmexport nd_scrobbler_now_playing
func ndScrobblerNowPlaying() int32 {
	var input NowPlayingInput
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}

	// Check for configured error
	hasErr, errMsg, errType := checkConfigError()
	if hasErr {
		output := ScrobblerOutput{
			Error:     errMsg,
			ErrorType: errType,
		}
		if err := pdk.OutputJSON(output); err != nil {
			pdk.SetError(err)
			return 1
		}
		return 0
	}

	// Log the now playing (for potential debugging)
	// In a real plugin, this would send to an external service
	pdk.Log(pdk.LogInfo, "NowPlaying: "+input.Track.Title+" by "+input.Track.Artist)

	// Success - no output needed
	return 0
}

//go:wasmexport nd_scrobbler_scrobble
func ndScrobblerScrobble() int32 {
	var input ScrobbleInput
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(err)
		return 1
	}

	// Check for configured error
	hasErr, errMsg, errType := checkConfigError()
	if hasErr {
		output := ScrobblerOutput{
			Error:     errMsg,
			ErrorType: errType,
		}
		if err := pdk.OutputJSON(output); err != nil {
			pdk.SetError(err)
			return 1
		}
		return 0
	}

	// Log the scrobble (for potential debugging)
	// In a real plugin, this would send to an external service
	pdk.Log(pdk.LogInfo, "Scrobble: "+input.Track.Title+" by "+input.Track.Artist)

	// Success - no output needed
	return 0
}

func main() {}
