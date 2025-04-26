package plugins

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/plugins/host/timer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TimerService", func() {
	var (
		ts      *TimerService
		manager *Manager
		ctx     context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		manager = createManager()
		ts = NewTimerService(manager)
	})

	Context("RegisterTimer", func() {
		It("should generate a timer ID if not provided", func() {
			resp, err := ts.RegisterTimer(ctx, &timer.TimerRequest{
				PluginName: "test_plugin",
				Payload:    []byte("test-payload"),
				Delay:      5,
			})

			Expect(err).To(BeNil())
			Expect(resp.Error).To(BeEmpty())
			Expect(resp.TimerId).ToNot(BeEmpty())
		})

		It("should use the provided timer ID", func() {
			resp, err := ts.RegisterTimer(ctx, &timer.TimerRequest{
				PluginName: "test_plugin",
				Payload:    []byte("test-payload"),
				Delay:      5,
				TimerId:    "custom-id",
			})

			Expect(err).To(BeNil())
			Expect(resp.Error).To(BeEmpty())
			Expect(resp.TimerId).To(Equal("custom-id"))
		})

		It("should store the timer in the internal map with a prefixed ID", func() {
			_, err := ts.RegisterTimer(ctx, &timer.TimerRequest{
				PluginName: "test_plugin",
				Payload:    []byte("test-payload"),
				Delay:      5,
				TimerId:    "custom-id",
			})

			Expect(err).To(BeNil())
			// Check internal map
			Expect(ts.HasTimer("test_plugin:custom-id")).To(BeTrue())
		})

		It("should fail if manager is not initialized", func() {
			ts = NewTimerService(nil)
			resp, err := ts.RegisterTimer(ctx, &timer.TimerRequest{
				PluginName: "test_plugin",
				TimerId:    "custom-id",
			})

			Expect(err).To(BeNil())
			Expect(resp.Error).ToNot(BeEmpty())
		})
	})

	Context("CancelTimer", func() {
		It("should cancel a timer correctly", func() {
			// First register a timer
			resp, err := ts.RegisterTimer(ctx, &timer.TimerRequest{
				PluginName: "test_plugin",
				Payload:    []byte("test-payload"),
				Delay:      60, // Long enough not to fire during test
				TimerId:    "custom-id",
			})

			Expect(err).To(BeNil())
			Expect(resp.TimerId).To(Equal("custom-id"))

			// Now cancel it
			cancelResp, err := ts.CancelTimer(ctx, &timer.CancelTimerRequest{
				TimerId: "custom-id",
			})

			Expect(err).To(BeNil())
			Expect(cancelResp.Success).To(BeTrue())
			Expect(cancelResp.Error).To(BeEmpty())

			// Timer should be removed from map
			Expect(ts.HasTimer("test_plugin:custom-id")).To(BeFalse())
		})

		It("should fail when canceling non-existent timer", func() {
			resp, err := ts.CancelTimer(ctx, &timer.CancelTimerRequest{
				TimerId: "non-existent",
			})

			Expect(err).To(BeNil())
			Expect(resp.Success).To(BeFalse())
			Expect(resp.Error).ToNot(BeEmpty())
		})
	})

	Context("Timer execution", func() {
		It("should remove timer from map after it fires", func() {
			// Register a timer with short delay
			resp, err := ts.RegisterTimer(ctx, &timer.TimerRequest{
				PluginName: "test_plugin",
				Payload:    []byte("test-payload"),
				Delay:      1, // 1 second
				TimerId:    "short-timer",
			})

			Expect(err).To(BeNil())
			Expect(resp.TimerId).To(Equal("short-timer"))

			// Check that timer exists in map
			Expect(ts.HasTimer("test_plugin:short-timer")).To(BeTrue())

			// Wait for timer to fire
			time.Sleep(1500 * time.Millisecond)

			// Timer should be removed from map after firing
			Expect(ts.HasTimer("test_plugin:short-timer")).To(BeFalse())
		})
	})

	Context("Multiple plugins with same timer ID", func() {
		It("should handle multiple plugins using the same timer ID", func() {
			// Register timers with the same ID but different plugins
			resp1, err := ts.RegisterTimer(ctx, &timer.TimerRequest{
				PluginName: "plugin1",
				Payload:    []byte("payload1"),
				Delay:      60,
				TimerId:    "same-id",
			})
			Expect(err).To(BeNil())
			Expect(resp1.TimerId).To(Equal("same-id"))

			resp2, err := ts.RegisterTimer(ctx, &timer.TimerRequest{
				PluginName: "plugin2",
				Payload:    []byte("payload2"),
				Delay:      60,
				TimerId:    "same-id",
			})
			Expect(err).To(BeNil())
			Expect(resp2.TimerId).To(Equal("same-id"))

			// Both timers should exist with prefixed IDs
			Expect(ts.HasTimer("plugin1:same-id")).To(BeTrue())
			Expect(ts.HasTimer("plugin2:same-id")).To(BeTrue())

			// Cancel one timer
			cancelResp, err := ts.CancelTimer(ctx, &timer.CancelTimerRequest{
				TimerId: "same-id",
			})
			Expect(err).To(BeNil())
			Expect(cancelResp.Success).To(BeTrue())

			// One timer should remain (we don't know which one will be canceled first)
			Expect(ts.TimerCount()).To(Equal(1))
		})
	})
})
