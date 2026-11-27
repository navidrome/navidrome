package gg_test

import (
	"testing"

	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/gg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGG(t *testing.T) {
	tests.Init(t, false)
	RegisterFailHandler(Fail)
	RunSpecs(t, "GG Suite")
}

var _ = Describe("GG", func() {
	Describe("V", func() {
		It("returns the value of the input pointer", func() {
			Expect(gg.V(new(123))).To(Equal(123))
		})

		It("returns a zero value if the input pointer is nil", func() {
			var v *int
			Expect(gg.V(v)).To(Equal(0))
		})
	})

	Describe("If", func() {
		It("returns the first value if the condition is true", func() {
			Expect(gg.If(true, 1, 2)).To(Equal(1))
		})

		It("returns the second value if the condition is false", func() {
			Expect(gg.If(false, 1, 2)).To(Equal(2))
		})

		It("works with string values", func() {
			Expect(gg.If(true, "a", "b")).To(Equal("a"))
			Expect(gg.If(false, "a", "b")).To(Equal("b"))
		})

		It("works with different types", func() {
			Expect(gg.If(true, 1.1, 2.2)).To(Equal(1.1))
			Expect(gg.If(false, 1.1, 2.2)).To(Equal(2.2))
		})
	})

	Describe("Clone", func() {
		It("returns a pointer to a copy of the value", func() {
			original := 123
			cloned := gg.Clone(&original)
			Expect(cloned).To(HaveValue(Equal(123)))
			Expect(cloned).NotTo(BeIdenticalTo(&original))
		})

		It("does not alias the original value", func() {
			original := 123
			cloned := gg.Clone(&original)
			original = 456
			Expect(*cloned).To(Equal(123))
		})

		It("returns nil when the input is nil", func() {
			var v *int
			Expect(gg.Clone(v)).To(BeNil())
		})
	})
})
