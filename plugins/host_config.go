package plugins

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/host"
)

// configServiceImpl implements the host.ConfigService interface.
// It provides access to plugin configuration values set in the Navidrome config file.
type configServiceImpl struct {
	pluginName string
	config     map[string]string
}

// newConfigService creates a new configServiceImpl instance.
func newConfigService(pluginName string, config map[string]string) *configServiceImpl {
	if config == nil {
		config = make(map[string]string)
	}
	return &configServiceImpl{
		pluginName: pluginName,
		config:     config,
	}
}

// Get retrieves a configuration value as a string.
func (s *configServiceImpl) Get(ctx context.Context, key string) (string, bool) {
	value, exists := s.config[key]
	log.Trace(ctx, "Config.Get", "plugin", s.pluginName, "key", key, "exists", exists)
	return value, exists
}

// GetInt retrieves a configuration value as an integer.
func (s *configServiceImpl) GetInt(ctx context.Context, key string) (int64, bool) {
	value, exists := s.config[key]
	if !exists {
		log.Trace(ctx, "Config.GetInt", "plugin", s.pluginName, "key", key, "exists", false)
		return 0, false
	}

	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		log.Trace(ctx, "Config.GetInt parse error", "plugin", s.pluginName, "key", key, "value", value, "error", err)
		return 0, false
	}

	log.Trace(ctx, "Config.GetInt", "plugin", s.pluginName, "key", key, "value", intValue)
	return intValue, true
}

// Keys returns configuration keys matching the given prefix.
func (s *configServiceImpl) Keys(ctx context.Context, prefix string) []string {
	keys := make([]string, 0, len(s.config))
	for k := range s.config {
		if prefix == "" || strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	log.Trace(ctx, "Config.Keys", "plugin", s.pluginName, "prefix", prefix, "keyCount", len(keys))
	return keys
}

var _ host.ConfigService = (*configServiceImpl)(nil)
