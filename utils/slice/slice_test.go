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

var _ = Describe("Slice Utils", func() {
	Describe("Map", func() {
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

	Describe("Group", func() {
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

	Describe("ToMap", func() {
		It("returns empty map for an empty input", func() {
			transformFunc := func(v int) (int, string) { return v, strconv.Itoa(v) }
			result := slice.ToMap([]int{}, transformFunc)
			Expect(result).To(BeEmpty())
		})

		It("returns a map with the result of the transform function", func() {
			transformFunc := func(v int) (int, string) { return v * 2, strconv.Itoa(v * 2) }
			result := slice.ToMap([]int{1, 2, 3, 4}, transformFunc)
			Expect(result).To(HaveLen(4))
			Expect(result).To(HaveKeyWithValue(2, "2"))
			Expect(result).To(HaveKeyWithValue(4, "4"))
			Expect(result).To(HaveKeyWithValue(6, "6"))
			Expect(result).To(HaveKeyWithValue(8, "8"))
		})
	})

	Describe("CompactByFrequency", func() {
		It("returns empty slice for an empty input", func() {
			Expect(slice.CompactByFrequency([]int{})).To(BeEmpty())
		})

		It("groups by frequency", func() {
			Expect(slice.CompactByFrequency([]int{1, 2, 1, 2, 3, 2})).To(HaveExactElements(2, 1, 3))
		})
	})

	Describe("MostFrequent", func() {
		It("returns zero value if no arguments are passed", func() {
			Expect(slice.MostFrequent([]int{})).To(BeZero())
		})

		It("returns the single item", func() {
			Expect(slice.MostFrequent([]string{"123"})).To(Equal("123"))
		})
		It("returns the item that appeared more times", func() {
			Expect(slice.MostFrequent([]string{"1", "2", "1", "2", "3", "2"})).To(Equal("2"))
		})
		It("ignores zero values", func() {
			Expect(slice.MostFrequent([]int{0, 0, 0, 2, 2})).To(Equal(2))
		})
	})

	Describe("Move", func() {
		It("moves item to end of slice", func() {
			Expect(slice.Move([]string{"1", "2", "3"}, 0, 2)).To(HaveExactElements("2", "3", "1"))
		})
		It("moves item to beginning of slice", func() {
			Expect(slice.Move([]string{"1", "2", "3"}, 2, 0)).To(HaveExactElements("3", "1", "2"))
		})
		It("keeps item in same position if srcIndex == dstIndex", func() {
			Expect(slice.Move([]string{"1", "2", "3"}, 1, 1)).To(HaveExactElements("1", "2", "3"))
		})
	})

	Describe("BreakUp", func() {
		It("returns no chunks if slice is empty", func() {
			var s []string
			chunks := slice.BreakUp(s, 10)
			Expect(chunks).To(HaveLen(0))
		})
		It("returns the slice in one chunk if len < chunkSize", func() {
			s := []string{"a", "b", "c"}
			chunks := slice.BreakUp(s, 10)
			Expect(chunks).To(HaveLen(1))
			Expect(chunks[0]).To(HaveExactElements("a", "b", "c"))
		})
		It("breaks up the slice if len > chunkSize", func() {
			s := []string{"a", "b", "c", "d", "e"}
			chunks := slice.BreakUp(s, 3)
			Expect(chunks).To(HaveLen(2))
			Expect(chunks[0]).To(HaveExactElements("a", "b", "c"))
			Expect(chunks[1]).To(HaveExactElements("d", "e"))
		})
	})
})
