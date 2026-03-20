package nanoid_test

import (
	"testing"

	"github.com/navidrome/navidrome/utils/nanoid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestNanoid(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Nanoid Suite")
}

var _ = Describe("Generate", func() {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	It("generates a string of the requested length", func() {
		id, err := nanoid.Generate(alphabet, 22)
		Expect(err).ToNot(HaveOccurred())
		Expect(id).To(HaveLen(22))
	})

	It("generates a short string of the requested length", func() {
		id, err := nanoid.Generate(alphabet, 10)
		Expect(err).ToNot(HaveOccurred())
		Expect(id).To(HaveLen(10))
	})

	It("only contains characters from the alphabet", func() {
		id, err := nanoid.Generate(alphabet, 100)
		Expect(err).ToNot(HaveOccurred())
		for _, c := range id {
			Expect(alphabet).To(ContainSubstring(string(c)))
		}
	})

	It("generates unique IDs", func() {
		seen := make(map[string]bool)
		for range 1000 {
			id, err := nanoid.Generate(alphabet, 22)
			Expect(err).ToNot(HaveOccurred())
			Expect(seen).ToNot(HaveKey(id))
			seen[id] = true
		}
	})

	It("works with a single-character alphabet", func() {
		id, err := nanoid.Generate("a", 5)
		Expect(err).ToNot(HaveOccurred())
		Expect(id).To(Equal("aaaaa"))
	})

	It("works with a small alphabet", func() {
		id, err := nanoid.Generate("ab", 10)
		Expect(err).ToNot(HaveOccurred())
		Expect(id).To(HaveLen(10))
		for _, c := range id {
			Expect(string(c)).To(BeElementOf("a", "b"))
		}
	})

	It("returns error on empty alphabet", func() {
		_, err := nanoid.Generate("", 10)
		Expect(err).To(HaveOccurred())
	})

	It("returns error on alphabet larger than 255 characters", func() {
		bigAlphabet := make([]byte, 256)
		for i := range bigAlphabet {
			bigAlphabet[i] = byte(i)
		}
		_, err := nanoid.Generate(string(bigAlphabet), 10)
		Expect(err).To(HaveOccurred())
	})

	It("returns error on non-positive size", func() {
		_, err := nanoid.Generate(alphabet, 0)
		Expect(err).To(HaveOccurred())

		_, err = nanoid.Generate(alphabet, -1)
		Expect(err).To(HaveOccurred())
	})
})
