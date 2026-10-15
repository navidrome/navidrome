package artwork

import (
	"errors"
	"time"

	"github.com/navidrome/navidrome/core/agents"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// allowed drops the generation token when a caller only cares about admission.
func allowed(b *breaker) bool { ok, _ := b.allow(); return ok }

var _ = Describe("breaker", func() {
	// The worker drains concurrently, so when the breaker opens there are already calls past
	// allow(), queued in the rate limiter or waiting on a response. Their answers arrive
	// afterwards. Counting those as recovery closes the breaker with no probe interval elapsed,
	// which is the burst the ramp exists to prevent. No clock is involved: the race is an
	// ordering, so it is reproduced by making the calls in the order concurrency produces.
	It("ignores answers from calls admitted before it opened", func() {
		b := newBreaker()

		// A batch clears allow() while the breaker is still closed.
		for range breakerThreshold + breakerRecoveries {
			ok, gen := b.allow()
			Expect(ok).To(BeTrue())
			Expect(gen).To(BeZero(), "admitted with the breaker closed, so not a probe")
		}

		// The fast failures in that batch open it.
		for range breakerThreshold {
			b.record("agentA", 0, errors.New("blocked"))
		}
		Expect(allowed(b)).To(BeFalse(), "breaker is open")

		// The slower answers from the same batch land now.
		for range breakerRecoveries {
			b.record("agentA", 0, nil)
		}

		Expect(allowed(b)).To(BeFalse(),
			"answers from calls admitted before the breaker opened must not close it")
	})

	It("opens at once when a provider asks to retry later, honoring its delay", func() {
		b := newBreaker()
		Expect(allowed(b)).To(BeTrue(), "starts closed")

		// A single explicit back-off opens the breaker without reaching the failure threshold.
		b.record("agentA", 0, &agents.RetryLaterError{RetryIn: 5 * time.Second})

		Expect(allowed(b)).To(BeFalse(), "an explicit back-off opens the breaker immediately")
		Expect(b.probeAfter).To(Equal(5*time.Second), "the provider's delay drives the probe interval")
	})
})
