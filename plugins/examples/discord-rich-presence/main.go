// Discord Rich Presence Plugin for Navidrome
//
// This plugin integrates Navidrome with Discord Rich Presence. It shows how a plugin can
// keep a real-time connection to an external service while remaining completely stateless.
//
// Capabilities: Scrobbler, SchedulerCallback, WebSocketCallback
//
// NOTE: This plugin is for demonstration purposes only. It relies on the user's Discord
// token being stored in the Navidrome configuration file, which is not secure and may be
// against Discord's terms of service. Use it at your own risk.
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/extism/go-pdk"
	host "github.com/navidrome/navidrome/plugins/pdk/go/host"
)

// Configuration keys
const (
	clientIDKey = "clientid"
	usersKey    = "users"
)

// getConfig loads the plugin configuration.
func getConfig() (clientID string, users map[string]string, err error) {
	clientID, ok := pdk.GetConfig(clientIDKey)
	if !ok || clientID == "" {
		pdk.Log(pdk.LogWarn, "missing ClientID in configuration")
		return "", nil, nil
	}

	cfgUsers, ok := pdk.GetConfig(usersKey)
	if !ok || cfgUsers == "" {
		pdk.Log(pdk.LogWarn, "no users configured")
		return clientID, nil, nil
	}

	users = make(map[string]string)
	for _, user := range strings.Split(cfgUsers, ",") {
		tuple := strings.Split(user, ":")
		if len(tuple) != 2 {
			return clientID, nil, fmt.Errorf("invalid user config: %s", user)
		}
		users[strings.TrimSpace(tuple[0])] = strings.TrimSpace(tuple[1])
	}
	return clientID, users, nil
}

// getImageURL retrieves the track artwork URL.
func getImageURL(trackID string) string {
	resp, err := host.ArtworkGetTrackUrl(trackID, 300)
	if err != nil {
		pdk.Log(pdk.LogWarn, fmt.Sprintf("Failed to get artwork URL: %v", err))
		return ""
	}

	// Don't use localhost URLs
	if strings.HasPrefix(resp.Url, "http://localhost") {
		return ""
	}
	return resp.Url
}

// ============================================================================
// Scrobbler Implementation
// ============================================================================

// NdScrobblerIsAuthorized checks if a user is authorized for Discord Rich Presence.
func NdScrobblerIsAuthorized(input AuthInput) (AuthOutput, error) {
	_, users, err := getConfig()
	if err != nil {
		return AuthOutput{}, fmt.Errorf("failed to check user authorization: %w", err)
	}

	_, authorized := users[input.Username]
	pdk.Log(pdk.LogInfo, fmt.Sprintf("IsAuthorized for user %s: %v", input.Username, authorized))
	return AuthOutput{Authorized: authorized}, nil
}

// NdScrobblerNowPlaying sends a now playing notification to Discord.
func NdScrobblerNowPlaying(input NowPlayingInput) (ScrobblerOutput, error) {
	pdk.Log(pdk.LogInfo, fmt.Sprintf("Setting presence for user %s, track: %s", input.Username, input.Track.Title))

	// Load configuration
	clientID, users, err := getConfig()
	if err != nil {
		return ScrobblerOutput{}, fmt.Errorf("failed to get config: %w", err)
	}

	// Check authorization
	userToken, authorized := users[input.Username]
	if !authorized {
		errMsg := fmt.Sprintf("user '%s' not authorized", input.Username)
		return ScrobblerOutput{Error: &errMsg, ErrorType: ScrobblerErrorTypeNotAuthorized}, nil
	}

	// Connect to Discord
	if err := connect(input.Username, userToken); err != nil {
		errMsg := fmt.Sprintf("failed to connect to Discord: %v", err)
		return ScrobblerOutput{Error: &errMsg, ErrorType: ScrobblerErrorTypeRetryLater}, nil
	}

	// Cancel any existing completion schedule
	_, _ = host.SchedulerCancelSchedule(fmt.Sprintf("%s-clear", input.Username))

	// Calculate timestamps
	now := time.Now().Unix()
	startTime := (now - int64(input.Position)) * 1000
	endTime := startTime + int64(input.Track.Duration)*1000

	// Send activity update
	if err := sendActivity(clientID, input.Username, userToken, activity{
		Application: clientID,
		Name:        "Navidrome",
		Type:        2, // Listening
		Details:     input.Track.Title,
		State:       input.Track.Artist,
		Timestamps: activityTimestamps{
			Start: startTime,
			End:   endTime,
		},
		Assets: activityAssets{
			LargeImage: getImageURL(input.Track.Id),
			LargeText:  input.Track.Album,
		},
	}); err != nil {
		errMsg := fmt.Sprintf("failed to send activity: %v", err)
		return ScrobblerOutput{Error: &errMsg, ErrorType: ScrobblerErrorTypeRetryLater}, nil
	}

	// Schedule a timer to clear the activity after the track completes
	remainingSeconds := int32(input.Track.Duration) - input.Position + 5
	_, err = host.SchedulerScheduleOneTime(remainingSeconds, payloadClearActivity, fmt.Sprintf("%s-clear", input.Username))
	if err != nil {
		pdk.Log(pdk.LogWarn, fmt.Sprintf("Failed to schedule completion timer: %v", err))
	}

	return ScrobblerOutput{}, nil
}

// NdScrobblerScrobble handles scrobble requests (no-op for Discord).
func NdScrobblerScrobble(_ ScrobbleInput) (ScrobblerOutput, error) {
	// Discord Rich Presence doesn't need scrobble events
	return ScrobblerOutput{}, nil
}

// ============================================================================
// Scheduler Callback Implementation
// ============================================================================

// NdSchedulerCallback handles scheduler callbacks.
func NdSchedulerCallback(input SchedulerCallbackInput) (SchedulerCallbackOutput, error) {
	pdk.Log(pdk.LogDebug, fmt.Sprintf("Scheduler callback: id=%s, payload=%s, recurring=%v", input.ScheduleId, input.Payload, input.IsRecurring))

	// Route based on payload
	switch input.Payload {
	case payloadHeartbeat:
		// Heartbeat callback - scheduleId is the username
		if err := handleHeartbeatCallback(input.ScheduleId); err != nil {
			errMsg := err.Error()
			return SchedulerCallbackOutput{Error: &errMsg}, nil
		}

	case payloadClearActivity:
		// Clear activity callback - scheduleId is "username-clear"
		username := strings.TrimSuffix(input.ScheduleId, "-clear")
		if err := handleClearActivityCallback(username); err != nil {
			errMsg := err.Error()
			return SchedulerCallbackOutput{Error: &errMsg}, nil
		}

	default:
		pdk.Log(pdk.LogWarn, fmt.Sprintf("Unknown scheduler callback payload: %s", input.Payload))
	}

	return SchedulerCallbackOutput{}, nil
}

// ============================================================================
// WebSocket Callback Implementation
// ============================================================================

// NdWebsocketOnTextMessage handles incoming WebSocket text messages.
func NdWebsocketOnTextMessage(input OnTextMessageInput) (OnTextMessageOutput, error) {
	if err := handleWebSocketMessage(input.ConnectionId, input.Message); err != nil {
		errMsg := err.Error()
		return OnTextMessageOutput{Error: &errMsg}, nil
	}
	return OnTextMessageOutput{}, nil
}

// NdWebsocketOnBinaryMessage handles incoming WebSocket binary messages.
func NdWebsocketOnBinaryMessage(input OnBinaryMessageInput) (OnBinaryMessageOutput, error) {
	pdk.Log(pdk.LogDebug, fmt.Sprintf("Received unexpected binary message for connection '%s'", input.ConnectionId))
	return OnBinaryMessageOutput{}, nil
}

// NdWebsocketOnError handles WebSocket errors.
func NdWebsocketOnError(input OnErrorInput) (OnErrorOutput, error) {
	pdk.Log(pdk.LogWarn, fmt.Sprintf("WebSocket error for connection '%s': %s", input.ConnectionId, input.Error))
	return OnErrorOutput{}, nil
}

// NdWebsocketOnClose handles WebSocket connection closure.
func NdWebsocketOnClose(input OnCloseInput) (OnCloseOutput, error) {
	pdk.Log(pdk.LogInfo, fmt.Sprintf("WebSocket connection '%s' closed with code %d: %s", input.ConnectionId, input.Code, input.Reason))
	return OnCloseOutput{}, nil
}

func main() {}
