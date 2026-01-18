package host

import "context"

// ConfigService provides access to plugin configuration values.
//
// This service allows plugins to retrieve configuration values and enumerate
// available configuration keys. Unlike the built-in pdk.GetConfig(key) which
// only retrieves individual values, this service provides methods to list all
// available keys, making it useful for plugins that need to discover dynamic
// configuration (e.g., user-to-token mappings).
//
// This service is always available and does not require a permission in the manifest.
//
//nd:hostservice name=Config
type ConfigService interface {
	// Get retrieves a configuration value as a string.
	//
	// Parameters:
	//   - key: The configuration key
	//
	// Returns the value and whether the key exists.
	//nd:hostfunc
	Get(ctx context.Context, key string) (value string, exists bool)

	// GetInt retrieves a configuration value as an integer.
	//
	// Parameters:
	//   - key: The configuration key
	//
	// Returns the value and whether the key exists. If the key exists but the
	// value cannot be parsed as an integer, exists will be false.
	//nd:hostfunc
	GetInt(ctx context.Context, key string) (value int64, exists bool)

	// Keys returns configuration keys matching the given prefix.
	//
	// Parameters:
	//   - prefix: Key prefix to filter by. If empty, returns all keys.
	//
	// Returns a sorted slice of matching configuration keys.
	//nd:hostfunc
	Keys(ctx context.Context, prefix string) (keys []string)
}
