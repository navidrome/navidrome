package utils

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Files", func() {
	Describe("IsAudioFile", func() {
		It("returns true for a MP3 file", func() {
			Expect(IsAudioFile(filepath.Join("path", "to", "test.mp3"))).To(BeTrue())
		})

		It("returns true for a FLAC file", func() {
			Expect(IsAudioFile("test.flac")).To(BeTrue())
		})

		It("returns false for a non-audio file", func() {
			Expect(IsAudioFile("test.jpg")).To(BeFalse())
		})

		It("returns false for m3u files", func() {
			Expect(IsAudioFile("test.m3u")).To(BeFalse())
		})

		It("returns false for pls files", func() {
			Expect(IsAudioFile("test.pls")).To(BeFalse())
		})
	})

	Describe("IsImageFile", func() {
		It("returns true for a PNG file", func() {
			Expect(IsImageFile(filepath.Join("path", "to", "test.png"))).To(BeTrue())
		})

		It("returns true for a JPEG file", func() {
			Expect(IsImageFile("test.JPEG")).To(BeTrue())
		})

		It("returns false for a non-image file", func() {
			Expect(IsImageFile("test.mp3")).To(BeFalse())
		})
	})
})
