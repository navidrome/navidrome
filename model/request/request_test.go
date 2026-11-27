package request

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Token epoch holder", func() {
	var ctx context.Context
	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	It("reports nothing when unset", func() {
		ctx = WithTokenEpochHolder(ctx)
		_, ok := TokenEpochFrom(ctx)
		Expect(ok).To(BeFalse())
	})

	It("round-trips a value set by the handler", func() {
		ctx = WithTokenEpochHolder(ctx)
		SetTokenEpoch(ctx, 7)

		epoch, ok := TokenEpochFrom(ctx)
		Expect(ok).To(BeTrue())
		Expect(epoch).To(Equal(7))
	})

	It("survives being wrapped in a derived context", func() {
		ctx = WithTokenEpochHolder(ctx)
		SetTokenEpoch(context.WithValue(ctx, contextKey("unrelated"), 1), 3)

		epoch, ok := TokenEpochFrom(ctx)
		Expect(ok).To(BeTrue())
		Expect(epoch).To(Equal(3))
	})

	It("is a no-op with no holder installed", func() {
		Expect(func() { SetTokenEpoch(ctx, 5) }).ToNot(Panic())
		_, ok := TokenEpochFrom(ctx)
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("Server address", func() {
	var ctx context.Context
	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	It("reports nothing when unset", func() {
		_, _, ok := ServerAddressFrom(ctx)
		Expect(ok).To(BeFalse())
	})

	It("round-trips the scheme and host", func() {
		ctx = WithServerAddress(ctx, "https", "music.example.com")

		scheme, host, ok := ServerAddressFrom(ctx)
		Expect(ok).To(BeTrue())
		Expect(scheme).To(Equal("https"))
		Expect(host).To(Equal("music.example.com"))
	})

	It("reports nothing when the host is empty", func() {
		ctx = WithServerAddress(ctx, "https", "")

		_, _, ok := ServerAddressFrom(ctx)
		Expect(ok).To(BeFalse())
	})

	It("is carried over to a background context by AddValues", func() {
		reqCtx := WithServerAddress(ctx, "https", "music.example.com")

		_, host, ok := ServerAddressFrom(AddValues(ctx, reqCtx))
		Expect(ok).To(BeTrue())
		Expect(host).To(Equal("music.example.com"))
	})
})
