package plugins

import (
	"context"

	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/tetratelabs/wazero"
)

func NewWasmScrobblerPlugin(wasmPath, pluginName string, runtime api.WazeroNewRuntime, mc wazero.ModuleConfig) WasmPlugin {
	loader, err := api.NewScrobblerServicePlugin(context.Background(), api.WazeroRuntime(runtime), api.WazeroModuleConfig(mc))
	if err != nil {
		log.Error("Error creating scrobbler service plugin", "plugin", pluginName, "path", wasmPath, err)
		return nil
	}
	return &wasmScrobblerPlugin{
		wasmBasePlugin: &wasmBasePlugin[api.ScrobblerService, *api.ScrobblerServicePlugin]{
			wasmPath: wasmPath,
			name:     pluginName,
			loader:   loader,
			loadFunc: func(ctx context.Context, l *api.ScrobblerServicePlugin, path string) (api.ScrobblerService, error) {
				return l.Load(ctx, path)
			},
		},
	}
}

type wasmScrobblerPlugin struct {
	*wasmBasePlugin[api.ScrobblerService, *api.ScrobblerServicePlugin]
}

func (w *wasmScrobblerPlugin) PluginName() string {
	return w.name
}

func (w *wasmScrobblerPlugin) IsAuthorized(ctx context.Context, userId string) bool {
	result, err := callMethod(ctx, w, "IsAuthorized", func(inst api.ScrobblerService) (bool, error) {
		resp, err := inst.IsAuthorized(ctx, &api.ScrobblerIsAuthorizedRequest{UserId: userId})
		if err != nil {
			return false, err
		}
		if resp.Error != "" {
			return false, nil
		}
		return resp.Authorized, nil
	})
	return err == nil && result
}

func (w *wasmScrobblerPlugin) NowPlaying(ctx context.Context, userId string, track *model.MediaFile) error {
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
	}
	_, err := callMethod(ctx, w, "NowPlaying", func(inst api.ScrobblerService) (struct{}, error) {
		resp, err := inst.NowPlaying(ctx, &api.ScrobblerNowPlayingRequest{
			UserId: userId,
			Track:  trackInfo,
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
	track := &s.MediaFile
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
	}
	_, err := callMethod(ctx, w, "Scrobble", func(inst api.ScrobblerService) (struct{}, error) {
		resp, err := inst.Scrobble(ctx, &api.ScrobblerScrobbleRequest{
			UserId:    userId,
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
