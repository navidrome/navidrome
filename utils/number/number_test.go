package number_test

import (
	"testing"

	"github.com/navidrome/navidrome/utils/number"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestNumber(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Number Suite")
}

var _ = Describe("number package", func() {
	Describe("RandomInt64", func() {
		It("should return a random int64", func() {
			for i := 0; i < 10000; i++ {
				Expect(number.RandomInt64(100)).To(BeNumerically("<", 100))
			}
		})
	})

	Describe("ParseInt", func() {
		It("should parse a string into an int", func() {
			Expect(number.ParseInt[int64]("123")).To(Equal(int64(123)))
		})
		It("should parse a string into an int32", func() {
			Expect(number.ParseInt[int32]("123")).To(Equal(int32(123)))
		})
		It("should parse a string into an int64", func() {
			Expect(number.ParseInt[int]("123")).To(Equal(123))
		})
		It("should parse a string into an uint", func() {
			Expect(number.ParseInt[uint]("123")).To(Equal(uint(123)))
		})
	})
})
