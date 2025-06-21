package plugins

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/tetratelabs/wazero"
)

// newWasmWebSocketCallback creates a new adapter for a WebSocketCallback plugin
func newWasmWebSocketCallback(wasmPath, pluginID string, runtime api.WazeroNewRuntime, mc wazero.ModuleConfig) WasmPlugin {
	loader, err := api.NewWebSocketCallbackPlugin(context.Background(), api.WazeroRuntime(runtime), api.WazeroModuleConfig(mc))
	if err != nil {
		log.Error("Error creating WebSocket callback plugin", "plugin", pluginID, "path", wasmPath, err)
		return nil
	}
	return &wasmWebSocketCallback{
		wasmBasePlugin: &wasmBasePlugin[api.WebSocketCallback, *api.WebSocketCallbackPlugin]{
			wasmPath:   wasmPath,
			id:         pluginID,
			capability: CapabilityWebSocketCallback,
			loader:     loader,
			loadFunc: func(ctx context.Context, l *api.WebSocketCallbackPlugin, path string) (api.WebSocketCallback, error) {
				return l.Load(ctx, path)
			},
		},
	}
}

// wasmWebSocketCallback adapts a WebSocketCallback plugin
type wasmWebSocketCallback struct {
	*wasmBasePlugin[api.WebSocketCallback, *api.WebSocketCallbackPlugin]
}
