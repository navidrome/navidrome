package plugins

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/plugins/capabilities"
	"github.com/navidrome/navidrome/plugins/host"
	"golang.org/x/time/rate"
)

const (
	defaultConcurrency int32 = 1
	defaultBackoffMs   int64 = 1000
	defaultRetentionMs int64 = 3_600_000   // 1 hour
	minRetentionMs     int64 = 60_000      // 1 minute
	maxRetentionMs     int64 = 604_800_000 // 1 week
	maxQueueNameLength       = 128
	maxPayloadSize           = 1 * 1024 * 1024 // 1MB
	maxBackoffMs             = 3_600_000       // 1 hour
	cleanupInterval          = 5 * time.Minute
	pollInterval             = 5 * time.Second
	shutdownTimeout          = 10 * time.Second
)

const (
	taskStatusPending   = "pending"
	taskStatusRunning   = "running"
	taskStatusCompleted = "completed"
	taskStatusFailed    = "failed"
	taskStatusCancelled = "cancelled"
)

// CapabilityTaskWorker indicates the plugin can receive task execution callbacks.
// Detected when the plugin exports the task worker callback function.
const CapabilityTaskWorker Capability = "TaskWorker"

const FuncTaskWorkerCallback = "nd_task_execute"

func init() {
	registerCapability(CapabilityTaskWorker, FuncTaskWorkerCallback)
}

// queueState holds in-memory state for a single task queue.
type queueState struct {
	config  host.QueueConfig
	signal  chan struct{}
	limiter *rate.Limiter // rate limiter for delay enforcement between dispatches
}

// taskQueueServiceImpl implements host.TaskQueueService with SQLite persistence
// and background worker goroutines for task execution.
type taskQueueServiceImpl struct {
	pluginName     string
	manager        *Manager
	maxConcurrency int32
	db             *sql.DB
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	mu             sync.Mutex
	queues         map[string]*queueState

	// For testing: override how callbacks are invoked
	invokeCallbackFn func(ctx context.Context, queueName, taskID string, payload []byte, attempt int32) error
}

// newTaskQueueService creates a new taskQueueServiceImpl with its own SQLite database.
func newTaskQueueService(pluginName string, manager *Manager, maxConcurrency int32) (*taskQueueServiceImpl, error) {
	// Create plugin data directory
	dataDir := filepath.Join(conf.Server.DataFolder, "plugins", pluginName)
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("creating plugin data directory: %w", err)
	}

	// Open SQLite database
	dbPath := filepath.Join(dataDir, "taskqueue.db")
	db, err := sql.Open("sqlite3", dbPath+"?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=off")
	if err != nil {
		return nil, fmt.Errorf("opening taskqueue database: %w", err)
	}

	db.SetMaxOpenConns(3)
	db.SetMaxIdleConns(1)

	// Create schema
	if err := createTaskQueueSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating taskqueue schema: %w", err)
	}

	ctx, cancel := context.WithCancel(manager.ctx)

	s := &taskQueueServiceImpl{
		pluginName:     pluginName,
		manager:        manager,
		maxConcurrency: maxConcurrency,
		db:             db,
		ctx:            ctx,
		cancel:         cancel,
		queues:         make(map[string]*queueState),
	}
	s.invokeCallbackFn = s.defaultInvokeCallback

	// Start cleanup goroutine
	s.wg.Add(1)
	go s.cleanupLoop()

	log.Debug("Initialized plugin taskqueue", "plugin", pluginName, "path", dbPath, "maxConcurrency", maxConcurrency)
	return s, nil
}

func createTaskQueueSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS queues (
			name TEXT PRIMARY KEY,
			concurrency INTEGER NOT NULL DEFAULT 1,
			max_retries INTEGER NOT NULL DEFAULT 0,
			backoff_ms INTEGER NOT NULL DEFAULT 1000,
			delay_ms INTEGER NOT NULL DEFAULT 0,
			retention_ms INTEGER NOT NULL DEFAULT 3600000
		);

		CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			queue_name TEXT NOT NULL REFERENCES queues(name),
			payload BLOB NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			attempt INTEGER NOT NULL DEFAULT 0,
			max_retries INTEGER NOT NULL,
			next_run_at INTEGER NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_tasks_dequeue ON tasks(queue_name, status, next_run_at);
	`)
	return err
}

// CreateQueue creates a named task queue with the given configuration.
func (s *taskQueueServiceImpl) CreateQueue(ctx context.Context, name string, config host.QueueConfig) error {
	// Validate queue name
	if len(name) == 0 {
		return fmt.Errorf("queue name cannot be empty")
	}
	if len(name) > maxQueueNameLength {
		return fmt.Errorf("queue name exceeds maximum length of %d bytes", maxQueueNameLength)
	}

	// Apply defaults
	if config.Concurrency <= 0 {
		config.Concurrency = defaultConcurrency
	}
	if config.BackoffMs <= 0 {
		config.BackoffMs = defaultBackoffMs
	}
	if config.RetentionMs <= 0 {
		config.RetentionMs = defaultRetentionMs
	}

	// Clamp retention
	if config.RetentionMs < minRetentionMs {
		log.Warn(ctx, "TaskQueue retention clamped to minimum", "plugin", s.pluginName, "queue", name,
			"requested", config.RetentionMs, "min", minRetentionMs)
		config.RetentionMs = minRetentionMs
	}
	if config.RetentionMs > maxRetentionMs {
		log.Warn(ctx, "TaskQueue retention clamped to maximum", "plugin", s.pluginName, "queue", name,
			"requested", config.RetentionMs, "max", maxRetentionMs)
		config.RetentionMs = maxRetentionMs
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Clamp concurrency based on maxConcurrency minus already-allocated concurrency
	var allocated int32
	for _, qs := range s.queues {
		allocated += qs.config.Concurrency
	}
	available := s.maxConcurrency - allocated
	if available <= 0 {
		available = 1 // Always allow at least 1
	}
	if config.Concurrency > available {
		log.Warn(ctx, "TaskQueue concurrency clamped", "plugin", s.pluginName, "queue", name,
			"requested", config.Concurrency, "available", available, "maxConcurrency", s.maxConcurrency)
		config.Concurrency = available
	}

	// Check queue name doesn't already exist
	if _, exists := s.queues[name]; exists {
		return fmt.Errorf("queue %q already exists", name)
	}

	// Upsert into queues table (idempotent across restarts)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO queues (name, concurrency, max_retries, backoff_ms, delay_ms, retention_ms)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			concurrency = excluded.concurrency,
			max_retries = excluded.max_retries,
			backoff_ms = excluded.backoff_ms,
			delay_ms = excluded.delay_ms,
			retention_ms = excluded.retention_ms
	`, name, config.Concurrency, config.MaxRetries, config.BackoffMs, config.DelayMs, config.RetentionMs)
	if err != nil {
		return fmt.Errorf("creating queue: %w", err)
	}

	// Reset stale running tasks from previous crash
	now := time.Now().UnixMilli()
	_, err = s.db.ExecContext(ctx, `
		UPDATE tasks SET status = ?, updated_at = ? WHERE queue_name = ? AND status = ?
	`, taskStatusPending, now, name, taskStatusRunning)
	if err != nil {
		return fmt.Errorf("resetting stale tasks: %w", err)
	}

	// Store queue state
	qs := &queueState{
		config: config,
		signal: make(chan struct{}, 1),
	}
	if config.DelayMs > 0 {
		// Rate limit dispatches to enforce delay between tasks.
		// Burst of 1 allows one immediate dispatch, then enforces the delay interval.
		qs.limiter = rate.NewLimiter(rate.Every(time.Duration(config.DelayMs)*time.Millisecond), 1)
	}
	s.queues[name] = qs

	// Start worker goroutines
	for i := int32(0); i < config.Concurrency; i++ {
		s.wg.Add(1)
		go s.worker(name, qs)
	}

	log.Debug(ctx, "Created task queue", "plugin", s.pluginName, "queue", name,
		"concurrency", config.Concurrency, "maxRetries", config.MaxRetries,
		"backoffMs", config.BackoffMs, "delayMs", config.DelayMs, "retentionMs", config.RetentionMs)
	return nil
}

// Enqueue adds a task to the named queue and returns the task ID.
func (s *taskQueueServiceImpl) Enqueue(ctx context.Context, queueName string, payload []byte) (string, error) {
	s.mu.Lock()
	qs, exists := s.queues[queueName]
	s.mu.Unlock()

	if !exists {
		return "", fmt.Errorf("queue %q does not exist", queueName)
	}

	if len(payload) > maxPayloadSize {
		return "", fmt.Errorf("payload size %d exceeds maximum of %d bytes", len(payload), maxPayloadSize)
	}

	taskID := id.NewRandom()
	now := time.Now().UnixMilli()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tasks (id, queue_name, payload, status, attempt, max_retries, next_run_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, ?, ?, ?, ?)
	`, taskID, queueName, payload, taskStatusPending, qs.config.MaxRetries, now, now, now)
	if err != nil {
		return "", fmt.Errorf("enqueuing task: %w", err)
	}

	// Signal workers (non-blocking)
	select {
	case qs.signal <- struct{}{}:
	default:
	}

	log.Trace(ctx, "Enqueued task", "plugin", s.pluginName, "queue", queueName, "taskID", taskID)
	return taskID, nil
}

// GetTaskStatus returns the status of a task.
func (s *taskQueueServiceImpl) GetTaskStatus(ctx context.Context, taskID string) (string, error) {
	var status string
	err := s.db.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = ?`, taskID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("task %q not found", taskID)
	}
	if err != nil {
		return "", fmt.Errorf("getting task status: %w", err)
	}
	return status, nil
}

// CancelTask cancels a pending task.
func (s *taskQueueServiceImpl) CancelTask(ctx context.Context, taskID string) error {
	now := time.Now().UnixMilli()
	result, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET status = ?, updated_at = ? WHERE id = ? AND status = ?
	`, taskStatusCancelled, now, taskID, taskStatusPending)
	if err != nil {
		return fmt.Errorf("cancelling task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking cancel result: %w", err)
	}

	if rowsAffected == 0 {
		// Check if task exists at all
		var status string
		err := s.db.QueryRowContext(ctx, `SELECT status FROM tasks WHERE id = ?`, taskID).Scan(&status)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("task %q not found", taskID)
		}
		if err != nil {
			return fmt.Errorf("checking task existence: %w", err)
		}
		return fmt.Errorf("task %q cannot be cancelled (status: %s)", taskID, status)
	}

	log.Trace(ctx, "Cancelled task", "plugin", s.pluginName, "taskID", taskID)
	return nil
}

// worker is the main loop for a single worker goroutine.
func (s *taskQueueServiceImpl) worker(queueName string, qs *queueState) {
	defer s.wg.Done()

	// Process any existing pending tasks immediately on startup
	s.drainQueue(queueName, qs)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-qs.signal:
			s.drainQueue(queueName, qs)
		case <-ticker.C:
			s.drainQueue(queueName, qs)
		}
	}
}

// drainQueue processes tasks until the queue is empty.
func (s *taskQueueServiceImpl) drainQueue(queueName string, qs *queueState) {
	for {
		if s.ctx.Err() != nil {
			return
		}
		if !s.processTask(queueName, qs) {
			return
		}
	}
}

// processTask dequeues and processes a single task. Returns true if a task was processed.
func (s *taskQueueServiceImpl) processTask(queueName string, qs *queueState) bool {
	now := time.Now().UnixMilli()

	// Atomically dequeue a task
	var taskID string
	var payload []byte
	var attempt int32
	var maxRetries int32
	err := s.db.QueryRowContext(s.ctx, `
		UPDATE tasks SET status = ?, attempt = attempt + 1, updated_at = ?
		WHERE id = (
			SELECT id FROM tasks
			WHERE queue_name = ? AND status = ? AND next_run_at <= ?
			ORDER BY next_run_at, created_at LIMIT 1
		)
		RETURNING id, payload, attempt, max_retries
	`, taskStatusRunning, now, queueName, taskStatusPending, now).Scan(&taskID, &payload, &attempt, &maxRetries)
	if errors.Is(err, sql.ErrNoRows) {
		return false
	}
	if err != nil {
		log.Error(s.ctx, "Failed to dequeue task", "plugin", s.pluginName, "queue", queueName, err)
		return false
	}

	// Enforce delay between task dispatches using a rate limiter.
	// This is done after dequeue so that empty polls don't consume rate tokens.
	if qs.limiter != nil {
		if err := qs.limiter.Wait(s.ctx); err != nil {
			// Context cancelled during wait — revert task to pending for recovery
			s.revertTaskToPending(taskID)
			return false
		}
	}

	// Invoke callback
	log.Debug(s.ctx, "Executing task", "plugin", s.pluginName, "queue", queueName, "taskID", taskID, "attempt", attempt)
	callbackErr := s.invokeCallbackFn(s.ctx, queueName, taskID, payload, attempt)

	// If context was cancelled (shutdown), revert task to pending for recovery
	if s.ctx.Err() != nil {
		s.revertTaskToPending(taskID)
		return false
	}

	now = time.Now().UnixMilli()
	if callbackErr == nil {
		// Success: mark as completed
		_, err = s.db.ExecContext(s.ctx, `UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?`, taskStatusCompleted, now, taskID)
		if err != nil {
			log.Error(s.ctx, "Failed to mark task as completed", "plugin", s.pluginName, "taskID", taskID, err)
		}
		log.Debug(s.ctx, "Task completed", "plugin", s.pluginName, "queue", queueName, "taskID", taskID)
	} else {
		// Failure: retry or mark as failed
		log.Warn(s.ctx, "Task execution failed", "plugin", s.pluginName, "queue", queueName,
			"taskID", taskID, "attempt", attempt, "maxRetries", maxRetries, "err", callbackErr)

		if attempt <= maxRetries {
			// Retry with exponential backoff: backoffMs * 2^(attempt-1)
			backoff := qs.config.BackoffMs * int64(math.Pow(2, float64(attempt-1)))
			if backoff < 0 || backoff > maxBackoffMs {
				backoff = maxBackoffMs
			}
			nextRunAt := now + backoff
			_, err = s.db.ExecContext(s.ctx, `
				UPDATE tasks SET status = ?, next_run_at = ?, updated_at = ? WHERE id = ?
			`, taskStatusPending, nextRunAt, now, taskID)
			if err != nil {
				log.Error(s.ctx, "Failed to reschedule task for retry", "plugin", s.pluginName, "taskID", taskID, err)
			}

			// Schedule a delayed signal so the worker picks up the retried task
			// after the backoff period, rather than waiting for the next poll.
			backoffDuration := time.Duration(backoff) * time.Millisecond
			time.AfterFunc(backoffDuration, func() {
				select {
				case qs.signal <- struct{}{}:
				default:
				}
			})
		} else {
			// Exhausted retries: mark as failed
			_, err = s.db.ExecContext(s.ctx, `UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?`, taskStatusFailed, now, taskID)
			if err != nil {
				log.Error(s.ctx, "Failed to mark task as failed", "plugin", s.pluginName, "taskID", taskID, err)
			}
			log.Warn(s.ctx, "Task failed after all retries", "plugin", s.pluginName, "queue", queueName, "taskID", taskID)
		}
	}

	return true
}

// revertTaskToPending puts a running task back to pending status and decrements the attempt
// counter (used during shutdown to ensure the interrupted attempt doesn't count).
func (s *taskQueueServiceImpl) revertTaskToPending(taskID string) {
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(`UPDATE tasks SET status = ?, attempt = MAX(attempt - 1, 0), updated_at = ? WHERE id = ? AND status = ?`, taskStatusPending, now, taskID, taskStatusRunning)
	if err != nil {
		log.Error("Failed to revert task to pending", "plugin", s.pluginName, "taskID", taskID, err)
	}
}

// defaultInvokeCallback calls the plugin's nd_task_execute function.
func (s *taskQueueServiceImpl) defaultInvokeCallback(ctx context.Context, queueName, taskID string, payload []byte, attempt int32) error {
	s.manager.mu.RLock()
	p, ok := s.manager.plugins[s.pluginName]
	s.manager.mu.RUnlock()

	if !ok {
		return fmt.Errorf("plugin %s not loaded", s.pluginName)
	}

	input := capabilities.TaskExecuteRequest{
		QueueName: queueName,
		TaskID:    taskID,
		Payload:   payload,
		Attempt:   attempt,
	}

	result, err := callPluginFunction[capabilities.TaskExecuteRequest, capabilities.TaskExecuteResponse](ctx, p, FuncTaskWorkerCallback, input)
	if err != nil {
		return err
	}
	if result.Error != "" {
		return fmt.Errorf("%s", result.Error)
	}
	return nil
}

// cleanupLoop periodically removes terminal tasks past their retention period.
func (s *taskQueueServiceImpl) cleanupLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.runCleanup()
		}
	}
}

// runCleanup deletes terminal tasks past their retention period.
func (s *taskQueueServiceImpl) runCleanup() {
	s.mu.Lock()
	queues := make(map[string]*queueState, len(s.queues))
	for k, v := range s.queues {
		queues[k] = v
	}
	s.mu.Unlock()

	now := time.Now().UnixMilli()
	for name, qs := range queues {
		result, err := s.db.ExecContext(s.ctx, `
			DELETE FROM tasks WHERE queue_name = ? AND status IN (?, ?, ?) AND updated_at + ? < ?
		`, name, taskStatusCompleted, taskStatusFailed, taskStatusCancelled, qs.config.RetentionMs, now)
		if err != nil {
			log.Error(s.ctx, "Failed to cleanup tasks", "plugin", s.pluginName, "queue", name, err)
			continue
		}
		if deleted, _ := result.RowsAffected(); deleted > 0 {
			log.Debug(s.ctx, "Cleaned up terminal tasks", "plugin", s.pluginName, "queue", name, "deleted", deleted)
		}
	}
}

// Close shuts down the task queue service, stopping all workers and closing the database.
func (s *taskQueueServiceImpl) Close() error {
	// Cancel context to signal all goroutines
	s.cancel()

	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(shutdownTimeout):
		log.Warn("TaskQueue shutdown timed out", "plugin", s.pluginName)
	}

	// Mark running tasks as pending for recovery on next startup
	if s.db != nil {
		now := time.Now().UnixMilli()
		_, err := s.db.Exec(`UPDATE tasks SET status = ?, updated_at = ? WHERE status = ?`, taskStatusPending, now, taskStatusRunning)
		if err != nil {
			log.Error("Failed to reset running tasks on shutdown", "plugin", s.pluginName, err)
		}

		log.Debug("Closing plugin taskqueue", "plugin", s.pluginName)
		return s.db.Close()
	}
	return nil
}

// Compile-time verification
var _ host.TaskQueueService = (*taskQueueServiceImpl)(nil)
var _ io.Closer = (*taskQueueServiceImpl)(nil)
