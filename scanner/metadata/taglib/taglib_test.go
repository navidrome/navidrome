package taglib

import (
	"io/fs"
	"os"

	"github.com/navidrome/navidrome/scanner/metadata"
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

	Describe("Parse", func() {
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

			Expect(m).To(Or(
				HaveKeyWithValue("compilation", []string{"1"}),
				HaveKeyWithValue("tcmp", []string{"1"}))) // Compilation
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
			Expect(m).ToNot(HaveKey("lyrics"))
			Expect(m).To(Or(HaveKeyWithValue("lyrics-eng", []string{
				"[00:00.00]This is\n[00:02.50]English SYLT\n",
				"[00:00.00]This is\n[00:02.50]English",
			}), HaveKeyWithValue("lyrics-eng", []string{
				"[00:00.00]This is\n[00:02.50]English",
				"[00:00.00]This is\n[00:02.50]English SYLT\n",
			})))
			Expect(m).To(Or(HaveKeyWithValue("lyrics-xxx", []string{
				"[00:00.00]This is\n[00:02.50]unspecified SYLT\n",
				"[00:00.00]This is\n[00:02.50]unspecified",
			}), HaveKeyWithValue("lyrics-xxx", []string{
				"[00:00.00]This is\n[00:02.50]unspecified",
				"[00:00.00]This is\n[00:02.50]unspecified SYLT\n",
			})))
			Expect(m).To(HaveKeyWithValue("bpm", []string{"123"}))
			Expect(m).To(HaveKeyWithValue("replaygain_album_gain", []string{"+3.21518 dB"}))
			Expect(m).To(HaveKeyWithValue("replaygain_album_peak", []string{"0.9125"}))
			Expect(m).To(HaveKeyWithValue("replaygain_track_gain", []string{"-1.48 dB"}))
			Expect(m).To(HaveKeyWithValue("replaygain_track_peak", []string{"0.4512"}))

			Expect(m).To(HaveKeyWithValue("tracknumber", []string{"2/10"}))
			m = m.Map(e.CustomMappings())
			Expect(m).To(HaveKeyWithValue("tracknumber", []string{"2/10", "2/10", "2"}))

			m = mds["tests/fixtures/test.ogg"]
			Expect(err).To(BeNil())
			Expect(m).ToNot(HaveKey("has_picture"))
			Expect(m).To(HaveKeyWithValue("duration", []string{"1.04"}))
			Expect(m).To(HaveKeyWithValue("fbpm", []string{"141.7"}))

			// TabLib 1.12 returns 18, previous versions return 39.
			// See https://github.com/taglib/taglib/commit/2f238921824741b2cfe6fbfbfc9701d9827ab06b
			Expect(m).To(HaveKey("bitrate"))
			Expect(m["bitrate"][0]).To(BeElementOf("18", "39", "40", "43", "49"))
		})

		DescribeTable("Format-Specific tests",
			func(file, duration, channels, albumGain, albumPeak, trackGain, trackPeak string, id3Lyrics bool) {
				file = "tests/fixtures/" + file
				mds, err := e.Parse(file)
				Expect(err).NotTo(HaveOccurred())
				Expect(mds).To(HaveLen(1))

				m := mds[file]

				Expect(m).To(HaveKeyWithValue("replaygain_album_gain", []string{albumGain}))
				Expect(m).To(HaveKeyWithValue("replaygain_album_peak", []string{albumPeak}))
				Expect(m).To(HaveKeyWithValue("replaygain_track_gain", []string{trackGain}))
				Expect(m).To(HaveKeyWithValue("replaygain_track_peak", []string{trackPeak}))

				Expect(m).To(HaveKeyWithValue("title", []string{"Title", "Title"}))
				Expect(m).To(HaveKeyWithValue("album", []string{"Album", "Album"}))
				Expect(m).To(HaveKeyWithValue("artist", []string{"Artist", "Artist"}))
				Expect(m).To(HaveKeyWithValue("albumartist", []string{"Album Artist"}))
				Expect(m).To(HaveKeyWithValue("genre", []string{"Rock"}))
				Expect(m).To(HaveKeyWithValue("date", []string{"2014", "2014"}))

				// Special for M4A, do not catch keys that have no actual name
				Expect(m).ToNot(HaveKey(""))

				Expect(m).To(HaveKey("discnumber"))
				discno := m["discnumber"]
				Expect(discno).To(HaveLen(1))
				Expect(discno[0]).To(BeElementOf([]string{"1", "1/2"}))

				// WMA does not have a "compilation" tag, but "wm/iscompilation"
				if _, ok := m["compilation"]; ok {
					Expect(m).To(HaveKeyWithValue("compilation", []string{"1"}))
				} else {
					Expect(m).To(HaveKeyWithValue("wm/iscompilation", []string{"1"}))
				}

				Expect(m).NotTo(HaveKeyWithValue("has_picture", []string{"true"}))
				Expect(m).To(HaveKeyWithValue("duration", []string{duration}))

				Expect(m).To(HaveKeyWithValue("channels", []string{channels}))
				Expect(m).To(HaveKeyWithValue("comment", []string{"Comment1\nComment2"}))

				if id3Lyrics {
					Expect(m).To(HaveKeyWithValue("lyrics-eng", []string{
						"[00:00.00]This is\n[00:02.50]English",
					}))
					Expect(m).To(HaveKeyWithValue("lyrics-xxx", []string{
						"[00:00.00]This is\n[00:02.50]unspecified",
					}))
				} else {
					Expect(m).To(HaveKeyWithValue("lyrics", []string{
						"[00:00.00]This is\n[00:02.50]unspecified",
						"[00:00.00]This is\n[00:02.50]English",
					}))
				}

				Expect(m).To(HaveKeyWithValue("bpm", []string{"123"}))

				Expect(m).To(HaveKey("tracknumber"))
				trackNo := m["tracknumber"]
				Expect(trackNo).To(HaveLen(1))
				Expect(trackNo[0]).To(BeElementOf([]string{"3", "3/10"}))
			},

			// ffmpeg -f lavfi -i "sine=frequency=1200:duration=1" test.flac
			Entry("correctly parses flac tags", "test.flac", "1.00", "1", "+4.06 dB", "0.12496948", "+4.06 dB", "0.12496948", false),

			Entry("Correctly parses m4a (aac) gain tags", "01 Invisible (RED) Edit Version.m4a", "1.04", "2", "0.37", "0.48", "0.37", "0.48", false),
			Entry("Correctly parses m4a (aac) gain tags (uppercase)", "test.m4a", "1.04", "2", "0.37", "0.48", "0.37", "0.48", false),

			Entry("correctly parses ogg (vorbis) tags", "test.ogg", "1.04", "2", "+7.64 dB", "0.11772506", "+7.64 dB", "0.11772506", false),

			// ffmpeg -f lavfi -i "sine=frequency=900:duration=1" test.wma
			// Weird note: for the tag parsing to work, the lyrics are actually stored in the reverse order
			Entry("correctly parses wma/asf tags", "test.wma", "1.02", "1", "3.27 dB", "0.132914", "3.27 dB", "0.132914", false),

			// ffmpeg -f lavfi -i "sine=frequency=800:duration=1" test.wv
			Entry("correctly parses wv (wavpak) tags", "test.wv", "1.00", "1", "3.43 dB", "0.125061", "3.43 dB", "0.125061", false),

			// TODO - these breaks in the pipeline as it uses TabLib 1.11. Once Ubuntu 24.04 is released we can uncomment these tests
			// ffmpeg -f lavfi -i "sine=frequency=1000:duration=1" test.wav
			// Entry("correctly parses wav tags", "test.wav", "1.00", "1", "3.06 dB", "0.125056", "3.06 dB", "0.125056", true),

			// ffmpeg -f lavfi -i "sine=frequency=1400:duration=1" test.aiff
			// Entry("correctly parses aiff tags", "test.aiff", "1.00", "1", "2.00 dB", "0.124972", "2.00 dB", "0.124972", true),
		)
	})

	Describe("Error Checking", func() {
		It("correctly handle unreadable file due to insufficient read permission", func() {
			_, err := e.extractMetadata(accessForbiddenFile)
			Expect(err).To(MatchError(os.ErrPermission))
		})
		It("returns a generic ErrPath if file does not exist", func() {
			testFilePath := "tests/fixtures/NON_EXISTENT.ogg"
			_, err := e.extractMetadata(testFilePath)
			Expect(err).To(MatchError(fs.ErrNotExist))
		})
		It("does not throw a SIGSEGV error when reading a file with an invalid frame", func() {
			// File has an empty TDAT frame
			md, err := e.extractMetadata("tests/fixtures/invalid-files/test-invalid-frame.mp3")
			Expect(err).ToNot(HaveOccurred())
			Expect(md).To(HaveKeyWithValue("albumartist", []string{"Elvis Presley"}))
		})
	})

	Describe("parseTIPL", func() {
		var tags metadata.ParsedTags

		BeforeEach(func() {
			tags = metadata.ParsedTags{}
		})

		Context("when the TIPL string is populated", func() {
			It("correctly parses roles and names", func() {
				tags["tipl"] = []string{"arranger Andrew Powell dj-mix François Kevorkian engineer Chris Blair"}
				parseTIPL(tags)
				Expect(tags["arranger"]).To(ConsistOf("Andrew Powell"))
				Expect(tags["engineer"]).To(ConsistOf("Chris Blair"))
				Expect(tags["djmixer"]).To(ConsistOf("François Kevorkian"))
			})

			It("handles multiple names for a single role", func() {
				tags["tipl"] = []string{"engineer Pat Stapley producer Eric Woolfson engineer Chris Blair"}
				parseTIPL(tags)
				Expect(tags["producer"]).To(ConsistOf("Eric Woolfson"))
				Expect(tags["engineer"]).To(ConsistOf("Pat Stapley", "Chris Blair"))
			})

			It("discards roles without names", func() {
				tags["tipl"] = []string{"engineer Pat Stapley producer engineer Chris Blair"}
				parseTIPL(tags)
				Expect(tags).ToNot(HaveKey("producer"))
				Expect(tags["engineer"]).To(ConsistOf("Pat Stapley", "Chris Blair"))
			})
		})

		Context("when the TIPL string is empty", func() {
			It("does nothing", func() {
				tags["tipl"] = []string{""}
				parseTIPL(tags)
				Expect(tags).To(BeEmpty())
			})
		})

		Context("when the TIPL is not present", func() {
			It("does nothing", func() {
				parseTIPL(tags)
				Expect(tags).To(BeEmpty())
			})
		})
	})

})
