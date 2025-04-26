package plugins

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/tetratelabs/wazero"
)

// NewWasmTimerCallback creates a new adapter for a TimerCallback plugin
func NewWasmTimerCallback(wasmPath, pluginName string, runtime api.WazeroNewRuntime, mc wazero.ModuleConfig) WasmPlugin {
	loader, err := api.NewTimerCallbackPlugin(context.Background(), api.WazeroRuntime(runtime), api.WazeroModuleConfig(mc))
	if err != nil {
		log.Error("Error creating timer callback plugin", "plugin", pluginName, "path", wasmPath, err)
		return nil
	}
	return &wasmTimerCallback{
		wasmBasePlugin: &wasmBasePlugin[api.TimerCallback, *api.TimerCallbackPlugin]{
			wasmPath: wasmPath,
			name:     pluginName,
			loader:   loader,
			loadFunc: func(ctx context.Context, l *api.TimerCallbackPlugin, path string) (api.TimerCallback, error) {
				return l.Load(ctx, path)
			},
		},
	}
}

// wasmTimerCallback adapts a TimerCallback plugin
type wasmTimerCallback struct {
	*wasmBasePlugin[api.TimerCallback, *api.TimerCallbackPlugin]
}

func (w *wasmTimerCallback) PluginName() string {
	return w.name
}
