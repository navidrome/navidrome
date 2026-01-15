package capabilities

// SchedulerCallback provides scheduled task handling.
// This capability allows plugins to receive callbacks when their scheduled tasks execute.
// Plugins that use the scheduler host service must implement this capability
// to handle task execution.
//
//nd:capability name=scheduler
type SchedulerCallback interface {
	// OnCallback is called when a scheduled task fires.
	// Errors are logged but do not affect the scheduling system.
	//nd:export name=nd_scheduler_callback
	OnCallback(SchedulerCallbackRequest) error
}

// SchedulerCallbackRequest is the request provided when a scheduled task fires.
type SchedulerCallbackRequest struct {
	// ScheduleID is the unique identifier for this scheduled task.
	// This is either the ID provided when scheduling, or an auto-generated UUID if none was specified.
	ScheduleID string `json:"scheduleId"`
	// Payload is the payload data that was provided when the task was scheduled.
	// Can be used to pass context or parameters to the callback handler.
	Payload string `json:"payload"`
	// IsRecurring is true if this is a recurring schedule (created via ScheduleRecurring),
	// false if it's a one-time schedule (created via ScheduleOneTime).
	IsRecurring bool `json:"isRecurring"`
}
