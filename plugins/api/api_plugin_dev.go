//go:build !wasip1

package api

import "github.com/navidrome/navidrome/plugins/host/scheduler"

// This file exists to provide stubs for the plugin registration functions when building for non-WASM targets.
// This is useful for testing and development purposes, as it allows you to build and run your plugin code
// without having to compile it to WASM.
// In a real-world scenario, you would compile your plugin to WASM and use the generated registration functions.

func RegisterMetadataAgent(MetadataAgent) {
	panic("not implemented")
}

func RegisterScrobbler(Scrobbler) {
	panic("not implemented")
}

func RegisterSchedulerCallback(SchedulerCallback) {
	panic("not implemented")
}

func RegisterLifecycleManagement(LifecycleManagement) {
	panic("not implemented")
}

func RegisterWebSocketCallback(WebSocketCallback) {
	panic("not implemented")
}

func RegisterNamedSchedulerCallback(name string, cb SchedulerCallback) scheduler.SchedulerService {
	panic("not implemented")
}
