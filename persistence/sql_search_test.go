package persistence

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("sqlRepository", func() {
	Describe("getFullText", func() {
		It("returns all lowercase chars", func() {
			Expect(getFullText("Some Text")).To(Equal(" some text"))
		})

		It("removes accents", func() {
			Expect(getFullText("Quintão")).To(Equal(" quintao"))
		})

		It("remove extra spaces", func() {
			Expect(getFullText(" some  text  ")).To(Equal(" some text"))
		})

		It("remove duplicated words", func() {
			Expect(getFullText("legião urbana urbana legiÃo")).To(Equal(" legiao urbana"))
		})

		It("remove symbols", func() {
			Expect(getFullText("Tom’s Diner ' “40” ‘A’")).To(Equal(" 40 a diner toms"))
		})

		It("remove opening brackets", func() {
			Expect(getFullText("[Five Years]")).To(Equal(" five years"))
		})
	})
})
