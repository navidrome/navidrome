package artwork

import (
	"errors"

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
})
