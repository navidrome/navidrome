package artwork

import (
	"errors"
	"testing"
	"testing/synctest"
	"time"

	. "github.com/onsi/gomega"
)

// Drives the real breaker state machine with the fake clock. Plain test: testing/synctest
// needs a *testing.T, which Ginkgo doesn't give.
func TestArtworkBreakerHalfOpen(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := NewWithT(t)
		b := newBreaker()

		for range breakerThreshold {
			b.record(errors.New("boom"))
		}
		g.Expect(b.allow()).To(BeFalse(), "breaker opens after consecutive errors")

		time.Sleep(breakerProbeAfter - time.Nanosecond)
		g.Expect(b.allow()).To(BeFalse(), "still open before the probe interval")

		time.Sleep(time.Nanosecond)
		g.Expect(b.allow()).To(BeTrue(), "half-open: one probe is granted")
		g.Expect(b.allow()).To(BeFalse(), "only a single probe per interval")

		b.record(errors.New("boom")) // probe fails -> stay open
		time.Sleep(breakerProbeAfter)
		g.Expect(b.allow()).To(BeTrue(), "another probe after the next interval")

		b.record(nil) // probe succeeds -> close
		g.Expect(b.allow()).To(BeTrue(), "closed breaker admits freely")
		g.Expect(b.allow()).To(BeTrue())
	})
}
