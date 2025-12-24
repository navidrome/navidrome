package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/scheduler"
)

const FuncSchedulerCallback = "nd_scheduler_callback"

// scheduleEntry stores metadata about a scheduled task.
type scheduleEntry struct {
	pluginName  string
	payload     string
	isRecurring bool
	entryID     int // Internal scheduler entry ID
}

// callbackRecord stores information about a callback that was invoked (for testing).
type callbackRecord struct {
	ScheduleID  string
	Payload     string
	IsRecurring bool
	Count       int
}

// schedulerServiceImpl implements host.SchedulerService.
// It provides plugins with scheduling capabilities and invokes callbacks when schedules fire.
type schedulerServiceImpl struct {
	pluginName string
	manager    *Manager
	scheduler  scheduler.Scheduler

	mu        sync.Mutex
	schedules map[string]*scheduleEntry

	// Callback tracking (for testing) - tracks callbacks invoked on host side
	callbackMu      sync.Mutex
	callbackRecords map[string]*callbackRecord
	callbackCount   int
}

// newSchedulerService creates a new SchedulerService for a plugin.
func newSchedulerService(pluginName string, manager *Manager, sched scheduler.Scheduler) host.SchedulerService {
	return &schedulerServiceImpl{
		pluginName:      pluginName,
		manager:         manager,
		scheduler:       sched,
		schedules:       make(map[string]*scheduleEntry),
		callbackRecords: make(map[string]*callbackRecord),
	}
}

func (s *schedulerServiceImpl) ScheduleOneTime(ctx context.Context, delaySeconds int32, payload string, scheduleID string) (string, error) {
	if scheduleID == "" {
		scheduleID = uuid.New().String()
	}

	s.mu.Lock()
	if _, exists := s.schedules[scheduleID]; exists {
		s.mu.Unlock()
		return "", fmt.Errorf("schedule ID %q already exists", scheduleID)
	}

	entry := &scheduleEntry{
		pluginName:  s.pluginName,
		payload:     payload,
		isRecurring: false,
	}
	s.schedules[scheduleID] = entry
	s.mu.Unlock()

	// Use @every syntax for one-time delay
	cronExpr := fmt.Sprintf("@every %ds", delaySeconds)

	// Create callback that will fire once and then cancel itself
	schedID := scheduleID // capture for closure
	callback := func() {
		s.invokeCallback(schedID)
		// One-time schedules cancel themselves after firing
		_ = s.CancelSchedule(context.Background(), schedID)
	}

	entryID, err := s.scheduler.Add(cronExpr, callback)
	if err != nil {
		s.mu.Lock()
		delete(s.schedules, scheduleID)
		s.mu.Unlock()
		return "", fmt.Errorf("failed to schedule one-time task: %w", err)
	}

	s.mu.Lock()
	entry.entryID = entryID
	s.mu.Unlock()

	log.Debug(ctx, "Scheduled one-time task", "plugin", s.pluginName, "scheduleID", scheduleID, "delay", delaySeconds)
	return scheduleID, nil
}

func (s *schedulerServiceImpl) ScheduleRecurring(ctx context.Context, cronExpression string, payload string, scheduleID string) (string, error) {
	if scheduleID == "" {
		scheduleID = uuid.New().String()
	}

	s.mu.Lock()
	if _, exists := s.schedules[scheduleID]; exists {
		s.mu.Unlock()
		return "", fmt.Errorf("schedule ID %q already exists", scheduleID)
	}

	entry := &scheduleEntry{
		pluginName:  s.pluginName,
		payload:     payload,
		isRecurring: true,
	}
	s.schedules[scheduleID] = entry
	s.mu.Unlock()

	schedID := scheduleID // capture for closure
	callback := func() {
		s.invokeCallback(schedID)
	}

	entryID, err := s.scheduler.Add(cronExpression, callback)
	if err != nil {
		s.mu.Lock()
		delete(s.schedules, scheduleID)
		s.mu.Unlock()
		return "", fmt.Errorf("failed to schedule recurring task: %w", err)
	}

	s.mu.Lock()
	entry.entryID = entryID
	s.mu.Unlock()

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
	entryID := entry.entryID
	s.mu.Unlock()

	s.scheduler.Remove(entryID)
	log.Debug(ctx, "Cancelled schedule", "plugin", s.pluginName, "scheduleID", scheduleID)
	return nil
}

// CancelAllForPlugin cancels all schedules for this plugin.
// This is called when the plugin is unloaded.
func (s *schedulerServiceImpl) CancelAllForPlugin() {
	s.mu.Lock()
	schedules := make(map[string]*scheduleEntry, len(s.schedules))
	for k, v := range s.schedules {
		schedules[k] = v
	}
	s.schedules = make(map[string]*scheduleEntry)
	s.mu.Unlock()

	for scheduleID, entry := range schedules {
		s.scheduler.Remove(entry.entryID)
		log.Debug(context.Background(), "Cancelled schedule on plugin unload", "plugin", s.pluginName, "scheduleID", scheduleID)
	}
}

// schedulerCallbackInput is the input format for the nd_scheduler_callback function.
type schedulerCallbackInput struct {
	ScheduleID  string `json:"schedule_id"`
	Payload     string `json:"payload"`
	IsRecurring bool   `json:"is_recurring"`
}

// schedulerCallbackOutput is the output format for the nd_scheduler_callback function.
type schedulerCallbackOutput struct {
	Error string `json:"error,omitempty"`
}

// invokeCallback calls the plugin's nd_scheduler_callback function.
func (s *schedulerServiceImpl) invokeCallback(scheduleID string) {
	ctx := context.Background()
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
	input := schedulerCallbackInput{
		ScheduleID:  scheduleID,
		Payload:     payload,
		IsRecurring: isRecurring,
	}

	start := time.Now()
	result, err := callPluginFunction[schedulerCallbackInput, schedulerCallbackOutput](ctx, instance, FuncSchedulerCallback, input)
	if err != nil {
		log.Error(ctx, "Scheduler callback failed", "plugin", s.pluginName, "scheduleID", scheduleID, "duration", time.Since(start), err)
		return
	}

	if result.Error != "" {
		log.Error(ctx, "Scheduler callback returned error", "plugin", s.pluginName, "scheduleID", scheduleID, "error", result.Error, "duration", time.Since(start))
		return
	}

	// Track callback invocation on host side (for testing)
	s.trackCallback(scheduleID, payload, isRecurring)

	log.Debug(ctx, "Scheduler callback completed", "plugin", s.pluginName, "scheduleID", scheduleID, "duration", time.Since(start))
}

// trackCallback records a callback invocation (for testing).
func (s *schedulerServiceImpl) trackCallback(scheduleID, payload string, isRecurring bool) {
	s.callbackMu.Lock()
	defer s.callbackMu.Unlock()

	s.callbackCount++
	if record, exists := s.callbackRecords[scheduleID]; exists {
		record.Count++
	} else {
		s.callbackRecords[scheduleID] = &callbackRecord{
			ScheduleID:  scheduleID,
			Payload:     payload,
			IsRecurring: isRecurring,
			Count:       1,
		}
	}
}

// GetCallbackCount returns the total number of callbacks invoked for this service.
// This is primarily used for testing.
func (s *schedulerServiceImpl) GetCallbackCount() int {
	s.callbackMu.Lock()
	defer s.callbackMu.Unlock()
	return s.callbackCount
}

// GetCallbackRecords returns the callback records for this service.
// This is primarily used for testing.
func (s *schedulerServiceImpl) GetCallbackRecords() map[string]*callbackRecord {
	s.callbackMu.Lock()
	defer s.callbackMu.Unlock()
	// Return a copy
	records := make(map[string]*callbackRecord, len(s.callbackRecords))
	for k, v := range s.callbackRecords {
		records[k] = &callbackRecord{
			ScheduleID:  v.ScheduleID,
			Payload:     v.Payload,
			IsRecurring: v.IsRecurring,
			Count:       v.Count,
		}
	}
	return records
}

// ResetCallbackRecords clears the callback tracking state.
// This is primarily used for testing.
func (s *schedulerServiceImpl) ResetCallbackRecords() {
	s.callbackMu.Lock()
	defer s.callbackMu.Unlock()
	s.callbackRecords = make(map[string]*callbackRecord)
	s.callbackCount = 0
}

// GetScheduleCount returns the number of active schedules for this service.
// This is primarily used for testing.
func (s *schedulerServiceImpl) GetScheduleCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.schedules)
}

// Verify interface implementation
var _ host.SchedulerService = (*schedulerServiceImpl)(nil)

// schedulerServiceRegistry keeps track of scheduler services per plugin for cleanup.
type schedulerServiceRegistry struct {
	mu       sync.RWMutex
	services map[string]*schedulerServiceImpl
}

var schedulerRegistry = &schedulerServiceRegistry{
	services: make(map[string]*schedulerServiceImpl),
}

// registerSchedulerService registers a scheduler service for a plugin.
func registerSchedulerService(pluginName string, service *schedulerServiceImpl) {
	schedulerRegistry.mu.Lock()
	defer schedulerRegistry.mu.Unlock()
	schedulerRegistry.services[pluginName] = service
}

// unregisterSchedulerService unregisters and cancels all schedules for a plugin.
func unregisterSchedulerService(pluginName string) {
	schedulerRegistry.mu.Lock()
	service, exists := schedulerRegistry.services[pluginName]
	if exists {
		delete(schedulerRegistry.services, pluginName)
	}
	schedulerRegistry.mu.Unlock()

	if exists && service != nil {
		service.CancelAllForPlugin()
	}
}

// getSchedulerService returns the scheduler service for a plugin.
func getSchedulerService(pluginName string) *schedulerServiceImpl {
	schedulerRegistry.mu.RLock()
	defer schedulerRegistry.mu.RUnlock()
	return schedulerRegistry.services[pluginName]
}

// CreateSchedulerHostFunctions creates scheduler host functions for a plugin.
// This should be called during plugin load if the plugin has the scheduler permission.
func CreateSchedulerHostFunctions(pluginName string, manager *Manager) []func() {
	sched := scheduler.GetInstance()
	service := newSchedulerService(pluginName, manager, sched).(*schedulerServiceImpl)
	registerSchedulerService(pluginName, service)

	// Return a cleanup function
	return []func(){
		func() { unregisterSchedulerService(pluginName) },
	}
}
