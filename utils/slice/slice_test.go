package slice_test

import (
	"os"
	"slices"
	"strconv"
	"testing"

	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/slice"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSlice(t *testing.T) {
	tests.Init(t, false)
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

	Describe("MapWithArg", func() {
		It("returns empty slice for an empty input", func() {
			mapFunc := func(a int, v int) string { return strconv.Itoa(a + v) }
			result := slice.MapWithArg([]int{}, 10, mapFunc)
			Expect(result).To(BeEmpty())
		})

		It("returns a new slice with elements mapped", func() {
			mapFunc := func(a int, v int) string { return strconv.Itoa(a + v) }
			result := slice.MapWithArg([]int{1, 2, 3, 4}, 10, mapFunc)
			Expect(result).To(ConsistOf("11", "12", "13", "14"))
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

	DescribeTable("LinesFrom",
		func(path string, expected int) {
			count := 0
			file, _ := os.Open(path)
			defer file.Close()
			for _ = range slice.LinesFrom(file) {
				count++
			}
			Expect(count).To(Equal(expected))
		},
		Entry("returns empty slice for an empty input", "tests/fixtures/empty.txt", 0),
		Entry("returns the lines of a file", "tests/fixtures/playlists/pls1.m3u", 3),
		Entry("returns empty if file does not exist", "tests/fixtures/NON-EXISTENT", 0),
	)

	DescribeTable("CollectChunks",
		func(input []int, n int, expected [][]int) {
			var result [][]int
			for chunks := range slice.CollectChunks(slices.Values(input), n) {
				result = append(result, chunks)
			}
			Expect(result).To(Equal(expected))
		},
		Entry("returns empty slice (nil) for an empty input", []int{}, 1, nil),
		Entry("returns the slice in one chunk if len < chunkSize", []int{1, 2, 3}, 10, [][]int{{1, 2, 3}}),
		Entry("breaks up the slice if len > chunkSize", []int{1, 2, 3, 4, 5}, 3, [][]int{{1, 2, 3}, {4, 5}}),
	)

	Describe("SeqFunc", func() {
		It("returns empty slice for an empty input", func() {
			it := slice.SeqFunc([]int{}, func(v int) int { return v })

			result := slices.Collect(it)
			Expect(result).To(BeEmpty())
		})

		It("returns a new slice with mapped elements", func() {
			it := slice.SeqFunc([]int{1, 2, 3, 4}, func(v int) string { return strconv.Itoa(v * 2) })

			result := slices.Collect(it)
			Expect(result).To(ConsistOf("2", "4", "6", "8"))
		})
	})
})
