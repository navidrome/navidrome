package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host/cache"
	"github.com/navidrome/navidrome/plugins/host/http"
	"github.com/navidrome/navidrome/plugins/host/websocket"
)

type discordRPC struct {
	ws  websocket.WebSocketService
	web http.HttpService
	mem cache.CacheService
}

// Discord WebSocket Gateway constants
const (
	heartbeatOpCode = 1 // Heartbeat operation code
	gateOpCode      = 2 // Identify operation code
	presenceOpCode  = 3 // Presence update operation code
)

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
}

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

type presencePayload struct {
	Activities []activity `json:"activities"`
	Since      int64      `json:"since"`
	Status     string     `json:"status"`
	Afk        bool       `json:"afk"`
}

func (r *discordRPC) sendActivity(ctx context.Context, username string, data activity) error {
	log.Printf("Sending activity to for user %s: %#v", username, data)
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
	resp, err := r.web.Get(ctx, &http.HttpRequest{
		Url: "https://discord.com/api/gateway",
	})
	if err != nil {
		return "", fmt.Errorf("failed to get Discord gateway: %w", err)
	}
	var result map[string]string
	err = json.Unmarshal(resp.Body, &result)
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

func (r *discordRPC) isConnected(ctx context.Context, username string) bool {
	err := r.sendHeartbeat(ctx, username)
	return err == nil
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

	log.Printf("Successfully authenticated user %s", username)
	return nil
}

func (r *discordRPC) OnTextMessage(ctx context.Context, req *api.OnTextMessageRequest) (*api.OnTextMessageResponse, error) {
	if len(req.Message) < 1024 {
		log.Printf("Received WebSocket message for connection '%s': %s", req.ConnectionId, req.Message)
	} else {
		log.Printf("Received WebSocket message for connection '%s' (truncated): %s...", req.ConnectionId, req.Message[:1021])
	}
	message := map[string]any{}
	err := json.Unmarshal([]byte(req.Message), &message)
	if err != nil {
		return nil, fmt.Errorf("failed to parse WebSocket message: %w", err)
	}
	if seq := message["s"]; seq != nil {
		log.Printf("Received heartbeat_ack for connection '%s': %f", req.ConnectionId, seq)
		resp, _ := r.mem.SetInt(ctx, &cache.SetIntRequest{
			Key:        fmt.Sprintf("discord.seq.%s", req.ConnectionId),
			Value:      int64(seq.(float64)),
			TtlSeconds: 300,
		})
		if !resp.Success {
			return nil, fmt.Errorf("failed to store sequence number for user %s", req.ConnectionId)
		}
	}
	return nil, nil
}

func (r *discordRPC) OnBinaryMessage(ctx context.Context, req *api.OnBinaryMessageRequest) (*api.OnBinaryMessageResponse, error) {
	log.Printf("Received unexpected binary message for connection '%s'", req.ConnectionId)
	return nil, nil
}

func (r *discordRPC) OnError(ctx context.Context, req *api.OnErrorRequest) (*api.OnErrorResponse, error) {
	log.Printf("WebSocket error for connection '%s': %s", req.ConnectionId, req.Error)
	return nil, nil
}

func (r *discordRPC) OnClose(ctx context.Context, req *api.OnCloseRequest) (*api.OnCloseResponse, error) {
	log.Printf("WebSocket connection '%s' closed with code %d: %s", req.ConnectionId, req.Code, req.Reason)
	return nil, nil
}
