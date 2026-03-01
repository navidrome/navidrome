//go:build !windows

package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TaskQueueService", func() {
	var tmpDir string
	var service *taskQueueServiceImpl
	var ctx context.Context
	var manager *Manager

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		var err error
		tmpDir, err = os.MkdirTemp("", "taskqueue-test-*")
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(configtest.SetupConfig())
		conf.Server.DataFolder = tmpDir

		// Create a mock manager with context
		managerCtx, cancel := context.WithCancel(ctx)
		manager = &Manager{
			plugins: make(map[string]*plugin),
			ctx:     managerCtx,
		}
		DeferCleanup(cancel)

		service, err = newTaskQueueService("test_plugin", manager, 5)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if service != nil {
			service.Close()
		}
		os.RemoveAll(tmpDir)
	})

	Describe("CreateQueue", func() {
		It("creates a queue successfully", func() {
			err := service.CreateQueue(ctx, "my-queue", host.QueueConfig{
				Concurrency: 2,
				MaxRetries:  3,
				BackoffMs:   2000,
				RetentionMs: 7200000,
			})
			Expect(err).ToNot(HaveOccurred())

			service.mu.Lock()
			qs, exists := service.queues["my-queue"]
			service.mu.Unlock()
			Expect(exists).To(BeTrue())
			Expect(qs.config.Concurrency).To(Equal(int32(2)))
			Expect(qs.config.MaxRetries).To(Equal(int32(3)))
			Expect(qs.config.BackoffMs).To(Equal(int64(2000)))
			Expect(qs.config.RetentionMs).To(Equal(int64(7200000)))
		})

		It("returns error for duplicate queue name", func() {
			err := service.CreateQueue(ctx, "dup-queue", host.QueueConfig{})
			Expect(err).ToNot(HaveOccurred())

			err = service.CreateQueue(ctx, "dup-queue", host.QueueConfig{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))
		})
	})

	Describe("CreateQueue name validation", func() {
		It("rejects empty queue name", func() {
			err := service.CreateQueue(ctx, "", host.QueueConfig{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("queue name cannot be empty"))
		})

		It("rejects over-length queue name", func() {
			longName := strings.Repeat("a", maxQueueNameLength+1)
			err := service.CreateQueue(ctx, longName, host.QueueConfig{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exceeds maximum length"))
		})

		It("accepts queue name at maximum length", func() {
			service.invokeCallbackFn = func(_ context.Context, _, _ string, _ []byte, _ int32) (string, error) {
				return "", nil
			}
			exactName := strings.Repeat("a", maxQueueNameLength)
			err := service.CreateQueue(ctx, exactName, host.QueueConfig{})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("CreateQueue defaults", func() {
		It("applies defaults for zero-value config", func() {
			err := service.CreateQueue(ctx, "defaults-queue", host.QueueConfig{})
			Expect(err).ToNot(HaveOccurred())

			service.mu.Lock()
			qs := service.queues["defaults-queue"]
			service.mu.Unlock()
			Expect(qs.config.Concurrency).To(Equal(defaultConcurrency))
			Expect(qs.config.BackoffMs).To(Equal(defaultBackoffMs))
			Expect(qs.config.RetentionMs).To(Equal(defaultRetentionMs))
		})
	})

	Describe("CreateQueue defaults with negative values", func() {
		It("applies default RetentionMs for negative value", func() {
			service.invokeCallbackFn = func(_ context.Context, _, _ string, _ []byte, _ int32) (string, error) {
				return "", nil
			}
			err := service.CreateQueue(ctx, "neg-retention", host.QueueConfig{
				RetentionMs: -500,
			})
			Expect(err).ToNot(HaveOccurred())

			service.mu.Lock()
			qs := service.queues["neg-retention"]
			service.mu.Unlock()
			Expect(qs.config.RetentionMs).To(Equal(defaultRetentionMs))
		})
	})

	Describe("CreateQueue clamping", func() {
		It("clamps concurrency exceeding maxConcurrency", func() {
			// maxConcurrency is 5; request 10
			err := service.CreateQueue(ctx, "clamped-queue", host.QueueConfig{
				Concurrency: 10,
			})
			Expect(err).ToNot(HaveOccurred())

			service.mu.Lock()
			qs := service.queues["clamped-queue"]
			service.mu.Unlock()
			Expect(qs.config.Concurrency).To(Equal(int32(5)))
		})

		It("returns error when concurrency budget is exhausted", func() {
			// maxConcurrency is 5; create a queue that uses all 5
			err := service.CreateQueue(ctx, "full-budget", host.QueueConfig{
				Concurrency: 5,
			})
			Expect(err).ToNot(HaveOccurred())

			// Next queue should fail — no budget remaining
			err = service.CreateQueue(ctx, "over-budget", host.QueueConfig{
				Concurrency: 1,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("concurrency budget exhausted"))
		})

		It("clamps retention below minimum", func() {
			err := service.CreateQueue(ctx, "low-retention", host.QueueConfig{
				RetentionMs: 100, // below minRetentionMs
			})
			Expect(err).ToNot(HaveOccurred())

			service.mu.Lock()
			qs := service.queues["low-retention"]
			service.mu.Unlock()
			Expect(qs.config.RetentionMs).To(Equal(minRetentionMs))
		})

		It("clamps retention above maximum", func() {
			err := service.CreateQueue(ctx, "high-retention", host.QueueConfig{
				RetentionMs: 999_999_999_999, // above maxRetentionMs
			})
			Expect(err).ToNot(HaveOccurred())

			service.mu.Lock()
			qs := service.queues["high-retention"]
			service.mu.Unlock()
			Expect(qs.config.RetentionMs).To(Equal(maxRetentionMs))
		})
	})

	Describe("Enqueue", func() {
		BeforeEach(func() {
			// Use a no-op callback to prevent actual execution attempts
			service.invokeCallbackFn = func(_ context.Context, _, _ string, _ []byte, _ int32) (string, error) {
				return "", nil
			}
			err := service.CreateQueue(ctx, "enqueue-test", host.QueueConfig{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("enqueues a task and returns task ID", func() {
			taskID, err := service.Enqueue(ctx, "enqueue-test", []byte("payload"))
			Expect(err).ToNot(HaveOccurred())
			Expect(taskID).ToNot(BeEmpty())
		})

		It("returns error for non-existent queue", func() {
			_, err := service.Enqueue(ctx, "no-such-queue", []byte("payload"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not exist"))
		})

		It("rejects payload exceeding maximum size", func() {
			bigPayload := make([]byte, maxPayloadSize+1)
			_, err := service.Enqueue(ctx, "enqueue-test", bigPayload)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exceeds maximum"))
		})

		It("accepts payload at maximum size", func() {
			exactPayload := make([]byte, maxPayloadSize)
			taskID, err := service.Enqueue(ctx, "enqueue-test", exactPayload)
			Expect(err).ToNot(HaveOccurred())
			Expect(taskID).ToNot(BeEmpty())
		})
	})

	Describe("GetTaskStatus", func() {
		BeforeEach(func() {
			// Use a callback that blocks until context is cancelled so tasks stay pending
			service.invokeCallbackFn = func(ctx context.Context, _, _ string, _ []byte, _ int32) (string, error) {
				<-ctx.Done()
				return "", ctx.Err()
			}
		})

		It("returns pending for a new task", func() {
			err := service.CreateQueue(ctx, "status-test", host.QueueConfig{})
			Expect(err).ToNot(HaveOccurred())

			taskID, err := service.Enqueue(ctx, "status-test", []byte("data"))
			Expect(err).ToNot(HaveOccurred())

			// The task may get picked up quickly; check initial status
			// Since the callback blocks, it should be either pending or running
			info, err := service.Get(ctx, taskID)
			Expect(err).ToNot(HaveOccurred())
			Expect(info).ToNot(BeNil())
			Expect(info.Status).To(BeElementOf("pending", "running"))
		})

		It("returns error for unknown task ID", func() {
			_, err := service.Get(ctx, "nonexistent-id")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})
	})

	Describe("CancelTask", func() {
		BeforeEach(func() {
			// Block callback so tasks stay in pending/running
			service.invokeCallbackFn = func(ctx context.Context, _, _ string, _ []byte, _ int32) (string, error) {
				<-ctx.Done()
				return "", ctx.Err()
			}
		})

		It("cancels a pending task", func() {
			// Block the callback so the first task occupies the worker
			started := make(chan struct{})
			service.invokeCallbackFn = func(ctx context.Context, _, _ string, _ []byte, _ int32) (string, error) {
				close(started)
				<-ctx.Done()
				return "", ctx.Err()
			}

			err := service.CreateQueue(ctx, "cancel-test", host.QueueConfig{
				Concurrency: 1,
			})
			Expect(err).ToNot(HaveOccurred())

			// Enqueue a blocker task to occupy the single worker
			_, err = service.Enqueue(ctx, "cancel-test", []byte("blocker"))
			Expect(err).ToNot(HaveOccurred())

			// Wait for the blocker task to start running
			Eventually(started).WithTimeout(5 * time.Second).Should(BeClosed())

			// Enqueue a second task — it stays pending since the worker is busy
			taskID, err := service.Enqueue(ctx, "cancel-test", []byte("cancel-me"))
			Expect(err).ToNot(HaveOccurred())

			err = service.Cancel(ctx, taskID)
			Expect(err).ToNot(HaveOccurred())

			info, err := service.Get(ctx, taskID)
			Expect(err).ToNot(HaveOccurred())
			Expect(info.Status).To(Equal("cancelled"))
		})

		It("returns error for unknown task ID", func() {
			err := service.Cancel(ctx, "nonexistent-id")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("returns error for non-pending task", func() {
			// Create a queue where tasks complete immediately
			service.invokeCallbackFn = func(_ context.Context, _, _ string, _ []byte, _ int32) (string, error) {
				return "", nil
			}
			err := service.CreateQueue(ctx, "completed-test", host.QueueConfig{})
			Expect(err).ToNot(HaveOccurred())

			taskID, err := service.Enqueue(ctx, "completed-test", []byte("data"))
			Expect(err).ToNot(HaveOccurred())

			// Wait for task to complete
			Eventually(func() string {
				info, err := service.Get(ctx, taskID)
				if err != nil || info == nil {
					return ""
				}
				return info.Status
			}).WithTimeout(5 * time.Second).WithPolling(50 * time.Millisecond).Should(Equal("completed"))

			// Try to cancel completed task
			err = service.Cancel(ctx, taskID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot be cancelled"))
		})
	})

	Describe("ClearQueue", func() {
		It("clears all pending tasks from a queue", func() {
			// Block the callback so the first task occupies the worker
			started := make(chan struct{})
			service.invokeCallbackFn = func(ctx context.Context, _, _ string, _ []byte, _ int32) (string, error) {
				close(started)
				<-ctx.Done()
				return "", ctx.Err()
			}

			err := service.CreateQueue(ctx, "clear-test", host.QueueConfig{
				Concurrency: 1,
			})
			Expect(err).ToNot(HaveOccurred())

			// Enqueue a blocker task to occupy the single worker
			_, err = service.Enqueue(ctx, "clear-test", []byte("blocker"))
			Expect(err).ToNot(HaveOccurred())

			// Wait for the blocker task to start running
			Eventually(started).WithTimeout(5 * time.Second).Should(BeClosed())

			// Enqueue several more tasks — they stay pending since the worker is busy
			var pendingIDs []string
			for i := 0; i < 3; i++ {
				taskID, err := service.Enqueue(ctx, "clear-test", []byte(fmt.Sprintf("task-%d", i)))
				Expect(err).ToNot(HaveOccurred())
				pendingIDs = append(pendingIDs, taskID)
			}

			// Clear the queue
			cleared, err := service.ClearQueue(ctx, "clear-test")
			Expect(err).ToNot(HaveOccurred())
			Expect(cleared).To(Equal(int64(3)))

			// Verify all pending tasks are now cancelled
			for _, id := range pendingIDs {
				info, err := service.Get(ctx, id)
				Expect(err).ToNot(HaveOccurred())
				Expect(info.Status).To(Equal("cancelled"))
			}
		})

		It("returns zero when queue has no pending tasks", func() {
			service.invokeCallbackFn = func(_ context.Context, _, _ string, _ []byte, _ int32) (string, error) {
				return "", nil
			}
			err := service.CreateQueue(ctx, "empty-clear", host.QueueConfig{})
			Expect(err).ToNot(HaveOccurred())

			cleared, err := service.ClearQueue(ctx, "empty-clear")
			Expect(err).ToNot(HaveOccurred())
			Expect(cleared).To(Equal(int64(0)))
		})

		It("returns error for non-existent queue", func() {
			_, err := service.ClearQueue(ctx, "no-such-queue")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not exist"))
		})

		It("does not affect running tasks", func() {
			// Block the callback so tasks stay running
			started := make(chan struct{}, 1)
			service.invokeCallbackFn = func(ctx context.Context, _, _ string, _ []byte, _ int32) (string, error) {
				select {
				case started <- struct{}{}:
				default:
				}
				<-ctx.Done()
				return "", ctx.Err()
			}

			err := service.CreateQueue(ctx, "clear-running", host.QueueConfig{
				Concurrency: 1,
			})
			Expect(err).ToNot(HaveOccurred())

			// Enqueue a task that will start running
			runningID, err := service.Enqueue(ctx, "clear-running", []byte("running-task"))
			Expect(err).ToNot(HaveOccurred())

			// Wait for it to start running
			Eventually(started).WithTimeout(5 * time.Second).Should(Receive())

			// Clear the queue — should not affect the running task
			cleared, err := service.ClearQueue(ctx, "clear-running")
			Expect(err).ToNot(HaveOccurred())
			Expect(cleared).To(Equal(int64(0)))

			// Verify the running task is still running
			info, err := service.Get(ctx, runningID)
			Expect(err).ToNot(HaveOccurred())
			Expect(info.Status).To(Equal("running"))
		})
	})

	Describe("Worker execution", func() {
		It("invokes callback and completes task", func() {
			var callCount atomic.Int32
			var receivedQueueName, receivedTaskID string
			var receivedPayload []byte
			var receivedAttempt int32

			service.invokeCallbackFn = func(_ context.Context, queueName, taskID string, payload []byte, attempt int32) (string, error) {
				callCount.Add(1)
				receivedQueueName = queueName
				receivedTaskID = taskID
				receivedPayload = payload
				receivedAttempt = attempt
				return "", nil
			}

			err := service.CreateQueue(ctx, "worker-test", host.QueueConfig{})
			Expect(err).ToNot(HaveOccurred())

			taskID, err := service.Enqueue(ctx, "worker-test", []byte("test-payload"))
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() string {
				info, err := service.Get(ctx, taskID)
				if err != nil || info == nil {
					return ""
				}
				return info.Status
			}).WithTimeout(5 * time.Second).WithPolling(50 * time.Millisecond).Should(Equal("completed"))

			Expect(callCount.Load()).To(Equal(int32(1)))
			Expect(receivedQueueName).To(Equal("worker-test"))
			Expect(receivedTaskID).To(Equal(taskID))
			Expect(receivedPayload).To(Equal([]byte("test-payload")))
			Expect(receivedAttempt).To(Equal(int32(1)))
		})
	})

	Describe("Message storage", func() {
		It("stores message on successful completion", func() {
			service.invokeCallbackFn = func(_ context.Context, _, _ string, _ []byte, _ int32) (string, error) {
				return "task completed successfully", nil
			}

			err := service.CreateQueue(ctx, "msg-success", host.QueueConfig{})
			Expect(err).ToNot(HaveOccurred())

			taskID, err := service.Enqueue(ctx, "msg-success", []byte("data"))
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() string {
				info, err := service.Get(ctx, taskID)
				if err != nil || info == nil {
					return ""
				}
				return info.Status
			}).WithTimeout(5 * time.Second).WithPolling(50 * time.Millisecond).Should(Equal("completed"))

			info, err := service.Get(ctx, taskID)
			Expect(err).ToNot(HaveOccurred())
			Expect(info.Message).To(Equal("task completed successfully"))
			Expect(info.Attempt).To(Equal(int32(1)))
		})

		It("stores error message on failure", func() {
			service.invokeCallbackFn = func(_ context.Context, _, _ string, _ []byte, _ int32) (string, error) {
				return "", fmt.Errorf("something went wrong")
			}

			err := service.CreateQueue(ctx, "msg-fail", host.QueueConfig{
				MaxRetries: 0,
			})
			Expect(err).ToNot(HaveOccurred())

			taskID, err := service.Enqueue(ctx, "msg-fail", []byte("data"))
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() string {
				info, err := service.Get(ctx, taskID)
				if err != nil || info == nil {
					return ""
				}
				return info.Status
			}).WithTimeout(5 * time.Second).WithPolling(50 * time.Millisecond).Should(Equal("failed"))

			info, err := service.Get(ctx, taskID)
			Expect(err).ToNot(HaveOccurred())
			Expect(info.Message).To(Equal("something went wrong"))
		})

		It("uses explicit message over error message on failure", func() {
			service.invokeCallbackFn = func(_ context.Context, _, _ string, _ []byte, _ int32) (string, error) {
				return "partial progress made", fmt.Errorf("timeout exceeded")
			}

			err := service.CreateQueue(ctx, "msg-fail-with-msg", host.QueueConfig{
				MaxRetries: 0,
			})
			Expect(err).ToNot(HaveOccurred())

			taskID, err := service.Enqueue(ctx, "msg-fail-with-msg", []byte("data"))
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() string {
				info, err := service.Get(ctx, taskID)
				if err != nil || info == nil {
					return ""
				}
				return info.Status
			}).WithTimeout(5 * time.Second).WithPolling(50 * time.Millisecond).Should(Equal("failed"))

			info, err := service.Get(ctx, taskID)
			Expect(err).ToNot(HaveOccurred())
			Expect(info.Message).To(Equal("partial progress made"))
		})
	})

	Describe("Retry on failure", func() {
		It("retries and eventually fails after exhausting retries", func() {
			var callCount atomic.Int32

			service.invokeCallbackFn = func(_ context.Context, _, _ string, _ []byte, _ int32) (string, error) {
				callCount.Add(1)
				return "", fmt.Errorf("task failed")
			}

			err := service.CreateQueue(ctx, "retry-test", host.QueueConfig{
				MaxRetries: 2,
				BackoffMs:  10, // Very short for testing
			})
			Expect(err).ToNot(HaveOccurred())

			taskID, err := service.Enqueue(ctx, "retry-test", []byte("retry-payload"))
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() string {
				info, err := service.Get(ctx, taskID)
				if err != nil || info == nil {
					return ""
				}
				return info.Status
			}).WithTimeout(10 * time.Second).WithPolling(50 * time.Millisecond).Should(Equal("failed"))

			// 1 initial attempt + 2 retries = 3 total calls
			Expect(callCount.Load()).To(Equal(int32(3)))
		})
	})

	Describe("Retry then succeed", func() {
		It("retries and succeeds on second attempt", func() {
			var callCount atomic.Int32

			service.invokeCallbackFn = func(_ context.Context, _, _ string, _ []byte, attempt int32) (string, error) {
				callCount.Add(1)
				if attempt == 1 {
					return "", fmt.Errorf("temporary error")
				}
				return "success", nil
			}

			err := service.CreateQueue(ctx, "retry-succeed", host.QueueConfig{
				MaxRetries: 1,
				BackoffMs:  10, // Very short for testing
			})
			Expect(err).ToNot(HaveOccurred())

			taskID, err := service.Enqueue(ctx, "retry-succeed", []byte("data"))
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() string {
				info, err := service.Get(ctx, taskID)
				if err != nil || info == nil {
					return ""
				}
				return info.Status
			}).WithTimeout(10 * time.Second).WithPolling(50 * time.Millisecond).Should(Equal("completed"))

			Expect(callCount.Load()).To(Equal(int32(2)))
		})
	})

	Describe("Backoff overflow cap", func() {
		It("caps backoff at maxRetentionMs to prevent overflow", func() {
			var callCount atomic.Int32

			service.invokeCallbackFn = func(_ context.Context, _, _ string, _ []byte, _ int32) (string, error) {
				callCount.Add(1)
				return "", fmt.Errorf("always fail")
			}

			err := service.CreateQueue(ctx, "backoff-overflow", host.QueueConfig{
				MaxRetries: 3,
				BackoffMs:  1_000_000_000, // Very large backoff to trigger overflow on exponentiation
			})
			Expect(err).ToNot(HaveOccurred())

			taskID, err := service.Enqueue(ctx, "backoff-overflow", []byte("overflow-test"))
			Expect(err).ToNot(HaveOccurred())

			// Wait for first attempt to fail
			Eventually(func() int32 {
				return callCount.Load()
			}).WithTimeout(5 * time.Second).WithPolling(50 * time.Millisecond).Should(BeNumerically(">=", int32(1)))

			// Check next_run_at is positive and reasonable (capped at maxRetentionMs from now)
			var nextRunAt int64
			err = service.db.QueryRow(`SELECT next_run_at FROM tasks WHERE id = ?`, taskID).Scan(&nextRunAt)
			Expect(err).ToNot(HaveOccurred())

			now := time.Now().UnixMilli()
			Expect(nextRunAt).To(BeNumerically(">", int64(0)), "next_run_at should be positive")
			Expect(nextRunAt).To(BeNumerically("<=", now+maxBackoffMs+1000), "next_run_at should be at most maxBackoffMs from now")
		})
	})

	Describe("Delay enforcement with concurrent workers", func() {
		It("enforces delay between dispatches even with multiple workers", func() {
			var mu sync.Mutex
			var dispatchTimes []time.Time

			service.invokeCallbackFn = func(_ context.Context, _, _ string, _ []byte, _ int32) (string, error) {
				mu.Lock()
				dispatchTimes = append(dispatchTimes, time.Now())
				mu.Unlock()
				return "", nil
			}

			err := service.CreateQueue(ctx, "delay-concurrent", host.QueueConfig{
				Concurrency: 3,
				DelayMs:     200,
			})
			Expect(err).ToNot(HaveOccurred())

			// Enqueue 5 tasks
			for i := 0; i < 5; i++ {
				_, err := service.Enqueue(ctx, "delay-concurrent", []byte(fmt.Sprintf("task-%d", i)))
				Expect(err).ToNot(HaveOccurred())
			}

			// Wait for all tasks to complete
			Eventually(func() int {
				mu.Lock()
				defer mu.Unlock()
				return len(dispatchTimes)
			}).WithTimeout(10 * time.Second).WithPolling(50 * time.Millisecond).Should(Equal(5))

			// Sort dispatch times and verify gaps
			mu.Lock()
			sort.Slice(dispatchTimes, func(i, j int) bool {
				return dispatchTimes[i].Before(dispatchTimes[j])
			})
			times := make([]time.Time, len(dispatchTimes))
			copy(times, dispatchTimes)
			mu.Unlock()

			// Consecutive dispatches should have at least ~160ms gap (80% of 200ms)
			for i := 1; i < len(times); i++ {
				gap := times[i].Sub(times[i-1])
				Expect(gap).To(BeNumerically(">=", 160*time.Millisecond),
					fmt.Sprintf("gap between dispatch %d and %d was %v, expected >= 160ms", i-1, i, gap))
			}
		})
	})

	Describe("Shutdown recovery", func() {
		It("resets stale running tasks on CreateQueue", func() {
			// Create a first service and queue, enqueue a task
			service.invokeCallbackFn = func(ctx context.Context, _, _ string, _ []byte, _ int32) (string, error) {
				<-ctx.Done()
				return "", ctx.Err()
			}
			err := service.CreateQueue(ctx, "recovery-queue", host.QueueConfig{})
			Expect(err).ToNot(HaveOccurred())

			taskID, err := service.Enqueue(ctx, "recovery-queue", []byte("stale-task"))
			Expect(err).ToNot(HaveOccurred())

			// Wait for the task to start running
			Eventually(func() string {
				info, err := service.Get(ctx, taskID)
				if err != nil || info == nil {
					return ""
				}
				return info.Status
			}).WithTimeout(5 * time.Second).WithPolling(50 * time.Millisecond).Should(Equal("running"))

			// Close the service (simulates crash - tasks left in running state)
			service.Close()

			// Create a new service pointing to the same DB
			managerCtx2, cancel2 := context.WithCancel(ctx)
			DeferCleanup(cancel2)
			manager2 := &Manager{
				plugins: make(map[string]*plugin),
				ctx:     managerCtx2,
			}

			service, err = newTaskQueueService("test_plugin", manager2, 5)
			Expect(err).ToNot(HaveOccurred())

			// Override callback to succeed
			service.invokeCallbackFn = func(_ context.Context, _, _ string, _ []byte, _ int32) (string, error) {
				return "", nil
			}

			// Re-create the queue - the upsert handles the existing row from the old service
			err = service.CreateQueue(ctx, "recovery-queue", host.QueueConfig{})
			Expect(err).ToNot(HaveOccurred())

			// The stale running task should now be reset to pending and eventually completed
			Eventually(func() string {
				info, err := service.Get(ctx, taskID)
				if err != nil || info == nil {
					return ""
				}
				return info.Status
			}).WithTimeout(10 * time.Second).WithPolling(50 * time.Millisecond).Should(Equal("completed"))
		})
	})

	Describe("Close", func() {
		It("prevents subsequent operations after close", func() {
			err := service.CreateQueue(ctx, "close-test", host.QueueConfig{})
			Expect(err).ToNot(HaveOccurred())

			service.Close()

			// After close, operations should fail
			_, err = service.Enqueue(ctx, "close-test", []byte("data"))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Plugin isolation", func() {
		It("uses separate databases for different plugins", func() {
			managerCtx2, cancel2 := context.WithCancel(ctx)
			DeferCleanup(cancel2)
			manager2 := &Manager{
				plugins: make(map[string]*plugin),
				ctx:     managerCtx2,
			}

			service2, err := newTaskQueueService("other_plugin", manager2, 5)
			Expect(err).ToNot(HaveOccurred())
			defer service2.Close()

			// Check that separate database files exist
			_, err = os.Stat(filepath.Join(tmpDir, "plugins", "test_plugin", "taskqueue.db"))
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Stat(filepath.Join(tmpDir, "plugins", "other_plugin", "taskqueue.db"))
			Expect(err).ToNot(HaveOccurred())

			// Both services should be able to create queues with the same name independently
			service.invokeCallbackFn = func(_ context.Context, _, _ string, _ []byte, _ int32) (string, error) { return "", nil }
			service2.invokeCallbackFn = func(_ context.Context, _, _ string, _ []byte, _ int32) (string, error) { return "", nil }

			err = service.CreateQueue(ctx, "shared-name", host.QueueConfig{})
			Expect(err).ToNot(HaveOccurred())
			err = service2.CreateQueue(ctx, "shared-name", host.QueueConfig{})
			Expect(err).ToNot(HaveOccurred())

			// Enqueue to each and verify they work independently
			taskID1, err := service.Enqueue(ctx, "shared-name", []byte("plugin1"))
			Expect(err).ToNot(HaveOccurred())
			taskID2, err := service2.Enqueue(ctx, "shared-name", []byte("plugin2"))
			Expect(err).ToNot(HaveOccurred())

			Expect(taskID1).ToNot(Equal(taskID2))

			// Both should complete
			Eventually(func() string {
				info, err := service.Get(ctx, taskID1)
				if err != nil || info == nil {
					return ""
				}
				return info.Status
			}).WithTimeout(5 * time.Second).WithPolling(50 * time.Millisecond).Should(Equal("completed"))

			Eventually(func() string {
				info, err := service2.Get(ctx, taskID2)
				if err != nil || info == nil {
					return ""
				}
				return info.Status
			}).WithTimeout(5 * time.Second).WithPolling(50 * time.Millisecond).Should(Equal("completed"))
		})
	})
})

var _ = Describe("TaskQueueService Integration", Ordered, func() {
	var manager *Manager
	var tmpDir string

	BeforeAll(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "taskqueue-integration-test-*")
		Expect(err).ToNot(HaveOccurred())

		// Copy the test-taskqueue plugin
		srcPath := filepath.Join(testdataDir, "test-taskqueue"+PackageExtension)
		destPath := filepath.Join(tmpDir, "test-taskqueue"+PackageExtension)
		data, err := os.ReadFile(srcPath)
		Expect(err).ToNot(HaveOccurred())
		err = os.WriteFile(destPath, data, 0600)
		Expect(err).ToNot(HaveOccurred())

		// Compute SHA256 for the plugin
		hash := sha256.Sum256(data)
		hashHex := hex.EncodeToString(hash[:])

		// Setup config
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = tmpDir
		conf.Server.Plugins.AutoReload = false
		conf.Server.CacheFolder = filepath.Join(tmpDir, "cache")
		conf.Server.DataFolder = tmpDir

		// Setup mock DataStore with pre-enabled plugin
		mockPluginRepo := tests.CreateMockPluginRepo()
		mockPluginRepo.Permitted = true
		mockPluginRepo.SetData(model.Plugins{{
			ID:      "test-taskqueue",
			Path:    destPath,
			SHA256:  hashHex,
			Enabled: true,
		}})
		dataStore := &tests.MockDataStore{MockedPlugin: mockPluginRepo}

		// Create and start manager
		manager = &Manager{
			plugins:        make(map[string]*plugin),
			ds:             dataStore,
			metrics:        noopMetricsRecorder{},
			subsonicRouter: http.NotFoundHandler(),
		}
		err = manager.Start(GinkgoT().Context())
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(func() {
			_ = manager.Stop()
			_ = os.RemoveAll(tmpDir)
		})
	})

	// Helper types for calling the test plugin
	type testQueueConfig struct {
		Concurrency int32 `json:"concurrency,omitempty"`
		MaxRetries  int32 `json:"maxRetries,omitempty"`
		BackoffMs   int64 `json:"backoffMs,omitempty"`
		DelayMs     int64 `json:"delayMs,omitempty"`
		RetentionMs int64 `json:"retentionMs,omitempty"`
	}

	type testTaskQueueInput struct {
		Operation string           `json:"operation"`
		QueueName string           `json:"queueName,omitempty"`
		Config    *testQueueConfig `json:"config,omitempty"`
		Payload   []byte           `json:"payload,omitempty"`
		TaskID    string           `json:"taskId,omitempty"`
	}

	type testTaskQueueOutput struct {
		TaskID  string  `json:"taskId,omitempty"`
		Status  string  `json:"status,omitempty"`
		Message string  `json:"message,omitempty"`
		Attempt int32   `json:"attempt,omitempty"`
		Cleared int64   `json:"cleared,omitempty"`
		Error   *string `json:"error,omitempty"`
	}

	callTestTaskQueue := func(ctx context.Context, input testTaskQueueInput) (*testTaskQueueOutput, error) {
		manager.mu.RLock()
		p := manager.plugins["test-taskqueue"]
		manager.mu.RUnlock()

		instance, err := p.instance(ctx)
		if err != nil {
			return nil, err
		}
		defer instance.Close(ctx)

		inputBytes, _ := json.Marshal(input)
		_, outputBytes, err := instance.Call("nd_test_taskqueue", inputBytes)
		if err != nil {
			return nil, err
		}

		var output testTaskQueueOutput
		if err := json.Unmarshal(outputBytes, &output); err != nil {
			return nil, err
		}
		if output.Error != nil {
			return nil, errors.New(*output.Error)
		}
		return &output, nil
	}

	Describe("Plugin Loading", func() {
		It("should load plugin with taskqueue permission and TaskWorker capability", func() {
			manager.mu.RLock()
			p, ok := manager.plugins["test-taskqueue"]
			manager.mu.RUnlock()
			Expect(ok).To(BeTrue())
			Expect(p.manifest.Permissions).ToNot(BeNil())
			Expect(p.manifest.Permissions.Taskqueue).ToNot(BeNil())
			Expect(p.manifest.Permissions.Taskqueue.MaxConcurrency).To(Equal(10))
			Expect(p.capabilities).To(ContainElement(CapabilityTaskWorker))
		})
	})

	Describe("Create Queue", func() {
		It("should create a queue without error", func() {
			ctx := GinkgoT().Context()
			_, err := callTestTaskQueue(ctx, testTaskQueueInput{
				Operation: "create_queue",
				QueueName: "test-create",
			})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return error for duplicate queue name", func() {
			ctx := GinkgoT().Context()
			_, err := callTestTaskQueue(ctx, testTaskQueueInput{
				Operation: "create_queue",
				QueueName: "test-dup",
			})
			Expect(err).ToNot(HaveOccurred())

			_, err = callTestTaskQueue(ctx, testTaskQueueInput{
				Operation: "create_queue",
				QueueName: "test-dup",
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))
		})
	})

	Describe("Enqueue and Task Completion", func() {
		It("should enqueue a task and complete successfully", func() {
			ctx := GinkgoT().Context()

			// Create queue
			_, err := callTestTaskQueue(ctx, testTaskQueueInput{
				Operation: "create_queue",
				QueueName: "test-complete",
			})
			Expect(err).ToNot(HaveOccurred())

			// Enqueue task with payload "hello"
			output, err := callTestTaskQueue(ctx, testTaskQueueInput{
				Operation: "enqueue",
				QueueName: "test-complete",
				Payload:   []byte("hello"),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.TaskID).ToNot(BeEmpty())

			taskID := output.TaskID

			// Poll until completed
			Eventually(func() string {
				out, err := callTestTaskQueue(ctx, testTaskQueueInput{
					Operation: "get_task_status",
					TaskID:    taskID,
				})
				if err != nil {
					return "error"
				}
				return out.Status
			}).WithTimeout(5 * time.Second).WithPolling(100 * time.Millisecond).Should(Equal("completed"))
		})
	})

	Describe("Enqueue with Failure, No Retries", func() {
		It("should fail when payload is 'fail' and maxRetries is 0", func() {
			ctx := GinkgoT().Context()

			// Create queue with no retries
			_, err := callTestTaskQueue(ctx, testTaskQueueInput{
				Operation: "create_queue",
				QueueName: "test-fail-no-retry",
				Config: &testQueueConfig{
					MaxRetries: 0,
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// Enqueue task that will fail
			output, err := callTestTaskQueue(ctx, testTaskQueueInput{
				Operation: "enqueue",
				QueueName: "test-fail-no-retry",
				Payload:   []byte("fail"),
			})
			Expect(err).ToNot(HaveOccurred())

			taskID := output.TaskID

			// Poll until failed
			Eventually(func() string {
				out, err := callTestTaskQueue(ctx, testTaskQueueInput{
					Operation: "get_task_status",
					TaskID:    taskID,
				})
				if err != nil {
					return "error"
				}
				return out.Status
			}).WithTimeout(5 * time.Second).WithPolling(100 * time.Millisecond).Should(Equal("failed"))
		})
	})

	Describe("Enqueue with Retry Then Success", func() {
		It("should retry and eventually succeed with 'fail-then-succeed' payload", func() {
			ctx := GinkgoT().Context()

			// Create queue with retries and short backoff
			_, err := callTestTaskQueue(ctx, testTaskQueueInput{
				Operation: "create_queue",
				QueueName: "test-retry-succeed",
				Config: &testQueueConfig{
					MaxRetries: 2,
					BackoffMs:  100,
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// Enqueue task that fails on attempt < 2, then succeeds
			output, err := callTestTaskQueue(ctx, testTaskQueueInput{
				Operation: "enqueue",
				QueueName: "test-retry-succeed",
				Payload:   []byte("fail-then-succeed"),
			})
			Expect(err).ToNot(HaveOccurred())

			taskID := output.TaskID

			// Poll until completed
			Eventually(func() string {
				out, err := callTestTaskQueue(ctx, testTaskQueueInput{
					Operation: "get_task_status",
					TaskID:    taskID,
				})
				if err != nil {
					return "error"
				}
				return out.Status
			}).WithTimeout(5 * time.Second).WithPolling(100 * time.Millisecond).Should(Equal("completed"))
		})
	})

	Describe("Cancel Pending Task", func() {
		It("should cancel a pending task", func() {
			ctx := GinkgoT().Context()

			// Create queue with concurrency=1 and a large delay between dispatches.
			// The first task completes immediately (burst token), the second is dequeued
			// but blocks on the rate limiter. Tasks 3+ remain in 'pending' and can be cancelled.
			_, err := callTestTaskQueue(ctx, testTaskQueueInput{
				Operation: "create_queue",
				QueueName: "test-cancel",
				Config: &testQueueConfig{
					Concurrency: 1,
					DelayMs:     60000,
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// Enqueue several tasks - the first will complete immediately,
			// the second will be dequeued but block on the rate limiter (status=running),
			// the rest will stay pending.
			var taskIDs []string
			for i := 0; i < 5; i++ {
				output, err := callTestTaskQueue(ctx, testTaskQueueInput{
					Operation: "enqueue",
					QueueName: "test-cancel",
					Payload:   []byte("hello"),
				})
				Expect(err).ToNot(HaveOccurred())
				taskIDs = append(taskIDs, output.TaskID)
			}

			// Wait for the first task to complete (it has no delay)
			Eventually(func() string {
				out, err := callTestTaskQueue(ctx, testTaskQueueInput{
					Operation: "get_task_status",
					TaskID:    taskIDs[0],
				})
				if err != nil {
					return "error"
				}
				return out.Status
			}).WithTimeout(5 * time.Second).WithPolling(50 * time.Millisecond).Should(Equal("completed"))

			// Give the worker a moment to dequeue the second task (which will
			// block on the delay) so tasks 3+ stay in 'pending'
			time.Sleep(100 * time.Millisecond)

			// Cancel the last task - it should still be pending
			lastTaskID := taskIDs[len(taskIDs)-1]
			_, err = callTestTaskQueue(ctx, testTaskQueueInput{
				Operation: "cancel_task",
				TaskID:    lastTaskID,
			})
			Expect(err).ToNot(HaveOccurred())

			// Verify status is cancelled
			statusOut, err := callTestTaskQueue(ctx, testTaskQueueInput{
				Operation: "get_task_status",
				TaskID:    lastTaskID,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(statusOut.Status).To(Equal("cancelled"))
		})
	})

	Describe("Enqueue to Non-Existent Queue", func() {
		It("should return error when enqueueing to a queue that does not exist", func() {
			ctx := GinkgoT().Context()

			_, err := callTestTaskQueue(ctx, testTaskQueueInput{
				Operation: "enqueue",
				QueueName: "nonexistent-queue",
				Payload:   []byte("payload"),
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not exist"))
		})
	})

	Describe("Clear Queue", func() {
		It("should clear pending tasks and return the count", func() {
			ctx := GinkgoT().Context()

			// Create queue with large delay so tasks stay pending after the first completes
			_, err := callTestTaskQueue(ctx, testTaskQueueInput{
				Operation: "create_queue",
				QueueName: "test-clear",
				Config: &testQueueConfig{
					Concurrency: 1,
					DelayMs:     60000,
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// Enqueue several tasks
			for i := 0; i < 4; i++ {
				_, err := callTestTaskQueue(ctx, testTaskQueueInput{
					Operation: "enqueue",
					QueueName: "test-clear",
					Payload:   []byte(fmt.Sprintf("task-%d", i)),
				})
				Expect(err).ToNot(HaveOccurred())
			}

			// Wait for the first task to complete (burst token)
			time.Sleep(200 * time.Millisecond)

			// Clear the queue — should cancel remaining pending tasks
			output, err := callTestTaskQueue(ctx, testTaskQueueInput{
				Operation: "clear_queue",
				QueueName: "test-clear",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Cleared).To(BeNumerically(">=", int64(1)))
		})

		It("should return error for non-existent queue", func() {
			ctx := GinkgoT().Context()

			_, err := callTestTaskQueue(ctx, testTaskQueueInput{
				Operation: "clear_queue",
				QueueName: "nonexistent-clear",
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not exist"))
		})
	})
})
