package plugins

import (
	"context"
	"sync"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/navidrome/navidrome/log"
	cacheproto "github.com/navidrome/navidrome/plugins/host/cache"
)

const (
	defaultCacheTTL = 24 * time.Hour
)

// cacheService is a singleton that manages caches for all plugins
type cacheService struct {
	caches     map[string]*ttlcache.Cache[string, interface{}]
	manager    *Manager
	mu         sync.RWMutex
	defaultTTL time.Duration
}

// cacheServiceImpl implements the cache.CacheService interface
type cacheServiceImpl struct {
	pluginName string
	manager    *Manager
}

// newCacheService creates a new cacheService instance
func newCacheService(manager *Manager) *cacheService {
	return &cacheService{
		caches:     make(map[string]*ttlcache.Cache[string, interface{}]),
		manager:    manager,
		defaultTTL: defaultCacheTTL,
	}
}

// HostFunctions returns the host functions for a specific plugin
func (s *cacheService) HostFunctions(pluginName string) *cacheServiceImpl {
	return &cacheServiceImpl{
		pluginName: pluginName,
		manager:    s.manager,
	}
}

// getCache gets or creates a cache for a plugin
func (s *cacheService) getCache(ctx context.Context, pluginName string) *ttlcache.Cache[string, interface{}] {
	s.mu.RLock()
	cache, ok := s.caches[pluginName]
	s.mu.RUnlock()

	if ok {
		return cache
	}

	// Need to create a new cache
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check, another goroutine might have created it
	if cache, ok = s.caches[pluginName]; ok {
		return cache
	}

	// Create a new cache
	opts := []ttlcache.Option[string, interface{}]{
		ttlcache.WithTTL[string, interface{}](s.defaultTTL),
		ttlcache.WithDisableTouchOnHit[string, interface{}](),
	}
	cache = ttlcache.New[string, interface{}](opts...)

	// Start the janitor goroutine to clean up expired entries
	go cache.Start()

	s.caches[pluginName] = cache
	log.Debug(ctx, "Created new plugin cache", "plugin", pluginName)
	return cache
}

// getTTL converts seconds to a duration, using default if 0
func (s *cacheService) getTTL(seconds int64) time.Duration {
	if seconds <= 0 {
		return s.defaultTTL
	}
	return time.Duration(seconds) * time.Second
}

// setCacheValue is a generic function to set a value in the cache
func setCacheValue[T any](ctx context.Context, cs *cacheServiceImpl, key string, value T, ttlSeconds int64) (*cacheproto.SetResponse, error) {
	c := cs.manager.cacheService.getCache(ctx, cs.pluginName)
	ttl := cs.manager.cacheService.getTTL(ttlSeconds)
	c.Set(key, value, ttl)
	return &cacheproto.SetResponse{Success: true}, nil
}

// getCacheValue is a generic function to get a value from the cache
func getCacheValue[T any](ctx context.Context, cs *cacheServiceImpl, key string, typeName string) (T, bool, error) {
	var zero T
	c := cs.manager.cacheService.getCache(ctx, cs.pluginName)
	item := c.Get(key)
	if item == nil {
		return zero, false, nil
	}

	value, ok := item.Value().(T)
	if !ok {
		log.Debug(ctx, "Type mismatch in cache", "plugin", cs.pluginName, "key", key, "expected", typeName)
		return zero, false, nil
	}
	return value, true, nil
}

// SetString sets a string value in the cache
func (s *cacheServiceImpl) SetString(ctx context.Context, req *cacheproto.SetStringRequest) (*cacheproto.SetResponse, error) {
	return setCacheValue(ctx, s, req.Key, req.Value, req.TtlSeconds)
}

// GetString gets a string value from the cache
func (s *cacheServiceImpl) GetString(ctx context.Context, req *cacheproto.GetRequest) (*cacheproto.GetStringResponse, error) {
	value, exists, err := getCacheValue[string](ctx, s, req.Key, "string")
	if err != nil {
		return nil, err
	}
	return &cacheproto.GetStringResponse{Exists: exists, Value: value}, nil
}

// SetInt sets an integer value in the cache
func (s *cacheServiceImpl) SetInt(ctx context.Context, req *cacheproto.SetIntRequest) (*cacheproto.SetResponse, error) {
	return setCacheValue(ctx, s, req.Key, req.Value, req.TtlSeconds)
}

// GetInt gets an integer value from the cache
func (s *cacheServiceImpl) GetInt(ctx context.Context, req *cacheproto.GetRequest) (*cacheproto.GetIntResponse, error) {
	value, exists, err := getCacheValue[int64](ctx, s, req.Key, "int64")
	if err != nil {
		return nil, err
	}
	return &cacheproto.GetIntResponse{Exists: exists, Value: value}, nil
}

// SetFloat sets a float value in the cache
func (s *cacheServiceImpl) SetFloat(ctx context.Context, req *cacheproto.SetFloatRequest) (*cacheproto.SetResponse, error) {
	return setCacheValue(ctx, s, req.Key, req.Value, req.TtlSeconds)
}

// GetFloat gets a float value from the cache
func (s *cacheServiceImpl) GetFloat(ctx context.Context, req *cacheproto.GetRequest) (*cacheproto.GetFloatResponse, error) {
	value, exists, err := getCacheValue[float64](ctx, s, req.Key, "float64")
	if err != nil {
		return nil, err
	}
	return &cacheproto.GetFloatResponse{Exists: exists, Value: value}, nil
}

// SetBytes sets a byte slice value in the cache
func (s *cacheServiceImpl) SetBytes(ctx context.Context, req *cacheproto.SetBytesRequest) (*cacheproto.SetResponse, error) {
	return setCacheValue(ctx, s, req.Key, req.Value, req.TtlSeconds)
}

// GetBytes gets a byte slice value from the cache
func (s *cacheServiceImpl) GetBytes(ctx context.Context, req *cacheproto.GetRequest) (*cacheproto.GetBytesResponse, error) {
	value, exists, err := getCacheValue[[]byte](ctx, s, req.Key, "[]byte")
	if err != nil {
		return nil, err
	}
	return &cacheproto.GetBytesResponse{Exists: exists, Value: value}, nil
}

// Remove removes a value from the cache
func (s *cacheServiceImpl) Remove(ctx context.Context, req *cacheproto.RemoveRequest) (*cacheproto.RemoveResponse, error) {
	c := s.manager.cacheService.getCache(ctx, s.pluginName)
	c.Delete(req.Key)
	return &cacheproto.RemoveResponse{Success: true}, nil
}

// Has checks if a key exists in the cache
func (s *cacheServiceImpl) Has(ctx context.Context, req *cacheproto.HasRequest) (*cacheproto.HasResponse, error) {
	c := s.manager.cacheService.getCache(ctx, s.pluginName)
	item := c.Get(req.Key)
	return &cacheproto.HasResponse{Exists: item != nil}, nil
}

// stopAllCaches stops all cache janitor routines
func (s *cacheService) stopAllCaches() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for plugin, c := range s.caches {
		c.Stop()
		log.Debug("Stopped cache janitor", "plugin", plugin)
	}
}
