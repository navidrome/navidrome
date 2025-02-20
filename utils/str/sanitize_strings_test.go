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
			Expect(str.SanitizeStrings(" some  text  ", "text some")).To(Equal("some text"))
		})

		It("remove duplicated words", func() {
			Expect(str.SanitizeStrings("legião urbana", "urbana legiÃo")).To(Equal("legiao urbana"))
		})

		It("remove symbols", func() {
			Expect(str.SanitizeStrings("Tom’s Diner ' “40” ‘A’")).To(Equal("40 a diner toms"))
		})

		It("remove opening brackets", func() {
			Expect(str.SanitizeStrings("[Five Years]")).To(Equal("five years"))
		})

		It("remove slashes", func() {
			Expect(str.SanitizeStrings("folder/file\\yyyy")).To(Equal("file folder yyyy"))
		})

		It("normalizes utf chars", func() {
			// These uses different types of hyphens
			Expect(str.SanitizeStrings("k—os", "k−os")).To(Equal("k-os"))
		})

		It("remove commas", func() {
			// This is specially useful for handling cases where the Sort field uses comma.
			// It reduces the size of the resulting string, thus reducing the size of the DB table and indexes.
			Expect(str.SanitizeStrings("Bob Marley", "Marley, Bob")).To(Equal("bob marley"))
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

	Describe("RemoveArticle", func() {
		Context("Empty articles list", func() {
			BeforeEach(func() {
				conf.Server.IgnoredArticles = ""
			})
			It("returns empty if string is empty", func() {
				Expect(str.RemoveArticle("")).To(BeEmpty())
			})
			It("returns same string", func() {
				Expect(str.RemoveArticle("The Beatles")).To(Equal("The Beatles"))
			})
		})
		Context("Default articles", func() {
			BeforeEach(func() {
				conf.Server.IgnoredArticles = "The El La Los Las Le Les Os As O A"
			})
			It("returns empty if string is empty", func() {
				Expect(str.RemoveArticle("")).To(BeEmpty())
			})
			It("remove prefix article from string", func() {
				Expect(str.RemoveArticle("Os Paralamas do Sucesso")).To(Equal("Paralamas do Sucesso"))
			})
			It("does not remove article if it is part of the first word", func() {
				Expect(str.RemoveArticle("Thelonious Monk")).To(Equal("Thelonious Monk"))
			})
		})
	})
})
