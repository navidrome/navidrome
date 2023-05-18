package taglib

import (
	"io/fs"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Extractor", func() {
	var e *Extractor
	// This file will have 0222 (no read) permission during these tests
	var accessForbiddenFile = "tests/fixtures/test_no_read_permission.ogg"

	BeforeEach(func() {
		e = &Extractor{}

		err := os.Chmod(accessForbiddenFile, 0222)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() {
			err = os.Chmod(accessForbiddenFile, 0644)
			Expect(err).ToNot(HaveOccurred())
		})
	})
	Context("Parse", func() {
		It("correctly parses metadata from all files in folder", func() {
			mds, err := e.Parse(
				"tests/fixtures/test.mp3",
				"tests/fixtures/test.ogg",
				accessForbiddenFile,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(mds).To(HaveLen(2))
			Expect(mds).ToNot(HaveKey(accessForbiddenFile))

			m := mds["tests/fixtures/test.mp3"]
			Expect(m).To(HaveKeyWithValue("title", []string{"Song", "Song"}))
			Expect(m).To(HaveKeyWithValue("album", []string{"Album", "Album"}))
			Expect(m).To(HaveKeyWithValue("artist", []string{"Artist", "Artist"}))
			Expect(m).To(HaveKeyWithValue("albumartist", []string{"Album Artist"}))
			Expect(m).To(HaveKeyWithValue("tcmp", []string{"1"})) // Compilation
			Expect(m).To(HaveKeyWithValue("genre", []string{"Rock"}))
			Expect(m).To(HaveKeyWithValue("date", []string{"2014-05-21", "2014"}))
			Expect(m).To(HaveKeyWithValue("originaldate", []string{"1996-11-21"}))
			Expect(m).To(HaveKeyWithValue("releasedate", []string{"2020-12-31"}))
			Expect(m).To(HaveKeyWithValue("discnumber", []string{"1/2"}))
			Expect(m).To(HaveKeyWithValue("has_picture", []string{"true"}))
			Expect(m).To(HaveKeyWithValue("duration", []string{"1.02"}))
			Expect(m).To(HaveKeyWithValue("bitrate", []string{"192"}))
			Expect(m).To(HaveKeyWithValue("channels", []string{"2"}))
			Expect(m).To(HaveKeyWithValue("comment", []string{"Comment1\nComment2"}))
			Expect(m).To(HaveKeyWithValue("lyrics", []string{"Lyrics 1\rLyrics 2"}))
			Expect(m).To(HaveKeyWithValue("bpm", []string{"123"}))
			Expect(m).To(HaveKeyWithValue("tpub", []string{"Publisher"}))

			Expect(m).To(HaveKeyWithValue("tracknumber", []string{"2/10"}))
			m = m.Map(e.CustomMappings())
			Expect(m).To(HaveKeyWithValue("tracknumber", []string{"2/10", "2/10", "2"}))

			m = mds["tests/fixtures/test.ogg"]
			Expect(err).To(BeNil())
			Expect(m).ToNot(HaveKey("title"))
			Expect(m).ToNot(HaveKey("has_picture"))
			Expect(m).To(HaveKeyWithValue("duration", []string{"1.04"}))
			Expect(m).To(HaveKeyWithValue("fbpm", []string{"141.7"}))

			// TabLib 1.12 returns 18, previous versions return 39.
			// See https://github.com/taglib/taglib/commit/2f238921824741b2cfe6fbfbfc9701d9827ab06b
			Expect(m).To(HaveKey("bitrate"))
			Expect(m["bitrate"][0]).To(BeElementOf("18", "39", "40"))
		})

		DescribeTable("ReplayGain",
			func(file, albumGain, albumPeak, trackGain, trackPeak string) {
				file = "tests/fixtures/" + file
				mds, err := e.Parse(file)
				Expect(err).NotTo(HaveOccurred())
				Expect(mds).To(HaveLen(1))

				m := mds[file]

				Expect(m).To(HaveKeyWithValue("replaygain_album_gain", []string{albumGain}))
				Expect(m).To(HaveKeyWithValue("replaygain_album_peak", []string{albumPeak}))
				Expect(m).To(HaveKeyWithValue("replaygain_track_gain", []string{trackGain}))
				Expect(m).To(HaveKeyWithValue("replaygain_track_peak", []string{trackPeak}))
			},
			Entry("Correctly parses m4a (aac) gain tags", "01 Invisible (RED) Edit Version.m4a", "0.37", "0.48", "0.37", "0.48"),
			Entry("correctly parses mp3 tags", "test.mp3", "+3.21518 dB", "0.9125", "-1.48 dB", "0.4512"),
			Entry("correctly parses ogg (vorbis) tags", "test.ogg", "+7.64 dB", "0.11772506", "+7.64 dB", "0.11772506"),
		)
	})

	Context("Error Checking", func() {
		It("correctly handle unreadable file due to insufficient read permission", func() {
			_, err := e.extractMetadata(accessForbiddenFile)
			Expect(err).To(MatchError(os.ErrPermission))
		})
		It("returns a generic ErrPath if file does not exist", func() {
			testFilePath := "tests/fixtures/NON_EXISTENT.ogg"
			_, err := e.extractMetadata(testFilePath)
			Expect(err).To(MatchError(fs.ErrNotExist))
		})
	})

})
