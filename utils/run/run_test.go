package run_test

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/navidrome/navidrome/utils/run"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRun(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Run Suite")
}

var _ = Describe("Sequentially", func() {
	It("should return nil if no functions are provided", func() {
		err := run.Sequentially()
		Expect(err).To(BeNil())
	})

	It("should return nil if all functions succeed", func() {
		err := run.Sequentially(
			func() error { return nil },
			func() error { return nil },
		)
		Expect(err).To(BeNil())
	})

	It("should return the error from the first failing function", func() {
		expectedErr := errors.New("error in function 2")
		err := run.Sequentially(
			func() error { return nil },
			func() error { return expectedErr },
			func() error { return errors.New("error in function 3") },
		)
		Expect(err).To(Equal(expectedErr))
	})

	It("should not run functions after the first failing function", func() {
		expectedErr := errors.New("error in function 1")
		var runCount int
		err := run.Sequentially(
			func() error { runCount++; return expectedErr },
			func() error { runCount++; return nil },
		)
		Expect(err).To(Equal(expectedErr))
		Expect(runCount).To(Equal(1))
	})
})

var _ = Describe("Parallel", func() {
	It("should return a function that returns nil if no functions are provided", func() {
		parallelFunc := run.Parallel()
		err := parallelFunc()
		Expect(err).To(BeNil())
	})

	It("should return a function that returns nil if all functions succeed", func() {
		parallelFunc := run.Parallel(
			func() error { return nil },
			func() error { return nil },
			func() error { return nil },
		)
		err := parallelFunc()
		Expect(err).To(BeNil())
	})

	It("should return the first error encountered when functions fail", func() {
		expectedErr := errors.New("parallel error")
		parallelFunc := run.Parallel(
			func() error { return nil },
			func() error { return expectedErr },
			func() error { return errors.New("another error") },
		)
		err := parallelFunc()
		Expect(err).To(HaveOccurred())
		// Note: We can't guarantee which error will be returned first in parallel execution
		// but we can ensure an error is returned
	})

	It("should run all functions in parallel", func() {
		var runCount atomic.Int32
		sync := make(chan struct{})

		parallelFunc := run.Parallel(
			func() error {
				runCount.Add(1)
				<-sync
				runCount.Add(-1)
				return nil
			},
			func() error {
				runCount.Add(1)
				<-sync
				runCount.Add(-1)
				return nil
			},
			func() error {
				runCount.Add(1)
				<-sync
				runCount.Add(-1)
				return nil
			},
		)

		// Run the parallel function in a goroutine
		go func() {
			Expect(parallelFunc()).To(Succeed())
		}()

		// Wait for all functions to start running
		Eventually(func() int32 { return runCount.Load() }).Should(Equal(int32(3)))

		// Release the functions to complete
		close(sync)

		// Wait for all functions to finish
		Eventually(func() int32 { return runCount.Load() }).Should(Equal(int32(0)))
	})

	It("should wait for all functions to complete before returning", func() {
		var completedCount atomic.Int32

		parallelFunc := run.Parallel(
			func() error {
				completedCount.Add(1)
				return nil
			},
			func() error {
				completedCount.Add(1)
				return nil
			},
			func() error {
				completedCount.Add(1)
				return nil
			},
		)

		Expect(parallelFunc()).To(Succeed())
		Expect(completedCount.Load()).To(Equal(int32(3)))
	})

	It("should return an error even if other functions are still running", func() {
		expectedErr := errors.New("fast error")
		var slowFunctionCompleted bool

		parallelFunc := run.Parallel(
			func() error {
				return expectedErr // Return error immediately
			},
			func() error {
				time.Sleep(50 * time.Millisecond) // Slow function
				slowFunctionCompleted = true
				return nil
			},
		)

		start := time.Now()
		err := parallelFunc()
		duration := time.Since(start)

		Expect(err).To(HaveOccurred())
		// Should wait for all functions to complete, even if one fails early
		Expect(duration).To(BeNumerically(">=", 50*time.Millisecond))
		Expect(slowFunctionCompleted).To(BeTrue())
	})
})
