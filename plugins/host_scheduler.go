package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/host/scheduler"
	navidsched "github.com/navidrome/navidrome/scheduler"
)

const (
	ScheduleTypeOneTime   = "one-time"
	ScheduleTypeRecurring = "recurring"
)

// ScheduledCallback represents a registered schedule callback
type ScheduledCallback struct {
	ID       string
	PluginID string
	Type     string // "one-time" or "recurring"
	Payload  []byte
	EntryID  int                // Used for recurring schedules via the scheduler
	Cancel   context.CancelFunc // Used for one-time schedules
}

// SchedulerHostFunctions implements the scheduler.SchedulerService interface
type SchedulerHostFunctions struct {
	ss       *schedulerService
	pluginID string
}

func (s SchedulerHostFunctions) ScheduleOneTime(ctx context.Context, req *scheduler.ScheduleOneTimeRequest) (*scheduler.ScheduleResponse, error) {
	return s.ss.scheduleOneTime(ctx, s.pluginID, req)
}

func (s SchedulerHostFunctions) ScheduleRecurring(ctx context.Context, req *scheduler.ScheduleRecurringRequest) (*scheduler.ScheduleResponse, error) {
	return s.ss.scheduleRecurring(ctx, s.pluginID, req)
}

func (s SchedulerHostFunctions) CancelSchedule(ctx context.Context, req *scheduler.CancelRequest) (*scheduler.CancelResponse, error) {
	return s.ss.cancelSchedule(ctx, s.pluginID, req)
}

func (s SchedulerHostFunctions) TimeNow(ctx context.Context, req *scheduler.TimeNowRequest) (*scheduler.TimeNowResponse, error) {
	return s.ss.timeNow(ctx, req)
}

type schedulerService struct {
	// Map of schedule IDs to their callback info
	schedules  map[string]*ScheduledCallback
	manager    *managerImpl
	navidSched navidsched.Scheduler // Navidrome scheduler for recurring jobs
	mu         sync.Mutex
}

// newSchedulerService creates a new schedulerService instance
func newSchedulerService(manager *managerImpl) *schedulerService {
	return &schedulerService{
		schedules:  make(map[string]*ScheduledCallback),
		manager:    manager,
		navidSched: navidsched.GetInstance(),
	}
}

func (s *schedulerService) HostFunctions(pluginID string) SchedulerHostFunctions {
	return SchedulerHostFunctions{
		ss:       s,
		pluginID: pluginID,
	}
}

// Safe accessor methods for tests

// hasSchedule safely checks if a schedule exists
func (s *schedulerService) hasSchedule(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.schedules[id]
	return exists
}

// scheduleCount safely returns the number of schedules
func (s *schedulerService) scheduleCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.schedules)
}

// getScheduleType safely returns the type of a schedule
func (s *schedulerService) getScheduleType(id string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cb, exists := s.schedules[id]; exists {
		return cb.Type
	}
	return ""
}

// scheduleJob is a helper function that handles the common logic for scheduling jobs
func (s *schedulerService) scheduleJob(pluginID string, scheduleId string, jobType string, payload []byte) (string, *ScheduledCallback, context.CancelFunc, error) {
	if s.manager == nil {
		return "", nil, nil, fmt.Errorf("scheduler service not properly initialized")
	}

	// Original scheduleId (what the plugin will see)
	originalScheduleId := scheduleId
	if originalScheduleId == "" {
		// Generate a random ID if one wasn't provided
		originalScheduleId, _ = gonanoid.New(10)
	}

	// Internal scheduleId (prefixed with plugin name to avoid conflicts)
	internalScheduleId := pluginID + ":" + originalScheduleId

	// Store any existing cancellation function to call after we've updated the map
	var cancelExisting context.CancelFunc

	// Check if there's an existing schedule with the same ID, we'll cancel it after updating the map
	if existingSchedule, ok := s.schedules[internalScheduleId]; ok {
		log.Debug("Replacing existing schedule with same ID", "plugin", pluginID, "scheduleID", originalScheduleId)

		// Store cancel information but don't call it yet
		if existingSchedule.Type == ScheduleTypeOneTime && existingSchedule.Cancel != nil {
			// We'll set the Cancel to nil to prevent the old job from removing the new one
			cancelExisting = existingSchedule.Cancel
			existingSchedule.Cancel = nil
		} else if existingSchedule.Type == ScheduleTypeRecurring {
			existingRecurringEntryID := existingSchedule.EntryID
			if existingRecurringEntryID != 0 {
				s.navidSched.Remove(existingRecurringEntryID)
			}
		}
	}

	// Create the callback object
	callback := &ScheduledCallback{
		ID:       originalScheduleId,
		PluginID: pluginID,
		Type:     jobType,
		Payload:  payload,
	}

	return internalScheduleId, callback, cancelExisting, nil
}

// scheduleOneTime registers a new one-time scheduled job
func (s *schedulerService) scheduleOneTime(_ context.Context, pluginID string, req *scheduler.ScheduleOneTimeRequest) (*scheduler.ScheduleResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	internalScheduleId, callback, cancelExisting, err := s.scheduleJob(pluginID, req.ScheduleId, ScheduleTypeOneTime, req.Payload)
	if err != nil {
		return nil, err
	}

	// Create a context with cancel for this one-time schedule
	scheduleCtx, cancel := context.WithCancel(context.Background())
	callback.Cancel = cancel

	// Store the callback info
	s.schedules[internalScheduleId] = callback

	// Now that the new job is in the map, we can safely cancel the old one
	if cancelExisting != nil {
		// Cancel in a goroutine to avoid deadlock since we're already holding the lock
		go cancelExisting()
	}

	log.Debug("One-time schedule registered", "plugin", pluginID, "scheduleID", callback.ID, "internalID", internalScheduleId)

	// Start the timer goroutine with the internal ID
	go s.runOneTimeSchedule(scheduleCtx, internalScheduleId, time.Duration(req.DelaySeconds)*time.Second)

	// Return the original ID to the plugin
	return &scheduler.ScheduleResponse{
		ScheduleId: callback.ID,
	}, nil
}

// scheduleRecurring registers a new recurring scheduled job
func (s *schedulerService) scheduleRecurring(_ context.Context, pluginID string, req *scheduler.ScheduleRecurringRequest) (*scheduler.ScheduleResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	internalScheduleId, callback, cancelExisting, err := s.scheduleJob(pluginID, req.ScheduleId, ScheduleTypeRecurring, req.Payload)
	if err != nil {
		return nil, err
	}

	// Schedule the job with the Navidrome scheduler
	entryID, err := s.navidSched.Add(req.CronExpression, func() {
		s.executeCallback(context.Background(), internalScheduleId, true)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to schedule recurring job: %w", err)
	}

	// Store the entry ID so we can cancel it later
	callback.EntryID = entryID

	// Store the callback info
	s.schedules[internalScheduleId] = callback

	// Now that the new job is in the map, we can safely cancel the old one
	if cancelExisting != nil {
		// Cancel in a goroutine to avoid deadlock since we're already holding the lock
		go cancelExisting()
	}

	log.Debug("Recurring schedule registered", "plugin", pluginID, "scheduleID", callback.ID, "internalID", internalScheduleId, "cron", req.CronExpression)

	// Return the original ID to the plugin
	return &scheduler.ScheduleResponse{
		ScheduleId: callback.ID,
	}, nil
}

// cancelSchedule cancels a scheduled job (either one-time or recurring)
func (s *schedulerService) cancelSchedule(_ context.Context, pluginID string, req *scheduler.CancelRequest) (*scheduler.CancelResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	internalScheduleId := pluginID + ":" + req.ScheduleId
	callback, exists := s.schedules[internalScheduleId]
	if !exists {
		return &scheduler.CancelResponse{
			Success: false,
			Error:   "schedule not found",
		}, nil
	}

	// Store the cancel functions to call after we've updated the schedule map
	var cancelFunc context.CancelFunc
	var recurringEntryID int

	// Store cancel information but don't call it yet
	if callback.Type == ScheduleTypeOneTime && callback.Cancel != nil {
		cancelFunc = callback.Cancel
		callback.Cancel = nil // Set to nil to prevent the cancel handler from removing the job
	} else if callback.Type == ScheduleTypeRecurring {
		recurringEntryID = callback.EntryID
	}

	// First remove from the map
	delete(s.schedules, internalScheduleId)

	// Now perform the cancellation safely
	if cancelFunc != nil {
		// Execute in a goroutine to avoid deadlock since we're already holding the lock
		go cancelFunc()
	}
	if recurringEntryID != 0 {
		s.navidSched.Remove(recurringEntryID)
	}

	log.Debug("Schedule canceled", "plugin", pluginID, "scheduleID", req.ScheduleId, "internalID", internalScheduleId, "type", callback.Type)

	return &scheduler.CancelResponse{
		Success: true,
	}, nil
}

// timeNow returns the current time in multiple formats
func (s *schedulerService) timeNow(_ context.Context, req *scheduler.TimeNowRequest) (*scheduler.TimeNowResponse, error) {
	now := time.Now()

	return &scheduler.TimeNowResponse{
		Rfc3339Nano:   now.Format(time.RFC3339Nano),
		UnixMilli:     now.UnixMilli(),
		LocalTimeZone: now.Location().String(),
	}, nil
}

// runOneTimeSchedule handles the one-time schedule execution and callback
func (s *schedulerService) runOneTimeSchedule(ctx context.Context, internalScheduleId string, delay time.Duration) {
	tmr := time.NewTimer(delay)
	defer tmr.Stop()

	select {
	case <-ctx.Done():
		// Schedule was cancelled via its context
		// We're no longer removing the schedule here because that's handled by the code that
		// cancelled the context
		log.Debug("One-time schedule context canceled", "internalID", internalScheduleId)
		return

	case <-tmr.C:
		// Timer fired, execute the callback
		s.executeCallback(ctx, internalScheduleId, false)
	}
}

// executeCallback calls the plugin's OnSchedulerCallback method
func (s *schedulerService) executeCallback(ctx context.Context, internalScheduleId string, isRecurring bool) {
	s.mu.Lock()
	callback := s.schedules[internalScheduleId]
	// Only remove one-time schedules from the map after execution
	if callback != nil && callback.Type == ScheduleTypeOneTime {
		delete(s.schedules, internalScheduleId)
	}
	s.mu.Unlock()

	if callback == nil {
		log.Error("Schedule not found for callback", "internalID", internalScheduleId)
		return
	}

	ctx = log.NewContext(ctx, "plugin", callback.PluginID, "scheduleID", callback.ID, "type", callback.Type)
	log.Debug("Executing schedule callback")
	start := time.Now()

	// Get the plugin
	p := s.manager.LoadPlugin(callback.PluginID, CapabilitySchedulerCallback)
	if p == nil {
		log.Error("Plugin not found for callback", "plugin", callback.PluginID)
		return
	}

	// Type-check the plugin
	plugin, ok := p.(*wasmSchedulerCallback)
	if !ok {
		log.Error("Plugin does not implement SchedulerCallback", "plugin", callback.PluginID)
		return
	}

	// Call the plugin's OnSchedulerCallback method
	log.Trace(ctx, "Executing schedule callback")
	err := plugin.OnSchedulerCallback(ctx, callback.ID, callback.Payload, isRecurring)
	if err != nil {
		log.Error("Error executing schedule callback", "elapsed", time.Since(start), err)
		return
	}
	log.Debug("Schedule callback executed", "elapsed", time.Since(start))
}
