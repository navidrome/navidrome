package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

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

// getInstance gets an instance from the pool, and returns a function to return it to the pool
func (w *wasmAgent) getInstance(ctx context.Context, methodName string) (api.ArtistMetadataService, func(error), error) {
	var inst api.ArtistMetadataService
	var closer func(context.Context) error
	var pooled *pooledInstance

	v := w.pool.Get()
	if v == nil {
		log.Error(ctx, "wasmAgent: sync.Pool returned nil instance", "plugin", w.name, "path", w.wasmPath)
		return nil, nil, fmt.Errorf("wasmAgent: sync.Pool returned nil instance for plugin %s", w.name)
	}

	// Type assert to the pooledInstance struct
	pooled, ok := v.(*pooledInstance)
	if !ok || pooled == nil || pooled.instance == nil {
		// This shouldn't happen if pool.New is correct, but handle defensively
		log.Error(ctx, "wasmAgent: pool returned invalid type or nil instance", "plugin", w.name, "path", w.wasmPath, "type", fmt.Sprintf("%T", v))
		// Attempt cleanup if possible
		if pooled != nil {
			pooled.cleanup.Stop()
		}
		if closer, canClose := v.(interface{ Close(context.Context) error }); canClose {
			_ = closer.Close(ctx)
		}
		return nil, nil, fmt.Errorf("wasmAgent: pool returned invalid instance for plugin %s", w.name)
	}

	log.Trace(ctx, "wasmAgent: got instance from pool", "plugin", w.name, "method", methodName)
	inst = pooled.instance.(api.ArtistMetadataService)
	start := time.Now()

	// Get the closer function if the instance supports it
	closerInst, canClose := pooled.instance.(interface{ Close(context.Context) error })
	if !canClose {
		// Should not happen if pool.New logic is correct and registers cleanup
		log.Error(ctx, "wasmAgent: pooled instance does not implement Close", "plugin", w.name, "path", w.wasmPath)
		// Return the instance, but the closeFn won't do much
		closeFn := func(err error) {
			if err == nil || err.Error() == api.ErrNotFound.Error() || err.Error() == api.ErrNotImplemented.Error() {
				w.pool.Put(pooled) // Put the wrapper back
				log.Trace(ctx, "wasmAgent: returned instance (no close) to pool", "plugin", w.name, "method", methodName, "elapsed", time.Since(start), err)
			} else {
				log.Trace(ctx, "wasmAgent: discarding instance (no close) due to error", "plugin", w.name, "method", methodName, "elapsed", time.Since(start), err)
			}
		}
		return inst, closeFn, nil
	}
	closer = closerInst.Close

	closeFn := func(err error) {
		if err == nil || err.Error() == api.ErrNotFound.Error() || err.Error() == api.ErrNotImplemented.Error() {
			// Return the wrapper to the pool. Do NOT stop the GC cleanup.
			w.pool.Put(pooled)
			log.Trace(ctx, "wasmAgent: returned instance to pool", "plugin", w.name, "method", methodName, "elapsed", time.Since(start), err)
		} else {
			// First, stop the GC cleanup to prevent double closing.
			// Calling Stop() on a zero Cleanup is a no-op.
			pooled.cleanup.Stop()
			log.Trace(ctx, "wasmAgent: stopped GC cleanup", "plugin", w.name, "method", methodName)

			_ = closer(ctx)
			log.Trace(ctx, "wasmAgent: closed instance due to error", "plugin", w.name, "method", methodName, "elapsed", time.Since(start), err)
		}
	}
	return inst, closeFn, nil
}

// callMethod calls the given method on the wasm instance, and returns the result
func callMethod[R any](ctx context.Context, w *wasmAgent, methodName string, fn func(inst api.ArtistMetadataService) (R, error)) (R, error) {
	inst, done, err := w.getInstance(ctx, methodName)
	var r R
	if err != nil {
		return r, err
	}
	defer func() { done(err) }()
	r, err = fn(inst)
	return r, w.error(err)
}

// error maps the plugin errors to the agent errors
// It uses the error message to match the error, since the error is serialized and deserialized and cannot be compared
// using errors.Is
func (w *wasmAgent) error(err error) error {
	if err != nil && (err.Error() == api.ErrNotFound.Error() || err.Error() == api.ErrNotImplemented.Error()) {
		return agents.ErrNotFound
	}
	return err
}

func (w *wasmAgent) GetArtistMBID(ctx context.Context, id string, name string) (string, error) {
	return callMethod(ctx, w, "GetArtistMBID", func(inst api.ArtistMetadataService) (string, error) {
		res, err := inst.GetArtistMBID(ctx, &api.ArtistMBIDRequest{Id: id, Name: name})
		if err != nil {
			return "", err
		}
		return res.GetMbid(), nil
	})
}

func (w *wasmAgent) GetArtistURL(ctx context.Context, id, name, mbid string) (string, error) {
	return callMethod(ctx, w, "GetArtistURL", func(inst api.ArtistMetadataService) (string, error) {
		res, err := inst.GetArtistURL(ctx, &api.ArtistURLRequest{Id: id, Name: name, Mbid: mbid})
		if err != nil {
			return "", err
		}
		return res.GetUrl(), nil
	})
}

func (w *wasmAgent) GetArtistBiography(ctx context.Context, id, name, mbid string) (string, error) {
	return callMethod(ctx, w, "GetArtistBiography", func(inst api.ArtistMetadataService) (string, error) {
		res, err := inst.GetArtistBiography(ctx, &api.ArtistBiographyRequest{Id: id, Name: name, Mbid: mbid})
		if err != nil {
			return "", err
		}
		return res.GetBiography(), nil
	})
}

func (w *wasmAgent) GetSimilarArtists(ctx context.Context, id, name, mbid string, limit int) ([]agents.Artist, error) {
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

func (w *wasmAgent) GetArtistImages(ctx context.Context, id, name, mbid string) ([]agents.ExternalImage, error) {
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

func (w *wasmAgent) GetArtistTopSongs(ctx context.Context, id, artistName, mbid string, count int) ([]agents.Song, error) {
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

func (w *wasmAgent) Close(ctx context.Context) error {
	// Drain and close all instances in the pool
	for {
		v := w.pool.Get()
		if v == nil {
			break // Pool is empty
		}

		pooled, ok := v.(*pooledInstance)
		if !ok || pooled == nil || pooled.instance == nil {
			log.Warn(ctx, "wasmAgent: found invalid type or nil instance in pool during agent close", "plugin", w.name, "path", w.wasmPath, "type", fmt.Sprintf("%T", v))
			if pooled != nil {
				pooled.cleanup.Stop()
			}
			if closer, canClose := v.(interface{ Close(context.Context) error }); canClose {
				_ = closer.Close(ctx)
			}
			continue
		}

		// Calling Stop() on a zero Cleanup is a no-op.
		pooled.cleanup.Stop()
		log.Trace(ctx, "wasmAgent: stopped GC cleanup during agent close", "plugin", w.name)

		if closer, ok := pooled.instance.(interface{ Close(context.Context) error }); ok {
			_ = closer.Close(ctx)
			log.Trace(ctx, "wasmAgent: closed instance during agent close", "plugin", w.name, "path", w.wasmPath)
		} else {
			log.Warn(ctx, "wasmAgent: instance in pool during agent close does not implement Close", "plugin", w.name, "path", w.wasmPath)
		}
	}
	log.Trace(ctx, "wasmAgent: agent closed", "plugin", w.name, "path", w.wasmPath)
	return nil
}
