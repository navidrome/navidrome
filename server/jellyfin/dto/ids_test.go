package dto

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("id codec", func() {
	It("round-trips a base62 nanoid through Encode/Decode", func() {
		id := "5QFKvMsJrd57QE2Le2dKKo"
		Expect(DecodeID(EncodeID(id))).To(Equal(id))
	})

	It("passes a raw (non-hex) id through DecodeID unchanged", func() {
		Expect(DecodeID("5QFKvMsJrd57QE2Le2dKKo")).To(Equal("5QFKvMsJrd57QE2Le2dKKo"))
	})

	It("produces valid lowercase hex", func() {
		encoded := EncodeID("song-1")
		Expect(encoded).To(MatchRegexp("^[0-9a-f]+$"))
		Expect(encoded).To(HaveLen(len("song-1") * 2))
	})

	It("round-trips the empty string", func() {
		Expect(EncodeID("")).To(Equal(""))
		Expect(DecodeID("")).To(Equal(""))
	})

	It("decodes a hex-looking raw id incorrectly only when re-encoded consistently (encode/decode is always internally consistent)", func() {
		// "a1" happens to be valid hex on its own; DecodeID can't tell a coincidental hex
		// string apart from one we encoded. Callers must always encode ids on emission and
		// decode them on receipt so this ambiguity never surfaces in practice.
		Expect(DecodeID(EncodeID("a1"))).To(Equal("a1"))
	})
})
