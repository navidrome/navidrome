package math2_test

import (
	"testing"

	"github.com/navidrome/navidrome/utils/math2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMath2(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Math2 Suite")
}

var _ = Describe("Min", func() {
	It("returns zero value if no arguments are passed", func() {
		Expect(math2.Min[int]()).To(BeZero())
	})
	It("returns the smallest int", func() {
		Expect(math2.Min(1, 2)).To(Equal(1))
	})
	It("returns the smallest float", func() {
		Expect(math2.Min(-4.1, -4.2, -4.0)).To(Equal(-4.2))
	})
})

var _ = Describe("Max", func() {
	It("returns zero value if no arguments are passed", func() {
		Expect(math2.Max[int]()).To(BeZero())
	})
	It("returns the biggest int", func() {
		Expect(math2.Max(1, 2)).To(Equal(2))
	})
	It("returns the biggest float", func() {
		Expect(math2.Max(-4.1, -4.2, -4.0)).To(Equal(-4.0))
	})
})
