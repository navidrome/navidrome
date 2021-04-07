package cache

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"time"

	"github.com/djherbis/fscache"
	"github.com/dustin/go-humanize"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
)

type Item interface {
	Key() string
}

type ReadFunc func(ctx context.Context, item Item) (io.Reader, error)

type FileCache interface {
	Get(ctx context.Context, item Item) (*CachedStream, error)
	Ready(ctx context.Context) bool
	Available(ctx context.Context) bool
}

func NewFileCache(name, cacheSize, cacheFolder string, maxItems int, getReader ReadFunc) *fileCache {
	fc := &fileCache{
		name:        name,
		cacheSize:   cacheSize,
		cacheFolder: filepath.FromSlash(cacheFolder),
		maxItems:    maxItems,
		getReader:   getReader,
		mutex:       &sync.RWMutex{},
	}

	go func() {
		start := time.Now()
		cache, err := newFSCache(fc.name, fc.cacheSize, fc.cacheFolder, fc.maxItems)
		fc.mutex.Lock()
		defer fc.mutex.Unlock()
		if err == nil {
			fc.cache = cache
			fc.disabled = cache == nil
		}
		log.Info("Finished initializing cache", "cache", fc.name, "maxSize", fc.cacheSize, "elapsedTime", time.Since(start))
		fc.ready = true
		if fc.disabled {
			log.Debug("Cache DISABLED", "cache", fc.name, "size", fc.cacheSize)
		}
	}()

	return fc
}

type fileCache struct {
	name        string
	cacheSize   string
	cacheFolder string
	maxItems    int
	cache       fscache.Cache
	getReader   ReadFunc
	disabled    bool
	ready       bool
	mutex       *sync.RWMutex
}

func (fc *fileCache) Ready(ctx context.Context) bool {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()
	return fc.ready
}

func (fc *fileCache) Available(ctx context.Context) bool {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()

	if !fc.ready {
		log.Debug(ctx, "Cache not initialized yet", "cache", fc.name)
	}

	return fc.ready && !fc.disabled
}

func (fc *fileCache) Get(ctx context.Context, arg Item) (*CachedStream, error) {
	if !fc.Available(ctx) {
		reader, err := fc.getReader(ctx, arg)
		if err != nil {
			return nil, err
		}
		return &CachedStream{Reader: reader}, nil
	}

	key := arg.Key()
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
				Cached: true,
			}, nil
		} else {
			log.Trace(ctx, "Cache HIT", "cache", fc.name, "key", key)
		}
	}

	// All other cases, just return a Reader, without Seek capabilities
	return &CachedStream{Reader: r, Cached: cached}, nil
}

type CachedStream struct {
	io.Reader
	io.Seeker
	Cached bool
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
		log.Error("Invalid cache size. Using default size", "cache", name, "size", cacheSize,
			"defaultSize", humanize.Bytes(consts.DefaultCacheSize))
		size = consts.DefaultCacheSize
	}
	if size == 0 {
		log.Warn(fmt.Sprintf("%s cache disabled", name))
		return nil, nil
	}

	lru := fscache.NewLRUHaunter(maxItems, int64(size), consts.DefaultCacheCleanUpInterval)
	h := fscache.NewLRUHaunterStrategy(lru)
	cacheFolder = filepath.Join(conf.Server.DataFolder, cacheFolder)

	var fs fscache.FileSystem
	log.Info(fmt.Sprintf("Creating %s cache", name), "path", cacheFolder, "maxSize", humanize.Bytes(size))
	if conf.Server.DevOldCacheLayout {
		fs, err = fscache.NewFs(cacheFolder, 0755)
	} else {
		fs, err = NewSpreadFS(cacheFolder, 0755)
	}
	if err != nil {
		log.Error(fmt.Sprintf("Error initializing %s cache", name), err)
		return nil, err
	}

	ck, err := fscache.NewCacheWithHaunter(fs, h)
	if err != nil {
		return nil, err
	}

	if !conf.Server.DevOldCacheLayout {
		ck.SetKeyMapper(fs.(*spreadFS).KeyMapper)
	}

	return ck, nil
}
