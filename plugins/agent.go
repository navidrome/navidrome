package plugins

import (
	"context"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/plugins/api"
)

type wasmAgent struct {
	inst interface {
		api.ArtistMetadataService
		Close(context.Context) error
	}
	name string
}

func (w *wasmAgent) AgentName() string {
	return w.name
}

func (w *wasmAgent) GetArtistMBID(ctx context.Context, id string, name string) (string, error) {
	resp, err := w.inst.GetArtistMBID(ctx, &api.ArtistMBIDRequest{Id: id, Name: name})
	if err != nil {
		return "", err
	}
	return resp.GetMbid(), nil
}

func (w *wasmAgent) GetArtistURL(ctx context.Context, id, name, mbid string) (string, error) {
	resp, err := w.inst.GetArtistURL(ctx, &api.ArtistURLRequest{Id: id, Name: name, Mbid: mbid})
	if err != nil {
		return "", err
	}
	return resp.GetUrl(), nil
}

func (w *wasmAgent) GetArtistBiography(ctx context.Context, id, name, mbid string) (string, error) {
	resp, err := w.inst.GetArtistBiography(ctx, &api.ArtistBiographyRequest{Id: id, Name: name, Mbid: mbid})
	if err != nil {
		return "", err
	}
	return resp.GetBiography(), nil
}

func (w *wasmAgent) GetSimilarArtists(ctx context.Context, id, name, mbid string, limit int) ([]agents.Artist, error) {
	resp, err := w.inst.GetSimilarArtists(ctx, &api.ArtistSimilarRequest{Id: id, Name: name, Mbid: mbid, Limit: int32(limit)})
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
}

func (w *wasmAgent) GetArtistImages(ctx context.Context, id, name, mbid string) ([]agents.ExternalImage, error) {
	resp, err := w.inst.GetArtistImages(ctx, &api.ArtistImageRequest{Id: id, Name: name, Mbid: mbid})
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
}

func (w *wasmAgent) GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]agents.Song, error) {
	resp, err := w.inst.GetArtistTopSongs(ctx, &api.ArtistTopSongsRequest{Id: id, ArtistName: artistName, Mbid: mbid, Count: int32(count)})
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
}

func (w *wasmAgent) Close(ctx context.Context) error {
	return w.inst.Close(ctx)
}
