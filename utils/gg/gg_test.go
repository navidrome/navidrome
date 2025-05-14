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

var _ = Describe("IfZero", func() {
	DescribeTable("string",
		func(v, orElse, expected string) {
			Expect(gg.IfZero(v, orElse)).To(Equal(expected))
		},
		Entry("zero value", "", "default", "default"),
		Entry("non-zero value", "anything", "default", "anything"),
	)
	DescribeTable("numeric",
		func(v, orElse, expected int) {
			Expect(gg.IfZero(v, orElse)).To(Equal(expected))
		},
		Entry("zero value", 0, 2, 2),
		Entry("non-zero value", -1, 2, -1),
	)
	type testStruct struct {
		field1 int
	}
	DescribeTable("struct",
		func(v, orElse, expected testStruct) {
			Expect(gg.IfZero(v, orElse)).To(Equal(expected))
		},
		Entry("zero value", testStruct{}, testStruct{123}, testStruct{123}),
		Entry("non-zero value", testStruct{456}, testStruct{123}, testStruct{456}),
	)
})
