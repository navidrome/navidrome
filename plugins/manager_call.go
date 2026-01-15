package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	extism "github.com/extism/go-sdk"
	"github.com/navidrome/navidrome/log"
)

var errFunctionNotFound = errors.New("function not found")
var errNotImplemented = errors.New("function not implemented")

// notImplementedCode is the standard return code from plugin PDKs
// indicating a function exists but is not implemented by this plugin.
// The plugin returns -2 as int32, which becomes 0xFFFFFFFE as uint32.
const notImplementedCode uint32 = 0xFFFFFFFE

// callPluginFunctionNoInput is a helper to call a plugin function with no input and output.
func callPluginFunctionNoInput(ctx context.Context, plugin *plugin, funcName string) error {
	_, err := callPluginFunction[struct{}, struct{}](ctx, plugin, funcName, struct{}{})
	return err
}

// callPluginFunctionNoOutput is a helper to call a plugin function with input and no output.
func callPluginFunctionNoOutput[I any](ctx context.Context, plugin *plugin, funcName string, input I) error {
	_, err := callPluginFunction[I, struct{}](ctx, plugin, funcName, input)
	return err
}

// callPluginFunction is a helper to call a plugin function with input and output types.
// It handles JSON marshalling/unmarshalling and error checking.
// The context is used for cancellation - if cancelled during the call, the plugin
// instance will be terminated and context.Canceled or context.DeadlineExceeded will be returned.
func callPluginFunction[I any, O any](ctx context.Context, plugin *plugin, funcName string, input I) (O, error) {
	start := time.Now()

	var result O

	// Create plugin instance with context for cancellation support
	p, err := plugin.instance(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to create plugin: %w", err)
	}
	defer p.Close(ctx)

	if !p.FunctionExists(funcName) {
		log.Trace(ctx, "Plugin function not found", "plugin", plugin.name, "function", funcName)
		return result, fmt.Errorf("%w: %s", errFunctionNotFound, funcName)
	}

	inputBytes, err := json.Marshal(input)
	if err != nil {
		return result, fmt.Errorf("failed to marshal input: %w", err)
	}

	startCall := time.Now()
	exit, output, err := p.CallWithContext(ctx, funcName, inputBytes)
	elapsed := time.Since(startCall)
	if err != nil {
		// If context was cancelled, return that error instead of the plugin error
		if ctx.Err() != nil {
			log.Debug(ctx, "Plugin call cancelled", "plugin", plugin.name, "function", funcName, "pluginDuration", elapsed)
			return result, ctx.Err()
		}
		plugin.metrics.RecordPluginRequest(ctx, plugin.name, funcName, false, elapsed.Milliseconds())
		log.Trace(ctx, "Plugin call failed", "plugin", plugin.name, "function", funcName, "pluginDuration", elapsed, "navidromeDuration", startCall.Sub(start), err)
		return result, fmt.Errorf("plugin call failed: %w", err)
	}
	if exit != 0 {
		if exit == notImplementedCode {
			plugin.metrics.RecordPluginRequest(ctx, plugin.name, funcName, false, elapsed.Milliseconds())
			return result, fmt.Errorf("%w: %s", errNotImplemented, funcName)
		}
		plugin.metrics.RecordPluginRequest(ctx, plugin.name, funcName, false, elapsed.Milliseconds())
		return result, fmt.Errorf("plugin call exited with code %d", exit)
	}

	if len(output) > 0 {
		err = json.Unmarshal(output, &result)
		if err != nil {
			log.Trace(ctx, "Plugin call failed", "plugin", plugin.name, "function", funcName, "pluginDuration", elapsed, "navidromeDuration", startCall.Sub(start), err)
		}
	}

	// Record metrics for successful calls (or JSON unmarshal failures)
	plugin.metrics.RecordPluginRequest(ctx, plugin.name, funcName, err == nil, elapsed.Milliseconds())

	log.Trace(ctx, "Plugin call succeeded", "plugin", plugin.name, "function", funcName, "pluginDuration", time.Since(startCall), "navidromeDuration", startCall.Sub(start))
	return result, err
}

// extismLogger is a helper to log messages from Extism plugins
func extismLogger(pluginName string) func(level extism.LogLevel, msg string) {
	return func(level extism.LogLevel, msg string) {
		if level == extism.LogLevelOff {
			return
		}
		log.Log(log.ParseLogLevel(level.String()), msg, "plugin", pluginName)
	}
}

// toExtismLogLevel converts a Navidrome log level to an extism LogLevel
func toExtismLogLevel(level log.Level) extism.LogLevel {
	switch level {
	case log.LevelTrace:
		return extism.LogLevelTrace
	case log.LevelDebug:
		return extism.LogLevelDebug
	case log.LevelInfo:
		return extism.LogLevelInfo
	case log.LevelWarn:
		return extism.LogLevelWarn
	case log.LevelError, log.LevelFatal:
		return extism.LogLevelError
	default:
		return extism.LogLevelInfo
	}
}
