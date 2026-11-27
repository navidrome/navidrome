package utils_test

import (
	"context"

	"github.com/navidrome/navidrome/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("IsCtxDone", func() {
	var ctx context.Context
	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	It("returns false if the context is not done", func() {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		Expect(utils.IsCtxDone(ctx)).To(BeFalse())
	})

	It("returns true if the context is done", func() {
		ctx, cancel := context.WithCancel(ctx)
		cancel()
		Expect(utils.IsCtxDone(ctx)).To(BeTrue())
	})
})
