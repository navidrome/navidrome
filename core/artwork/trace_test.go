package artwork

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("chainTrace", func() {
	It("returns nil when no trace is attached", func() {
		Expect(traceFrom(context.Background())).To(BeNil())
	})

	It("collects steps in order", func() {
		t := &chainTrace{}
		ctx := withTrace(context.Background(), t)

		traceFrom(ctx).add(traceStep{Candidate: "cover.*", Outcome: outcomeMiss})
		traceFrom(ctx).add(traceStep{Candidate: "embedded", Outcome: outcomeHit, Detail: "/music/a.flac"})

		Expect(t.Steps()).To(Equal([]traceStep{
			{Candidate: "cover.*", Outcome: outcomeMiss},
			{Candidate: "embedded", Outcome: outcomeHit, Detail: "/music/a.flac"},
		}))
	})

	It("is safe to use concurrently", func() {
		t := &chainTrace{}
		done := make(chan struct{})
		for range 10 {
			go func() {
				defer GinkgoRecover()
				t.add(traceStep{Candidate: "x", Outcome: outcomeMiss})
				done <- struct{}{}
			}()
		}
		for range 10 {
			<-done
		}
		Expect(t.Steps()).To(HaveLen(10))
	})
})
