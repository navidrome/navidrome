//go:build !windows

package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/scheduler"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SchedulerService", Ordered, func() {
	var (
		manager     *Manager
		tmpDir      string
		mockSched   *mockScheduler
		mockTimers  *mockTimerRegistry
		testService *testableSchedulerService
		origAfterFn func(time.Duration, func()) *time.Timer
	)

	BeforeAll(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "scheduler-test-*")
		Expect(err).ToNot(HaveOccurred())

		// Copy the test-scheduler plugin
		srcPath := filepath.Join(testdataDir, "test-scheduler"+PackageExtension)
		destPath := filepath.Join(tmpDir, "test-scheduler"+PackageExtension)
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

		// Create mock scheduler and timer registry
		mockSched = newMockScheduler()
		mockTimers = newMockTimerRegistry()

		// Replace timeAfterFunc with mock
		origAfterFn = timeAfterFunc
		timeAfterFunc = mockTimers.AfterFunc

		// Setup mock DataStore with pre-enabled plugin
		mockPluginRepo := tests.CreateMockPluginRepo()
		mockPluginRepo.Permitted = true
		mockPluginRepo.SetData(model.Plugins{{
			ID:      "test-scheduler",
			Path:    destPath,
			SHA256:  hashHex,
			Enabled: true,
		}})
		dataStore := &tests.MockDataStore{MockedPlugin: mockPluginRepo}

		// Create and start manager
		manager = &Manager{
			plugins:        make(map[string]*plugin),
			ds:             dataStore,
			subsonicRouter: http.NotFoundHandler(),
			metrics:        noopMetricsRecorder{},
		}
		err = manager.Start(GinkgoT().Context())
		Expect(err).ToNot(HaveOccurred())

		// Get scheduler service from plugin's closers and wrap it for testing
		service := findSchedulerService(manager, "test-scheduler")
		Expect(service).ToNot(BeNil())
		testService = &testableSchedulerService{schedulerServiceImpl: service}
		testService.scheduler = mockSched

		DeferCleanup(func() {
			timeAfterFunc = origAfterFn
			_ = manager.Stop()
			_ = os.RemoveAll(tmpDir)
		})
	})

	BeforeEach(func() {
		mockSched.Reset()
		mockTimers.Reset()
		testService.ClearSchedules()
	})

	Describe("Plugin Loading", func() {
		It("should detect scheduler capability", func() {
			names := manager.PluginNames(string(CapabilityScheduler))
			Expect(names).To(ContainElement("test-scheduler"))
		})

		It("should register scheduler service for plugin", func() {
			service := findSchedulerService(manager, "test-scheduler")
			Expect(service).ToNot(BeNil())
		})
	})

	Describe("ScheduleOneTime", func() {
		It("should schedule a one-time task", func() {
			scheduleID, err := testService.ScheduleOneTime(GinkgoT().Context(), 1, "test-payload", "test-id")
			Expect(err).ToNot(HaveOccurred())
			Expect(scheduleID).To(Equal("test-id"))

			// Verify schedule was registered
			Expect(testService.GetScheduleCount()).To(Equal(1))
			Expect(mockTimers.GetTimerCount()).To(Equal(1))
		})

		It("should invoke plugin callback and auto-cleanup after firing", func() {
			_, err := testService.ScheduleOneTime(GinkgoT().Context(), 1, "data", "cleanup-id")
			Expect(err).ToNot(HaveOccurred())
			Expect(testService.GetScheduleCount()).To(Equal(1))

			// Trigger fires the callback which calls the plugin's nd_scheduler_callback
			// One-time schedules clean up after the callback completes
			mockTimers.TriggerAll()

			// One-time schedules should self-cleanup
			Expect(testService.GetScheduleCount()).To(Equal(0))
		})

		It("should reject duplicate schedule ID", func() {
			_, err := testService.ScheduleOneTime(GinkgoT().Context(), 60, "data", "dup-id")
			Expect(err).ToNot(HaveOccurred())

			_, err = testService.ScheduleOneTime(GinkgoT().Context(), 60, "data2", "dup-id")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))
		})

		It("should auto-generate schedule ID when empty", func() {
			scheduleID, err := testService.ScheduleOneTime(GinkgoT().Context(), 1, "data", "")
			Expect(err).ToNot(HaveOccurred())
			Expect(scheduleID).ToNot(BeEmpty())
		})
	})

	Describe("ScheduleRecurring", func() {
		It("should schedule recurring tasks", func() {
			scheduleID, err := testService.ScheduleRecurring(GinkgoT().Context(), "@every 1s", "recurring-data", "recurring-id")
			Expect(err).ToNot(HaveOccurred())
			Expect(scheduleID).To(Equal("recurring-id"))

			// Verify schedule was registered
			Expect(testService.GetScheduleCount()).To(Equal(1))
			entry := testService.GetSchedule("recurring-id")
			Expect(entry).ToNot(BeNil())
			Expect(entry.isRecurring).To(BeTrue())
		})

		It("should invoke plugin callback multiple times without self-canceling", func() {
			_, err := testService.ScheduleRecurring(GinkgoT().Context(), "@every 1s", "data", "persist-id")
			Expect(err).ToNot(HaveOccurred())

			// Trigger multiple times - recurring schedules should persist
			mockSched.TriggerAll()
			mockSched.TriggerAll()

			// Recurring schedules should persist
			Expect(testService.GetScheduleCount()).To(Equal(1))
		})
	})

	Describe("Plugin Calling Host Functions", func() {
		It("should allow plugin to schedule a one-time task from callback", func() {
			// Schedule with magic payload that triggers plugin to call SchedulerScheduleOneTime
			_, err := testService.ScheduleRecurring(GinkgoT().Context(), "@every 1s", "schedule-followup", "trigger-id")
			Expect(err).ToNot(HaveOccurred())
			Expect(testService.GetScheduleCount()).To(Equal(1))

			// Trigger - plugin callback will schedule a follow-up task
			mockSched.TriggerAll()

			// Verify the plugin created a new schedule via host function
			Expect(testService.GetScheduleCount()).To(Equal(2)) // original + followup

			// Verify the follow-up schedule was created with correct ID and properties
			followup := testService.GetSchedule("followup-id")
			Expect(followup).ToNot(BeNil())
			Expect(followup.payload).To(Equal("followup-created"))
			Expect(followup.isRecurring).To(BeFalse())
			Expect(followup.timer).ToNot(BeNil()) // One-time tasks use timers
		})

		It("should allow plugin to schedule a recurring task from callback", func() {
			_, err := testService.ScheduleRecurring(GinkgoT().Context(), "@every 1s", "schedule-recurring", "trigger-id")
			Expect(err).ToNot(HaveOccurred())

			mockSched.TriggerAll()

			// Verify the plugin created a recurring schedule
			entry := testService.GetSchedule("recurring-from-plugin")
			Expect(entry).ToNot(BeNil())
			Expect(entry.isRecurring).To(BeTrue())
			Expect(entry.payload).To(Equal("recurring-created"))
		})
	})

	Describe("CancelSchedule", func() {
		It("should cancel a recurring task", func() {
			_, err := testService.ScheduleRecurring(GinkgoT().Context(), "@every 1s", "data", "cancel-id")
			Expect(err).ToNot(HaveOccurred())
			Expect(testService.GetScheduleCount()).To(Equal(1))

			err = testService.CancelSchedule(GinkgoT().Context(), "cancel-id")
			Expect(err).ToNot(HaveOccurred())
			Expect(testService.GetScheduleCount()).To(Equal(0))
		})

		It("should cancel a one-time task", func() {
			_, err := testService.ScheduleOneTime(GinkgoT().Context(), 60, "data", "cancel-onetime-id")
			Expect(err).ToNot(HaveOccurred())
			Expect(testService.GetScheduleCount()).To(Equal(1))
			Expect(mockTimers.GetTimerCount()).To(Equal(1))

			err = testService.CancelSchedule(GinkgoT().Context(), "cancel-onetime-id")
			Expect(err).ToNot(HaveOccurred())
			Expect(testService.GetScheduleCount()).To(Equal(0))
		})

		It("should remove callback from scheduler for recurring tasks", func() {
			_, err := testService.ScheduleRecurring(GinkgoT().Context(), "@every 1s", "data", "cancel-id")
			Expect(err).ToNot(HaveOccurred())
			Expect(mockSched.GetCallbackCount()).To(Equal(1))

			err = testService.CancelSchedule(GinkgoT().Context(), "cancel-id")
			Expect(err).ToNot(HaveOccurred())
			Expect(mockSched.GetCallbackCount()).To(Equal(0))
		})

		It("should return error for non-existent schedule", func() {
			err := testService.CancelSchedule(GinkgoT().Context(), "non-existent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})
	})

	Describe("Scheduler Service Isolation", func() {
		It("should share the same scheduler service across multiple plugin instances", func() {
			// This test verifies that when we call plugin.instance() multiple times
			// (creating multiple instances from the same compiled plugin), they all
			// share the same scheduler service. This is the expected behavior since
			// the scheduler service is registered once per plugin at compile time.

			// Get the plugin
			manager.mu.RLock()
			plugin, ok := manager.plugins["test-scheduler"]
			manager.mu.RUnlock()
			Expect(ok).To(BeTrue())

			// Schedule a task using the service directly
			_, err := testService.ScheduleOneTime(GinkgoT().Context(), 60, "shared-data", "shared-id")
			Expect(err).ToNot(HaveOccurred())
			Expect(testService.GetScheduleCount()).To(Equal(1))

			// Create a plugin instance
			instance, err := plugin.instance(GinkgoT().Context())
			Expect(err).ToNot(HaveOccurred())
			defer instance.Close(GinkgoT().Context())

			// The scheduler service is shared, so the schedule ID should clash
			// if another instance tries to use the same ID
			_, err = testService.ScheduleOneTime(GinkgoT().Context(), 60, "other-data", "shared-id")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))

			// But different IDs should work fine
			_, err = testService.ScheduleOneTime(GinkgoT().Context(), 60, "instance2-data", "otherx-id")
			Expect(err).ToNot(HaveOccurred())
			Expect(testService.GetScheduleCount()).To(Equal(2))
		})
	})

	Describe("Plugin Unload", func() {
		It("should cancel all schedules when plugin is unloaded", func() {
			_, err := testService.ScheduleRecurring(GinkgoT().Context(), "@every 10s", "data1", "unload-1")
			Expect(err).ToNot(HaveOccurred())
			_, err = testService.ScheduleOneTime(GinkgoT().Context(), 60, "data2", "unload-2")
			Expect(err).ToNot(HaveOccurred())
			Expect(testService.GetScheduleCount()).To(Equal(2))
			Expect(mockSched.GetCallbackCount()).To(Equal(1)) // Only recurring task uses scheduler
			Expect(mockTimers.GetTimerCount()).To(Equal(1))   // Only one-time task uses timer

			err = manager.unloadPlugin("test-scheduler")
			Expect(err).ToNot(HaveOccurred())

			Expect(findSchedulerService(manager, "test-scheduler")).To(BeNil())
			Expect(mockSched.GetCallbackCount()).To(Equal(0)) // Recurring task removed
		})
	})
})

// testableSchedulerService wraps schedulerServiceImpl with test helpers.
type testableSchedulerService struct {
	*schedulerServiceImpl
}

func (t *testableSchedulerService) GetScheduleCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.schedules)
}

func (t *testableSchedulerService) GetSchedule(id string) *scheduleEntry {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.schedules[id]
}

func (t *testableSchedulerService) ClearSchedules() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.schedules = make(map[string]*scheduleEntry)
}

// mockScheduler implements scheduler.Scheduler for testing without timing dependencies.
type mockScheduler struct {
	mu        sync.Mutex
	callbacks map[int]func()
	nextID    int
}

func newMockScheduler() *mockScheduler {
	return &mockScheduler{
		callbacks: make(map[int]func()),
		nextID:    1,
	}
}

func (s *mockScheduler) Run(_ context.Context) {}

func (s *mockScheduler) Add(_ string, cmd func()) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID
	s.nextID++
	s.callbacks[id] = cmd
	return id, nil
}

func (s *mockScheduler) Remove(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.callbacks, id)
}

func (s *mockScheduler) TriggerAll() {
	s.mu.Lock()
	callbacks := make([]func(), 0, len(s.callbacks))
	for _, cb := range s.callbacks {
		callbacks = append(callbacks, cb)
	}
	s.mu.Unlock()
	for _, cb := range callbacks {
		cb()
	}
}

func (s *mockScheduler) GetCallbackCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.callbacks)
}

func (s *mockScheduler) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callbacks = make(map[int]func())
	s.nextID = 1
}

var _ scheduler.Scheduler = (*mockScheduler)(nil)

// mockTimerRegistry tracks mock timers created during tests.
type mockTimerRegistry struct {
	mu        sync.Mutex
	callbacks []func()
	timers    []*time.Timer
}

func newMockTimerRegistry() *mockTimerRegistry {
	return &mockTimerRegistry{
		callbacks: make([]func(), 0),
		timers:    make([]*time.Timer, 0),
	}
}

// AfterFunc creates a timer that we control for testing.
func (r *mockTimerRegistry) AfterFunc(_ time.Duration, f func()) *time.Timer {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Store callback for TriggerAll
	r.callbacks = append(r.callbacks, f)

	// Create a real timer that won't fire (very long duration, immediately stopped)
	t := time.NewTimer(time.Hour * 24 * 365)
	t.Stop()
	r.timers = append(r.timers, t)

	return t
}

// TriggerAll fires all pending timer callbacks.
func (r *mockTimerRegistry) TriggerAll() {
	r.mu.Lock()
	callbacks := make([]func(), len(r.callbacks))
	copy(callbacks, r.callbacks)
	r.mu.Unlock()

	for _, cb := range callbacks {
		cb()
	}
}

func (r *mockTimerRegistry) GetTimerCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.callbacks)
}

func (r *mockTimerRegistry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.callbacks = make([]func(), 0)
	r.timers = make([]*time.Timer, 0)
}

// findSchedulerService finds the scheduler service from a plugin's closers.
func findSchedulerService(m *Manager, pluginName string) *schedulerServiceImpl {
	m.mu.RLock()
	instance, ok := m.plugins[pluginName]
	m.mu.RUnlock()
	if !ok {
		return nil
	}
	for _, closer := range instance.closers {
		if svc, ok := closer.(*schedulerServiceImpl); ok {
			return svc
		}
	}
	return nil
}
