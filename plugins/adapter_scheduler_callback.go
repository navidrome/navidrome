package plugins

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/tetratelabs/wazero"
)

// newWasmSchedulerCallback creates a new adapter for a SchedulerCallback plugin
func newWasmSchedulerCallback(wasmPath, pluginID string, runtime api.WazeroNewRuntime, mc wazero.ModuleConfig) WasmPlugin {
	loader, err := api.NewSchedulerCallbackPlugin(context.Background(), api.WazeroRuntime(runtime), api.WazeroModuleConfig(mc))
	if err != nil {
		log.Error("Error creating scheduler callback plugin", "plugin", pluginID, "path", wasmPath, err)
		return nil
	}
	return &wasmSchedulerCallback{
		wasmBasePlugin: &wasmBasePlugin[api.SchedulerCallback, *api.SchedulerCallbackPlugin]{
			wasmPath:   wasmPath,
			id:         pluginID,
			capability: CapabilitySchedulerCallback,
			loader:     loader,
			loadFunc: func(ctx context.Context, l *api.SchedulerCallbackPlugin, path string) (api.SchedulerCallback, error) {
				return l.Load(ctx, path)
			},
		},
	}
}

// wasmSchedulerCallback adapts a SchedulerCallback plugin
type wasmSchedulerCallback struct {
	*wasmBasePlugin[api.SchedulerCallback, *api.SchedulerCallbackPlugin]
}
