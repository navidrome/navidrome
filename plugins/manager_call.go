package plugins

import (
	"context"
	"encoding/binary"
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

	success := false
	skipMetrics := false
	defer func() {
		if !skipMetrics {
			plugin.metrics.RecordPluginRequest(ctx, plugin.name, funcName, success, elapsed.Milliseconds())
		}
	}()

	if err != nil {
		// If context was cancelled, return that error instead of the plugin error
		if ctx.Err() != nil {
			skipMetrics = true
			log.Debug(ctx, "Plugin call cancelled", "plugin", plugin.name, "function", funcName, "pluginDuration", elapsed)
			return result, ctx.Err()
		}
		log.Trace(ctx, "Plugin call failed", "plugin", plugin.name, "function", funcName, "pluginDuration", elapsed, "navidromeDuration", startCall.Sub(start), err)
		return result, fmt.Errorf("plugin call failed: %w", err)
	}
	if exit != 0 {
		if exit == notImplementedCode {
			skipMetrics = true
			log.Trace(ctx, "Plugin function not implemented", "plugin", plugin.name, "function", funcName, "pluginDuration", elapsed, "navidromeDuration", startCall.Sub(start))
			return result, fmt.Errorf("%w: %s", errNotImplemented, funcName)
		}
		return result, fmt.Errorf("plugin call exited with code %d", exit)
	}

	if len(output) > 0 {
		if err = json.Unmarshal(output, &result); err != nil {
			log.Trace(ctx, "Plugin call failed", "plugin", plugin.name, "function", funcName, "pluginDuration", elapsed, "navidromeDuration", startCall.Sub(start), err)
			return result, err
		}
	}

	success = true
	log.Trace(ctx, "Plugin call succeeded", "plugin", plugin.name, "function", funcName, "pluginDuration", time.Since(startCall), "navidromeDuration", startCall.Sub(start))
	return result, nil
}

// callPluginFunctionRaw calls a plugin function using binary framing for []byte fields.
// The input is JSON-encoded (with []byte field excluded via json:"-"), followed by raw bytes.
// The output frame is: [status:1B][json_len:4B][JSON][raw bytes] for success (0x00),
// or [0x01][UTF-8 error message] for errors.
func callPluginFunctionRaw[I any, O any](
	ctx context.Context, plugin *plugin, funcName string,
	input I, rawInputBytes []byte,
	setRawOutput func(*O, []byte),
) (O, error) {
	start := time.Now()

	var result O

	p, err := plugin.instance(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to create plugin: %w", err)
	}
	defer p.Close(ctx)

	if !p.FunctionExists(funcName) {
		log.Trace(ctx, "Plugin function not found", "plugin", plugin.name, "function", funcName)
		return result, fmt.Errorf("%w: %s", errFunctionNotFound, funcName)
	}

	// Build input frame: [json_len:4B][JSON][raw bytes]
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		return result, fmt.Errorf("failed to marshal input: %w", err)
	}
	totalSize := 4 + len(jsonBytes) + len(rawInputBytes)
	if totalSize < len(jsonBytes) || totalSize < len(rawInputBytes) {
		return result, fmt.Errorf("input frame too large")
	}
	frame := make([]byte, totalSize)
	binary.BigEndian.PutUint32(frame[:4], uint32(len(jsonBytes)))
	copy(frame[4:4+len(jsonBytes)], jsonBytes)
	copy(frame[4+len(jsonBytes):], rawInputBytes)

	startCall := time.Now()
	exit, output, err := p.CallWithContext(ctx, funcName, frame)
	elapsed := time.Since(startCall)

	success := false
	skipMetrics := false
	defer func() {
		if !skipMetrics {
			plugin.metrics.RecordPluginRequest(ctx, plugin.name, funcName, success, elapsed.Milliseconds())
		}
	}()

	if err != nil {
		if ctx.Err() != nil {
			skipMetrics = true
			log.Debug(ctx, "Plugin call cancelled", "plugin", plugin.name, "function", funcName, "pluginDuration", elapsed)
			return result, ctx.Err()
		}
		log.Trace(ctx, "Plugin call failed", "plugin", plugin.name, "function", funcName, "pluginDuration", elapsed, "navidromeDuration", startCall.Sub(start), err)
		return result, fmt.Errorf("plugin call failed: %w", err)
	}
	if exit != 0 {
		if exit == notImplementedCode {
			skipMetrics = true
			log.Trace(ctx, "Plugin function not implemented", "plugin", plugin.name, "function", funcName, "pluginDuration", elapsed, "navidromeDuration", startCall.Sub(start))
			return result, fmt.Errorf("%w: %s", errNotImplemented, funcName)
		}
		return result, fmt.Errorf("plugin call exited with code %d", exit)
	}

	// Parse output frame
	if len(output) < 1 {
		return result, fmt.Errorf("empty response from plugin")
	}

	statusByte := output[0]
	if statusByte == 0x01 {
		return result, fmt.Errorf("plugin error: %s", string(output[1:]))
	}
	if statusByte != 0x00 {
		return result, fmt.Errorf("unknown response status byte: 0x%02x", statusByte)
	}

	// Success frame: [0x00][json_len:4B][JSON][raw bytes]
	if len(output) < 5 {
		return result, fmt.Errorf("malformed success response from plugin")
	}

	jsonLen := binary.BigEndian.Uint32(output[1:5])
	if uint32(len(output)-5) < jsonLen {
		return result, fmt.Errorf("invalid json length in response frame: %d exceeds available %d bytes", jsonLen, len(output)-5)
	}
	jsonData := output[5 : 5+jsonLen]
	rawData := output[5+jsonLen:]

	if err := json.Unmarshal(jsonData, &result); err != nil {
		return result, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	setRawOutput(&result, rawData)

	success = true
	log.Trace(ctx, "Plugin call succeeded", "plugin", plugin.name, "function", funcName, "pluginDuration", time.Since(startCall), "navidromeDuration", startCall.Sub(start))
	return result, nil
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
