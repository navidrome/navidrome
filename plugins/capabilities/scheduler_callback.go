package capabilities

// SchedulerCallback provides scheduled task handling.
// This capability allows plugins to receive callbacks when their scheduled tasks execute.
// Plugins that use the scheduler host service must implement this capability
// to handle task execution.
//
//nd:capability name=scheduler
type SchedulerCallback interface {
	// OnSchedulerCallback is called when a scheduled task fires.
	//nd:export name=nd_scheduler_callback
	OnSchedulerCallback(SchedulerCallbackInput) (SchedulerCallbackOutput, error)
}

// SchedulerCallbackInput is the input provided when a scheduled task fires.
type SchedulerCallbackInput struct {
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

// SchedulerCallbackOutput is the output from the scheduler callback.
type SchedulerCallbackOutput struct {
	// Error is the error message if the callback failed to process the scheduled task.
	// Empty or null indicates success. The error is logged but does not
	// affect the scheduling system.
	Error *string `json:"error,omitempty"`
}
