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
	Describe("If", func() {
		DescribeTable("string",
			func(v, orElse, expected string) {
				Expect(gg.If(v, orElse)).To(Equal(expected))
			},
			Entry("zero value", "", "default", "default"),
			Entry("non-zero value", "anything", "default", "anything"),
		)
		DescribeTable("numeric",
			func(v, orElse, expected int) {
				Expect(gg.If(v, orElse)).To(Equal(expected))
			},
			Entry("zero value", 0, 2, 2),
			Entry("non-zero value", -1, 2, -1),
		)
		type testStruct struct {
			field1 int
		}
		DescribeTable("struct",
			func(v, orElse, expected testStruct) {
				Expect(gg.If(v, orElse)).To(Equal(expected))
			},
			Entry("zero value", testStruct{}, testStruct{123}, testStruct{123}),
			Entry("non-zero value", testStruct{456}, testStruct{123}, testStruct{456}),
		)
	})

	Describe("FirstOr", func() {
		Context("when given a list of strings", func() {
			It("returns the first non-empty value", func() {
				Expect(gg.FirstOr("default", "foo", "bar", "baz")).To(Equal("foo"))
				Expect(gg.FirstOr("default", "", "", "qux")).To(Equal("qux"))
			})

			It("returns the default value if all values are empty", func() {
				Expect(gg.FirstOr("default", "", "", "")).To(Equal("default"))
				Expect(gg.FirstOr("", "", "", "")).To(Equal(""))
			})
		})
	})

	Describe("P", func() {
		It("returns a pointer to the input value", func() {
			v := 123
			Expect(gg.P(123)).To(Equal(&v))
		})

		It("returns nil if the input value is zero", func() {
			v := 0
			Expect(gg.P(0)).To(Equal(&v))
		})
	})

	Describe("V", func() {
		It("returns the value of the input pointer", func() {
			v := 123
			Expect(gg.V(&v)).To(Equal(123))
		})

		It("returns a zero value if the input pointer is nil", func() {
			var v *int
			Expect(gg.V(v)).To(Equal(0))
		})
	})
})
