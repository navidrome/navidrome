package scanner

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("TagScanner", func() {
	dirFS := os.DirFS(".")

	Describe("loadAllAudioFiles", func() {
		It("return all audio files from the folder", func() {
			files, err := loadAllAudioFiles(dirFS, "tests/fixtures")
			Expect(err).ToNot(HaveOccurred())
			Expect(files).To(HaveLen(3))
			Expect(files).To(HaveKey("tests/fixtures/test.ogg"))
			Expect(files).To(HaveKey("tests/fixtures/test.mp3"))
			Expect(files).To(HaveKey("tests/fixtures/01 Invisible (RED) Edit Version.mp3"))
			Expect(files).ToNot(HaveKey("tests/fixtures/._02 Invisible.mp3"))
			Expect(files).ToNot(HaveKey("tests/fixtures/playlist.m3u"))
		})

		It("returns error if path does not exist", func() {
			_, err := loadAllAudioFiles(dirFS, "./INVALID/PATH")
			Expect(err).To(HaveOccurred())
		})

		It("returns empty map if there are no audio files in path", func() {
			Expect(loadAllAudioFiles(dirFS, "tests/fixtures/empty_folder")).To(BeEmpty())
		})
	})
})
