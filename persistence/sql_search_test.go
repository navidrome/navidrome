package persistence

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("sqlRepository", func() {
	var sqlRepository = &sqlRepository{}

	Describe("getFullText", func() {
		It("returns all lowercase chars", func() {
			Expect(sqlRepository.getFullText("Some Text")).To(Equal("some text"))
		})

		It("removes accents", func() {
			Expect(sqlRepository.getFullText("Quintão")).To(Equal("quintao"))
		})

		It("remove extra spaces", func() {
			Expect(sqlRepository.getFullText(" some  text  ")).To(Equal("some text"))
		})

		It("remove duplicated words", func() {
			Expect(sqlRepository.getFullText("legião urbana urbana legiÃo")).To(Equal("legiao urbana"))
		})
	})
})
