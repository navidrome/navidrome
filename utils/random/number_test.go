package random_test

import (
	"testing"

	"github.com/navidrome/navidrome/utils/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRandom(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Random Suite")
}

var _ = Describe("number package", func() {
	Describe("Int64N", func() {
		It("should return a random int64", func() {
			for i := 0; i < 10000; i++ {
				Expect(random.Int64N(100)).To(BeNumerically("<", 100))
			}
		})
	})
})
