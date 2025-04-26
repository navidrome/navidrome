//go:build !wasip1

package plugins

import (
	"context"
	"strings"
	"sync"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host/timer"
)

// TimerCallback represents a registered callback
type TimerCallback struct {
	PluginName string
	Payload    []byte
	Cancel     context.CancelFunc
}

// TimerService implements the timer.TimerService interface
type TimerService struct {
	// Map of timer IDs to their callback info
	timers  map[string]*TimerCallback
	manager *Manager
	mu      sync.Mutex
}

// NewTimerService creates a new TimerService instance
func NewTimerService(manager *Manager) *TimerService {
	return &TimerService{
		timers:  make(map[string]*TimerCallback),
		manager: manager,
	}
}

// RegisterTimer implements the TimerService interface
func (t *TimerService) RegisterTimer(ctx context.Context, req *timer.TimerRequest) (*timer.TimerResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.manager == nil {
		return &timer.TimerResponse{
			Error: "timer service not properly initialized",
		}, nil
	}

	// Original timerId (what the plugin will see)
	originalTimerId := req.TimerId
	if originalTimerId == "" {
		// Generate a random ID if one wasn't provided
		originalTimerId, _ = gonanoid.New(10)
	}

	// Internal timerId (prefixed with plugin name to avoid conflicts)
	internalTimerId := req.PluginName + ":" + originalTimerId

	// Create a context with cancel for this timer
	timerCtx, cancel := context.WithCancel(context.Background())

	// Store the callback info using the prefixed internal ID
	t.timers[internalTimerId] = &TimerCallback{
		PluginName: req.PluginName,
		Payload:    req.Payload,
		Cancel:     cancel,
	}

	// Start the timer goroutine with the internal ID
	go t.runTimer(timerCtx, internalTimerId, originalTimerId, time.Duration(req.Delay)*time.Second)

	// Return the original ID to the plugin
	return &timer.TimerResponse{
		TimerId: originalTimerId,
	}, nil
}

// CancelTimer implements the TimerService interface
func (t *TimerService) CancelTimer(ctx context.Context, req *timer.CancelTimerRequest) (*timer.CancelTimerResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Extract plugin name from context or request somehow
	// For now, we'll need to look for all possible keys
	var found bool
	var callback *TimerCallback

	// Try to find a timer with this ID from any plugin
	for key, cb := range t.timers {
		// Check if the key ends with the requested timer ID
		parts := strings.Split(key, ":")
		if len(parts) == 2 && parts[1] == req.TimerId {
			found = true
			callback = cb
			delete(t.timers, key)
			break
		}
	}

	if !found {
		return &timer.CancelTimerResponse{
			Success: false,
			Error:   "timer not found",
		}, nil
	}

	// Cancel the timer
	callback.Cancel()

	return &timer.CancelTimerResponse{
		Success: true,
	}, nil
}

// runTimer handles the timer execution and callback
func (t *TimerService) runTimer(ctx context.Context, internalTimerId, originalTimerId string, delay time.Duration) {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		// Timer was cancelled
		t.mu.Lock()
		delete(t.timers, internalTimerId)
		t.mu.Unlock()
		return

	case <-timer.C:
		// Timer fired, execute the callback
		var callback *TimerCallback

		t.mu.Lock()
		callback = t.timers[internalTimerId]
		delete(t.timers, internalTimerId)
		t.mu.Unlock()

		if callback != nil {
			// Pass the original (non-prefixed) timer ID to the callback
			t.executeCallback(ctx, originalTimerId, callback)
		}
	}
}

// executeCallback calls the plugin's OnTimerCallback method
func (t *TimerService) executeCallback(ctx context.Context, originalTimerId string, callback *TimerCallback) {
	log.Debug("Executing timer callback", "plugin", callback.PluginName, "timerID", originalTimerId)
	start := time.Now()

	// Create a TimerCallbackRequest with the original (unprefixed) timer ID
	req := &api.TimerCallbackRequest{
		TimerId: originalTimerId,
		Payload: callback.Payload,
	}

	// Get the callback plugin info
	pluginInfo := t.manager.GetPluginInfo(callback.PluginName)
	if pluginInfo == nil {
		log.Error("Plugin not registered for timer callback", "plugin", callback.PluginName)
		return
	}

	// It must be a TimerCallbackPlugin
	loader, err := api.NewTimerCallbackServicePlugin(ctx, api.WazeroRuntime(pluginInfo.Runtime), api.WazeroModuleConfig(pluginInfo.ModConfig))
	if loader == nil || err != nil {
		log.Error("Plugin not found for timer callback", "plugin", callback.PluginName, err)
		return
	}

	plugin, err := loader.Load(ctx, pluginInfo.WasmPath)
	if err != nil {
		log.Error("Error loading plugin", "plugin", callback.PluginName, "path", pluginInfo.WasmPath, err)
		return
	}
	defer plugin.Close(ctx)

	// Call the plugin's OnTimerCallback method
	resp, err := plugin.OnTimerCallback(ctx, req)
	if err != nil {
		log.Error("Error executing timer callback", "plugin", callback.PluginName, "elapsed", time.Since(start), err)
		return
	}
	log.Debug("Timer callback executed", "plugin", callback.PluginName, "elapsed", time.Since(start))

	if resp.Error != "" {
		log.Error("Plugin reported error in timer callback", "plugin", callback.PluginName, resp.Error)
	}
}
