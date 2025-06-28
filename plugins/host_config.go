package plugins

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/plugins/host/config"
)

type configServiceImpl struct {
	pluginID string
}

func (c *configServiceImpl) GetPluginConfig(ctx context.Context, req *config.GetPluginConfigRequest) (*config.GetPluginConfigResponse, error) {
	cfg, ok := conf.Server.PluginConfig[c.pluginID]
	if !ok {
		cfg = map[string]string{}
	}
	return &config.GetPluginConfigResponse{
		Config: cfg,
	}, nil
}
