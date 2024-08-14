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
