package utils

import (
	"github.com/deluan/navidrome/conf"
	. "github.com/onsi/ginkgo"
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

	Describe("StringInSlice", func() {
		It("returns false if slice is empty", func() {
			Expect(StringInSlice("test", nil)).To(BeFalse())
		})

		It("returns false if string is not found in slice", func() {
			Expect(StringInSlice("aaa", []string{"bbb", "ccc"})).To(BeFalse())
		})

		It("returns true if string is found in slice", func() {
			Expect(StringInSlice("bbb", []string{"bbb", "aaa", "ccc"})).To(BeTrue())
		})
	})

	Describe("MoveString", func() {
		It("moves item to end of slice", func() {
			Expect(MoveString([]string{"1", "2", "3"}, 0, 2)).To(ConsistOf("2", "3", "1"))
		})
		It("moves item to beginning of slice", func() {
			Expect(MoveString([]string{"1", "2", "3"}, 2, 0)).To(ConsistOf("3", "1", "2"))
		})
		It("keeps item in same position if srcIndex == dstIndex", func() {
			Expect(MoveString([]string{"1", "2", "3"}, 1, 1)).To(ConsistOf("1", "2", "3"))
		})
	})
})
