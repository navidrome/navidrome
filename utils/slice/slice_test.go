package slice_test

import (
	"strconv"
	"testing"

	"github.com/navidrome/navidrome/utils/slice"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSlice(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Slice Suite")
}

var _ = Describe("Map", func() {
	It("returns empty slice for an empty input", func() {
		mapFunc := func(v int) string { return strconv.Itoa(v * 2) }
		result := slice.Map([]int{}, mapFunc)
		Expect(result).To(BeEmpty())
	})

	It("returns a new slice with elements mapped", func() {
		mapFunc := func(v int) string { return strconv.Itoa(v * 2) }
		result := slice.Map([]int{1, 2, 3, 4}, mapFunc)
		Expect(result).To(ConsistOf("2", "4", "6", "8"))
	})
})

var _ = Describe("Group", func() {
	It("returns empty map for an empty input", func() {
		keyFunc := func(v int) int { return v % 2 }
		result := slice.Group([]int{}, keyFunc)
		Expect(result).To(BeEmpty())
	})

	It("groups by the result of the key function", func() {
		keyFunc := func(v int) int { return v % 2 }
		result := slice.Group([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}, keyFunc)
		Expect(result).To(HaveLen(2))
		Expect(result[0]).To(ConsistOf(2, 4, 6, 8, 10))
		Expect(result[1]).To(ConsistOf(1, 3, 5, 7, 9, 11))
	})
})

var _ = Describe("MostFrequent", func() {
	It("returns zero value if no arguments are passed", func() {
		Expect(slice.MostFrequent([]int{})).To(BeZero())
	})

	It("returns the single item", func() {
		Expect(slice.MostFrequent([]string{"123"})).To(Equal("123"))
	})
	It("returns the item that appeared more times", func() {
		Expect(slice.MostFrequent([]string{"1", "2", "1", "2", "3", "2"})).To(Equal("2"))
	})
})
