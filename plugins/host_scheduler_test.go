package plugins

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/plugins/host/scheduler"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SchedulerService", func() {
	var (
		ss         *schedulerService
		manager    *managerImpl
		pluginName = "test_plugin"
	)

	BeforeEach(func() {
		manager = createManager(nil, metrics.NewNoopInstance())
		ss = manager.schedulerService
	})

	Describe("One-time scheduling", func() {
		It("schedules one-time jobs successfully", func() {
			req := &scheduler.ScheduleOneTimeRequest{
				DelaySeconds: 1,
				Payload:      []byte("test payload"),
				ScheduleId:   "test-job",
			}

			resp, err := ss.scheduleOneTime(context.Background(), pluginName, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.ScheduleId).To(Equal("test-job"))
			Expect(ss.hasSchedule(pluginName + ":" + "test-job")).To(BeTrue())
			Expect(ss.getScheduleType(pluginName + ":" + "test-job")).To(Equal(ScheduleTypeOneTime))

			// Test auto-generated ID
			req.ScheduleId = ""
			resp, err = ss.scheduleOneTime(context.Background(), pluginName, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.ScheduleId).ToNot(BeEmpty())
		})

		It("cancels one-time jobs successfully", func() {
			req := &scheduler.ScheduleOneTimeRequest{
				DelaySeconds: 10,
				ScheduleId:   "test-job",
			}

			_, err := ss.scheduleOneTime(context.Background(), pluginName, req)
			Expect(err).ToNot(HaveOccurred())

			cancelReq := &scheduler.CancelRequest{
				ScheduleId: "test-job",
			}

			resp, err := ss.cancelSchedule(context.Background(), pluginName, cancelReq)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Success).To(BeTrue())
			Expect(ss.hasSchedule(pluginName + ":" + "test-job")).To(BeFalse())
		})
	})

	Describe("Recurring scheduling", func() {
		It("schedules recurring jobs successfully", func() {
			req := &scheduler.ScheduleRecurringRequest{
				CronExpression: "* * * * *", // Every minute
				Payload:        []byte("test payload"),
				ScheduleId:     "test-cron",
			}

			resp, err := ss.scheduleRecurring(context.Background(), pluginName, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.ScheduleId).To(Equal("test-cron"))
			Expect(ss.hasSchedule(pluginName + ":" + "test-cron")).To(BeTrue())
			Expect(ss.getScheduleType(pluginName + ":" + "test-cron")).To(Equal(ScheduleTypeRecurring))

			// Test auto-generated ID
			req.ScheduleId = ""
			resp, err = ss.scheduleRecurring(context.Background(), pluginName, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.ScheduleId).ToNot(BeEmpty())
		})

		It("cancels recurring jobs successfully", func() {
			req := &scheduler.ScheduleRecurringRequest{
				CronExpression: "* * * * *", // Every minute
				ScheduleId:     "test-cron",
			}

			_, err := ss.scheduleRecurring(context.Background(), pluginName, req)
			Expect(err).ToNot(HaveOccurred())

			cancelReq := &scheduler.CancelRequest{
				ScheduleId: "test-cron",
			}

			resp, err := ss.cancelSchedule(context.Background(), pluginName, cancelReq)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Success).To(BeTrue())
			Expect(ss.hasSchedule(pluginName + ":" + "test-cron")).To(BeFalse())
		})
	})

	Describe("Replace existing schedules", func() {
		It("replaces one-time jobs with new ones", func() {
			// Create first job
			req1 := &scheduler.ScheduleOneTimeRequest{
				DelaySeconds: 10,
				Payload:      []byte("test payload 1"),
				ScheduleId:   "replace-job",
			}
			_, err := ss.scheduleOneTime(context.Background(), pluginName, req1)
			Expect(err).ToNot(HaveOccurred())

			// Verify that the initial job exists
			scheduleId := pluginName + ":" + "replace-job"
			Expect(ss.hasSchedule(scheduleId)).To(BeTrue(), "Initial schedule should exist")

			beforeCount := ss.scheduleCount()

			// Replace with second job using same ID
			req2 := &scheduler.ScheduleOneTimeRequest{
				DelaySeconds: 60, // Use a longer delay to ensure it doesn't execute during the test
				Payload:      []byte("test payload 2"),
				ScheduleId:   "replace-job",
			}

			_, err = ss.scheduleOneTime(context.Background(), pluginName, req2)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				return ss.hasSchedule(scheduleId)
			}).Should(BeTrue(), "Schedule should exist after replacement")
			Expect(ss.scheduleCount()).To(Equal(beforeCount), "Job count should remain the same after replacement")
		})

		It("replaces recurring jobs with new ones", func() {
			// Create first job
			req1 := &scheduler.ScheduleRecurringRequest{
				CronExpression: "0 * * * *",
				Payload:        []byte("test payload 1"),
				ScheduleId:     "replace-cron",
			}
			_, err := ss.scheduleRecurring(context.Background(), pluginName, req1)
			Expect(err).ToNot(HaveOccurred())

			beforeCount := ss.scheduleCount()

			// Replace with second job using same ID
			req2 := &scheduler.ScheduleRecurringRequest{
				CronExpression: "*/5 * * * *",
				Payload:        []byte("test payload 2"),
				ScheduleId:     "replace-cron",
			}

			_, err = ss.scheduleRecurring(context.Background(), pluginName, req2)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				return ss.hasSchedule(pluginName + ":" + "replace-cron")
			}).Should(BeTrue(), "Schedule should exist after replacement")
			Expect(ss.scheduleCount()).To(Equal(beforeCount), "Job count should remain the same after replacement")
		})
	})

	Describe("TimeNow", func() {
		It("returns current time in RFC3339Nano, Unix milliseconds, and local timezone", func() {
			now := time.Now()
			req := &scheduler.TimeNowRequest{}
			resp, err := ss.timeNow(context.Background(), req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.UnixMilli).To(BeNumerically(">=", now.UnixMilli()))
			Expect(resp.LocalTimeZone).ToNot(BeEmpty())

			// Validate RFC3339Nano format can be parsed
			parsedTime, parseErr := time.Parse(time.RFC3339Nano, resp.Rfc3339Nano)
			Expect(parseErr).ToNot(HaveOccurred())

			// Validate that Unix milliseconds is reasonably close to the RFC3339Nano time
			expectedMillis := parsedTime.UnixMilli()
			Expect(resp.UnixMilli).To(Equal(expectedMillis))

			// Validate local timezone matches the current system timezone
			expectedTimezone := now.Location().String()
			Expect(resp.LocalTimeZone).To(Equal(expectedTimezone))
		})
	})
})
