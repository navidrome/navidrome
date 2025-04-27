package plugins

import (
	"context"
	"fmt"
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

type TimerHostFunctions struct {
	ts         *timerService
	pluginName string
}

// RegisterTimer implements the TimerService interface
func (t TimerHostFunctions) RegisterTimer(ctx context.Context, req *timer.TimerRequest) (*timer.TimerResponse, error) {
	return t.ts.register(ctx, t.pluginName, req)
}

// CancelTimer implements the TimerService interface
func (t TimerHostFunctions) CancelTimer(ctx context.Context, req *timer.CancelTimerRequest) (*timer.CancelTimerResponse, error) {
	return t.ts.cancel(ctx, t.pluginName, req)
}

// timerService implements the timer.TimerService interface
type timerService struct {
	// Map of timer IDs to their callback info
	timers  map[string]*TimerCallback
	manager *Manager
	mu      sync.Mutex
}

// newTimerService creates a new timerService instance
func newTimerService(manager *Manager) *timerService {
	return &timerService{
		timers:  make(map[string]*TimerCallback),
		manager: manager,
	}
}

func (t *timerService) HostFunctions(pluginName string) TimerHostFunctions {
	return TimerHostFunctions{
		ts:         t,
		pluginName: pluginName,
	}
}

// Safe accessor methods for tests

// hasTimer safely checks if a timer exists
func (t *timerService) hasTimer(id string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, exists := t.timers[id]
	return exists
}

// timerCount safely returns the number of timers
func (t *timerService) timerCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.timers)
}

func (t *timerService) register(_ context.Context, pluginName string, req *timer.TimerRequest) (*timer.TimerResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.manager == nil {
		return nil, fmt.Errorf("timer service not properly initialized")
	}

	// Original timerId (what the plugin will see)
	originalTimerId := req.TimerId
	if originalTimerId == "" {
		// Generate a random ID if one wasn't provided
		originalTimerId, _ = gonanoid.New(10)
	}

	// Internal timerId (prefixed with plugin name to avoid conflicts)
	internalTimerId := pluginName + ":" + originalTimerId

	// Create a context with cancel for this timer
	timerCtx, cancel := context.WithCancel(context.Background())

	// Store the callback info using the prefixed internal ID
	t.timers[internalTimerId] = &TimerCallback{
		PluginName: pluginName,
		Payload:    req.Payload,
		Cancel:     cancel,
	}
	log.Debug("Timer registered", "plugin", pluginName, "timerID", originalTimerId, "internalID", internalTimerId)

	// Start the timer goroutine with the internal ID
	go t.runTimer(timerCtx, internalTimerId, originalTimerId, time.Duration(req.Delay)*time.Second)

	// Return the original ID to the plugin
	return &timer.TimerResponse{
		TimerId: originalTimerId,
	}, nil
}

func (t *timerService) cancel(_ context.Context, pluginName string, req *timer.CancelTimerRequest) (*timer.CancelTimerResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	cb := t.timers[pluginName+":"+req.TimerId]
	if cb == nil {
		return &timer.CancelTimerResponse{
			Success: false,
		}, fmt.Errorf("timer not found")
	}

	delete(t.timers, pluginName+":"+req.TimerId)

	// Cancel the timer
	cb.Cancel()

	return &timer.CancelTimerResponse{
		Success: true,
	}, nil
}

// runTimer handles the timer execution and callback
func (t *timerService) runTimer(ctx context.Context, internalTimerId, originalTimerId string, delay time.Duration) {
	tmr := time.NewTimer(delay)
	defer tmr.Stop()

	select {
	case <-ctx.Done():
		// Timer was cancelled
		t.mu.Lock()
		delete(t.timers, internalTimerId)
		t.mu.Unlock()
		log.Debug("Timer canceled", "internalID", internalTimerId)
		return

	case <-tmr.C:
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
func (t *timerService) executeCallback(ctx context.Context, originalTimerId string, callback *TimerCallback) {
	log.Debug("Executing timer callback", "plugin", callback.PluginName, "timerID", originalTimerId)
	start := time.Now()

	// Create a TimerCallbackRequest with the original (unprefixed) timer ID
	req := &api.TimerCallbackRequest{
		TimerId: originalTimerId,
		Payload: callback.Payload,
	}

	// Get the plugin
	p := t.manager.LoadPlugin(callback.PluginName, CapabilityTimerCallback)
	if p == nil {
		log.Error("Plugin not found for callback", "plugin", callback.PluginName)
		return
	}

	// Get instance
	inst, closeFn, err := p.GetInstance(ctx)
	if err != nil {
		log.Error("Error getting plugin instance for callback", "plugin", callback.PluginName, err)
		return
	}
	defer closeFn()

	// Type-check the plugin
	plugin, ok := inst.(api.TimerCallback)
	if !ok {
		log.Error("Plugin does not implement TimerCallback", "plugin", callback.PluginName)
		return
	}

	// Call the plugin's OnTimerCallback method
	log.Trace(ctx, "Executing timer callback", "plugin", callback.PluginName, "timerID", originalTimerId)
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
