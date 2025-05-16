package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host/cache"
	"github.com/navidrome/navidrome/plugins/host/config"
	"github.com/navidrome/navidrome/plugins/host/http"
	"github.com/navidrome/navidrome/plugins/host/scheduler"
	"github.com/navidrome/navidrome/plugins/host/websocket"
)

type DiscordRPPlugin struct {
	*discordRPC
	cfg   config.ConfigService
	sched scheduler.SchedulerService
}

func (d DiscordRPPlugin) IsAuthorized(ctx context.Context, req *api.ScrobblerIsAuthorizedRequest) (*api.ScrobblerIsAuthorizedResponse, error) {
	// Get plugin configuration
	_, users, err := d.getConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check user authorization: %w", err)
	}

	// Check if the user has a Discord token configured
	_, authorized := users[req.Username]
	log.Printf("IsAuthorized for user %s: %v", req.Username, authorized)
	return &api.ScrobblerIsAuthorizedResponse{
		Authorized: authorized,
	}, nil
}

func (d DiscordRPPlugin) NowPlaying(ctx context.Context, request *api.ScrobblerNowPlayingRequest) (*api.ScrobblerNowPlayingResponse, error) {
	log.Printf("Setting presence for user %s, track: %s", request.Username, request.Track.Name)

	clientID, users, err := d.getConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	// Check if the user has a Discord token configured
	userToken, authorized := users[request.Username]
	if !authorized {
		return nil, fmt.Errorf("user '%s' not authorized", request.Username)
	}

	if err := d.connect(ctx, request.Username, userToken); err != nil {
		return nil, fmt.Errorf("failed to connect to Discord: %w", err)
	}

	if _, err := d.sched.CancelSchedule(ctx, &scheduler.CancelRequest{ScheduleId: request.Username}); err != nil {
		return nil, fmt.Errorf("failed to cancel schedule: %w", err)
	}

	if err := d.sendActivity(ctx, request.Username, activity{
		Application: clientID,
		Name:        "Navidrome",
		Type:        2,
		Details:     request.Track.Name,
		State:       fmt.Sprintf("by %s", request.Track.GetArtists()[0].Name),
		Timestamps: activityTimestamps{
			Start: request.Timestamp,
			End:   request.Timestamp + int64(request.Track.Length),
		},
		Assets: activityAssets{
			LargeImage: "https://raw.githubusercontent.com/navidrome/navidrome/refs/heads/master/resources/album-placeholder.webp",
		},
	}); err != nil {
		return nil, fmt.Errorf("failed to send activity: %w", err)
	}

	_, err = d.sched.ScheduleOneTime(ctx, &scheduler.ScheduleOneTimeRequest{
		ScheduleId:   request.Username,
		DelaySeconds: request.Track.Length + 5,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to schedule completion timer: %w", err)
	}

	return nil, nil
}

func (d DiscordRPPlugin) Scrobble(context.Context, *api.ScrobblerScrobbleRequest) (*api.ScrobblerScrobbleResponse, error) {
	return nil, nil
}

func (d DiscordRPPlugin) getConfig(ctx context.Context) (string, map[string]string, error) {
	const (
		clientIDKey = "clientid"
		usersKey    = "users"
	)
	confResp, err := d.cfg.GetPluginConfig(ctx, &config.GetPluginConfigRequest{})
	if err != nil {
		return "", nil, fmt.Errorf("unable to load config: %w", err)
	}
	conf := confResp.GetConfig()
	if len(conf) < 1 {
		log.Print("missing configuration")
		return "", nil, nil
	}
	clientID := conf[clientIDKey]
	if clientID == "" {
		log.Printf("missing ClientID: %v", conf)
		return "", nil, nil
	}
	cfgUsers := conf[usersKey]
	if len(cfgUsers) == 0 {
		log.Print("no users configured")
		return "", nil, nil
	}
	users := map[string]string{}
	for _, user := range strings.Split(cfgUsers, ",") {
		tuple := strings.Split(user, ":")
		if len(tuple) != 2 {
			return clientID, nil, fmt.Errorf("invalid user config: %s", user)
		}
		users[tuple[0]] = tuple[1]
	}
	return clientID, users, nil
}

func (d DiscordRPPlugin) OnSchedulerCallback(ctx context.Context, req *api.SchedulerCallbackRequest) (*api.SchedulerCallbackResponse, error) {
	log.Printf("Removing presence for user %s", req.ScheduleId)
	if err := d.clearActivity(ctx, req.ScheduleId); err != nil {
		return nil, fmt.Errorf("failed to clear activity: %w", err)
	}
	return nil, nil
}

var plugin = DiscordRPPlugin{
	cfg:   config.NewConfigService(),
	sched: scheduler.NewSchedulerService(),
	discordRPC: &discordRPC{
		ws:  websocket.NewWebSocketService(),
		web: http.NewHttpService(),
		mem: cache.NewCacheService(),
	},
}

func init() {
	log.SetFlags(0)
	log.SetPrefix("[Discord] ")

	api.RegisterScrobbler(plugin)
	api.RegisterSchedulerCallback(plugin)
	api.RegisterWebSocketCallback(plugin)
}

func main() {}
