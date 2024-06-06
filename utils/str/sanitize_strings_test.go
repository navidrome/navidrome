package str_test

import (
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/utils/str"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sanitize Strings", func() {
	Describe("SanitizeStrings", func() {
		It("returns all lowercase chars", func() {
			Expect(str.SanitizeStrings("Some Text")).To(Equal("some text"))
		})

		It("removes accents", func() {
			Expect(str.SanitizeStrings("Quintão")).To(Equal("quintao"))
		})

		It("remove extra spaces", func() {
			Expect(str.SanitizeStrings(" some  text  ")).To(Equal("some text"))
		})

		It("remove duplicated words", func() {
			Expect(str.SanitizeStrings("legião urbana urbana legiÃo")).To(Equal("legiao urbana"))
		})

		It("remove symbols", func() {
			Expect(str.SanitizeStrings("Tom’s Diner ' “40” ‘A’")).To(Equal("40 a diner toms"))
		})

		It("remove opening brackets", func() {
			Expect(str.SanitizeStrings("[Five Years]")).To(Equal("five years"))
		})
	})

	Describe("SanitizeFieldForSorting", func() {
		BeforeEach(func() {
			conf.Server.IgnoredArticles = "The O"
		})
		It("sanitize accents", func() {
			Expect(str.SanitizeFieldForSorting("Céu")).To(Equal("ceu"))
		})
		It("removes articles", func() {
			Expect(str.SanitizeFieldForSorting("The Beatles")).To(Equal("the beatles"))
		})
		It("removes accented articles", func() {
			Expect(str.SanitizeFieldForSorting("Õ Blésq Blom")).To(Equal("o blesq blom"))
		})
	})

	Describe("SanitizeFieldForSortingNoArticle", func() {
		BeforeEach(func() {
			conf.Server.IgnoredArticles = "The O"
		})
		It("sanitize accents", func() {
			Expect(str.SanitizeFieldForSortingNoArticle("Céu")).To(Equal("ceu"))
		})
		It("removes articles", func() {
			Expect(str.SanitizeFieldForSortingNoArticle("The Beatles")).To(Equal("beatles"))
		})
		It("removes accented articles", func() {
			Expect(str.SanitizeFieldForSortingNoArticle("Õ Blésq Blom")).To(Equal("blesq blom"))
		})
	})
})
