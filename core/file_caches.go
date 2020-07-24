package core

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/log"
	"github.com/djherbis/fscache"
	"github.com/dustin/go-humanize"
)

type ReadFunc func(ctx context.Context, arg fmt.Stringer) (io.Reader, error)

func NewFileCache(name, cacheSize, cacheFolder string, maxItems int, getReader ReadFunc) (*FileCache, error) {
	cache, err := newFSCache(name, cacheSize, cacheFolder, maxItems)
	if err != nil {
		return nil, err
	}
	return &FileCache{
		name:      name,
		disabled:  cache == nil,
		cache:     cache,
		getReader: getReader,
	}, nil
}

type FileCache struct {
	disabled  bool
	name      string
	cache     fscache.Cache
	getReader ReadFunc
}

func (fc *FileCache) Get(ctx context.Context, arg fmt.Stringer) (*CachedStream, error) {
	if fc.disabled {
		log.Debug(ctx, "Cache disabled", "cache", fc.name)
		reader, err := fc.getReader(ctx, arg)
		if err != nil {
			return nil, err
		}
		return &CachedStream{Reader: reader}, nil
	}

	key := arg.String()
	r, w, err := fc.cache.Get(key)
	if err != nil {
		return nil, err
	}

	cached := w == nil

	if !cached {
		log.Trace(ctx, "Cache MISS", "cache", fc.name, "key", key)
		reader, err := fc.getReader(ctx, arg)
		if err != nil {
			return nil, err
		}
		go copyAndClose(ctx, w, reader)
	}

	// If it is in the cache, check if the stream is done being written. If so, return a ReaderSeeker
	if cached {
		size := getFinalCachedSize(r)
		if size >= 0 {
			log.Trace(ctx, "Cache HIT", "cache", fc.name, "key", key, "size", size)
			sr := io.NewSectionReader(r, 0, size)
			return &CachedStream{
				Reader: sr,
				Seeker: sr,
			}, nil
		} else {
			log.Trace(ctx, "Cache HIT", "cache", fc.name, "key", key)
		}
	}

	// All other cases, just return a Reader, without Seek capabilities
	return &CachedStream{Reader: r}, nil
}

type CachedStream struct {
	io.Reader
	io.Seeker
}

func (s *CachedStream) Seekable() bool { return s.Seeker != nil }
func (s *CachedStream) Close() error {
	if c, ok := s.Reader.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func getFinalCachedSize(r fscache.ReadAtCloser) int64 {
	cr, ok := r.(*fscache.CacheReader)
	if ok {
		size, final, err := cr.Size()
		if final && err == nil {
			return size
		}
	}
	return -1
}

func copyAndClose(ctx context.Context, w io.WriteCloser, r io.Reader) {
	_, err := io.Copy(w, r)
	if err != nil {
		log.Error(ctx, "Error copying data to cache", err)
	}
	if c, ok := r.(io.Closer); ok {
		err = c.Close()
		if err != nil {
			log.Error(ctx, "Error closing source stream", err)
		}
	}
	err = w.Close()
	if err != nil {
		log.Error(ctx, "Error closing cache writer", err)
	}
}

func newFSCache(name, cacheSize, cacheFolder string, maxItems int) (fscache.Cache, error) {
	size, err := humanize.ParseBytes(cacheSize)
	if err != nil {
		log.Error("Invalid cache size. Using default size", "cache", name, "size", cacheSize, "defaultSize", consts.DefaultCacheSize)
		size = consts.DefaultCacheSize
	}
	if size == 0 {
		log.Warn(fmt.Sprintf("%s cache disabled", name))
		return nil, nil
	}
	lru := fscache.NewLRUHaunter(maxItems, int64(size), consts.DefaultCacheCleanUpInterval)
	h := fscache.NewLRUHaunterStrategy(lru)
	cacheFolder = filepath.Join(conf.Server.DataFolder, cacheFolder)
	log.Info(fmt.Sprintf("Creating %s cache", name), "path", cacheFolder, "maxSize", humanize.Bytes(size))
	fs, err := fscache.NewFs(cacheFolder, 0755)
	if err != nil {
		return nil, err
	}
	return fscache.NewCacheWithHaunter(fs, h)
}
