package scanner

import (
	"github.com/deluan/navidrome/conf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("TagScanner", func() {
	Describe("sanitizeFieldForSorting", func() {
		BeforeEach(func() {
			conf.Server.IgnoredArticles = "The"
		})
		It("sanitize accents", func() {
			Expect(sanitizeFieldForSorting("Céu")).To(Equal("Ceu"))
		})
		It("removes articles", func() {
			Expect(sanitizeFieldForSorting("The Beatles")).To(Equal("Beatles"))
		})
	})

	Describe("LoadAllAudioFiles", func() {
		It("return all audio files from the folder", func() {
			files, err := LoadAllAudioFiles("tests/fixtures")
			Expect(err).ToNot(HaveOccurred())
			Expect(files).To(HaveLen(3))
			Expect(files).To(HaveKey("tests/fixtures/test.ogg"))
			Expect(files).To(HaveKey("tests/fixtures/test.mp3"))
			Expect(files).To(HaveKey("tests/fixtures/01 Invisible (RED) Edit Version.mp3"))
			Expect(files).ToNot(HaveKey("tests/fixtures/playlist.m3u"))
		})
		It("returns error if path does not exist", func() {
			_, err := LoadAllAudioFiles("./INVALID/PATH")
			Expect(err).To(HaveOccurred())
		})

		It("returns empty map if there are no audio files in path", func() {
			Expect(LoadAllAudioFiles("tests/fixtures/empty_folder")).To(BeEmpty())
		})
	})
})
