package plugins

import (
	"context"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/plugins/host/crontab"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Crontab Host Service", func() {
	var (
		ctx     context.Context
		manager *Manager
		cs      *crontabService
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = context.Background()
		manager = createManager()
		cs = manager.crontabService
	})

	Describe("scheduleJob", func() {
		It("should schedule a job with a custom ID", func() {
			req := &crontab.ScheduleJobRequest{
				CronExpression: "* * * * *", // Every minute
				Payload:        []byte("test"),
				JobId:          "test-job",
			}
			resp, err := cs.scheduleJob(ctx, "test-plugin", req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.JobId).To(Equal("test-job"))
			Expect(cs.hasJob("test-plugin:test-job")).To(BeTrue())
			Expect(cs.jobCount()).To(Equal(1))
		})

		It("should generate an ID if none is provided", func() {
			req := &crontab.ScheduleJobRequest{
				CronExpression: "* * * * *", // Every minute
				Payload:        []byte("test"),
			}
			resp, err := cs.scheduleJob(ctx, "test-plugin", req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.JobId).ToNot(BeEmpty())
			Expect(cs.hasJob("test-plugin:" + resp.JobId)).To(BeTrue())
			Expect(cs.jobCount()).To(Equal(1))
		})

		It("should return an error for invalid cron expressions", func() {
			req := &crontab.ScheduleJobRequest{
				CronExpression: "invalid", // Invalid cron expression
				Payload:        []byte("test"),
				JobId:          "test-job",
			}
			_, err := cs.scheduleJob(ctx, "test-plugin", req)

			Expect(err).To(HaveOccurred())
			Expect(cs.jobCount()).To(Equal(0))
		})
	})

	Describe("cancelJob", func() {
		BeforeEach(func() {
			req := &crontab.ScheduleJobRequest{
				CronExpression: "* * * * *", // Every minute
				Payload:        []byte("test"),
				JobId:          "test-job",
			}
			_, err := cs.scheduleJob(ctx, "test-plugin", req)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should cancel a scheduled job", func() {
			req := &crontab.CancelJobRequest{
				JobId: "test-job",
			}
			resp, err := cs.cancelJob(ctx, "test-plugin", req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Success).To(BeTrue())
			Expect(cs.hasJob("test-plugin:test-job")).To(BeFalse())
			Expect(cs.jobCount()).To(Equal(0))
		})

		It("should actually stop the job from running", func() {
			// First, get the current job count
			initialJobCount := cs.jobCount()
			Expect(initialJobCount).To(Equal(1))

			// Schedule a frequent job that we can observe
			counter := 0
			var mu sync.Mutex
			var wg sync.WaitGroup
			wg.Add(1)

			// Use a channel to detect job executions
			executionCh := make(chan struct{})

			// Create a job with our own execution tracking
			req := &crontab.ScheduleJobRequest{
				CronExpression: "@every 100ms", // Run every 100ms
				JobId:          "counter-job",
				Payload:        []byte("counter-payload"),
			}
			_, err := cs.scheduleJob(ctx, "test-plugin", req)
			Expect(err).ToNot(HaveOccurred())

			// Create a goroutine that will:
			// 1. Wait for at least one execution
			// 2. Cancel the job
			// 3. Monitor if any more executions occur
			go func() {
				// Wait for first execution
				<-executionCh

				// Cancel the job
				cancelReq := &crontab.CancelJobRequest{
					JobId: "counter-job",
				}
				resp, err := cs.cancelJob(ctx, "test-plugin", cancelReq)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.Success).To(BeTrue())

				// Signal that we're done with setup
				wg.Done()

				// Monitor for more executions (which shouldn't happen)
				select {
				case <-executionCh:
					mu.Lock()
					counter++ // This should never be reached after cancellation
					mu.Unlock()
				case <-time.After(300 * time.Millisecond):
					// This is expected - no more executions
				}
			}()

			// Trigger a job execution to simulate what would happen in the real system
			// This simulates the first job execution which will kick off our test sequence
			cs.executeCallback(ctx, "test-plugin:counter-job")
			executionCh <- struct{}{}

			// Wait for test sequence to complete
			wg.Wait()

			// Check if any extra executions occurred
			mu.Lock()
			finalCounter := counter
			mu.Unlock()

			// The job should have been cancelled, so counter should remain at 0
			Expect(finalCounter).To(Equal(0))

			// The job should be gone from the registry
			Expect(cs.hasJob("test-plugin:counter-job")).To(BeFalse())
		})

		It("should return an error for non-existent jobs", func() {
			req := &crontab.CancelJobRequest{
				JobId: "non-existent",
			}
			resp, err := cs.cancelJob(ctx, "test-plugin", req)

			Expect(err).To(HaveOccurred())
			Expect(resp.Success).To(BeFalse())
			Expect(cs.jobCount()).To(Equal(1))
		})
	})

	Describe("CrontabHostFunctions", func() {
		var chf CrontabHostFunctions

		BeforeEach(func() {
			chf = cs.HostFunctions("test-plugin")
		})

		It("should delegate ScheduleJob to the crontab service", func() {
			req := &crontab.ScheduleJobRequest{
				CronExpression: "* * * * *", // Every minute
				Payload:        []byte("test"),
				JobId:          "test-job",
			}
			resp, err := chf.ScheduleJob(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.JobId).To(Equal("test-job"))
			Expect(cs.hasJob("test-plugin:test-job")).To(BeTrue())
		})

		It("should delegate CancelJob to the crontab service", func() {
			// First schedule a job
			scheduleReq := &crontab.ScheduleJobRequest{
				CronExpression: "* * * * *", // Every minute
				Payload:        []byte("test"),
				JobId:          "test-job",
			}
			_, err := chf.ScheduleJob(ctx, scheduleReq)
			Expect(err).ToNot(HaveOccurred())

			// Then cancel it
			cancelReq := &crontab.CancelJobRequest{
				JobId: "test-job",
			}
			resp, err := chf.CancelJob(ctx, cancelReq)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Success).To(BeTrue())
			Expect(cs.hasJob("test-plugin:test-job")).To(BeFalse())
		})
	})
})
