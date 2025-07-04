package plugins

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/tetratelabs/wazero"
)

// newWasmSchedulerCallback creates a new adapter for a SchedulerCallback plugin
func newWasmSchedulerCallback(wasmPath, pluginID string, m *managerImpl, runtime api.WazeroNewRuntime, mc wazero.ModuleConfig) WasmPlugin {
	loader, err := api.NewSchedulerCallbackPlugin(context.Background(), api.WazeroRuntime(runtime), api.WazeroModuleConfig(mc))
	if err != nil {
		log.Error("Error creating scheduler callback plugin", "plugin", pluginID, "path", wasmPath, err)
		return nil
	}
	return &wasmSchedulerCallback{
		baseCapability: newBaseCapability[api.SchedulerCallback, *api.SchedulerCallbackPlugin](
			wasmPath,
			pluginID,
			CapabilitySchedulerCallback,
			m.metrics,
			loader,
			func(ctx context.Context, l *api.SchedulerCallbackPlugin, path string) (api.SchedulerCallback, error) {
				return l.Load(ctx, path)
			},
		),
	}
}

// wasmSchedulerCallback adapts a SchedulerCallback plugin
type wasmSchedulerCallback struct {
	*baseCapability[api.SchedulerCallback, *api.SchedulerCallbackPlugin]
}

func (w *wasmSchedulerCallback) OnSchedulerCallback(ctx context.Context, scheduleID string, payload []byte, isRecurring bool) error {
	_, err := callMethod(ctx, w, "OnSchedulerCallback", func(inst api.SchedulerCallback) (*api.SchedulerCallbackResponse, error) {
		return inst.OnSchedulerCallback(ctx, &api.SchedulerCallbackRequest{
			ScheduleId:  scheduleID,
			Payload:     payload,
			IsRecurring: isRecurring,
		})
	})
	return err
}
