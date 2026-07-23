package artwork

import (
	"errors"
	"io"
	"testing"
	"testing/synctest"
	"time"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/tests"
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

// Drives the worker's per-name gate map with the fake clock: one agent's open breaker
// must neither block another agent nor short-circuit the other's probe recovery.
func TestArtworkGatePerAgentBreakerIsolation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := NewWithT(t)
		w := NewWorker(&tests.MockDataStore{}, NewImageStore(t.TempDir()),
			agents.GetAgents(&tests.MockDataStore{}, nil), tests.NewMockFFmpeg(""), &fakeEventBroker{}, nil)

		fail := func() (io.ReadCloser, string, error) { return nil, "", errors.New("boom") }
		for range breakerThreshold {
			_, _, _ = w.gate("A", fail)
		}
		_, _, err := w.gate("A", fail)
		g.Expect(err).To(MatchError(errBreakerOpen), "A opens after consecutive errors")

		// B has its own breaker, untouched by A being open.
		var bCalls int
		bStep := func() (io.ReadCloser, string, error) { bCalls++; return nil, "", errors.New("boom") }
		for range breakerThreshold - 1 {
			_, _, err := w.gate("B", bStep)
			g.Expect(err).To(MatchError("boom"))
		}
		g.Expect(bCalls).To(Equal(breakerThreshold-1), "B keeps being called while A is open")

		// After the probe window, A admits exactly one probe again.
		time.Sleep(breakerProbeAfter)
		var aCalls int
		aFail := func() (io.ReadCloser, string, error) { aCalls++; return nil, "", errors.New("boom") }
		_, _, _ = w.gate("A", aFail)
		g.Expect(aCalls).To(Equal(1), "A grants a single probe after the interval")
		_, _, err = w.gate("A", aFail)
		g.Expect(err).To(MatchError(errBreakerOpen), "the probe failed, so A stays open")
		g.Expect(aCalls).To(Equal(1))
	})
}
