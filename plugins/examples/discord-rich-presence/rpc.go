package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host/cache"
	"github.com/navidrome/navidrome/plugins/host/http"
	"github.com/navidrome/navidrome/plugins/host/scheduler"
	"github.com/navidrome/navidrome/plugins/host/websocket"
)

type discordRPC struct {
	ws    websocket.WebSocketService
	web   http.HttpService
	mem   cache.CacheService
	sched scheduler.SchedulerService
}

// Discord WebSocket Gateway constants
const (
	heartbeatOpCode = 1 // Heartbeat operation code
	gateOpCode      = 2 // Identify operation code
	presenceOpCode  = 3 // Presence update operation code
)

const (
	heartbeatInterval = 41 // Heartbeat interval in seconds
	defaultImage      = "https://i.imgur.com/hb3XPzA.png"
)

// Activity is a struct that represents an activity in Discord.
type activity struct {
	Name        string             `json:"name"`
	Type        int                `json:"type"`
	Details     string             `json:"details"`
	State       string             `json:"state"`
	Application string             `json:"application_id"`
	Timestamps  activityTimestamps `json:"timestamps"`
	Assets      activityAssets     `json:"assets"`
}

type activityTimestamps struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

type activityAssets struct {
	LargeImage string `json:"large_image"`
	LargeText  string `json:"large_text"`
}

// PresencePayload is a struct that represents a presence update in Discord.
type presencePayload struct {
	Activities []activity `json:"activities"`
	Since      int64      `json:"since"`
	Status     string     `json:"status"`
	Afk        bool       `json:"afk"`
}

// IdentifyPayload is a struct that represents an identify payload in Discord.
type identifyPayload struct {
	Token      string             `json:"token"`
	Intents    int                `json:"intents"`
	Properties identifyProperties `json:"properties"`
}

type identifyProperties struct {
	OS      string `json:"os"`
	Browser string `json:"browser"`
	Device  string `json:"device"`
}

func (r *discordRPC) processImage(ctx context.Context, imageURL string, clientID string, token string) (string, error) {
	return r.processImageWithFallback(ctx, imageURL, clientID, token, false)
}

func (r *discordRPC) processImageWithFallback(ctx context.Context, imageURL string, clientID string, token string, isDefaultImage bool) (string, error) {
	// Check if context is canceled
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("context canceled: %w", err)
	}

	if imageURL == "" {
		if isDefaultImage {
			// We're already processing the default image and it's empty, return error
			return "", fmt.Errorf("default image URL is empty")
		}
		return r.processImageWithFallback(ctx, defaultImage, clientID, token, true)
	}

	if strings.HasPrefix(imageURL, "mp:") {
		return imageURL, nil
	}

	// Check cache first
	cacheKey := fmt.Sprintf("discord.image.%x", imageURL)
	cacheResp, _ := r.mem.GetString(ctx, &cache.GetRequest{Key: cacheKey})
	if cacheResp.Exists {
		log.Printf("Cache hit for image URL: %s", imageURL)
		return cacheResp.Value, nil
	}

	resp, _ := r.web.Post(ctx, &http.HttpRequest{
		Url: fmt.Sprintf("https://discord.com/api/v9/applications/%s/external-assets", clientID),
		Headers: map[string]string{
			"Authorization": token,
			"Content-Type":  "application/json",
		},
		Body: fmt.Appendf(nil, `{"urls":[%q]}`, imageURL),
	})

	// Handle HTTP error responses
	if resp.Status >= 400 {
		if isDefaultImage {
			return "", fmt.Errorf("failed to process default image: HTTP %d %s", resp.Status, resp.Error)
		}
		return r.processImageWithFallback(ctx, defaultImage, clientID, token, true)
	}
	if resp.Error != "" {
		if isDefaultImage {
			// If we're already processing the default image and it fails, return error
			return "", fmt.Errorf("failed to process default image: %s", resp.Error)
		}
		// Try with default image
		return r.processImageWithFallback(ctx, defaultImage, clientID, token, true)
	}

	var data []map[string]string
	if err := json.Unmarshal(resp.Body, &data); err != nil {
		if isDefaultImage {
			// If we're already processing the default image and it fails, return error
			return "", fmt.Errorf("failed to unmarshal default image response: %w", err)
		}
		// Try with default image
		return r.processImageWithFallback(ctx, defaultImage, clientID, token, true)
	}

	if len(data) == 0 {
		if isDefaultImage {
			// If we're already processing the default image and it fails, return error
			return "", fmt.Errorf("no data returned for default image")
		}
		// Try with default image
		return r.processImageWithFallback(ctx, defaultImage, clientID, token, true)
	}

	image := data[0]["external_asset_path"]
	if image == "" {
		if isDefaultImage {
			// If we're already processing the default image and it fails, return error
			return "", fmt.Errorf("empty external_asset_path for default image")
		}
		// Try with default image
		return r.processImageWithFallback(ctx, defaultImage, clientID, token, true)
	}

	processedImage := fmt.Sprintf("mp:%s", image)

	// Cache the processed image URL
	var ttl = 4 * time.Hour // 4 hours for regular images
	if isDefaultImage {
		ttl = 48 * time.Hour // 48 hours for default image
	}

	_, _ = r.mem.SetString(ctx, &cache.SetStringRequest{
		Key:        cacheKey,
		Value:      processedImage,
		TtlSeconds: int64(ttl.Seconds()),
	})

	log.Printf("Cached processed image URL for %s (TTL: %s seconds)", imageURL, ttl)

	return processedImage, nil
}

func (r *discordRPC) sendActivity(ctx context.Context, clientID, username, token string, data activity) error {
	log.Printf("Sending activity to for user %s: %#v", username, data)

	processedImage, err := r.processImage(ctx, data.Assets.LargeImage, clientID, token)
	if err != nil {
		log.Printf("Failed to process image for user %s, continuing without image: %v", username, err)
		// Clear the image and continue without it
		data.Assets.LargeImage = ""
	} else {
		log.Printf("Processed image for URL %s: %s", data.Assets.LargeImage, processedImage)
		data.Assets.LargeImage = processedImage
	}

	presence := presencePayload{
		Activities: []activity{data},
		Status:     "dnd",
		Afk:        false,
	}
	return r.sendMessage(ctx, username, presenceOpCode, presence)
}

func (r *discordRPC) clearActivity(ctx context.Context, username string) error {
	log.Printf("Clearing activity for user %s", username)
	return r.sendMessage(ctx, username, presenceOpCode, presencePayload{})
}

func (r *discordRPC) sendMessage(ctx context.Context, username string, opCode int, payload any) error {
	message := map[string]any{
		"op": opCode,
		"d":  payload,
	}
	b, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal presence update: %w", err)
	}

	resp, _ := r.ws.SendText(ctx, &websocket.SendTextRequest{
		ConnectionId: username,
		Message:      string(b),
	})
	if resp.Error != "" {
		return fmt.Errorf("failed to send presence update: %s", resp.Error)
	}
	return nil
}

func (r *discordRPC) getDiscordGateway(ctx context.Context) (string, error) {
	resp, _ := r.web.Get(ctx, &http.HttpRequest{
		Url: "https://discord.com/api/gateway",
	})
	if resp.Error != "" {
		return "", fmt.Errorf("failed to get Discord gateway: %s", resp.Error)
	}
	var result map[string]string
	err := json.Unmarshal(resp.Body, &result)
	if err != nil {
		return "", fmt.Errorf("failed to parse Discord gateway response: %w", err)
	}
	return result["url"], nil
}

func (r *discordRPC) sendHeartbeat(ctx context.Context, username string) error {
	resp, _ := r.mem.GetInt(ctx, &cache.GetRequest{
		Key: fmt.Sprintf("discord.seq.%s", username),
	})
	log.Printf("Sending heartbeat for user %s: %d", username, resp.Value)
	return r.sendMessage(ctx, username, heartbeatOpCode, resp.Value)
}

func (r *discordRPC) cleanupFailedConnection(ctx context.Context, username string) {
	log.Printf("Cleaning up failed connection for user %s", username)

	// Cancel the heartbeat schedule
	if resp, _ := r.sched.CancelSchedule(ctx, &scheduler.CancelRequest{ScheduleId: username}); resp.Error != "" {
		log.Printf("Failed to cancel heartbeat schedule for user %s: %s", username, resp.Error)
	}

	// Close the WebSocket connection
	if resp, _ := r.ws.Close(ctx, &websocket.CloseRequest{
		ConnectionId: username,
		Code:         1000,
		Reason:       "Connection lost",
	}); resp.Error != "" {
		log.Printf("Failed to close WebSocket connection for user %s: %s", username, resp.Error)
	}

	// Clean up cache entries (just the sequence number, no failure tracking needed)
	_, _ = r.mem.Remove(ctx, &cache.RemoveRequest{Key: fmt.Sprintf("discord.seq.%s", username)})

	log.Printf("Cleaned up connection for user %s", username)
}

func (r *discordRPC) isConnected(ctx context.Context, username string) bool {
	// Try to send a heartbeat to test the connection
	err := r.sendHeartbeat(ctx, username)
	if err != nil {
		log.Printf("Heartbeat test failed for user %s: %v", username, err)
		return false
	}
	return true
}

func (r *discordRPC) connect(ctx context.Context, username string, token string) error {
	if r.isConnected(ctx, username) {
		log.Printf("Reusing existing connection for user %s", username)
		return nil
	}
	log.Printf("Creating new connection for user %s", username)

	// Get Discord Gateway URL
	gateway, err := r.getDiscordGateway(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Discord gateway: %w", err)
	}
	log.Printf("Using gateway: %s", gateway)

	// Connect to Discord Gateway
	resp, _ := r.ws.Connect(ctx, &websocket.ConnectRequest{
		ConnectionId: username,
		Url:          gateway,
	})
	if resp.Error != "" {
		return fmt.Errorf("failed to connect to WebSocket: %s", resp.Error)
	}

	// Send identify payload
	payload := identifyPayload{
		Token:   token,
		Intents: 0,
		Properties: identifyProperties{
			OS:      "Windows 10",
			Browser: "Discord Client",
			Device:  "Discord Client",
		},
	}
	err = r.sendMessage(ctx, username, gateOpCode, payload)
	if err != nil {
		return fmt.Errorf("failed to send identify payload: %w", err)
	}

	// Schedule heartbeats for this user/connection
	cronResp, _ := r.sched.ScheduleRecurring(ctx, &scheduler.ScheduleRecurringRequest{
		CronExpression: fmt.Sprintf("@every %ds", heartbeatInterval),
		ScheduleId:     username,
	})
	log.Printf("Scheduled heartbeat for user %s with ID %s", username, cronResp.ScheduleId)

	log.Printf("Successfully authenticated user %s", username)
	return nil
}

func (r *discordRPC) disconnect(ctx context.Context, username string) error {
	if resp, _ := r.sched.CancelSchedule(ctx, &scheduler.CancelRequest{ScheduleId: username}); resp.Error != "" {
		return fmt.Errorf("failed to cancel schedule: %s", resp.Error)
	}
	resp, _ := r.ws.Close(ctx, &websocket.CloseRequest{
		ConnectionId: username,
		Code:         1000,
		Reason:       "Navidrome disconnect",
	})
	if resp.Error != "" {
		return fmt.Errorf("failed to close WebSocket connection: %s", resp.Error)
	}
	return nil
}

func (r *discordRPC) OnTextMessage(ctx context.Context, req *api.OnTextMessageRequest) (*api.OnTextMessageResponse, error) {
	if len(req.Message) < 1024 {
		log.Printf("Received WebSocket message for connection '%s': %s", req.ConnectionId, req.Message)
	} else {
		log.Printf("Received WebSocket message for connection '%s' (truncated): %s...", req.ConnectionId, req.Message[:1021])
	}

	// Parse the message. If it's a heartbeat_ack, store the sequence number.
	message := map[string]any{}
	err := json.Unmarshal([]byte(req.Message), &message)
	if err != nil {
		return nil, fmt.Errorf("failed to parse WebSocket message: %w", err)
	}
	if v := message["s"]; v != nil {
		seq := int64(v.(float64))
		log.Printf("Received heartbeat_ack for connection '%s': %d", req.ConnectionId, seq)
		resp, _ := r.mem.SetInt(ctx, &cache.SetIntRequest{
			Key:        fmt.Sprintf("discord.seq.%s", req.ConnectionId),
			Value:      seq,
			TtlSeconds: heartbeatInterval * 2,
		})
		if !resp.Success {
			return nil, fmt.Errorf("failed to store sequence number for user %s", req.ConnectionId)
		}
	}
	return nil, nil
}

func (r *discordRPC) OnBinaryMessage(_ context.Context, req *api.OnBinaryMessageRequest) (*api.OnBinaryMessageResponse, error) {
	log.Printf("Received unexpected binary message for connection '%s'", req.ConnectionId)
	return nil, nil
}

func (r *discordRPC) OnError(_ context.Context, req *api.OnErrorRequest) (*api.OnErrorResponse, error) {
	log.Printf("WebSocket error for connection '%s': %s", req.ConnectionId, req.Error)
	return nil, nil
}

func (r *discordRPC) OnClose(_ context.Context, req *api.OnCloseRequest) (*api.OnCloseResponse, error) {
	log.Printf("WebSocket connection '%s' closed with code %d: %s", req.ConnectionId, req.Code, req.Reason)
	return nil, nil
}

func (r *discordRPC) OnSchedulerCallback(ctx context.Context, req *api.SchedulerCallbackRequest) (*api.SchedulerCallbackResponse, error) {
	err := r.sendHeartbeat(ctx, req.ScheduleId)
	if err != nil {
		// On first heartbeat failure, immediately clean up the connection
		// The next NowPlaying call will reconnect if needed
		log.Printf("Heartbeat failed for user %s, cleaning up connection: %v", req.ScheduleId, err)
		r.cleanupFailedConnection(ctx, req.ScheduleId)
		return nil, fmt.Errorf("heartbeat failed, connection cleaned up: %w", err)
	}

	return nil, nil
}
