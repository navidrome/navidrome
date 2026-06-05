package stream

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Aliases", func() {
	Describe("IsAACCodec", func() {
		It("returns true for AAC and its aliases", func() {
			Expect(IsAACCodec("aac")).To(BeTrue())
			Expect(IsAACCodec("AAC")).To(BeTrue())
			Expect(IsAACCodec("adts")).To(BeTrue())
			Expect(IsAACCodec("m4a")).To(BeTrue())
			Expect(IsAACCodec("mp4")).To(BeTrue())
			Expect(IsAACCodec("m4b")).To(BeTrue())
		})

		It("returns false for non-AAC formats", func() {
			Expect(IsAACCodec("mp3")).To(BeFalse())
			Expect(IsAACCodec("opus")).To(BeFalse())
			Expect(IsAACCodec("flac")).To(BeFalse())
			Expect(IsAACCodec("ogg")).To(BeFalse())
		})

		It("returns false for empty string", func() {
			Expect(IsAACCodec("")).To(BeFalse())
		})
	})
})
