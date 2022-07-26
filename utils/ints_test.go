package utils

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Int utils", func() {
	Describe("MinInt", func() {
		It("returns the first value if it is the smallest", func() {
			Expect(MinInt(1, 2)).To(Equal(1))
		})
		It("returns the second value if it is the smallest", func() {
			Expect(MinInt(-4, -6)).To(Equal(-6))
		})
	})

	Describe("MaxInt", func() {
		It("returns the first value if it is the biggest", func() {
			Expect(MaxInt(1, 2)).To(Equal(2))
		})
		It("returns the second value if it is the smallest", func() {
			Expect(MaxInt(-4, -6)).To(Equal(-4))
		})
	})

	Describe("IntInSlice", func() {
		It("returns false if slice is empty", func() {
			Expect(IntInSlice(1, nil)).To(BeFalse())
		})

		It("returns false if number is not in slice", func() {
			Expect(IntInSlice(1, []int{3, 4, 5})).To(BeFalse())
		})

		It("returns true if number is in slice", func() {
			Expect(IntInSlice(4, []int{3, 4, 5})).To(BeTrue())
		})
	})
})
