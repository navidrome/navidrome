package plugins

import (
	"context"
	"fmt"
	"sync"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/api"
)

type wasmAgent struct {
	pool     *sync.Pool
	wasmPath string
	name     string
}

func (w *wasmAgent) AgentName() string {
	return w.name
}

func (w *wasmAgent) getInstance(ctx context.Context) (api.ArtistMetadataService, func(error), error) {
	var inst api.ArtistMetadataService
	var closer func(context.Context) error
	v := w.pool.Get()
	if v == nil {
		log.Error(ctx, "wasmAgent: sync.Pool returned nil instance", "plugin", w.name, "path", w.wasmPath)
		return nil, nil, fmt.Errorf("wasmAgent: sync.Pool returned nil instance for plugin %s", w.name)
	}
	inst = v.(api.ArtistMetadataService)
	closer = v.(interface{ Close(context.Context) error }).Close
	closeFn := func(e error) {
		if e == nil {
			w.pool.Put(v)
		} else {
			_ = closer(ctx)
		}
	}
	return inst, closeFn, nil
}

func (w *wasmAgent) GetArtistMBID(ctx context.Context, id string, name string) (string, error) {
	inst, done, err := w.getInstance(ctx)
	if err != nil {
		return "", err
	}
	var resp *api.ArtistMBIDResponse
	var callErr error
	defer func() { done(callErr) }()
	resp, callErr = inst.GetArtistMBID(ctx, &api.ArtistMBIDRequest{Id: id, Name: name})
	if callErr != nil {
		return "", callErr
	}
	return resp.GetMbid(), nil
}

func (w *wasmAgent) GetArtistURL(ctx context.Context, id, name, mbid string) (string, error) {
	inst, done, err := w.getInstance(ctx)
	if err != nil {
		return "", err
	}
	var resp *api.ArtistURLResponse
	var callErr error
	defer func() { done(callErr) }()
	resp, callErr = inst.GetArtistURL(ctx, &api.ArtistURLRequest{Id: id, Name: name, Mbid: mbid})
	if callErr != nil {
		return "", callErr
	}
	return resp.GetUrl(), nil
}

func (w *wasmAgent) GetArtistBiography(ctx context.Context, id, name, mbid string) (string, error) {
	inst, done, err := w.getInstance(ctx)
	if err != nil {
		return "", err
	}
	var resp *api.ArtistBiographyResponse
	var callErr error
	defer func() { done(callErr) }()
	resp, callErr = inst.GetArtistBiography(ctx, &api.ArtistBiographyRequest{Id: id, Name: name, Mbid: mbid})
	if callErr != nil {
		return "", callErr
	}
	return resp.GetBiography(), nil
}

func (w *wasmAgent) GetSimilarArtists(ctx context.Context, id, name, mbid string, limit int) ([]agents.Artist, error) {
	inst, done, err := w.getInstance(ctx)
	if err != nil {
		return nil, err
	}
	var resp *api.ArtistSimilarResponse
	var callErr error
	defer func() { done(callErr) }()
	resp, callErr = inst.GetSimilarArtists(ctx, &api.ArtistSimilarRequest{Id: id, Name: name, Mbid: mbid, Limit: int32(limit)})
	if callErr != nil {
		return nil, callErr
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
	inst, done, err := w.getInstance(ctx)
	if err != nil {
		return nil, err
	}
	var resp *api.ArtistImageResponse
	var callErr error
	defer func() { done(callErr) }()
	resp, callErr = inst.GetArtistImages(ctx, &api.ArtistImageRequest{Id: id, Name: name, Mbid: mbid})
	if callErr != nil {
		return nil, callErr
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
	inst, done, err := w.getInstance(ctx)
	if err != nil {
		return nil, err
	}
	var resp *api.ArtistTopSongsResponse
	var callErr error
	defer func() { done(callErr) }()
	resp, callErr = inst.GetArtistTopSongs(ctx, &api.ArtistTopSongsRequest{Id: id, ArtistName: artistName, Mbid: mbid, Count: int32(count)})
	if callErr != nil {
		return nil, callErr
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
	// Drain and close all instances in the pool
	for {
		v := w.pool.Get()
		if v == nil {
			break
		}
		if closer, ok := v.(interface{ Close(context.Context) error }); ok {
			_ = closer.Close(ctx)
		}
	}
	return nil
}
