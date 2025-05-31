package chrono_test

import (
	"testing"
	"time"

	"github.com/navidrome/navidrome/tests"
	. "github.com/navidrome/navidrome/utils/chrono"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestChrono(t *testing.T) {
	tests.Init(t, false)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Chrono Suite")
}

// Note: These tests use longer sleep durations and generous tolerances to avoid flakiness
// due to system scheduling delays. For a more elegant approach in the future, consider
// using Go 1.24's experimental testing/synctest package with GOEXPERIMENT=synctest.

var _ = Describe("Meter", func() {
	var meter *Meter

	BeforeEach(func() {
		meter = &Meter{}
	})

	Describe("Stop", func() {
		It("should return the elapsed time", func() {
			meter.Start()
			time.Sleep(50 * time.Millisecond)
			elapsed := meter.Stop()
			// Use generous tolerance to account for system scheduling delays
			Expect(elapsed).To(BeNumerically(">=", 30*time.Millisecond))
			Expect(elapsed).To(BeNumerically("<=", 200*time.Millisecond))
		})

		It("should accumulate elapsed time on multiple starts and stops", func() {
			// First cycle
			meter.Start()
			time.Sleep(50 * time.Millisecond)
			firstElapsed := meter.Stop()

			// Second cycle
			meter.Start()
			time.Sleep(50 * time.Millisecond)
			totalElapsed := meter.Stop()

			// Test that time accumulates (second measurement should be greater than first)
			Expect(totalElapsed).To(BeNumerically(">", firstElapsed))

			// Test that accumulated time is reasonable (should be roughly double the first)
			Expect(totalElapsed).To(BeNumerically(">=", time.Duration(float64(firstElapsed)*1.5)))
			Expect(totalElapsed).To(BeNumerically("<=", firstElapsed*3))

			// Sanity check: total should be at least 60ms (allowing for some timing variance)
			Expect(totalElapsed).To(BeNumerically(">=", 60*time.Millisecond))
		})
	})

	Describe("Elapsed", func() {
		It("should return the total elapsed time", func() {
			meter.Start()
			time.Sleep(50 * time.Millisecond)
			meter.Stop()

			// Should not count the time the meter was stopped
			time.Sleep(50 * time.Millisecond)

			meter.Start()
			time.Sleep(50 * time.Millisecond)
			meter.Stop()

			elapsed := meter.Elapsed()
			// Should be roughly 100ms (2 x 50ms), but allow for significant variance
			Expect(elapsed).To(BeNumerically(">=", 60*time.Millisecond))
			Expect(elapsed).To(BeNumerically("<=", 300*time.Millisecond))
		})

		It("should include the current running time if started", func() {
			meter.Start()
			time.Sleep(50 * time.Millisecond)
			elapsed := meter.Elapsed()
			// Use generous tolerance to account for system scheduling delays
			Expect(elapsed).To(BeNumerically(">=", 30*time.Millisecond))
			Expect(elapsed).To(BeNumerically("<=", 200*time.Millisecond))
		})
	})
})
