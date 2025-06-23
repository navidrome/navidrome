package plugins

import (
	"context"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/tetratelabs/wazero"
)

// NewWasmMediaAgent creates a new adapter for a MetadataAgent plugin
func newWasmMediaAgent(wasmPath, pluginID string, runtime api.WazeroNewRuntime, mc wazero.ModuleConfig) WasmPlugin {
	loader, err := api.NewMetadataAgentPlugin(context.Background(), api.WazeroRuntime(runtime), api.WazeroModuleConfig(mc))
	if err != nil {
		log.Error("Error creating media metadata service plugin", "plugin", pluginID, "path", wasmPath, err)
		return nil
	}
	return &wasmMediaAgent{
		wasmBasePlugin: &wasmBasePlugin[api.MetadataAgent, *api.MetadataAgentPlugin]{
			wasmPath:   wasmPath,
			id:         pluginID,
			capability: CapabilityMetadataAgent,
			loader:     loader,
			loadFunc: func(ctx context.Context, l *api.MetadataAgentPlugin, path string) (api.MetadataAgent, error) {
				return l.Load(ctx, path)
			},
		},
	}
}

// wasmMediaAgent adapts a MetadataAgent plugin to implement the agents.Interface
type wasmMediaAgent struct {
	*wasmBasePlugin[api.MetadataAgent, *api.MetadataAgentPlugin]
}

func (w *wasmMediaAgent) AgentName() string {
	return w.id
}

func (w *wasmMediaAgent) mapError(err error) error {
	if err != nil && (err.Error() == api.ErrNotFound.Error() || err.Error() == api.ErrNotImplemented.Error()) {
		return agents.ErrNotFound
	}
	return err
}

// Album-related methods

func (w *wasmMediaAgent) GetAlbumInfo(ctx context.Context, name, artist, mbid string) (*agents.AlbumInfo, error) {
	return callMethod(ctx, w, "GetAlbumInfo", func(inst api.MetadataAgent) (*agents.AlbumInfo, error) {
		res, err := inst.GetAlbumInfo(ctx, &api.AlbumInfoRequest{Name: name, Artist: artist, Mbid: mbid})
		if err != nil {
			return nil, w.mapError(err)
		}
		if res == nil || res.Info == nil {
			return nil, agents.ErrNotFound
		}
		info := res.Info
		return &agents.AlbumInfo{
			Name:        info.Name,
			MBID:        info.Mbid,
			Description: info.Description,
			URL:         info.Url,
		}, nil
	})
}

func (w *wasmMediaAgent) GetAlbumImages(ctx context.Context, name, artist, mbid string) ([]agents.ExternalImage, error) {
	return callMethod(ctx, w, "GetAlbumImages", func(inst api.MetadataAgent) ([]agents.ExternalImage, error) {
		res, err := inst.GetAlbumImages(ctx, &api.AlbumImagesRequest{Name: name, Artist: artist, Mbid: mbid})
		if err != nil {
			return nil, w.mapError(err)
		}
		return convertExternalImages(res.Images), nil
	})
}

// Artist-related methods

func (w *wasmMediaAgent) GetArtistMBID(ctx context.Context, id string, name string) (string, error) {
	return callMethod(ctx, w, "GetArtistMBID", func(inst api.MetadataAgent) (string, error) {
		res, err := inst.GetArtistMBID(ctx, &api.ArtistMBIDRequest{Id: id, Name: name})
		if err != nil {
			return "", w.mapError(err)
		}
		return res.GetMbid(), nil
	})
}

func (w *wasmMediaAgent) GetArtistURL(ctx context.Context, id, name, mbid string) (string, error) {
	return callMethod(ctx, w, "GetArtistURL", func(inst api.MetadataAgent) (string, error) {
		res, err := inst.GetArtistURL(ctx, &api.ArtistURLRequest{Id: id, Name: name, Mbid: mbid})
		if err != nil {
			return "", w.mapError(err)
		}
		return res.GetUrl(), nil
	})
}

func (w *wasmMediaAgent) GetArtistBiography(ctx context.Context, id, name, mbid string) (string, error) {
	return callMethod(ctx, w, "GetArtistBiography", func(inst api.MetadataAgent) (string, error) {
		res, err := inst.GetArtistBiography(ctx, &api.ArtistBiographyRequest{Id: id, Name: name, Mbid: mbid})
		if err != nil {
			return "", w.mapError(err)
		}
		return res.GetBiography(), nil
	})
}

func (w *wasmMediaAgent) GetSimilarArtists(ctx context.Context, id, name, mbid string, limit int) ([]agents.Artist, error) {
	return callMethod(ctx, w, "GetSimilarArtists", func(inst api.MetadataAgent) ([]agents.Artist, error) {
		resp, err := inst.GetSimilarArtists(ctx, &api.ArtistSimilarRequest{Id: id, Name: name, Mbid: mbid, Limit: int32(limit)})
		if err != nil {
			return nil, w.mapError(err)
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

func (w *wasmMediaAgent) GetArtistImages(ctx context.Context, id, name, mbid string) ([]agents.ExternalImage, error) {
	return callMethod(ctx, w, "GetArtistImages", func(inst api.MetadataAgent) ([]agents.ExternalImage, error) {
		res, err := inst.GetArtistImages(ctx, &api.ArtistImageRequest{Id: id, Name: name, Mbid: mbid})
		if err != nil {
			return nil, w.mapError(err)
		}
		return convertExternalImages(res.Images), nil
	})
}

func (w *wasmMediaAgent) GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]agents.Song, error) {
	return callMethod(ctx, w, "GetArtistTopSongs", func(inst api.MetadataAgent) ([]agents.Song, error) {
		resp, err := inst.GetArtistTopSongs(ctx, &api.ArtistTopSongsRequest{Id: id, ArtistName: artistName, Mbid: mbid, Count: int32(count)})
		if err != nil {
			return nil, w.mapError(err)
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

// Helper function to convert ExternalImage objects from the API to the agents package
func convertExternalImages(images []*api.ExternalImage) []agents.ExternalImage {
	result := make([]agents.ExternalImage, 0, len(images))
	for _, img := range images {
		result = append(result, agents.ExternalImage{
			URL:  img.GetUrl(),
			Size: int(img.GetSize()),
		})
	}
	return result
}
