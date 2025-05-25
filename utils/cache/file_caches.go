package cache

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/djherbis/fscache"
	"github.com/dustin/go-humanize"
	"github.com/hashicorp/go-multierror"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
)

// Item represents an item that can be cached. It must implement the Key method that returns a unique key for a
// given item.
type Item interface {
	Key() string
}

// ReadFunc is a function that retrieves the data to be cached. It receives the Item to be cached and returns
// an io.Reader with the data and an error.
type ReadFunc func(ctx context.Context, item Item) (io.Reader, error)

// FileCache is designed to cache data on the filesystem to improve performance by avoiding repeated data
// retrieval operations.
//
// Errors are handled gracefully. If the cache is not initialized or an error occurs during data
// retrieval, it will log the error and proceed without caching.
type FileCache interface {

	// Get retrieves data from the cache. This method checks if the data is already cached. If it is, it
	// returns the cached data. If not, it retrieves the data using the provided getReader function and caches it.
	//
	// Example Usage:
	//
	//	s, err := fc.Get(context.Background(), cacheKey("testKey"))
	//	if err != nil {
	//	    log.Fatal(err)
	//	}
	//	defer s.Close()
	//
	//	data, err := io.ReadAll(s)
	//	if err != nil {
	//	    log.Fatal(err)
	//	}
	//	fmt.Println(string(data))
	Get(ctx context.Context, item Item) (*CachedStream, error)

	// Available checks if the cache is available
	Available(ctx context.Context) bool

	// Disabled reports if the cache has been permanently disabled
	Disabled(ctx context.Context) bool
}

// NewFileCache creates a new FileCache. This function initializes the cache and starts it in the background.
//
//	name: A string representing the name of the cache.
//	cacheSize: A string representing the maximum size of the cache (e.g., "1KB", "10MB").
//	cacheFolder: A string representing the folder where the cache files will be stored.
//	maxItems: An integer representing the maximum number of items the cache can hold.
//	getReader: A function of type ReadFunc that retrieves the data to be cached.
//
// Example Usage:
//
//	fc := NewFileCache("exampleCache", "10MB", "cacheFolder", 100, func(ctx context.Context, item Item) (io.Reader, error) {
//	    // Implement the logic to retrieve the data for the given item
//	    return strings.NewReader(item.Key()), nil
//	})
func NewFileCache(name, cacheSize, cacheFolder string, maxItems int, getReader ReadFunc) FileCache {
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
		fc.cache = cache
		fc.disabled = cache == nil || err != nil
		log.Info("Finished initializing cache", "cache", fc.name, "maxSize", fc.cacheSize, "elapsedTime", time.Since(start))
		fc.ready.Store(true)
		if err != nil {
			log.Error(fmt.Sprintf("Cache %s will be DISABLED due to previous errors", "name"), fc.name, err)
		}
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
	ready       atomic.Bool
	mutex       *sync.RWMutex
}

func (fc *fileCache) Available(_ context.Context) bool {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()

	return fc.ready.Load() && !fc.disabled
}

func (fc *fileCache) Disabled(_ context.Context) bool {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()

	return fc.disabled
}

func (fc *fileCache) invalidate(ctx context.Context, key string) error {
	if !fc.Available(ctx) {
		log.Debug(ctx, "Cache not initialized yet. Cannot invalidate key", "cache", fc.name, "key", key)
		return nil
	}
	if !fc.cache.Exists(key) {
		return nil
	}
	err := fc.cache.Remove(key)
	if err != nil {
		log.Warn(ctx, "Error removing key from cache", "cache", fc.name, "key", key, err)
	}
	return err
}

func (fc *fileCache) Get(ctx context.Context, arg Item) (*CachedStream, error) {
	if !fc.Available(ctx) {
		log.Debug(ctx, "Cache not initialized yet. Reading data directly from reader", "cache", fc.name)
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
			_ = r.Close()
			_ = w.Close()
			_ = fc.invalidate(ctx, key)
			return nil, err
		}
		go func() {
			if err := copyAndClose(w, reader); err != nil {
				log.Debug(ctx, "Error storing file in cache", "cache", fc.name, "key", key, err)
				_ = fc.invalidate(ctx, key)
			} else {
				log.Trace(ctx, "File successfully stored in cache", "cache", fc.name, "key", key)
			}
		}()
	}

	// If it is in the cache, check if the stream is done being written. If so, return a ReadSeeker
	if cached {
		size := getFinalCachedSize(r)
		if size >= 0 {
			log.Trace(ctx, "Cache HIT", "cache", fc.name, "key", key, "size", size)
			sr := io.NewSectionReader(r, 0, size)
			return &CachedStream{
				Reader: sr,
				Seeker: sr,
				Closer: r,
				Cached: true,
			}, nil
		} else {
			log.Trace(ctx, "Cache HIT", "cache", fc.name, "key", key)
		}
	}

	// All other cases, just return the cache reader, without Seek capabilities
	return &CachedStream{Reader: r, Cached: cached}, nil
}

// CachedStream is a wrapper around an io.ReadCloser that allows reading from a cache.
type CachedStream struct {
	io.Reader
	io.Seeker
	io.Closer
	Cached bool
}

func (s *CachedStream) Close() error {
	if s.Closer != nil {
		return s.Closer.Close()
	}
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

func copyAndClose(w io.WriteCloser, r io.Reader) error {
	_, err := io.Copy(w, r)
	if err != nil {
		err = fmt.Errorf("copying data to cache: %w", err)
	}
	if c, ok := r.(io.Closer); ok {
		if cErr := c.Close(); cErr != nil {
			err = multierror.Append(err, fmt.Errorf("closing source stream: %w", cErr))
		}
	}

	if cErr := w.Close(); cErr != nil {
		err = multierror.Append(err, fmt.Errorf("closing cache writer: %w", cErr))
	}
	return err
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

	lru := NewFileHaunter(name, maxItems, size, consts.DefaultCacheCleanUpInterval)
	h := fscache.NewLRUHaunterStrategy(lru)
	cacheFolder = filepath.Join(conf.Server.CacheFolder, cacheFolder)

	var fs *spreadFS
	log.Info(fmt.Sprintf("Creating %s cache", name), "path", cacheFolder, "maxSize", humanize.Bytes(size))
	fs, err = NewSpreadFS(cacheFolder, 0755)
	if err != nil {
		log.Error(fmt.Sprintf("Error initializing %s cache FS", name), err)
		return nil, err
	}

	ck, err := fscache.NewCacheWithHaunter(fs, h)
	if err != nil {
		log.Error(fmt.Sprintf("Error initializing %s cache", name), err)
		return nil, err
	}
	ck.SetKeyMapper(fs.KeyMapper)

	return ck, nil
}
