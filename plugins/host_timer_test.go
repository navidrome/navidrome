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
		ts      *timerService
		manager *Manager
		ctx     context.Context
		hf      TimerHostFunctions
	)

	BeforeEach(func() {
		ctx = context.Background()
		manager = createManager()
		ts = newTimerService(manager)
		hf = ts.HostFunctions("test_plugin")
	})

	Context("RegisterTimer", func() {
		It("should generate a timer ID if not provided", func() {
			resp, err := hf.RegisterTimer(ctx, &timer.TimerRequest{
				Payload: []byte("test-payload"),
				Delay:   5,
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.TimerId).ToNot(BeEmpty())
		})

		It("should use the provided timer ID", func() {
			resp, err := hf.RegisterTimer(ctx, &timer.TimerRequest{
				Payload: []byte("test-payload"),
				Delay:   5,
				TimerId: "custom-id",
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.TimerId).To(Equal("custom-id"))
		})

		It("should store the timer in the internal map with a prefixed ID", func() {
			_, err := hf.RegisterTimer(ctx, &timer.TimerRequest{
				Payload: []byte("test-payload"),
				Delay:   5,
				TimerId: "custom-id",
			})

			Expect(err).ToNot(HaveOccurred())
			// Check internal map
			Expect(ts.hasTimer("test_plugin:custom-id")).To(BeTrue())
		})

		It("should fail if manager is not initialized", func() {
			ts = newTimerService(nil)
			hf = ts.HostFunctions("test_plugin")
			_, err := hf.RegisterTimer(ctx, &timer.TimerRequest{
				TimerId: "custom-id",
			})

			Expect(err).To(HaveOccurred())
		})
	})

	Context("CancelTimer", func() {
		It("should cancel a timer correctly", func() {
			// First register a timer
			resp, err := hf.RegisterTimer(ctx, &timer.TimerRequest{
				Payload: []byte("test-payload"),
				Delay:   60, // Long enough not to fire during test
				TimerId: "custom-id",
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.TimerId).To(Equal("custom-id"))

			// Now cancel it
			cancelResp, err := hf.CancelTimer(ctx, &timer.CancelTimerRequest{
				TimerId: "custom-id",
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(cancelResp.Success).To(BeTrue())

			// Timer should be removed from map
			Expect(ts.hasTimer("test_plugin:custom-id")).To(BeFalse())
		})

		It("should fail when canceling non-existent timer", func() {
			resp, err := hf.CancelTimer(ctx, &timer.CancelTimerRequest{
				TimerId: "non-existent",
			})

			Expect(err).To(HaveOccurred())
			Expect(resp.Success).To(BeFalse())
		})
	})

	Context("Timer execution", func() {
		It("should remove timer from map after it fires", func() {
			// Register a timer with short delay
			resp, err := hf.RegisterTimer(ctx, &timer.TimerRequest{
				Payload: []byte("test-payload"),
				Delay:   1, // 1 second
				TimerId: "short-timer",
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.TimerId).To(Equal("short-timer"))

			// Check that timer exists in map
			Expect(ts.hasTimer("test_plugin:short-timer")).To(BeTrue())

			// Wait for timer to fire
			time.Sleep(1500 * time.Millisecond)

			// Timer should be removed from map after firing
			Expect(ts.hasTimer("test_plugin:short-timer")).To(BeFalse())
		})
	})

	Context("Multiple plugins with same timer ID", func() {
		It("should handle multiple plugins using the same timer ID", func() {
			hf1 := ts.HostFunctions("plugin1")
			hf2 := ts.HostFunctions("plugin2")

			// Register timers with the same ID but different plugins
			resp1, err := hf1.RegisterTimer(ctx, &timer.TimerRequest{
				Payload: []byte("payload1"),
				Delay:   60,
				TimerId: "same-id",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp1.TimerId).To(Equal("same-id"))

			resp2, err := hf2.RegisterTimer(ctx, &timer.TimerRequest{
				Payload: []byte("payload2"),
				Delay:   60,
				TimerId: "same-id",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp2.TimerId).To(Equal("same-id"))

			// Both timers should exist with prefixed IDs
			Expect(ts.hasTimer("plugin1:same-id")).To(BeTrue())
			Expect(ts.hasTimer("plugin2:same-id")).To(BeTrue())

			// Cancel one timer
			cancelResp, err := hf1.CancelTimer(ctx, &timer.CancelTimerRequest{
				TimerId: "same-id",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(cancelResp.Success).To(BeTrue())

			// One timer should remain (we don't know which one will be canceled first)
			Expect(ts.timerCount()).To(Equal(1))
			Expect(ts.hasTimer("plugin1:same-id")).To(BeFalse())
			Expect(ts.hasTimer("plugin2:same-id")).To(BeTrue())
		})
	})
})
