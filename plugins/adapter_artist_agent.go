package plugins

import (
	"context"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/tetratelabs/wazero"
)

func NewWasmArtistAgent(wasmPath, pluginName string, runtime api.WazeroNewRuntime, mc wazero.ModuleConfig) WasmPlugin {
	loader, _ := api.NewArtistMetadataServicePlugin(context.Background(), api.WazeroRuntime(runtime), api.WazeroModuleConfig(mc))
	return &wasmArtistAgent{
		wasmBasePlugin: &wasmBasePlugin[api.ArtistMetadataService, *api.ArtistMetadataServicePlugin]{
			wasmPath: wasmPath,
			name:     pluginName,
			loader:   loader,
			loadFunc: func(ctx context.Context, l *api.ArtistMetadataServicePlugin, path string) (api.ArtistMetadataService, error) {
				return l.Load(ctx, path)
			},
		},
	}
}

type wasmArtistAgent struct {
	*wasmBasePlugin[api.ArtistMetadataService, *api.ArtistMetadataServicePlugin]
}

func (w *wasmArtistAgent) AgentName() string {
	return w.name
}

func (w *wasmArtistAgent) PluginName() string {
	return w.name
}

func (w *wasmArtistAgent) mapError(err error) error {
	if err != nil && (err.Error() == api.ErrNotFound.Error() || err.Error() == api.ErrNotImplemented.Error()) {
		return agents.ErrNotFound
	}
	return err
}

func (w *wasmArtistAgent) GetArtistMBID(ctx context.Context, id string, name string) (string, error) {
	return callMethod(ctx, w, "GetArtistMBID", func(inst api.ArtistMetadataService) (string, error) {
		res, err := inst.GetArtistMBID(ctx, &api.ArtistMBIDRequest{Id: id, Name: name})
		if err != nil {
			return "", err
		}
		return res.GetMbid(), nil
	})
}

func (w *wasmArtistAgent) GetArtistURL(ctx context.Context, id, name, mbid string) (string, error) {
	return callMethod(ctx, w, "GetArtistURL", func(inst api.ArtistMetadataService) (string, error) {
		res, err := inst.GetArtistURL(ctx, &api.ArtistURLRequest{Id: id, Name: name, Mbid: mbid})
		if err != nil {
			return "", err
		}
		return res.GetUrl(), nil
	})
}

func (w *wasmArtistAgent) GetArtistBiography(ctx context.Context, id, name, mbid string) (string, error) {
	return callMethod(ctx, w, "GetArtistBiography", func(inst api.ArtistMetadataService) (string, error) {
		res, err := inst.GetArtistBiography(ctx, &api.ArtistBiographyRequest{Id: id, Name: name, Mbid: mbid})
		if err != nil {
			return "", err
		}
		return res.GetBiography(), nil
	})
}

func (w *wasmArtistAgent) GetSimilarArtists(ctx context.Context, id, name, mbid string, limit int) ([]agents.Artist, error) {
	return callMethod(ctx, w, "GetSimilarArtists", func(inst api.ArtistMetadataService) ([]agents.Artist, error) {
		resp, err := inst.GetSimilarArtists(ctx, &api.ArtistSimilarRequest{Id: id, Name: name, Mbid: mbid, Limit: int32(limit)})
		if err != nil {
			return nil, err
		}
		artists := make([]agents.Artist, 0, len(resp.GetArtists()))
		for _, a := range resp.GetArtists() {
			artists = append(artists, agents.Artist{
				Name: a.GetName(),
				MBID: a.GetMbid(),
			})
		}
		return artists, nil
	})
}

func (w *wasmArtistAgent) GetArtistImages(ctx context.Context, id, name, mbid string) ([]agents.ExternalImage, error) {
	return callMethod(ctx, w, "GetArtistImages", func(inst api.ArtistMetadataService) ([]agents.ExternalImage, error) {
		resp, err := inst.GetArtistImages(ctx, &api.ArtistImageRequest{Id: id, Name: name, Mbid: mbid})
		if err != nil {
			return nil, err
		}
		images := make([]agents.ExternalImage, 0, len(resp.GetImages()))
		for _, img := range resp.GetImages() {
			images = append(images, agents.ExternalImage{
				URL:  img.GetUrl(),
				Size: int(img.GetSize()),
			})
		}
		return images, nil
	})
}

func (w *wasmArtistAgent) GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]agents.Song, error) {
	return callMethod(ctx, w, "GetArtistTopSongs", func(inst api.ArtistMetadataService) ([]agents.Song, error) {
		resp, err := inst.GetArtistTopSongs(ctx, &api.ArtistTopSongsRequest{Id: id, ArtistName: artistName, Mbid: mbid, Count: int32(count)})
		if err != nil {
			return nil, err
		}
		songs := make([]agents.Song, 0, len(resp.GetSongs()))
		for _, s := range resp.GetSongs() {
			songs = append(songs, agents.Song{
				Name: s.GetName(),
				MBID: s.GetMbid(),
			})
		}
		return songs, nil
	})
}
