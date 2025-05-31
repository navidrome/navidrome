package chain_test

import (
	"errors"
	"testing"

	"github.com/navidrome/navidrome/utils/chain"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestChain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "chain Suite")
}

var _ = Describe("RunSequentially", func() {
	It("should return nil if no functions are provided", func() {
		err := chain.RunSequentially()
		Expect(err).To(BeNil())
	})

	It("should return nil if all functions succeed", func() {
		err := chain.RunSequentially(
			func() error { return nil },
			func() error { return nil },
		)
		Expect(err).To(BeNil())
	})

	It("should return the error from the first failing function", func() {
		expectedErr := errors.New("error in function 2")
		err := chain.RunSequentially(
			func() error { return nil },
			func() error { return expectedErr },
			func() error { return errors.New("error in function 3") },
		)
		Expect(err).To(Equal(expectedErr))
	})

	It("should not run functions after the first failing function", func() {
		expectedErr := errors.New("error in function 1")
		var runCount int
		err := chain.RunSequentially(
			func() error { runCount++; return expectedErr },
			func() error { runCount++; return nil },
		)
		Expect(err).To(Equal(expectedErr))
		Expect(runCount).To(Equal(1))
	})
})
