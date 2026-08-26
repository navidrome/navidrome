package request

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Token epoch holder", func() {
	It("reports nothing when unset", func() {
		ctx := WithTokenEpochHolder(context.TODO())
		_, ok := TokenEpochFrom(ctx)
		Expect(ok).To(BeFalse())
	})

	It("round-trips a value set by the handler", func() {
		ctx := WithTokenEpochHolder(context.TODO())
		SetTokenEpoch(ctx, 7)

		epoch, ok := TokenEpochFrom(ctx)
		Expect(ok).To(BeTrue())
		Expect(epoch).To(Equal(7))
	})

	It("survives being wrapped in a derived context", func() {
		ctx := WithTokenEpochHolder(context.TODO())
		SetTokenEpoch(context.WithValue(ctx, contextKey("unrelated"), 1), 3)

		epoch, ok := TokenEpochFrom(ctx)
		Expect(ok).To(BeTrue())
		Expect(epoch).To(Equal(3))
	})

	It("is a no-op with no holder installed", func() {
		Expect(func() { SetTokenEpoch(context.TODO(), 5) }).ToNot(Panic())
		_, ok := TokenEpochFrom(context.TODO())
		Expect(ok).To(BeFalse())
	})
})
