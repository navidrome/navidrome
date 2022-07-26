package utils

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseIndexGroup", func() {
	Context("Two simple entries", func() {
		It("returns the entries", func() {
			parsed := ParseIndexGroups("A The")

			Expect(parsed).To(HaveLen(2))
			Expect(parsed["A"]).To(Equal("A"))
			Expect(parsed["The"]).To(Equal("The"))
		})
	})
	Context("An entry with a group", func() {
		parsed := ParseIndexGroups("A-C(ABC) Z")

		It("parses the groups correctly", func() {
			Expect(parsed).To(HaveLen(4))
			Expect(parsed["A"]).To(Equal("A-C"))
			Expect(parsed["B"]).To(Equal("A-C"))
			Expect(parsed["C"]).To(Equal("A-C"))
			Expect(parsed["Z"]).To(Equal("Z"))
		})
	})
	Context("Correctly parses UTF-8", func() {
		parsed := ParseIndexGroups("UTF8(宇A海)")
		It("parses the groups correctly", func() {
			Expect(parsed).To(HaveLen(3))
			Expect(parsed["宇"]).To(Equal("UTF8"))
			Expect(parsed["A"]).To(Equal("UTF8"))
			Expect(parsed["海"]).To(Equal("UTF8"))
		})
	})
})
