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

// allowed drops the generation token when a test only cares about admission.
func allowed(b *breaker) bool { ok, _ := b.allow(); return ok }

// Drives the real breaker state machine with the fake clock. Plain test: testing/synctest
// needs a *testing.T, which Ginkgo doesn't give.
func TestArtworkBreakerHalfOpen(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := NewWithT(t)
		b := newBreaker()

		for range breakerThreshold {
			b.record("agentA", 0, errors.New("boom"))
		}
		g.Expect(allowed(b)).To(BeFalse(), "breaker opens after consecutive errors")

		time.Sleep(breakerProbeAfter - time.Nanosecond)
		g.Expect(allowed(b)).To(BeFalse(), "still open before the probe interval")

		time.Sleep(time.Nanosecond)
		ok, gen := b.allow()
		g.Expect(ok).To(BeTrue(), "half-open: one probe is granted")
		g.Expect(gen).ToNot(BeZero(), "a probe carries the open episode it belongs to")
		g.Expect(allowed(b)).To(BeFalse(), "only a single probe per interval")

		b.record("agentA", gen, errors.New("boom")) // probe fails -> stay open
		time.Sleep(breakerProbeAfter)
		ok, gen = b.allow()
		g.Expect(ok).To(BeTrue(), "another probe after the next interval")

		// One good answer must not reopen the floodgates: closing here is what let a burst out at
		// full rate and got the provider to escalate from throttling to blocking.
		b.record("agentA", gen, nil)
		g.Expect(allowed(b)).To(BeFalse(), "a single good answer does not close the breaker")

		for range breakerRecoveries - 1 {
			time.Sleep(breakerProbeAfter)
			ok, gen = b.allow()
			g.Expect(ok).To(BeTrue())
			b.record("agentA", gen, nil)
		}
		g.Expect(allowed(b)).To(BeTrue(), "closed breaker admits freely")
		g.Expect(allowed(b)).To(BeTrue())
	})
}

// The failure seen in production: while an agent was blocked, the occasional answer it did serve
// reset the breaker, releasing a burst that immediately re-tripped it. Open and closed pairs were
// seconds apart, over and over.
func TestArtworkBreakerDoesNotCloseOnAnIsolatedAnswer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := NewWithT(t)
		b := newBreaker()
		open := func() {
			for range breakerThreshold {
				b.record("agentA", 0, errors.New("blocked"))
			}
		}
		open()

		// A not-found is an answer, so it counts toward recovery, but never on its own.
		for range breakerRecoveries * 2 {
			time.Sleep(breakerProbeAfter)
			ok, gen := b.allow()
			g.Expect(ok).To(BeTrue(), "one probe per interval")
			b.record("agentA", gen, agents.ErrNotFound)
			b.record("agentA", 0, errors.New("blocked")) // the very next call is refused again
			g.Expect(allowed(b)).To(BeFalse(), "an answer between failures must not close the breaker")
		}
	})
}

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

// The worker drains concurrently, so when the breaker opens there are already calls past allow(),
// queued in the rate limiter or waiting on an HTTP response. Their answers arrive afterwards.
// Counting those as recovery closes the breaker with no probe interval elapsed, which is exactly
// the burst the ramp exists to prevent. No fake clock: the race is an ordering, not a duration.
func TestArtworkBreakerIgnoresAnswersAdmittedBeforeItOpened(t *testing.T) {
	g := NewWithT(t)
	b := newBreaker()

	// A batch clears allow() while the breaker is still closed.
	for range breakerThreshold + breakerRecoveries {
		ok, gen := b.allow()
		g.Expect(ok).To(BeTrue())
		g.Expect(gen).To(BeZero(), "admitted with the breaker closed, so not a probe")
	}

	// The fast failures in that batch open it.
	for range breakerThreshold {
		b.record("agentA", 0, errors.New("blocked"))
	}
	g.Expect(allowed(b)).To(BeFalse(), "breaker is open")

	// The slower answers from the same batch land now.
	for range breakerRecoveries {
		b.record("agentA", 0, nil)
	}

	g.Expect(allowed(b)).To(BeFalse(),
		"answers from calls admitted before the breaker opened must not close it")
}
