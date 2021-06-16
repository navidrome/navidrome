package utils_test

import (
	"context"

	"github.com/navidrome/navidrome/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IsCtxDone", func() {
	It("returns false if the context is not done", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		Expect(utils.IsCtxDone(ctx)).To(BeFalse())
	})

	It("returns true if the context is done", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		Expect(utils.IsCtxDone(ctx)).To(BeTrue())
	})
})
