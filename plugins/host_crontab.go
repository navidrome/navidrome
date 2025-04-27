package plugins

import (
	"context"
	"fmt"
	"sync"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/api"
	"github.com/navidrome/navidrome/plugins/host/crontab"
	"github.com/navidrome/navidrome/scheduler"
)

// CrontabCallback represents a registered cron job callback
type CrontabCallback struct {
	ID          string
	PluginID    string
	CallbackID  string
	Description string
	Schedule    string
	EntryID     int
}

type CrontabHostFunctions struct {
	cs         *crontabService
	pluginName string
}

// ScheduleJob implements the CrontabService interface
func (c CrontabHostFunctions) ScheduleJob(ctx context.Context, req *crontab.ScheduleJobRequest) (*crontab.ScheduleJobResponse, error) {
	return c.cs.scheduleJob(ctx, c.pluginName, req)
}

// CancelJob implements the CrontabService interface
func (c CrontabHostFunctions) CancelJob(ctx context.Context, req *crontab.CancelJobRequest) (*crontab.CancelJobResponse, error) {
	return c.cs.cancelJob(ctx, c.pluginName, req)
}

// crontabService implements the crontab.CrontabService interface
type crontabService struct {
	// Map of job IDs to their callback info
	jobs      map[string]*CrontabCallback
	manager   *Manager
	scheduler scheduler.Scheduler
	mu        sync.Mutex
}

// newCrontabService creates a new crontabService instance
func newCrontabService(manager *Manager) *crontabService {
	return &crontabService{
		jobs:      make(map[string]*CrontabCallback),
		manager:   manager,
		scheduler: scheduler.GetInstance(),
	}
}

func (c *crontabService) HostFunctions(pluginName string) CrontabHostFunctions {
	return CrontabHostFunctions{
		cs:         c,
		pluginName: pluginName,
	}
}

// Safe accessor methods for tests

// hasJob safely checks if a job exists
func (c *crontabService) hasJob(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, exists := c.jobs[id]
	return exists
}

// jobCount safely returns the number of jobs
func (c *crontabService) jobCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.jobs)
}

// scheduleJob registers a new cron job
func (c *crontabService) scheduleJob(ctx context.Context, pluginName string, req *crontab.ScheduleJobRequest) (*crontab.ScheduleJobResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.manager == nil {
		return nil, fmt.Errorf("crontab service not properly initialized")
	}

	// Original jobId (what the plugin will see)
	originalJobId := req.JobId
	if originalJobId == "" {
		// Generate a random ID if one wasn't provided
		originalJobId, _ = gonanoid.New(10)
	}

	// Internal jobId (prefixed with plugin name to avoid conflicts)
	internalJobId := pluginName + ":" + originalJobId

	// Check if there's an existing job with the same ID, cancel it first
	if existingJob, ok := c.jobs[internalJobId]; ok {
		log.Debug("Replacing existing cron job with same ID", "plugin", pluginName, "jobID", originalJobId)
		c.scheduler.Remove(existingJob.EntryID)
	}

	callback := &CrontabCallback{
		ID:       originalJobId,
		PluginID: pluginName,
		Schedule: req.CronExpression,
	}

	// Schedule the job with the Navidrome scheduler
	entryID, err := c.scheduler.Add(req.CronExpression, func() {
		c.executeCallback(context.Background(), internalJobId)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to schedule job: %w", err)
	}

	// Store the entry ID so we can cancel it later
	callback.EntryID = entryID
	c.jobs[internalJobId] = callback

	log.Debug("Cron job scheduled", "plugin", pluginName, "jobID", originalJobId, "internalID", internalJobId, "cron", req.CronExpression)

	// Return the original ID to the plugin
	return &crontab.ScheduleJobResponse{
		JobId: originalJobId,
	}, nil
}

// cancelJob cancels a scheduled job
func (c *crontabService) cancelJob(ctx context.Context, pluginName string, req *crontab.CancelJobRequest) (*crontab.CancelJobResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	internalJobId := pluginName + ":" + req.JobId
	callback, exists := c.jobs[internalJobId]
	if !exists {
		return &crontab.CancelJobResponse{
			Success: false,
		}, fmt.Errorf("job not found")
	}

	// Cancel the job in the scheduler using the stored entry ID
	c.scheduler.Remove(callback.EntryID)
	delete(c.jobs, internalJobId)

	log.Debug("Cron job canceled", "plugin", pluginName, "jobID", req.JobId, "internalID", internalJobId)

	return &crontab.CancelJobResponse{
		Success: true,
	}, nil
}

// executeCallback calls the plugin's OnCrontabCallback method
func (c *crontabService) executeCallback(ctx context.Context, internalJobId string) {
	c.mu.Lock()
	callback, exists := c.jobs[internalJobId]
	c.mu.Unlock()

	if !exists {
		log.Error("Cron job not found", "internalID", internalJobId)
		return
	}

	log.Debug("Executing cron job callback", "plugin", callback.PluginID, "jobID", callback.ID)

	// Create a TimerCallbackRequest - we reuse this since it has the same structure we need
	req := &api.TimerCallbackRequest{
		TimerId: callback.ID, // We reuse the TimerId field for JobId
	}

	// Get the plugin
	p := c.manager.LoadPlugin(callback.PluginID, CapabilityTimerCallback)
	if p == nil {
		log.Error("Plugin not found for callback", "plugin", callback.PluginID)
		return
	}

	// Get instance
	inst, closeFn, err := p.GetInstance(ctx)
	if err != nil {
		log.Error("Error getting plugin instance for callback", "plugin", callback.PluginID, err)
		return
	}
	defer closeFn()

	// Type-check the plugin
	plugin, ok := inst.(api.TimerCallback)
	if !ok {
		log.Error("Plugin does not implement TimerCallback", "plugin", callback.PluginID)
		return
	}

	// Call the plugin's OnTimerCallback method (which we reuse for cron callbacks too)
	log.Trace(ctx, "Executing cron job callback", "plugin", callback.PluginID, "jobID", callback.ID)
	resp, err := plugin.OnTimerCallback(ctx, req)
	if err != nil {
		log.Error("Error executing cron job callback", "plugin", callback.PluginID, err)
		return
	}

	if resp.Error != "" {
		log.Error("Plugin reported error in cron job callback", "plugin", callback.PluginID, resp.Error)
	}
}
