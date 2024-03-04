package scanner

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/navidrome/navidrome/consts"
)

var _ = Describe("TagScanner", func() {
	Describe("loadAllAudioFiles", func() {
		It("return all audio files from the folder (CUE support disabled)", func() {
			files, err := loadAllAudioFiles("tests/fixtures", consts.CUEDisable)
			Expect(err).ToNot(HaveOccurred())
			Expect(files).To(HaveLen(16))
			Expect(files).To(HaveKey("tests/fixtures/sample.ape"))
			Expect(files).To(HaveKey("tests/fixtures/sample.wv"))
			Expect(files).To(HaveKey("tests/fixtures/test.aiff"))
			Expect(files).To(HaveKey("tests/fixtures/test.flac"))
			Expect(files).To(HaveKey("tests/fixtures/test.m4a"))
			Expect(files).To(HaveKey("tests/fixtures/test.mp3"))
			Expect(files).To(HaveKey("tests/fixtures/test.tak"))
			Expect(files).To(HaveKey("tests/fixtures/test.ogg"))
			Expect(files).To(HaveKey("tests/fixtures/test.wav"))
			Expect(files).To(HaveKey("tests/fixtures/test.wma"))
			Expect(files).To(HaveKey("tests/fixtures/test.wv"))
			Expect(files).To(HaveKey("tests/fixtures/test_no_read_permission.ogg"))
			Expect(files).To(HaveKey("tests/fixtures/01 Invisible (RED) Edit Version.mp3"))
			Expect(files).To(HaveKey("tests/fixtures/01 Invisible (RED) Edit Version.m4a"))
			Expect(files).ToNot(HaveKey("tests/fixtures/._02 Invisible.mp3"))
			Expect(files).ToNot(HaveKey("tests/fixtures/playlist.m3u"))
			Expect(files).ToNot(HaveKey("tests/fixtures/invalid.cue"))
		})

		It("return all audio files from the folder (only embedded CUE support)", func() {
			files, err := loadAllAudioFiles("tests/fixtures", consts.CUEEmbedded)
			Expect(err).ToNot(HaveOccurred())
			Expect(files).To(HaveLen(16))
			Expect(files).ToNot(HaveKey("tests/fixtures/invalid.cue"))
		})

		It("return all audio files from the folder (only external CUE support)", func() {
			files, err := loadAllAudioFiles("tests/fixtures", consts.CUEExternal)
			Expect(err).ToNot(HaveOccurred())
			Expect(files).To(HaveLen(17))
			Expect(files).To(HaveKey("tests/fixtures/invalid.cue"))
		})

		It("return all audio files from the folder (PreferEmbedded) CUE support)", func() {
			files, err := loadAllAudioFiles("tests/fixtures", consts.CUEPreferEmbedded)
			Expect(err).ToNot(HaveOccurred())
			Expect(files).To(HaveLen(17))
			Expect(files).To(HaveKey("tests/fixtures/invalid.cue"))
		})

		It("return all audio files from the folder (CUEPreferEmbedded CUE support)", func() {
			files, err := loadAllAudioFiles("tests/fixtures", consts.CUEPreferEmbedded)
			Expect(err).ToNot(HaveOccurred())
			Expect(files).To(HaveLen(17))
			Expect(files).To(HaveKey("tests/fixtures/invalid.cue"))
		})

		It("returns error if path does not exist", func() {
			_, err := loadAllAudioFiles("./INVALID/PATH", consts.CUEDisable)
			Expect(err).To(HaveOccurred())
		})

		It("returns empty map if there are no audio files in path", func() {
			Expect(loadAllAudioFiles("tests/fixtures/empty_folder", consts.CUEDisable)).To(BeEmpty())
		})
	})
})
