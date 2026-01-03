package host

import "context"

// CacheService provides in-memory TTL-based caching capabilities for plugins.
//
// This service allows plugins to store and retrieve typed values (strings, integers,
// floats, and byte slices) with configurable time-to-live expiration. Each plugin's
// cache keys are automatically namespaced to prevent collisions between plugins.
//
// The cache is in-memory only and will be lost on server restart. Plugins should
// handle cache misses gracefully.
//
//nd:hostservice name=Cache permission=cache
type CacheService interface {
	// SetString stores a string value in the cache.
	//
	// Parameters:
	//   - key: The cache key (will be namespaced with plugin ID)
	//   - value: The string value to store
	//   - ttlSeconds: Time-to-live in seconds (0 uses default of 24 hours)
	//
	// Returns an error if the operation fails.
	//nd:hostfunc
	SetString(ctx context.Context, key string, value string, ttlSeconds int64) error

	// GetString retrieves a string value from the cache.
	//
	// Parameters:
	//   - key: The cache key (will be namespaced with plugin ID)
	//
	// Returns the value and whether the key exists. If the key doesn't exist
	// or the stored value is not a string, exists will be false.
	//nd:hostfunc
	GetString(ctx context.Context, key string) (value string, exists bool, err error)

	// SetInt stores an integer value in the cache.
	//
	// Parameters:
	//   - key: The cache key (will be namespaced with plugin ID)
	//   - value: The integer value to store
	//   - ttlSeconds: Time-to-live in seconds (0 uses default of 24 hours)
	//
	// Returns an error if the operation fails.
	//nd:hostfunc
	SetInt(ctx context.Context, key string, value int64, ttlSeconds int64) error

	// GetInt retrieves an integer value from the cache.
	//
	// Parameters:
	//   - key: The cache key (will be namespaced with plugin ID)
	//
	// Returns the value and whether the key exists. If the key doesn't exist
	// or the stored value is not an integer, exists will be false.
	//nd:hostfunc
	GetInt(ctx context.Context, key string) (value int64, exists bool, err error)

	// SetFloat stores a float value in the cache.
	//
	// Parameters:
	//   - key: The cache key (will be namespaced with plugin ID)
	//   - value: The float value to store
	//   - ttlSeconds: Time-to-live in seconds (0 uses default of 24 hours)
	//
	// Returns an error if the operation fails.
	//nd:hostfunc
	SetFloat(ctx context.Context, key string, value float64, ttlSeconds int64) error

	// GetFloat retrieves a float value from the cache.
	//
	// Parameters:
	//   - key: The cache key (will be namespaced with plugin ID)
	//
	// Returns the value and whether the key exists. If the key doesn't exist
	// or the stored value is not a float, exists will be false.
	//nd:hostfunc
	GetFloat(ctx context.Context, key string) (value float64, exists bool, err error)

	// SetBytes stores a byte slice in the cache.
	//
	// Parameters:
	//   - key: The cache key (will be namespaced with plugin ID)
	//   - value: The byte slice to store
	//   - ttlSeconds: Time-to-live in seconds (0 uses default of 24 hours)
	//
	// Returns an error if the operation fails.
	//nd:hostfunc
	SetBytes(ctx context.Context, key string, value []byte, ttlSeconds int64) error

	// GetBytes retrieves a byte slice from the cache.
	//
	// Parameters:
	//   - key: The cache key (will be namespaced with plugin ID)
	//
	// Returns the value and whether the key exists. If the key doesn't exist
	// or the stored value is not a byte slice, exists will be false.
	//nd:hostfunc
	GetBytes(ctx context.Context, key string) (value []byte, exists bool, err error)

	// Has checks if a key exists in the cache.
	//
	// Parameters:
	//   - key: The cache key (will be namespaced with plugin ID)
	//
	// Returns true if the key exists and has not expired.
	//nd:hostfunc
	Has(ctx context.Context, key string) (exists bool, err error)

	// Remove deletes a value from the cache.
	//
	// Parameters:
	//   - key: The cache key (will be namespaced with plugin ID)
	//
	// Returns an error if the operation fails. Does not return an error if the key doesn't exist.
	//nd:hostfunc
	Remove(ctx context.Context, key string) error
}
