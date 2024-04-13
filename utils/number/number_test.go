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

var _ = Describe("RandomInt64", func() {
	It("should return a random int64", func() {
		for i := 0; i < 10000; i++ {
			Expect(number.RandomInt64(100)).To(BeNumerically("<", 100))
		}
	})
})
