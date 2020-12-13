package persistence

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("sqlRepository", func() {
	Describe("getFullText", func() {
		It("prefixes with a space", func() {
			Expect(getFullText("legiao urbana")).To(Equal(" legiao urbana"))
		})
	})
})
