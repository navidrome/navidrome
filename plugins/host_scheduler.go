package plugins

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/plugins/capabilities"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/scheduler"
)

// CapabilityScheduler indicates the plugin can receive scheduled event callbacks.
// Detected when the plugin exports the scheduler callback function.
const CapabilityScheduler Capability = "Scheduler"

const FuncSchedulerCallback = "nd_scheduler_callback"

func init() {
	registerCapability(
		CapabilityScheduler,
		FuncSchedulerCallback,
	)
}

// timeAfterFunc is a variable for time.AfterFunc, allowing tests to override it.
var timeAfterFunc = time.AfterFunc

// scheduleEntry stores metadata about a scheduled task.
type scheduleEntry struct {
	pluginName  string
	payload     string
	isRecurring bool
	entryID     int         // Internal scheduler entry ID (for recurring tasks)
	timer       *time.Timer // Timer for one-time tasks (nil for recurring)
}

// schedulerServiceImpl implements host.SchedulerService.
// It provides plugins with scheduling capabilities and invokes callbacks when schedules fire.
type schedulerServiceImpl struct {
	pluginName string
	manager    *Manager
	scheduler  scheduler.Scheduler

	mu        sync.Mutex
	schedules map[string]*scheduleEntry
}

// newSchedulerService creates a new SchedulerService for a plugin.
func newSchedulerService(pluginName string, manager *Manager, sched scheduler.Scheduler) *schedulerServiceImpl {
	return &schedulerServiceImpl{
		pluginName: pluginName,
		manager:    manager,
		scheduler:  sched,
		schedules:  make(map[string]*scheduleEntry),
	}
}

func (s *schedulerServiceImpl) ScheduleOneTime(ctx context.Context, delaySeconds int32, payload string, scheduleID string) (string, error) {
	if scheduleID == "" {
		scheduleID = id.NewRandom()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.schedules[scheduleID]; exists {
		return "", fmt.Errorf("schedule ID %q already exists", scheduleID)
	}

	capturedID := scheduleID
	timer := timeAfterFunc(time.Duration(delaySeconds)*time.Second, func() {
		s.invokeCallback(context.Background(), capturedID)
		// Clean up the entry after firing
		s.mu.Lock()
		delete(s.schedules, capturedID)
		s.mu.Unlock()
	})

	s.schedules[scheduleID] = &scheduleEntry{
		pluginName:  s.pluginName,
		payload:     payload,
		isRecurring: false,
		timer:       timer,
	}

	log.Debug(ctx, "Scheduled one-time task", "plugin", s.pluginName, "scheduleID", scheduleID, "delaySeconds", delaySeconds)
	return scheduleID, nil
}

func (s *schedulerServiceImpl) ScheduleRecurring(ctx context.Context, cronExpression string, payload string, scheduleID string) (string, error) {
	if scheduleID == "" {
		scheduleID = id.NewRandom()
	}

	capturedID := scheduleID
	callback := func() {
		s.invokeCallback(context.Background(), capturedID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.schedules[scheduleID]; exists {
		return "", fmt.Errorf("schedule ID %q already exists", scheduleID)
	}

	entryID, err := s.scheduler.Add(cronExpression, callback)
	if err != nil {
		return "", fmt.Errorf("failed to schedule task: %w", err)
	}

	s.schedules[scheduleID] = &scheduleEntry{
		pluginName:  s.pluginName,
		payload:     payload,
		isRecurring: true,
		entryID:     entryID,
	}

	log.Debug(ctx, "Scheduled recurring task", "plugin", s.pluginName, "scheduleID", scheduleID, "cron", cronExpression)
	return scheduleID, nil
}

func (s *schedulerServiceImpl) CancelSchedule(ctx context.Context, scheduleID string) error {
	s.mu.Lock()
	entry, exists := s.schedules[scheduleID]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("schedule ID %q not found", scheduleID)
	}
	delete(s.schedules, scheduleID)
	s.mu.Unlock()

	if entry.timer != nil {
		entry.timer.Stop()
	} else {
		s.scheduler.Remove(entry.entryID)
	}
	log.Debug(ctx, "Cancelled schedule", "plugin", s.pluginName, "scheduleID", scheduleID)
	return nil
}

// Close cancels all schedules for this plugin.
// This is called when the plugin is unloaded.
func (s *schedulerServiceImpl) Close() error {
	s.mu.Lock()
	schedules := maps.Clone(s.schedules)
	s.schedules = make(map[string]*scheduleEntry)
	s.mu.Unlock()

	for scheduleID, entry := range schedules {
		if entry.timer != nil {
			entry.timer.Stop()
		} else {
			s.scheduler.Remove(entry.entryID)
		}
		log.Debug("Cancelled schedule on plugin unload", "plugin", s.pluginName, "scheduleID", scheduleID)
	}
	return nil
}

// invokeCallback calls the plugin's nd_scheduler_callback function.
func (s *schedulerServiceImpl) invokeCallback(ctx context.Context, scheduleID string) {
	log.Debug(ctx, "Scheduler callback invoked", "plugin", s.pluginName, "scheduleID", scheduleID)

	s.mu.Lock()
	entry, exists := s.schedules[scheduleID]
	if !exists {
		s.mu.Unlock()
		log.Warn(ctx, "Schedule entry not found during callback", "plugin", s.pluginName, "scheduleID", scheduleID)
		return
	}
	payload := entry.payload
	isRecurring := entry.isRecurring
	s.mu.Unlock()

	// Get the plugin instance from the manager
	s.manager.mu.RLock()
	instance, ok := s.manager.plugins[s.pluginName]
	s.manager.mu.RUnlock()

	if !ok {
		log.Warn(ctx, "Plugin not loaded when scheduler callback fired", "plugin", s.pluginName, "scheduleID", scheduleID)
		return
	}

	// Check if plugin has the scheduler capability
	if !hasCapability(instance.capabilities, CapabilityScheduler) {
		log.Warn(ctx, "Plugin does not have scheduler capability", "plugin", s.pluginName, "scheduleID", scheduleID)
		return
	}

	// Prepare callback input
	input := capabilities.SchedulerCallbackRequest{
		ScheduleID:  scheduleID,
		Payload:     payload,
		IsRecurring: isRecurring,
	}

	start := time.Now()
	err := callPluginFunctionNoOutput(ctx, instance, FuncSchedulerCallback, input)
	if err != nil {
		log.Error(ctx, "Scheduler callback failed", "plugin", s.pluginName, "scheduleID", scheduleID, "duration", time.Since(start), err)
		return
	}

	log.Debug(ctx, "Scheduler callback completed", "plugin", s.pluginName, "scheduleID", scheduleID, "duration", time.Since(start))
}

// Verify interface implementation
var _ host.SchedulerService = (*schedulerServiceImpl)(nil)
