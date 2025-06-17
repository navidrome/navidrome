package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host/artwork"
	"github.com/navidrome/navidrome/plugins/host/cache"
	"github.com/navidrome/navidrome/plugins/host/config"
	"github.com/navidrome/navidrome/plugins/host/http"
	"github.com/navidrome/navidrome/plugins/host/scheduler"
	"github.com/navidrome/navidrome/plugins/host/websocket"
	"github.com/navidrome/navidrome/utils/slice"
)

type DiscordRPPlugin struct {
	rpc     *discordRPC
	cfg     config.ConfigService
	artwork artwork.ArtworkService
	sched   scheduler.SchedulerService
}

func (d *DiscordRPPlugin) IsAuthorized(ctx context.Context, req *api.ScrobblerIsAuthorizedRequest) (*api.ScrobblerIsAuthorizedResponse, error) {
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

func (d *DiscordRPPlugin) NowPlaying(ctx context.Context, request *api.ScrobblerNowPlayingRequest) (*api.ScrobblerNowPlayingResponse, error) {
	log.Printf("Setting presence for user %s, track: %s", request.Username, request.Track.Name)

	// The plugin is stateless, we need to load the configuration every time
	clientID, users, err := d.getConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	// Check if the user has a Discord token configured
	userToken, authorized := users[request.Username]
	if !authorized {
		return nil, fmt.Errorf("user '%s' not authorized", request.Username)
	}

	// Make sure we have a connection
	if err := d.rpc.connect(ctx, request.Username, userToken); err != nil {
		return nil, fmt.Errorf("failed to connect to Discord: %w", err)
	}

	// Cancel any existing completion schedule
	if resp, _ := d.sched.CancelSchedule(ctx, &scheduler.CancelRequest{ScheduleId: request.Username}); resp.Error != "" {
		log.Printf("Ignoring failure to cancel schedule: %s", resp.Error)
	}

	// Send activity update
	if err := d.rpc.sendActivity(ctx, clientID, request.Username, userToken, activity{
		Application: clientID,
		Name:        "Navidrome",
		Type:        2,
		Details:     request.Track.Name,
		State:       d.getArtistList(request.Track),
		Timestamps: activityTimestamps{
			Start: (request.Timestamp - int64(request.Track.Position)) * 1000,
			End:   (request.Timestamp - int64(request.Track.Position) + int64(request.Track.Length)) * 1000,
		},
		Assets: activityAssets{
			LargeImage: d.imageURL(ctx, request),
			LargeText:  request.Track.Album,
		},
	}); err != nil {
		return nil, fmt.Errorf("failed to send activity: %w", err)
	}

	// Schedule a timer to clear the activity after the track completes
	_, err = d.sched.ScheduleOneTime(ctx, &scheduler.ScheduleOneTimeRequest{
		ScheduleId:   request.Username,
		DelaySeconds: request.Track.Length - request.Track.Position + 5,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to schedule completion timer: %w", err)
	}

	return nil, nil
}

func (d *DiscordRPPlugin) imageURL(ctx context.Context, request *api.ScrobblerNowPlayingRequest) string {
	imageResp, _ := d.artwork.GetTrackUrl(ctx, &artwork.GetArtworkUrlRequest{Id: request.Track.Id, Size: 300})
	imageURL := imageResp.Url
	if strings.HasPrefix(imageURL, "http://localhost") {
		return ""
	}
	return imageURL
}

func (d *DiscordRPPlugin) getArtistList(track *api.TrackInfo) string {
	return strings.Join(slice.Map(track.Artists, func(a *api.Artist) string { return a.Name }), " • ")
}

func (d *DiscordRPPlugin) Scrobble(context.Context, *api.ScrobblerScrobbleRequest) (*api.ScrobblerScrobbleResponse, error) {
	return nil, nil
}

func (d *DiscordRPPlugin) getConfig(ctx context.Context) (string, map[string]string, error) {
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

func (d *DiscordRPPlugin) OnSchedulerCallback(ctx context.Context, req *api.SchedulerCallbackRequest) (*api.SchedulerCallbackResponse, error) {
	log.Printf("Removing presence for user %s", req.ScheduleId)
	if err := d.rpc.clearActivity(ctx, req.ScheduleId); err != nil {
		return nil, fmt.Errorf("failed to clear activity: %w", err)
	}
	log.Printf("Disconnecting user %s", req.ScheduleId)
	if err := d.rpc.disconnect(ctx, req.ScheduleId); err != nil {
		return nil, fmt.Errorf("failed to disconnect from Discord: %w", err)
	}
	return nil, nil
}

// Creates a new instance of the DiscordRPPlugin, with all host services as dependencies
var plugin = &DiscordRPPlugin{
	cfg:     config.NewConfigService(),
	artwork: artwork.NewArtworkService(),
	rpc: &discordRPC{
		ws:  websocket.NewWebSocketService(),
		web: http.NewHttpService(),
		mem: cache.NewCacheService(),
	},
}

func init() {
	// Configure logging: No timestamps, no source file/line, prepend [Discord]
	log.SetFlags(0)
	log.SetPrefix("[Discord] ")

	// Register plugin capabilities
	api.RegisterScrobbler(plugin)
	api.RegisterWebSocketCallback(plugin.rpc)

	// Register named scheduler callbacks, and get the scheduler service for each
	plugin.sched = api.RegisterNamedSchedulerCallback("close-activity", plugin)
	plugin.rpc.sched = api.RegisterNamedSchedulerCallback("heartbeat", plugin.rpc)
}

func main() {}
