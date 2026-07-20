package id_test

import (
	"github.com/navidrome/navidrome/model/id"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Encode128/Decode128", func() {
	It("encodes 16 bytes as 22-char zero-padded base62", func() {
		Expect(id.Encode128([16]byte{})).To(Equal("0000000000000000000000"))
		allFF := [16]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
		Expect(id.Encode128(allFF)).To(Equal("7N42dgm5tFLK9N8MT7fHC7"))
	})

	It("round-trips arbitrary 16-byte values", func() {
		b := [16]byte{0xe3, 0xb7, 0xfc, 0x2a, 0xe9, 0x44, 0x7b, 0xbe,
			0xc3, 0x7a, 0x13, 0xbf, 0x91, 0x6e, 0x3c, 0xf6}
		s := id.Encode128(b)
		Expect(s).To(Equal("6VHl3uR4kss6sUPKA8Cwnk"))
		Expect(id.Decode128(s)).To(Equal(b[:]))
	})

	It("rejects invalid input", func() {
		_, err := id.Decode128("short")
		Expect(err).To(HaveOccurred())
		_, err = id.Decode128("!!!!!!!!!!!!!!!!!!!!!!") // 22 chars, not base62
		Expect(err).To(HaveOccurred())
		_, err = id.Decode128("-000000000000000000001") // sign is not part of the alphabet
		Expect(err).To(HaveOccurred())
		_, err = id.Decode128("zzzzzzzzzzzzzzzzzzzzzz") // > 2^128
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("NewRandom", func() {
	It("emits 22-char canonical ids that always fit 128 bits", func() {
		seen := make(map[string]struct{})
		for range 1000 {
			s := id.NewRandom()
			Expect(s).To(HaveLen(22))
			_, err := id.Decode128(s)
			Expect(err).ToNot(HaveOccurred(), "id %q must decode to 128 bits", s)
			seen[s] = struct{}{}
		}
		Expect(seen).To(HaveLen(1000))
	})
})

var _ = Describe("NewHash", func() {
	It("keeps its historical output byte-for-byte (golden)", func() {
		Expect(id.NewHash("test")).To(Equal("5cLJPkLA5DK2BADhoeotPk"))
		Expect(id.NewHash("[unknown artist]")).To(Equal("7lsE5pS09fPS1VuFqwXbia"))
		Expect(id.NewTagID("genre", "electronic")).To(Equal("7bLYq0Np81m1Wgy5N31nuG"))
	})

	It("always emits 22 decodable chars", func() {
		h := id.NewHash("anything", "at", "all")
		Expect(h).To(HaveLen(22))
		_, err := id.Decode128(h)
		Expect(err).ToNot(HaveOccurred())
	})
})
