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

// Note: These tests may be flaky due to the use of time.Sleep.
var _ = Describe("Meter", func() {
	var meter *Meter

	BeforeEach(func() {
		meter = &Meter{}
	})

	Describe("Stop", func() {
		It("should return the elapsed time", func() {
			meter.Start()
			time.Sleep(20 * time.Millisecond)
			elapsed := meter.Stop()
			Expect(elapsed).To(BeNumerically("~", 20*time.Millisecond, 10*time.Millisecond))
		})

		It("should accumulate elapsed time on multiple starts and stops", func() {
			meter.Start()
			time.Sleep(20 * time.Millisecond)
			meter.Stop()

			meter.Start()
			time.Sleep(20 * time.Millisecond)
			elapsed := meter.Stop()

			Expect(elapsed).To(BeNumerically("~", 40*time.Millisecond, 20*time.Millisecond))
		})
	})

	Describe("Elapsed", func() {
		It("should return the total elapsed time", func() {
			meter.Start()
			time.Sleep(20 * time.Millisecond)
			meter.Stop()

			// Should not count the time the meter was stopped
			time.Sleep(20 * time.Millisecond)

			meter.Start()
			time.Sleep(20 * time.Millisecond)
			meter.Stop()

			Expect(meter.Elapsed()).To(BeNumerically("~", 40*time.Millisecond, 20*time.Millisecond))
		})

		It("should include the current running time if started", func() {
			meter.Start()
			time.Sleep(20 * time.Millisecond)
			Expect(meter.Elapsed()).To(BeNumerically("~", 20*time.Millisecond, 10*time.Millisecond))
		})
	})
})
