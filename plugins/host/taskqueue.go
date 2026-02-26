package host

import "context"

// QueueConfig holds configuration for a task queue.
type QueueConfig struct {
	// Concurrency is the max number of parallel workers. Default: 1.
	// Capped by the plugin's manifest maxConcurrency.
	Concurrency int32 `json:"concurrency"`

	// MaxRetries is the number of times to retry a failed task. Default: 0.
	MaxRetries int32 `json:"maxRetries"`

	// BackoffMs is the initial backoff between retries in milliseconds.
	// Doubles each retry (exponential: backoffMs * 2^(attempt-1)). Default: 1000.
	BackoffMs int64 `json:"backoffMs"`

	// DelayMs is the minimum delay between starting consecutive tasks
	// in milliseconds. Useful for rate limiting. Default: 0.
	DelayMs int64 `json:"delayMs"`

	// RetentionMs is how long completed/failed/cancelled tasks are kept
	// in milliseconds. Default: 3600000 (1h). Min: 60000 (1m). Max: 604800000 (1w).
	RetentionMs int64 `json:"retentionMs"`
}

// TaskQueueService provides persistent task queues for plugins.
//
// This service allows plugins to create named queues with configurable concurrency,
// retry policies, and rate limiting. Tasks are persisted to SQLite and survive
// server restarts. When a task is ready to execute, the host calls the plugin's
// nd_task_execute callback function.
//
//nd:hostservice name=TaskQueue permission=taskqueue
type TaskQueueService interface {
	// CreateQueue creates a named task queue with the given configuration.
	// Zero-value fields in config use sensible defaults.
	// If a queue with the same name already exists, returns an error.
	// On startup, this also recovers any stale "running" tasks from a previous crash.
	//nd:hostfunc
	CreateQueue(ctx context.Context, name string, config QueueConfig) error

	// Enqueue adds a task to the named queue. Returns the task ID.
	// payload is opaque bytes passed back to the plugin on execution.
	//nd:hostfunc
	Enqueue(ctx context.Context, queueName string, payload []byte) (string, error)

	// GetTaskStatus returns the status of a task: "pending", "running",
	// "completed", "failed", or "cancelled".
	//nd:hostfunc
	GetTaskStatus(ctx context.Context, taskID string) (string, error)

	// CancelTask cancels a pending task. Returns error if already
	// running, completed, or failed.
	//nd:hostfunc
	CancelTask(ctx context.Context, taskID string) error
}
