package utils

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SanitizeStrings", func() {
	It("returns all lowercase chars", func() {
		Expect(SanitizeStrings("Some Text")).To(Equal("some text"))
	})

	It("removes accents", func() {
		Expect(SanitizeStrings("Quintão")).To(Equal("quintao"))
	})

	It("remove extra spaces", func() {
		Expect(SanitizeStrings(" some  text  ")).To(Equal("some text"))
	})

	It("remove duplicated words", func() {
		Expect(SanitizeStrings("legião urbana urbana legiÃo")).To(Equal("legiao urbana"))
	})

	It("remove symbols", func() {
		Expect(SanitizeStrings("Tom’s Diner ' “40” ‘A’")).To(Equal("40 a diner toms"))
	})

	It("remove opening brackets", func() {
		Expect(SanitizeStrings("[Five Years]")).To(Equal("five years"))
	})
})
