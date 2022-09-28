package utils

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HashFunc", func() {
	const input = "123e4567e89b12d3a456426614174000"

	It("hashes the input and returns the sum", func() {
		hashFunc := Hasher.HashFunc()
		sum := hashFunc(input)
		Expect(sum > 0).To(BeTrue())
	})

	It("hashes the input, reseeds and returns a different sum", func() {
		hashFunc := Hasher.HashFunc()
		sum := hashFunc(input)
		Hasher.Reseed()
		sum2 := hashFunc(input)
		Expect(sum).NotTo(Equal(sum2))
	})
})
