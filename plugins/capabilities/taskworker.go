package capabilities

// TaskWorker provides task execution handling.
// This capability allows plugins to receive callbacks when their queued tasks
// are ready to execute. Plugins that use the taskqueue host service must
// implement this capability.
//
//nd:capability name=taskworker
type TaskWorker interface {
	// OnTaskExecute is called when a queued task is ready to run.
	// The returned string is a status/result message stored in the tasks table.
	// Return an error to trigger retry (if retries are configured).
	//nd:export name=nd_task_execute
	OnTaskExecute(TaskExecuteRequest) (string, error)
}

// TaskExecuteRequest is the request provided when a task is ready to execute.
type TaskExecuteRequest struct {
	// QueueName is the name of the queue this task belongs to.
	QueueName string `json:"queueName"`
	// TaskID is the unique identifier for this task.
	TaskID string `json:"taskId"`
	// Payload is the opaque data provided when the task was enqueued.
	Payload []byte `json:"payload"`
	// Attempt is the current attempt number (1-based: first attempt = 1).
	Attempt int32 `json:"attempt"`
}
