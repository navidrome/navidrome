package plugins

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/tetratelabs/wazero"
)

func newWasmScrobblerPlugin(wasmPath, pluginID string, m *managerImpl, runtime api.WazeroNewRuntime, mc wazero.ModuleConfig) WasmPlugin {
	loader, err := api.NewScrobblerPlugin(context.Background(), api.WazeroRuntime(runtime), api.WazeroModuleConfig(mc))
	if err != nil {
		log.Error("Error creating scrobbler service plugin", "plugin", pluginID, "path", wasmPath, err)
		return nil
	}
	return &wasmScrobblerPlugin{
		baseCapability: newBaseCapability[api.Scrobbler, *api.ScrobblerPlugin](
			wasmPath,
			pluginID,
			CapabilityScrobbler,
			m.metrics,
			loader,
			func(ctx context.Context, l *api.ScrobblerPlugin, path string) (api.Scrobbler, error) {
				return l.Load(ctx, path)
			},
		),
	}
}

type wasmScrobblerPlugin struct {
	*baseCapability[api.Scrobbler, *api.ScrobblerPlugin]
}

func (w *wasmScrobblerPlugin) IsAuthorized(ctx context.Context, userId string) bool {
	username, _ := request.UsernameFrom(ctx)
	if username == "" {
		u, ok := request.UserFrom(ctx)
		if ok {
			username = u.UserName
		}
	}
	resp, err := callMethod(ctx, w, "IsAuthorized", func(inst api.Scrobbler) (*api.ScrobblerIsAuthorizedResponse, error) {
		return inst.IsAuthorized(ctx, &api.ScrobblerIsAuthorizedRequest{
			UserId:   userId,
			Username: username,
		})
	})
	if err != nil {
		log.Warn("Error calling IsAuthorized", "userId", userId, "pluginID", w.id, err)
	}
	return err == nil && resp.Authorized
}

func (w *wasmScrobblerPlugin) NowPlaying(ctx context.Context, userId string, track *model.MediaFile, position int) error {
	username, _ := request.UsernameFrom(ctx)
	if username == "" {
		u, ok := request.UserFrom(ctx)
		if ok {
			username = u.UserName
		}
	}

	trackInfo := w.toTrackInfo(track, position)
	_, err := callMethod(ctx, w, "NowPlaying", func(inst api.Scrobbler) (struct{}, error) {
		resp, err := inst.NowPlaying(ctx, &api.ScrobblerNowPlayingRequest{
			UserId:    userId,
			Username:  username,
			Track:     trackInfo,
			Timestamp: time.Now().Unix(),
		})
		if err != nil {
			return struct{}{}, err
		}
		if resp.Error != "" {
			return struct{}{}, nil
		}
		return struct{}{}, nil
	})
	return err
}

func (w *wasmScrobblerPlugin) Scrobble(ctx context.Context, userId string, s scrobbler.Scrobble) error {
	username, _ := request.UsernameFrom(ctx)
	if username == "" {
		u, ok := request.UserFrom(ctx)
		if ok {
			username = u.UserName
		}
	}
	trackInfo := w.toTrackInfo(&s.MediaFile, 0)
	_, err := callMethod(ctx, w, "Scrobble", func(inst api.Scrobbler) (struct{}, error) {
		resp, err := inst.Scrobble(ctx, &api.ScrobblerScrobbleRequest{
			UserId:    userId,
			Username:  username,
			Track:     trackInfo,
			Timestamp: s.TimeStamp.Unix(),
		})
		if err != nil {
			return struct{}{}, err
		}
		if resp.Error != "" {
			return struct{}{}, nil
		}
		return struct{}{}, nil
	})
	return err
}

func (w *wasmScrobblerPlugin) toTrackInfo(track *model.MediaFile, position int) *api.TrackInfo {
	artists := make([]*api.Artist, 0, len(track.Participants[model.RoleArtist]))

	for _, a := range track.Participants[model.RoleArtist] {
		artists = append(artists, &api.Artist{Name: a.Name, Mbid: a.MbzArtistID})
	}
	albumArtists := make([]*api.Artist, 0, len(track.Participants[model.RoleAlbumArtist]))
	for _, a := range track.Participants[model.RoleAlbumArtist] {
		albumArtists = append(albumArtists, &api.Artist{Name: a.Name, Mbid: a.MbzArtistID})
	}
	trackInfo := &api.TrackInfo{
		Id:           track.ID,
		Mbid:         track.MbzRecordingID,
		Name:         track.Title,
		Album:        track.Album,
		AlbumMbid:    track.MbzAlbumID,
		Artists:      artists,
		AlbumArtists: albumArtists,
		Length:       int32(track.Duration),
		Position:     int32(position),
	}
	return trackInfo
}
