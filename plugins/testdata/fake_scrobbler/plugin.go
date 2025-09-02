//go:build wasip1

package main

import (
	"context"
	"log"

	"github.com/navidrome/navidrome/plugins/api"
)

type FakeScrobbler struct{}

func (FakeScrobbler) IsAuthorized(ctx context.Context, req *api.ScrobblerIsAuthorizedRequest) (*api.ScrobblerIsAuthorizedResponse, error) {
	log.Printf("[FakeScrobbler] IsAuthorized called for user: %s (%s)", req.Username, req.UserId)
	return &api.ScrobblerIsAuthorizedResponse{Authorized: true}, nil
}

func (FakeScrobbler) NowPlaying(ctx context.Context, req *api.ScrobblerNowPlayingRequest) (*api.ScrobblerNowPlayingResponse, error) {
	log.Printf("[FakeScrobbler] NowPlaying called for user: %s (%s), track: %s", req.Username, req.UserId, req.Track.Name)
	return &api.ScrobblerNowPlayingResponse{}, nil
}

func (FakeScrobbler) Scrobble(ctx context.Context, req *api.ScrobblerScrobbleRequest) (*api.ScrobblerScrobbleResponse, error) {
	log.Printf("[FakeScrobbler] Scrobble called for user: %s (%s), track: %s, timestamp: %d", req.Username, req.UserId, req.Track.Name, req.Timestamp)
	return &api.ScrobblerScrobbleResponse{}, nil
}

func main() {}

func init() {
	api.RegisterScrobbler(FakeScrobbler{})
}
