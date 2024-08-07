package persistence

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sqlRepository", func() {
	Describe("formatFullText", func() {
		It("prefixes with a space", func() {
			Expect(formatFullText("legiao urbana")).To(Equal(" legiao urbana"))
		})
	})
})
