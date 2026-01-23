package plugins

import (
	"context"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/host"
)

const (
	defaultCacheTTL = 24 * time.Hour
)

// cacheServiceImpl implements the host.CacheService interface.
// Each plugin gets its own cache instance for isolation.
type cacheServiceImpl struct {
	pluginName string
	cache      *ttlcache.Cache[string, any]
	defaultTTL time.Duration
}

// newCacheService creates a new cacheServiceImpl instance with its own cache.
func newCacheService(pluginName string) *cacheServiceImpl {
	cache := ttlcache.New[string, any](
		ttlcache.WithTTL[string, any](defaultCacheTTL),
	)
	// Start the janitor goroutine to clean up expired entries
	go cache.Start()

	return &cacheServiceImpl{
		pluginName: pluginName,
		cache:      cache,
		defaultTTL: defaultCacheTTL,
	}
}

// getTTL converts seconds to a duration, using default if 0 or negative
func (s *cacheServiceImpl) getTTL(seconds int64) time.Duration {
	if seconds <= 0 {
		return s.defaultTTL
	}
	return time.Duration(seconds) * time.Second
}

// SetString stores a string value in the cache.
func (s *cacheServiceImpl) SetString(ctx context.Context, key string, value string, ttlSeconds int64) error {
	s.cache.Set(key, value, s.getTTL(ttlSeconds))
	return nil
}

// GetString retrieves a string value from the cache.
func (s *cacheServiceImpl) GetString(ctx context.Context, key string) (string, bool, error) {
	item := s.cache.Get(key)
	if item == nil {
		return "", false, nil
	}

	value, ok := item.Value().(string)
	if !ok {
		log.Debug(ctx, "Cache type mismatch", "plugin", s.pluginName, "key", key, "expected", "string")
		return "", false, nil
	}
	return value, true, nil
}

// SetInt stores an integer value in the cache.
func (s *cacheServiceImpl) SetInt(ctx context.Context, key string, value int64, ttlSeconds int64) error {
	s.cache.Set(key, value, s.getTTL(ttlSeconds))
	return nil
}

// GetInt retrieves an integer value from the cache.
func (s *cacheServiceImpl) GetInt(ctx context.Context, key string) (int64, bool, error) {
	item := s.cache.Get(key)
	if item == nil {
		return 0, false, nil
	}

	value, ok := item.Value().(int64)
	if !ok {
		log.Debug(ctx, "Cache type mismatch", "plugin", s.pluginName, "key", key, "expected", "int64")
		return 0, false, nil
	}
	return value, true, nil
}

// SetFloat stores a float value in the cache.
func (s *cacheServiceImpl) SetFloat(ctx context.Context, key string, value float64, ttlSeconds int64) error {
	s.cache.Set(key, value, s.getTTL(ttlSeconds))
	return nil
}

// GetFloat retrieves a float value from the cache.
func (s *cacheServiceImpl) GetFloat(ctx context.Context, key string) (float64, bool, error) {
	item := s.cache.Get(key)
	if item == nil {
		return 0, false, nil
	}

	value, ok := item.Value().(float64)
	if !ok {
		log.Debug(ctx, "Cache type mismatch", "plugin", s.pluginName, "key", key, "expected", "float64")
		return 0, false, nil
	}
	return value, true, nil
}

// SetBytes stores a byte slice in the cache.
func (s *cacheServiceImpl) SetBytes(ctx context.Context, key string, value []byte, ttlSeconds int64) error {
	s.cache.Set(key, value, s.getTTL(ttlSeconds))
	return nil
}

// GetBytes retrieves a byte slice from the cache.
func (s *cacheServiceImpl) GetBytes(ctx context.Context, key string) ([]byte, bool, error) {
	item := s.cache.Get(key)
	if item == nil {
		return nil, false, nil
	}

	value, ok := item.Value().([]byte)
	if !ok {
		log.Debug(ctx, "Cache type mismatch", "plugin", s.pluginName, "key", key, "expected", "[]byte")
		return nil, false, nil
	}
	return value, true, nil
}

// Has checks if a key exists in the cache.
func (s *cacheServiceImpl) Has(ctx context.Context, key string) (bool, error) {
	item := s.cache.Get(key)
	return item != nil, nil
}

// Remove deletes a value from the cache.
func (s *cacheServiceImpl) Remove(ctx context.Context, key string) error {
	s.cache.Delete(key)
	return nil
}

// Close stops the cache's janitor goroutine and clears all entries.
// This is called when the plugin is unloaded.
func (s *cacheServiceImpl) Close() error {
	s.cache.Stop()
	s.cache.DeleteAll()
	log.Debug("Closed plugin cache", "plugin", s.pluginName)
	return nil
}

// Ensure cacheServiceImpl implements host.CacheService
var _ host.CacheService = (*cacheServiceImpl)(nil)
