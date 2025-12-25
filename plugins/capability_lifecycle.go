package plugins

import (
	"context"

	"github.com/navidrome/navidrome/log"
)

// CapabilityLifecycle indicates the plugin has lifecycle callback functions.
// Detected when the plugin exports the nd_on_init function.
const CapabilityLifecycle Capability = "Lifecycle"

const FuncOnInit = "nd_on_init"

func init() {
	registerCapability(
		CapabilityLifecycle,
		FuncOnInit,
	)
}

// onInitInput is the input for nd_on_init (currently empty, reserved for future use)
type onInitInput struct{}

// onInitOutput is the output from nd_on_init
type onInitOutput struct {
	Error string `json:"error,omitempty"`
}

// callPluginInit calls the plugin's nd_on_init function if it has the Lifecycle capability.
// This is called after the plugin is fully loaded with all services registered.
func callPluginInit(ctx context.Context, instance *plugin) {
	if !hasCapability(instance.capabilities, CapabilityLifecycle) {
		return
	}

	log.Debug(ctx, "Calling plugin init function", "plugin", instance.name)

	result, err := callPluginFunction[onInitInput, onInitOutput](ctx, instance, FuncOnInit, onInitInput{})
	if err != nil {
		log.Error(ctx, "Plugin init function failed", "plugin", instance.name, err)
		return
	}

	if result.Error != "" {
		log.Error(ctx, "Plugin init function returned error", "plugin", instance.name, "error", result.Error)
		return
	}

	log.Debug(ctx, "Plugin init function completed", "plugin", instance.name)
}
