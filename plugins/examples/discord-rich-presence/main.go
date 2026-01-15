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

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/scheduler"
	"github.com/navidrome/navidrome/plugins/pdk/go/scrobbler"
	"github.com/navidrome/navidrome/plugins/pdk/go/websocket"
)

// Configuration keys
const (
	clientIDKey   = "clientid"
	userKeyPrefix = "user."
)

// discordPlugin implements the scrobbler and scheduler interfaces.
type discordPlugin struct{}

// rpc handles Discord gateway communication (via websockets).
var rpc = &discordRPC{}

// init registers the plugin capabilities
func init() {
	scrobbler.Register(&discordPlugin{})
	scheduler.Register(&discordPlugin{})
	websocket.Register(rpc)
}

// getConfig loads the plugin configuration.
func getConfig() (clientID string, users map[string]string, err error) {
	clientID, ok := pdk.GetConfig(clientIDKey)
	if !ok || clientID == "" {
		pdk.Log(pdk.LogWarn, "missing ClientID in configuration")
		return "", nil, nil
	}

	// Get all user keys with the "user." prefix
	userKeys := host.ConfigKeys(userKeyPrefix)
	if len(userKeys) == 0 {
		pdk.Log(pdk.LogWarn, "no users configured")
		return clientID, nil, nil
	}

	users = make(map[string]string)
	for _, key := range userKeys {
		username := strings.TrimPrefix(key, userKeyPrefix)
		token, exists := host.ConfigGet(key)
		if exists && token != "" {
			users[username] = token
		}
	}

	if len(users) == 0 {
		pdk.Log(pdk.LogWarn, "no users configured")
		return clientID, nil, nil
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
func (p *discordPlugin) IsAuthorized(input scrobbler.IsAuthorizedRequest) (bool, error) {
	_, users, err := getConfig()
	if err != nil {
		return false, fmt.Errorf("failed to check user authorization: %w", err)
	}

	_, authorized := users[input.Username]
	pdk.Log(pdk.LogInfo, fmt.Sprintf("IsAuthorized for user %s: %v", input.Username, authorized))
	return authorized, nil
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
	if err := rpc.connect(input.Username, userToken); err != nil {
		return fmt.Errorf("%w: failed to connect to Discord: %v", scrobbler.ScrobblerErrorRetryLater, err)
	}

	// Cancel any existing completion schedule
	_ = host.SchedulerCancelSchedule(fmt.Sprintf("%s-clear", input.Username))

	// Calculate timestamps
	now := time.Now().Unix()
	startTime := (now - int64(input.Position)) * 1000
	endTime := startTime + int64(input.Track.Duration)*1000

	// Send activity update
	if err := rpc.sendActivity(clientID, input.Username, userToken, activity{
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
		if err := rpc.handleHeartbeatCallback(input.ScheduleID); err != nil {
			return err
		}

	case payloadClearActivity:
		// Clear activity callback - scheduleId is "username-clear"
		username := strings.TrimSuffix(input.ScheduleID, "-clear")
		if err := rpc.handleClearActivityCallback(username); err != nil {
			return err
		}

	default:
		pdk.Log(pdk.LogWarn, fmt.Sprintf("Unknown scheduler callback payload: %s", input.Payload))
	}

	return nil
}

func main() {}
