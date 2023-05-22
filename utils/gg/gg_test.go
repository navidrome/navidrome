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

	Describe("Coalesce", func() {
		Context("when given a list of strings", func() {
			It("returns the first non-empty value", func() {
				Expect(gg.Coalesce("foo", "bar", "baz", "default")).To(Equal("foo"))
				Expect(gg.Coalesce("", "", "qux", "default")).To(Equal("qux"))
			})

			It("returns the default value if all values are empty", func() {
				Expect(gg.Coalesce("", "", "", "default")).To(Equal("default"))
				Expect(gg.Coalesce("", "", "", "")).To(Equal(""))
			})
		})
	})

	Describe("P", func() {
		Context("when given a non-zero value", func() {
			It("should return a non-nil pointer to that value", func() {
				value := 42
				result := gg.P(value)
				Expect(result).ToNot(BeNil())
				Expect(*result).To(Equal(value))
			})
		})

		Context("when given the zero value of a type", func() {
			It("should return nil", func() {
				var value string
				result := gg.P(value)
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("V", func() {
		Context("when given a non-nil pointer", func() {
			It("should return the value of that pointer", func() {
				value := 42
				pointer := &value
				result := gg.V(pointer)
				Expect(result).To(Equal(value))
			})
		})

		Context("when given a nil pointer", func() {
			It("should return the zero value of the type", func() {
				var pointer *string
				result := gg.V(pointer)
				Expect(result).To(Equal(""))
			})
		})
	})
})
