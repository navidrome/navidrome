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

// callPluginFunctionNoInput calls a plugin function that takes no input.
// It only checks for errors, with no response expected.
func callPluginFunctionNoInput(ctx context.Context, plugin *plugin, funcName string) error {
	start := time.Now()

	// Create plugin instance
	p, err := plugin.instance()
	if err != nil {
		return fmt.Errorf("failed to create plugin: %w", err)
	}
	defer p.Close(ctx)

	if !p.FunctionExists(funcName) {
		log.Trace(ctx, "Plugin function not found", "plugin", plugin.name, "function", funcName)
		return fmt.Errorf("%w: %s", errFunctionNotFound, funcName)
	}

	startCall := time.Now()
	exit, _, err := p.Call(funcName, nil)
	if err != nil {
		log.Trace(ctx, "Plugin call failed", "plugin", plugin.name, "function", funcName, "pluginDuration", time.Since(startCall), "navidromeDuration", startCall.Sub(start), err)
		return fmt.Errorf("plugin call failed: %w", err)
	}
	if exit != 0 {
		return fmt.Errorf("plugin call exited with code %d", exit)
	}

	log.Trace(ctx, "Plugin call succeeded", "plugin", plugin.name, "function", funcName, "pluginDuration", time.Since(startCall), "navidromeDuration", startCall.Sub(start))
	return nil
}

// callPluginFunctionNoOutput calls a plugin function with input but no response expected.
// It handles JSON marshalling for input and only checks for errors.
func callPluginFunctionNoOutput[I any](ctx context.Context, plugin *plugin, funcName string, input I) error {
	start := time.Now()

	// Create plugin instance
	p, err := plugin.instance()
	if err != nil {
		return fmt.Errorf("failed to create plugin: %w", err)
	}
	defer p.Close(ctx)

	if !p.FunctionExists(funcName) {
		log.Trace(ctx, "Plugin function not found", "plugin", plugin.name, "function", funcName)
		return fmt.Errorf("%w: %s", errFunctionNotFound, funcName)
	}

	inputBytes, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal input: %w", err)
	}

	startCall := time.Now()
	exit, _, err := p.Call(funcName, inputBytes)
	if err != nil {
		log.Trace(ctx, "Plugin call failed", "plugin", plugin.name, "function", funcName, "pluginDuration", time.Since(startCall), "navidromeDuration", startCall.Sub(start), err)
		return fmt.Errorf("plugin call failed: %w", err)
	}
	if exit != 0 {
		return fmt.Errorf("plugin call exited with code %d", exit)
	}

	log.Trace(ctx, "Plugin call succeeded", "plugin", plugin.name, "function", funcName, "pluginDuration", time.Since(startCall), "navidromeDuration", startCall.Sub(start))
	return nil
}

// callPluginFunction is a helper to call a plugin function with input and output types.
// It handles JSON marshalling/unmarshalling and error checking.
func callPluginFunction[I any, O any](ctx context.Context, plugin *plugin, funcName string, input I) (O, error) {
	start := time.Now()

	var result O

	// Create plugin instance
	p, err := plugin.instance()
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
	exit, output, err := p.Call(funcName, inputBytes)
	if err != nil {
		log.Trace(ctx, "Plugin call failed", "plugin", plugin.name, "function", funcName, "pluginDuration", time.Since(startCall), "navidromeDuration", startCall.Sub(start), err)
		return result, fmt.Errorf("plugin call failed: %w", err)
	}
	if exit != 0 {
		return result, fmt.Errorf("plugin call exited with code %d", exit)
	}

	if len(output) > 0 {
		err = json.Unmarshal(output, &result)
		if err != nil {
			log.Trace(ctx, "Plugin call failed", "plugin", plugin.name, "function", funcName, "pluginDuration", time.Since(startCall), "navidromeDuration", startCall.Sub(start), err)
		}
	}

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
