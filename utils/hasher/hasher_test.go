package hasher_test

import (
	"strconv"
	"testing"

	"github.com/navidrome/navidrome/utils/hasher"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHasher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Hasher Suite")
}

var _ = Describe("HashFunc", func() {
	const input = "123e4567e89b12d3a456426614174000"

	It("hashes the input and returns the sum", func() {
		hashFunc := hasher.HashFunc()
		sum := hashFunc("1", input)
		Expect(sum > 0).To(BeTrue())
	})

	It("hashes the input, reseeds and returns a different sum", func() {
		hashFunc := hasher.HashFunc()
		sum := hashFunc("1", input)
		hasher.Reseed("1")
		sum2 := hashFunc("1", input)
		Expect(sum).NotTo(Equal(sum2))
	})

	It("keeps different hashes for different ids", func() {
		hashFunc := hasher.HashFunc()
		sum := hashFunc("1", input)
		sum2 := hashFunc("2", input)

		Expect(sum).NotTo(Equal(sum2))

		Expect(sum).To(Equal(hashFunc("1", input)))
		Expect(sum2).To(Equal(hashFunc("2", input)))
	})

	It("keeps the same hash for the same id and seed", func() {
		id := "1"
		hashFunc := hasher.HashFunc()
		hasher.SetSeed(id, "original_seed")
		sum := hashFunc(id, input)
		Expect(sum).To(Equal(hashFunc(id, input)))

		hasher.Reseed(id)
		Expect(sum).NotTo(Equal(hashFunc(id, input)))

		hasher.SetSeed(id, "original_seed")
		Expect(sum).To(Equal(hashFunc(id, input)))
	})

	It("does not cause race conditions", func() {
		for i := 0; i < 1000; i++ {
			go func() {
				hashFunc := hasher.HashFunc()
				sum := hashFunc(strconv.Itoa(i), input)
				Expect(sum).ToNot(BeZero())
			}()
		}
	})
})
