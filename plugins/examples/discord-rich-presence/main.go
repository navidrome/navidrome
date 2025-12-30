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
	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/scheduler"
	"github.com/navidrome/navidrome/plugins/pdk/go/scrobbler"
	"github.com/navidrome/navidrome/plugins/pdk/go/websocket"
)

// Configuration keys
const (
	clientIDKey = "clientid"
	usersKey    = "users"
)

// discordPlugin implements the scrobbler, scheduler, and websocket interfaces.
type discordPlugin struct{}

// init registers the plugin capabilities
func init() {
	scrobbler.Register(&discordPlugin{})
	scheduler.Register(&discordPlugin{})
	websocket.Register(&discordPlugin{})
}

// Ensure discordPlugin implements the required provider interfaces
var (
	_ scrobbler.Scrobbler             = (*discordPlugin)(nil)
	_ scheduler.CallbackProvider      = (*discordPlugin)(nil)
	_ websocket.TextMessageProvider   = (*discordPlugin)(nil)
	_ websocket.BinaryMessageProvider = (*discordPlugin)(nil)
	_ websocket.ErrorProvider         = (*discordPlugin)(nil)
	_ websocket.CloseProvider         = (*discordPlugin)(nil)
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
	artworkURL, err := host.ArtworkGetTrackUrl(trackID, 300)
	if err != nil {
		pdk.Log(pdk.LogWarn, fmt.Sprintf("Failed to get artwork URL: %v", err))
		return ""
	}

	// Don't use localhost URLs
	if strings.HasPrefix(artworkURL, "http://localhost") {
		return ""
	}
	return artworkURL
}

// ============================================================================
// Scrobbler Implementation
// ============================================================================

// IsAuthorized checks if a user is authorized for Discord Rich Presence.
func (p *discordPlugin) IsAuthorized(input scrobbler.IsAuthorizedRequest) (*scrobbler.IsAuthorizedResponse, error) {
	_, users, err := getConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to check user authorization: %w", err)
	}

	_, authorized := users[input.Username]
	pdk.Log(pdk.LogInfo, fmt.Sprintf("IsAuthorized for user %s: %v", input.Username, authorized))
	return &scrobbler.IsAuthorizedResponse{Authorized: authorized}, nil
}

// NowPlaying sends a now playing notification to Discord.
func (p *discordPlugin) NowPlaying(input scrobbler.NowPlayingRequest) error {
	pdk.Log(pdk.LogInfo, fmt.Sprintf("Setting presence for user %s, track: %s", input.Username, input.Track.Title))

	// Load configuration
	clientID, users, err := getConfig()
	if err != nil {
		return fmt.Errorf("%w: failed to get config: %v", scrobbler.ScrobblerErrorRetryLater, err)
	}

	// Check authorization
	userToken, authorized := users[input.Username]
	if !authorized {
		return fmt.Errorf("%w: user '%s' not authorized", scrobbler.ScrobblerErrorNotAuthorized, input.Username)
	}

	// Connect to Discord
	if err := connect(input.Username, userToken); err != nil {
		return fmt.Errorf("%w: failed to connect to Discord: %v", scrobbler.ScrobblerErrorRetryLater, err)
	}

	// Cancel any existing completion schedule
	_ = host.SchedulerCancelSchedule(fmt.Sprintf("%s-clear", input.Username))

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
			LargeImage: getImageURL(input.Track.ID),
			LargeText:  input.Track.Album,
		},
	}); err != nil {
		return fmt.Errorf("%w: failed to send activity: %v", scrobbler.ScrobblerErrorRetryLater, err)
	}

	// Schedule a timer to clear the activity after the track completes
	remainingSeconds := int32(input.Track.Duration) - input.Position + 5
	_, err = host.SchedulerScheduleOneTime(remainingSeconds, payloadClearActivity, fmt.Sprintf("%s-clear", input.Username))
	if err != nil {
		pdk.Log(pdk.LogWarn, fmt.Sprintf("Failed to schedule completion timer: %v", err))
	}

	return nil
}

// Scrobble handles scrobble requests (no-op for Discord).
func (p *discordPlugin) Scrobble(_ scrobbler.ScrobbleRequest) error {
	// Discord Rich Presence doesn't need scrobble events
	return nil
}

// ============================================================================
// Scheduler Callback Implementation
// ============================================================================

// OnCallback handles scheduler callbacks.
func (p *discordPlugin) OnCallback(input scheduler.SchedulerCallbackRequest) error {
	pdk.Log(pdk.LogDebug, fmt.Sprintf("Scheduler callback: id=%s, payload=%s, recurring=%v", input.ScheduleID, input.Payload, input.IsRecurring))

	// Route based on payload
	switch input.Payload {
	case payloadHeartbeat:
		// Heartbeat callback - scheduleId is the username
		if err := handleHeartbeatCallback(input.ScheduleID); err != nil {
			return err
		}

	case payloadClearActivity:
		// Clear activity callback - scheduleId is "username-clear"
		username := strings.TrimSuffix(input.ScheduleID, "-clear")
		if err := handleClearActivityCallback(username); err != nil {
			return err
		}

	default:
		pdk.Log(pdk.LogWarn, fmt.Sprintf("Unknown scheduler callback payload: %s", input.Payload))
	}

	return nil
}

// ============================================================================
// WebSocket Callback Implementation
// ============================================================================

// OnTextMessage handles incoming WebSocket text messages.
func (p *discordPlugin) OnTextMessage(input websocket.OnTextMessageRequest) error {
	return handleWebSocketMessage(input.ConnectionID, input.Message)
}

// OnBinaryMessage handles incoming WebSocket binary messages.
func (p *discordPlugin) OnBinaryMessage(input websocket.OnBinaryMessageRequest) error {
	pdk.Log(pdk.LogDebug, fmt.Sprintf("Received unexpected binary message for connection '%s'", input.ConnectionID))
	return nil
}

// OnError handles WebSocket errors.
func (p *discordPlugin) OnError(input websocket.OnErrorRequest) error {
	pdk.Log(pdk.LogWarn, fmt.Sprintf("WebSocket error for connection '%s': %s", input.ConnectionID, input.Error))
	return nil
}

// OnClose handles WebSocket connection closure.
func (p *discordPlugin) OnClose(input websocket.OnCloseRequest) error {
	pdk.Log(pdk.LogInfo, fmt.Sprintf("WebSocket connection '%s' closed with code %d: %s", input.ConnectionID, input.Code, input.Reason))
	return nil
}

func main() {}
