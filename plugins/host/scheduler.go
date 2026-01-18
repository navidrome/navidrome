package host

import "context"

// SchedulerService provides task scheduling capabilities for plugins.
//
// This service allows plugins to schedule both one-time and recurring tasks using
// cron expressions. All scheduled tasks can be cancelled using their schedule ID.
//
//nd:hostservice name=Scheduler permission=scheduler
type SchedulerService interface {
	// ScheduleOneTime schedules a one-time event to be triggered after the specified delay.
	// Plugins that use this function must also implement the SchedulerCallback capability
	//
	// Parameters:
	//   - delaySeconds: Number of seconds to wait before triggering the event
	//   - payload: Data to be passed to the scheduled event handler
	//   - scheduleID: Optional unique identifier for the scheduled job. If empty, one will be generated
	//
	// Returns the schedule ID that can be used to cancel the job, or an error if scheduling fails.
	//nd:hostfunc
	ScheduleOneTime(ctx context.Context, delaySeconds int32, payload string, scheduleID string) (newScheduleID string, err error)

	// ScheduleRecurring schedules a recurring event using a cron expression.
	// Plugins that use this function must also implement the SchedulerCallback capability
	//
	// Parameters:
	//   - cronExpression: Standard cron format expression (e.g., "0 0 * * *" for daily at midnight)
	//   - payload: Data to be passed to each scheduled event handler invocation
	//   - scheduleID: Optional unique identifier for the scheduled job. If empty, one will be generated
	//
	// Returns the schedule ID that can be used to cancel the job, or an error if scheduling fails.
	//nd:hostfunc
	ScheduleRecurring(ctx context.Context, cronExpression string, payload string, scheduleID string) (newScheduleID string, err error)

	// CancelSchedule cancels a scheduled job identified by its schedule ID.
	//
	// This works for both one-time and recurring schedules. Once cancelled, the job will not trigger
	// any future events.
	//
	// Returns an error if the schedule ID is not found or if cancellation fails.
	//nd:hostfunc
	CancelSchedule(ctx context.Context, scheduleID string) error
}
