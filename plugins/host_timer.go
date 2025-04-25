package plugins

import (
	"context"
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

	timerID, _ := gonanoid.New(10)

	// Create a context with cancel for this timer
	timerCtx, cancel := context.WithCancel(context.Background())

	// Store the callback info
	t.timers[timerID] = &TimerCallback{
		PluginName: req.PluginName,
		Payload:    req.Payload,
		Cancel:     cancel,
	}

	// Start the timer goroutine
	go t.runTimer(timerCtx, timerID, time.Duration(req.Delay)*time.Second)

	return &timer.TimerResponse{
		TimerId: timerID,
	}, nil
}

// CancelTimer implements the TimerService interface
func (t *TimerService) CancelTimer(ctx context.Context, req *timer.CancelTimerRequest) (*timer.CancelTimerResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	callback, exists := t.timers[req.TimerId]
	if !exists {
		return &timer.CancelTimerResponse{
			Success: false,
			Error:   "timer not found",
		}, nil
	}

	// Cancel the timer
	callback.Cancel()

	// Remove from map
	delete(t.timers, req.TimerId)

	return &timer.CancelTimerResponse{
		Success: true,
	}, nil
}

// runTimer handles the timer execution and callback
func (t *TimerService) runTimer(ctx context.Context, timerID string, delay time.Duration) {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		// Timer was cancelled
		t.mu.Lock()
		delete(t.timers, timerID)
		t.mu.Unlock()
		return

	case <-timer.C:
		// Timer fired, execute the callback
		var callback *TimerCallback

		t.mu.Lock()
		callback = t.timers[timerID]
		delete(t.timers, timerID)
		t.mu.Unlock()

		if callback != nil {
			t.executeCallback(ctx, timerID, callback)
		}
	}
}

// executeCallback calls the plugin's OnTimerCallback method
func (t *TimerService) executeCallback(ctx context.Context, timerID string, callback *TimerCallback) {
	log.Debug("Executing timer callback", "plugin", callback.PluginName, "timerID", timerID)

	// Create a TimerCallbackRequest
	req := &api.TimerCallbackRequest{
		TimerId: timerID,
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
		log.Error("Error executing timer callback", "plugin", callback.PluginName, err)
		return
	}

	if resp.Error != "" {
		log.Error("Plugin reported error in timer callback", "plugin", callback.PluginName, resp.Error)
	}
}
