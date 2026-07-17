package dto

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("blurHash", func() {
	It("returns a 6-char valid blurhash starting with the 1x1 component prefix", func() {
		h := blurHash("x")
		Expect(h).To(HaveLen(6))
		Expect(h).To(HavePrefix("00"))
		for _, c := range h {
			Expect(strings.ContainsRune(base83Alphabet, c)).To(BeTrue(), "unexpected char %q", c)
		}
	})

	It("is deterministic for the same seed", func() {
		Expect(blurHash("cover-tag-1")).To(Equal(blurHash("cover-tag-1")))
	})

	It("differs for different seeds", func() {
		Expect(blurHash("cover-tag-1")).ToNot(Equal(blurHash("cover-tag-2")))
	})
})
