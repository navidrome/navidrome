package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
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
		inflight:    map[string]*atomic.Pointer[error]{},
	}

	go func() {
		start := time.Now()
		cache, sfs, err := newFSCache(fc.name, fc.cacheSize, fc.cacheFolder, fc.maxItems)
		fc.mutex.Lock()
		defer fc.mutex.Unlock()
		fc.cache = cache
		fc.fs = sfs
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
	fs          *spreadFS
	getReader   ReadFunc
	disabled    bool
	ready       atomic.Bool
	mutex       *sync.RWMutex
	// Write outcome of each entry still being filled, so every reader of it learns a failure.
	inflightMu sync.Mutex
	inflight   map[string]*atomic.Pointer[error]
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

// reserve opens the cache entry and binds its write outcome in one step, so a reader
// cannot pick up the outcome of a newer entry that reused the same key.
func (fc *fileCache) reserve(ctx context.Context, key string) (fscache.ReadAtCloser, io.WriteCloser, *atomic.Pointer[error], error) {
	fc.inflightMu.Lock()
	defer fc.inflightMu.Unlock()

	r, w, err := fc.cache.Get(key)
	if errors.Is(err, fs.ErrNotExist) {
		// The entry outlived its data file. Drop it and retry, or every future Get
		// for this key fails for the rest of the process's life.
		log.Debug(ctx, "Cache entry lost its data file. Re-fetching", "cache", fc.name, "key", key)
		// invalidate waits on open handles, so it must never run under the lock.
		fc.inflightMu.Unlock()
		_ = fc.invalidate(ctx, key)
		fc.inflightMu.Lock()
		r, w, err = fc.cache.Get(key)
	}
	if err != nil {
		return nil, nil, nil, err
	}
	if w != nil {
		outcome := &atomic.Pointer[error]{}
		fc.inflight[key] = outcome
		return r, w, outcome, nil
	}
	return r, w, fc.inflight[key], nil
}

// forget drops the outcome only if it is still ours: a later entry may have replaced it.
func (fc *fileCache) forget(key string, outcome *atomic.Pointer[error]) {
	fc.inflightMu.Lock()
	defer fc.inflightMu.Unlock()
	if fc.inflight[key] == outcome {
		delete(fc.inflight, key)
	}
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
	r, w, writeErr, err := fc.reserve(ctx, key)
	if err != nil {
		return nil, err
	}

	cached := w == nil

	if !cached {
		log.Trace(ctx, "Cache MISS", "cache", fc.name, "key", key)
		reader, err := fc.getReader(ctx, arg)
		if err != nil {
			writeErr.Store(&err)
			_ = r.Close()
			_ = w.Close()
			_ = fc.invalidate(ctx, key)
			fc.forget(key, writeErr)
			return nil, err
		}
		go func() {
			defer fc.forget(key, writeErr)
			if err := fc.copyAndClose(ctx, key, w, reader, writeErr); err != nil {
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
		// Read the outcome after the size: a final entry means the writer already closed,
		// and it stores any failure before closing.
		if writeErr != nil {
			if err := writeErr.Load(); err != nil {
				_ = r.Close()
				return nil, *err
			}
		}
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
	return &CachedStream{Reader: r, Cached: cached, writeErr: writeErr}, nil
}

// CachedStream is a wrapper around an io.ReadCloser that allows reading from a cache.
type CachedStream struct {
	io.Reader
	io.Seeker
	io.Closer
	Cached   bool
	writeErr *atomic.Pointer[error]
}

// Err reports a failure of the goroutine filling the entry. Only meaningful after EOF,
// since a truncated entry ends in a clean EOF.
func (s *CachedStream) Err() error {
	if s.writeErr == nil {
		return nil
	}
	if err := s.writeErr.Load(); err != nil {
		return *err
	}
	return nil
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

// copyAndClose marks the entry complete before closing w, so EOF implies the entry is settled on disk.
func (fc *fileCache) copyAndClose(ctx context.Context, key string, w io.WriteCloser, r io.Reader, writeErr *atomic.Pointer[error]) error {
	_, err := io.Copy(w, r)
	if err != nil {
		err = fmt.Errorf("copying data to cache: %w", err)
	}
	if c, ok := r.(io.Closer); ok {
		if cErr := c.Close(); cErr != nil {
			err = multierror.Append(err, fmt.Errorf("closing source stream: %w", cErr))
		}
	}
	if err == nil {
		fc.markComplete(ctx, key)
	} else {
		// Store before Close: closing w is what releases readers waiting on EOF.
		writeErr.Store(&err)
	}
	if cErr := w.Close(); cErr != nil {
		err = multierror.Append(err, fmt.Errorf("closing cache writer: %w", cErr))
	}
	return err
}

// markComplete records on disk that the entry for key was written in full,
// so it is eligible for adoption after a restart (see spreadFS.Reload).
func (fc *fileCache) markComplete(ctx context.Context, key string) {
	if fc.fs == nil {
		return
	}
	if err := fc.fs.MarkComplete(key); err != nil {
		log.Warn(ctx, "Error writing cache completion marker", "cache", fc.name, "key", key, err)
	}
}

func newFSCache(name, cacheSize, cacheFolder string, maxItems int) (fscache.Cache, *spreadFS, error) {
	size, err := humanize.ParseBytes(cacheSize)
	if err != nil {
		log.Error("Invalid cache size. Using default size", "cache", name, "size", cacheSize,
			"defaultSize", humanize.Bytes(consts.DefaultCacheSize))
		size = consts.DefaultCacheSize
	}
	if size == 0 {
		log.Warn(fmt.Sprintf("%s cache disabled", name))
		return nil, nil, nil
	}

	lru := NewFileHaunter(name, maxItems, size, consts.DefaultCacheCleanUpInterval)
	h := fscache.NewLRUHaunterStrategy(lru)
	cacheFolder = filepath.Join(conf.Server.CacheFolder.MustPath(), cacheFolder)

	var fs *spreadFS
	log.Info(fmt.Sprintf("Creating %s cache", name), "path", cacheFolder, "maxSize", humanize.Bytes(size))
	fs, err = NewSpreadFS(cacheFolder, 0755)
	if err != nil {
		log.Error(fmt.Sprintf("Error initializing %s cache FS", name), err)
		return nil, nil, err
	}

	ck, err := fscache.NewCacheWithHaunter(fs, h)
	if err != nil {
		log.Error(fmt.Sprintf("Error initializing %s cache", name), err)
		return nil, nil, err
	}
	ck.SetKeyMapper(fs.KeyMapper)

	return ck, fs, nil
}
