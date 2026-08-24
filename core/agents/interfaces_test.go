package agents_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/scrobbler"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RetryLaterError", func() {
	It("matches the ErrRetryLater sentinel via errors.Is", func() {
		err := &agents.RetryLaterError{RetryIn: 30 * time.Second}
		Expect(errors.Is(err, agents.ErrRetryLater)).To(BeTrue())
	})

	It("matches through errors.Join and wrapping", func() {
		err := fmt.Errorf("calling LB: %w", errors.Join(errors.New("http 429"), &agents.RetryLaterError{}))
		Expect(errors.Is(err, agents.ErrRetryLater)).To(BeTrue())
	})

	It("exposes the delay via RetryIn", func() {
		err := errors.Join(errors.New("http 429"), &agents.RetryLaterError{RetryIn: 42 * time.Second})
		d, ok := agents.RetryIn(err)
		Expect(ok).To(BeTrue())
		Expect(d).To(Equal(42 * time.Second))
	})

	It("reports no delay for a plain sentinel", func() {
		d, ok := agents.RetryIn(agents.ErrRetryLater)
		Expect(ok).To(BeFalse())
		Expect(d).To(BeZero())
	})

	It("is the same sentinel as scrobbler.ErrRetryLater", func() {
		Expect(errors.Is(scrobbler.ErrRetryLater, agents.ErrRetryLater)).To(BeTrue())
		Expect(errors.Is(&agents.RetryLaterError{}, scrobbler.ErrRetryLater)).To(BeTrue())
	})
})
