package number_test

import (
	"testing"

	"github.com/navidrome/navidrome/utils/number"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestNumber(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Number Suite")
}

var _ = Describe("Min", func() {
	It("returns zero value if no arguments are passed", func() {
		Expect(number.Min[int]()).To(BeZero())
	})
	It("returns the smallest int", func() {
		Expect(number.Min(1, 2)).To(Equal(1))
	})
	It("returns the smallest float", func() {
		Expect(number.Min(-4.1, -4.2, -4.0)).To(Equal(-4.2))
	})
})

var _ = Describe("Max", func() {
	It("returns zero value if no arguments are passed", func() {
		Expect(number.Max[int]()).To(BeZero())
	})
	It("returns the biggest int", func() {
		Expect(number.Max(1, 2)).To(Equal(2))
	})
	It("returns the biggest float", func() {
		Expect(number.Max(-4.1, -4.2, -4.0)).To(Equal(-4.0))
	})
})
