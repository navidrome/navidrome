package model_test

import (
	"path/filepath"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("File Types()", func() {
	Describe("IsAudioFile", func() {
		It("returns true for a MP3 file", func() {
			Expect(model.IsAudioFile(filepath.Join("path", "to", "test.mp3"))).To(BeTrue())
		})

		It("returns true for a FLAC file", func() {
			Expect(model.IsAudioFile("test.flac")).To(BeTrue())
		})

		It("returns false for a non-audio file", func() {
			Expect(model.IsAudioFile("test.jpg")).To(BeFalse())
		})

		It("returns false for m3u files", func() {
			Expect(model.IsAudioFile("test.m3u")).To(BeFalse())
		})

		It("returns false for pls files", func() {
			Expect(model.IsAudioFile("test.pls")).To(BeFalse())
		})
	})

	Describe("IsImageFile()", func() {
		It("returns true for a PNG file", func() {
			Expect(model.IsImageFile(filepath.Join("path", "to", "test.png"))).To(BeTrue())
		})

		It("returns true for a JPEG file", func() {
			Expect(model.IsImageFile("test.JPEG")).To(BeTrue())
		})

		It("returns false for a non-image file", func() {
			Expect(model.IsImageFile("test.mp3")).To(BeFalse())
		})
	})

	Describe("IsValidPlaylist()", func() {
		It("returns true for a M3U file", func() {
			Expect(model.IsValidPlaylist(filepath.Join("path", "to", "test.m3u"))).To(BeTrue())
		})

		It("returns true for a M3U8 file", func() {
			Expect(model.IsValidPlaylist(filepath.Join("path", "to", "test.m3u8"))).To(BeTrue())
		})

		It("returns false for a non-playlist file", func() {
			Expect(model.IsValidPlaylist("testm3u")).To(BeFalse())
		})
	})
})
