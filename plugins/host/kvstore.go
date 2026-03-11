package host

import "context"

// KVStoreService provides persistent key-value storage for plugins.
//
// Unlike CacheService which is in-memory only, KVStoreService persists data
// to disk and survives server restarts. Each plugin has its own isolated
// storage with configurable size limits.
//
// Values are stored as raw bytes, giving plugins full control over
// serialization (JSON, protobuf, etc.).
//
//nd:hostservice name=KVStore permission=kvstore
type KVStoreService interface {
	// Set stores a byte value with the given key.
	//
	// Parameters:
	//   - key: The storage key (max 256 bytes, UTF-8)
	//   - value: The byte slice to store
	//
	// Returns an error if the storage limit would be exceeded or the operation fails.
	//nd:hostfunc
	Set(ctx context.Context, key string, value []byte) error

	// SetWithTTL stores a byte value with the given key and a time-to-live.
	//
	// After ttlSeconds, the key is treated as non-existent and will be
	// cleaned up lazily. ttlSeconds must be greater than 0.
	//
	// Parameters:
	//   - key: The storage key (max 256 bytes, UTF-8)
	//   - value: The byte slice to store
	//   - ttlSeconds: Time-to-live in seconds (must be > 0)
	//
	// Returns an error if the storage limit would be exceeded or the operation fails.
	//nd:hostfunc
	SetWithTTL(ctx context.Context, key string, value []byte, ttlSeconds int64) error

	// Get retrieves a byte value from storage.
	//
	// Parameters:
	//   - key: The storage key
	//
	// Returns the value and whether the key exists.
	//nd:hostfunc
	Get(ctx context.Context, key string) (value []byte, exists bool, err error)

	// GetMany retrieves multiple values in a single call.
	//
	// Parameters:
	//   - keys: The storage keys to retrieve
	//
	// Returns a map of key to value for keys that exist and have not expired.
	// Missing or expired keys are omitted from the result.
	//nd:hostfunc
	GetMany(ctx context.Context, keys []string) (values map[string][]byte, err error)

	// Has checks if a key exists in storage.
	//
	// Parameters:
	//   - key: The storage key
	//
	// Returns true if the key exists.
	//nd:hostfunc
	Has(ctx context.Context, key string) (exists bool, err error)

	// List returns all keys matching the given prefix.
	//
	// Parameters:
	//   - prefix: Key prefix to filter by (empty string returns all keys)
	//
	// Returns a slice of matching keys.
	//nd:hostfunc
	List(ctx context.Context, prefix string) (keys []string, err error)

	// Delete removes a value from storage.
	//
	// Parameters:
	//   - key: The storage key
	//
	// Returns an error if the operation fails. Does not return an error if the key doesn't exist.
	//nd:hostfunc
	Delete(ctx context.Context, key string) error

	// DeleteByPrefix removes all keys matching the given prefix.
	//
	// Parameters:
	//   - prefix: Key prefix to match (must not be empty)
	//
	// Returns the number of keys deleted. Includes expired keys.
	//nd:hostfunc
	DeleteByPrefix(ctx context.Context, prefix string) (deletedCount int64, err error)

	// GetStorageUsed returns the total storage used by this plugin in bytes.
	//nd:hostfunc
	GetStorageUsed(ctx context.Context) (bytes int64, err error)
}
