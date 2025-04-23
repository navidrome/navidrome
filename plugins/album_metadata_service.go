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

type wasmAlbumAgent struct {
	pool     *sync.Pool
	wasmPath string
	name     string
}

func (w *wasmAlbumAgent) AgentName() string {
	return w.name
}

func (w *wasmAlbumAgent) getValidPooledInstance(ctx context.Context, methodName string) (*pooledInstance, error) {
	v := w.pool.Get()
	if v == nil {
		log.Error(ctx, "wasmAlbumAgent: sync.Pool returned nil instance", "plugin", w.name, "path", w.wasmPath)
		return nil, fmt.Errorf("wasmAlbumAgent: sync.Pool returned nil instance for plugin %s", w.name)
	}
	pooled, ok := v.(*pooledInstance)
	if !ok || pooled == nil || pooled.instance == nil {
		log.Error(ctx, "wasmAlbumAgent: pool returned invalid type or nil instance", "plugin", w.name, "path", w.wasmPath, "type", fmt.Sprintf("%T", v))
		if pooled != nil {
			pooled.cleanup.Stop()
		}
		if closer, canClose := v.(interface{ Close(context.Context) error }); canClose {
			_ = closer.Close(ctx)
		}
		return nil, fmt.Errorf("wasmAlbumAgent: pool returned invalid instance for plugin %s", w.name)
	}
	return pooled, nil
}

func (w *wasmAlbumAgent) createPoolCleanupFunc(ctx context.Context, pooled *pooledInstance, closer func(context.Context) error, start time.Time, methodName string) func(error) {
	return func(err error) {
		if err == nil || err.Error() == api.ErrNotFound.Error() || err.Error() == api.ErrNotImplemented.Error() {
			w.pool.Put(pooled)
			log.Trace(ctx, "wasmAlbumAgent: returned instance to pool", "plugin", w.name, "method", methodName, "elapsed", time.Since(start), err)
		} else {
			pooled.cleanup.Stop()
			log.Trace(ctx, "wasmAlbumAgent: stopped GC cleanup", "plugin", w.name, "method", methodName)
			if closer != nil {
				_ = closer(ctx)
				log.Trace(ctx, "wasmAlbumAgent: closed instance due to error", "plugin", w.name, "method", methodName, "elapsed", time.Since(start), err)
			} else {
				log.Error(ctx, "wasmAlbumAgent: attempted to close instance due to error, but closer was nil", "plugin", w.name, "method", methodName, "elapsed", time.Since(start), err)
			}
		}
	}
}

func (w *wasmAlbumAgent) getInstance(ctx context.Context, methodName string) (api.AlbumMetadataService, func(error), error) {
	pooled, err := w.getValidPooledInstance(ctx, methodName)
	if err != nil {
		return nil, nil, err
	}
	log.Trace(ctx, "wasmAlbumAgent: got instance from pool", "plugin", w.name, "method", methodName)
	inst := pooled.instance.(api.AlbumMetadataService)
	start := time.Now()
	closerInst := pooled.instance.(interface{ Close(context.Context) error })
	closeFn := w.createPoolCleanupFunc(ctx, pooled, closerInst.Close, start, methodName)
	return inst, closeFn, nil
}

func callAlbumMethod[R any](ctx context.Context, w *wasmAlbumAgent, methodName string, fn func(inst api.AlbumMetadataService) (R, error)) (R, error) {
	inst, done, err := w.getInstance(ctx, methodName)
	var r R
	if err != nil {
		return r, err
	}
	defer func() { done(err) }()
	r, err = fn(inst)
	return r, w.error(err)
}

func (w *wasmAlbumAgent) error(err error) error {
	if err != nil && (err.Error() == api.ErrNotFound.Error() || err.Error() == api.ErrNotImplemented.Error()) {
		return agents.ErrNotFound
	}
	return err
}

// AlbumMetadataService methods
func (w *wasmAlbumAgent) GetAlbumInfo(ctx context.Context, name, artist, mbid string) (*agents.AlbumInfo, error) {
	return callAlbumMethod(ctx, w, "GetAlbumInfo", func(inst api.AlbumMetadataService) (*agents.AlbumInfo, error) {
		res, err := inst.GetAlbumInfo(ctx, &api.AlbumInfoRequest{Name: name, Artist: artist, Mbid: mbid})
		if err != nil {
			return nil, err
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
			Images:      nil, // TODO: Break agents.AlbumInfo into two, to match proto (no images here)
		}, nil
	})
}

func (w *wasmAlbumAgent) GetAlbumImages(ctx context.Context, name, artist, mbid string) ([]agents.ExternalImage, error) {
	return callAlbumMethod(ctx, w, "GetAlbumImages", func(inst api.AlbumMetadataService) ([]agents.ExternalImage, error) {
		res, err := inst.GetAlbumImages(ctx, &api.AlbumImagesRequest{Name: name, Artist: artist, Mbid: mbid})
		if err != nil {
			return nil, err
		}
		return convertExternalImages(res.Images), nil
	})
}

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

func (w *wasmAlbumAgent) Close(ctx context.Context) error {
	for {
		v := w.pool.Get()
		if v == nil {
			break
		}
		pooled, ok := v.(*pooledInstance)
		if !ok || pooled == nil || pooled.instance == nil {
			log.Warn(ctx, "wasmAlbumAgent: found invalid type or nil instance in pool during agent close", "plugin", w.name, "path", w.wasmPath, "type", fmt.Sprintf("%T", v))
			if pooled != nil {
				pooled.cleanup.Stop()
			}
			if closer, canClose := v.(interface{ Close(context.Context) error }); canClose {
				_ = closer.Close(ctx)
			}
			continue
		}
		pooled.cleanup.Stop()
		log.Trace(ctx, "wasmAlbumAgent: stopped GC cleanup during agent close", "plugin", w.name)
		if closer, ok := pooled.instance.(interface{ Close(context.Context) error }); ok {
			_ = closer.Close(ctx)
			log.Trace(ctx, "wasmAlbumAgent: closed instance during agent close", "plugin", w.name, "path", w.wasmPath)
		} else {
			log.Warn(ctx, "wasmAlbumAgent: instance in pool during agent close does not implement Close", "plugin", w.name, "path", w.wasmPath)
		}
	}
	log.Trace(ctx, "wasmAlbumAgent: agent closed", "plugin", w.name, "path", w.wasmPath)
	return nil
}
