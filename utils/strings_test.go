package utils

import (
	"strings"

	"github.com/navidrome/navidrome/conf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Strings", func() {
	Describe("NoArticle", func() {
		Context("Empty articles list", func() {
			BeforeEach(func() {
				conf.Server.IgnoredArticles = ""
			})
			It("returns empty if string is empty", func() {
				Expect(NoArticle("")).To(BeEmpty())
			})
			It("returns same string", func() {
				Expect(NoArticle("The Beatles")).To(Equal("The Beatles"))
			})
		})
		Context("Default articles", func() {
			BeforeEach(func() {
				conf.Server.IgnoredArticles = "The El La Los Las Le Les Os As O A"
			})
			It("returns empty if string is empty", func() {
				Expect(NoArticle("")).To(BeEmpty())
			})
			It("remove prefix article from string", func() {
				Expect(NoArticle("Os Paralamas do Sucesso")).To(Equal("Paralamas do Sucesso"))
			})
			It("does not remove article if it is part of the first word", func() {
				Expect(NoArticle("Thelonious Monk")).To(Equal("Thelonious Monk"))
			})
		})
	})

	Describe("BreakUpStringSlice", func() {
		It("returns no chunks if slice is empty", func() {
			var slice []string
			chunks := BreakUpStringSlice(slice, 10)
			Expect(chunks).To(HaveLen(0))
		})
		It("returns the slice in one chunk if len < chunkSize", func() {
			slice := []string{"a", "b", "c"}
			chunks := BreakUpStringSlice(slice, 10)
			Expect(chunks).To(HaveLen(1))
			Expect(chunks[0]).To(ConsistOf("a", "b", "c"))
		})
		It("breaks up the slice if len > chunkSize", func() {
			slice := []string{"a", "b", "c", "d", "e"}
			chunks := BreakUpStringSlice(slice, 3)
			Expect(chunks).To(HaveLen(2))
			Expect(chunks[0]).To(ConsistOf("a", "b", "c"))
			Expect(chunks[1]).To(ConsistOf("d", "e"))
		})
	})

	Describe("LongestCommonPrefix", func() {
		var testPaths = []string{
			"/Music/iTunes 1/iTunes Media/Music/ABBA/Gold_ Greatest Hits/Dancing Queen.m4a",
			"/Music/iTunes 1/iTunes Media/Music/ABBA/Gold_ Greatest Hits/Mamma Mia.m4a",
			"/Music/iTunes 1/iTunes Media/Music/Bachman-Turner Overdrive/Gold/Down Down.m4a",
			"/Music/iTunes 1/iTunes Media/Music/Bachman-Turner Overdrive/Gold/Hey You.m4a",
			"/Music/iTunes 1/iTunes Media/Music/Bachman-Turner Overdrive/Gold/Hold Back The Water.m4a",
			"/Music/iTunes 1/iTunes Media/Music/Compilations/Saturday Night Fever/01 Stayin' Alive.m4a",
			"/Music/iTunes 1/iTunes Media/Music/Compilations/Saturday Night Fever/03 Night Fever.m4a",
			"/Music/iTunes 1/iTunes Media/Music/Yes/Fragile/01 Roundabout.m4a",
		}

		It("finds the longest common prefix", func() {
			Expect(LongestCommonPrefix(testPaths)).To(Equal("/Music/iTunes 1/iTunes Media/Music/"))
		})
	})

	Describe("SplitFunc", func() {
		DescribeTable("when splitting strings with a delimiter",
			func(delimiter rune, input string, expected []string) {
				splitFunc := SplitFunc(delimiter)
				actual := strings.FieldsFunc(input, splitFunc)
				Expect(actual).To(Equal(expected))
			},
			Entry("should split strings without parentheses", ',', "name,age,email", []string{"name", "age", "email"}),
			Entry("should not split strings within parentheses", ',', "name, substr(email, 0, 3), age", []string{"name", " substr(email, 0, 3)", " age"}),
			Entry("should handle multiple delimiters outside parentheses", ';', "name;age;email", []string{"name", "age", "email"}),
			Entry("should return the whole input as a single element if the delimiter is not found", ';', "name,age,email", []string{"name,age,email"}),
			Entry("should handle empty input", ',', "", []string{}),
			Entry("should handle input with only delimiters", ',', ",,,", []string{}),
		)
	})
})
