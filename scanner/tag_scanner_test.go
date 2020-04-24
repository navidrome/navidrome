package scanner

import (
	"github.com/deluan/navidrome/conf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("TagScanner", func() {
	Describe("sanitizeFieldForSorting", func() {
		BeforeEach(func() {
			conf.Server.IgnoredArticles = "The"
		})
		It("sanitize accents", func() {
			Expect(sanitizeFieldForSorting("Céu")).To(Equal("Ceu"))
		})
		It("removes articles", func() {
			Expect(sanitizeFieldForSorting("The Beatles")).To(Equal("Beatles"))
		})
	})
})
