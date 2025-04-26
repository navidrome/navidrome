package plugins

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/tetratelabs/wazero"
)

// NewWasmTimerCallback creates a new adapter for a TimerCallbackService plugin
func NewWasmTimerCallback(wasmPath, pluginName string, runtime api.WazeroNewRuntime, mc wazero.ModuleConfig) WasmPlugin {
	loader, err := api.NewTimerCallbackServicePlugin(context.Background(), api.WazeroRuntime(runtime), api.WazeroModuleConfig(mc))
	if err != nil {
		log.Error("Error creating timer callback service plugin", "plugin", pluginName, "path", wasmPath, err)
		return nil
	}
	return &wasmTimerCallback{
		wasmBasePlugin: &wasmBasePlugin[api.TimerCallbackService, *api.TimerCallbackServicePlugin]{
			wasmPath: wasmPath,
			name:     pluginName,
			loader:   loader,
			loadFunc: func(ctx context.Context, l *api.TimerCallbackServicePlugin, path string) (api.TimerCallbackService, error) {
				return l.Load(ctx, path)
			},
		},
	}
}

// wasmTimerCallback adapts a TimerCallbackService plugin
type wasmTimerCallback struct {
	*wasmBasePlugin[api.TimerCallbackService, *api.TimerCallbackServicePlugin]
}

func (w *wasmTimerCallback) PluginName() string {
	return w.name
}
