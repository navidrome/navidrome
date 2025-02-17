package taglib

import (
	"io/fs"
	"os"
	"strings"

	"github.com/navidrome/navidrome/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Extractor", func() {
	var e *extractor

	BeforeEach(func() {
		e = &extractor{}
	})

	Describe("Parse", func() {
		It("correctly parses metadata from all files in folder", func() {
			mds, err := e.Parse(
				"tests/fixtures/test.mp3",
				"tests/fixtures/test.ogg",
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(mds).To(HaveLen(2))

			// Test MP3
			m := mds["tests/fixtures/test.mp3"]
			Expect(m.Tags).To(HaveKeyWithValue("title", []string{"Song"}))
			Expect(m.Tags).To(HaveKeyWithValue("album", []string{"Album"}))
			Expect(m.Tags).To(HaveKeyWithValue("artist", []string{"Artist"}))
			Expect(m.Tags).To(HaveKeyWithValue("albumartist", []string{"Album Artist"}))

			Expect(m.HasPicture).To(BeTrue())
			Expect(m.AudioProperties.Duration.String()).To(Equal("1.02s"))
			Expect(m.AudioProperties.BitRate).To(Equal(192))
			Expect(m.AudioProperties.Channels).To(Equal(2))
			Expect(m.AudioProperties.SampleRate).To(Equal(44100))

			Expect(m.Tags).To(Or(
				HaveKeyWithValue("compilation", []string{"1"}),
				HaveKeyWithValue("tcmp", []string{"1"})),
			)
			Expect(m.Tags).To(HaveKeyWithValue("genre", []string{"Rock"}))
			Expect(m.Tags).To(HaveKeyWithValue("date", []string{"2014-05-21"}))
			Expect(m.Tags).To(HaveKeyWithValue("originaldate", []string{"1996-11-21"}))
			Expect(m.Tags).To(HaveKeyWithValue("releasedate", []string{"2020-12-31"}))
			Expect(m.Tags).To(HaveKeyWithValue("discnumber", []string{"1"}))
			Expect(m.Tags).To(HaveKeyWithValue("disctotal", []string{"2"}))
			Expect(m.Tags).To(HaveKeyWithValue("comment", []string{"Comment1\nComment2"}))
			Expect(m.Tags).To(HaveKeyWithValue("bpm", []string{"123"}))
			Expect(m.Tags).To(HaveKeyWithValue("replaygain_album_gain", []string{"+3.21518 dB"}))
			Expect(m.Tags).To(HaveKeyWithValue("replaygain_album_peak", []string{"0.9125"}))
			Expect(m.Tags).To(HaveKeyWithValue("replaygain_track_gain", []string{"-1.48 dB"}))
			Expect(m.Tags).To(HaveKeyWithValue("replaygain_track_peak", []string{"0.4512"}))

			Expect(m.Tags).To(HaveKeyWithValue("tracknumber", []string{"2"}))
			Expect(m.Tags).To(HaveKeyWithValue("tracktotal", []string{"10"}))

			Expect(m.Tags).ToNot(HaveKey("lyrics"))
			Expect(m.Tags).To(Or(HaveKeyWithValue("lyrics:eng", []string{
				"[00:00.00]This is\n[00:02.50]English SYLT\n",
				"[00:00.00]This is\n[00:02.50]English",
			}), HaveKeyWithValue("lyrics:eng", []string{
				"[00:00.00]This is\n[00:02.50]English",
				"[00:00.00]This is\n[00:02.50]English SYLT\n",
			})))
			Expect(m.Tags).To(Or(HaveKeyWithValue("lyrics:xxx", []string{
				"[00:00.00]This is\n[00:02.50]unspecified SYLT\n",
				"[00:00.00]This is\n[00:02.50]unspecified",
			}), HaveKeyWithValue("lyrics:xxx", []string{
				"[00:00.00]This is\n[00:02.50]unspecified",
				"[00:00.00]This is\n[00:02.50]unspecified SYLT\n",
			})))

			// Test OGG
			m = mds["tests/fixtures/test.ogg"]
			Expect(err).To(BeNil())
			Expect(m.Tags).To(HaveKeyWithValue("fbpm", []string{"141.7"}))

			// TabLib 1.12 returns 18, previous versions return 39.
			// See https://github.com/taglib/taglib/commit/2f238921824741b2cfe6fbfbfc9701d9827ab06b
			Expect(m.AudioProperties.BitRate).To(BeElementOf(18, 39, 40, 43, 49))
			Expect(m.AudioProperties.Channels).To(BeElementOf(2))
			Expect(m.AudioProperties.SampleRate).To(BeElementOf(8000))
			Expect(m.AudioProperties.SampleRate).To(BeElementOf(8000))
			Expect(m.HasPicture).To(BeFalse())
		})

		DescribeTable("Format-Specific tests",
			func(file, duration string, channels, samplerate, bitdepth int, albumGain, albumPeak, trackGain, trackPeak string, id3Lyrics bool) {
				file = "tests/fixtures/" + file
				mds, err := e.Parse(file)
				Expect(err).NotTo(HaveOccurred())
				Expect(mds).To(HaveLen(1))

				m := mds[file]

				Expect(m.HasPicture).To(BeFalse())
				Expect(m.AudioProperties.Duration.String()).To(Equal(duration))
				Expect(m.AudioProperties.Channels).To(Equal(channels))
				Expect(m.AudioProperties.SampleRate).To(Equal(samplerate))
				Expect(m.AudioProperties.BitDepth).To(Equal(bitdepth))

				Expect(m.Tags).To(Or(
					HaveKeyWithValue("replaygain_album_gain", []string{albumGain}),
					HaveKeyWithValue("----:com.apple.itunes:replaygain_track_gain", []string{albumGain}),
				))

				Expect(m.Tags).To(Or(
					HaveKeyWithValue("replaygain_album_peak", []string{albumPeak}),
					HaveKeyWithValue("----:com.apple.itunes:replaygain_album_peak", []string{albumPeak}),
				))
				Expect(m.Tags).To(Or(
					HaveKeyWithValue("replaygain_track_gain", []string{trackGain}),
					HaveKeyWithValue("----:com.apple.itunes:replaygain_track_gain", []string{trackGain}),
				))
				Expect(m.Tags).To(Or(
					HaveKeyWithValue("replaygain_track_peak", []string{trackPeak}),
					HaveKeyWithValue("----:com.apple.itunes:replaygain_track_peak", []string{trackPeak}),
				))

				Expect(m.Tags).To(HaveKeyWithValue("title", []string{"Title"}))
				Expect(m.Tags).To(HaveKeyWithValue("album", []string{"Album"}))
				Expect(m.Tags).To(HaveKeyWithValue("artist", []string{"Artist"}))
				Expect(m.Tags).To(HaveKeyWithValue("albumartist", []string{"Album Artist"}))
				Expect(m.Tags).To(HaveKeyWithValue("genre", []string{"Rock"}))
				Expect(m.Tags).To(HaveKeyWithValue("date", []string{"2014"}))

				Expect(m.Tags).To(HaveKeyWithValue("bpm", []string{"123"}))
				Expect(m.Tags).To(Or(
					HaveKeyWithValue("tracknumber", []string{"3"}),
					HaveKeyWithValue("tracknumber", []string{"3/10"}),
				))
				if !strings.HasSuffix(file, "test.wma") {
					// TODO Not sure why this is not working for WMA
					Expect(m.Tags).To(HaveKeyWithValue("tracktotal", []string{"10"}))
				}
				Expect(m.Tags).To(Or(
					HaveKeyWithValue("discnumber", []string{"1"}),
					HaveKeyWithValue("discnumber", []string{"1/2"}),
				))
				Expect(m.Tags).To(HaveKeyWithValue("disctotal", []string{"2"}))

				// WMA does not have a "compilation" tag, but "wm/iscompilation"
				Expect(m.Tags).To(Or(
					HaveKeyWithValue("compilation", []string{"1"}),
					HaveKeyWithValue("wm/iscompilation", []string{"1"})),
				)

				if id3Lyrics {
					Expect(m.Tags).To(HaveKeyWithValue("lyrics:eng", []string{
						"[00:00.00]This is\n[00:02.50]English",
					}))
					Expect(m.Tags).To(HaveKeyWithValue("lyrics:xxx", []string{
						"[00:00.00]This is\n[00:02.50]unspecified",
					}))
				} else {
					Expect(m.Tags).To(HaveKeyWithValue("lyrics:xxx", []string{
						"[00:00.00]This is\n[00:02.50]unspecified",
						"[00:00.00]This is\n[00:02.50]English",
					}))
				}

				Expect(m.Tags).To(HaveKeyWithValue("comment", []string{"Comment1\nComment2"}))
			},

			// ffmpeg -f lavfi -i "sine=frequency=1200:duration=1" test.flac
			Entry("correctly parses flac tags", "test.flac", "1s", 1, 44100, 16, "+4.06 dB", "0.12496948", "+4.06 dB", "0.12496948", false),

			Entry("correctly parses m4a (aac) gain tags", "01 Invisible (RED) Edit Version.m4a", "1.04s", 2, 44100, 16, "0.37", "0.48", "0.37", "0.48", false),
			Entry("correctly parses m4a (aac) gain tags (uppercase)", "test.m4a", "1.04s", 2, 44100, 16, "0.37", "0.48", "0.37", "0.48", false),
			Entry("correctly parses ogg (vorbis) tags", "test.ogg", "1.04s", 2, 8000, 0, "+7.64 dB", "0.11772506", "+7.64 dB", "0.11772506", false),

			// ffmpeg -f lavfi -i "sine=frequency=900:duration=1" test.wma
			// Weird note: for the tag parsing to work, the lyrics are actually stored in the reverse order
			Entry("correctly parses wma/asf tags", "test.wma", "1.02s", 1, 44100, 16, "3.27 dB", "0.132914", "3.27 dB", "0.132914", false),

			// ffmpeg -f lavfi -i "sine=frequency=800:duration=1" test.wv
			Entry("correctly parses wv (wavpak) tags", "test.wv", "1s", 1, 44100, 16, "3.43 dB", "0.125061", "3.43 dB", "0.125061", false),

			// ffmpeg -f lavfi -i "sine=frequency=1000:duration=1" test.wav
			Entry("correctly parses wav tags", "test.wav", "1s", 1, 44100, 16, "3.06 dB", "0.125056", "3.06 dB", "0.125056", true),

			// ffmpeg -f lavfi -i "sine=frequency=1400:duration=1" test.aiff
			Entry("correctly parses aiff tags", "test.aiff", "1s", 1, 44100, 16, "2.00 dB", "0.124972", "2.00 dB", "0.124972", true),
		)

		// Skip these tests when running as root
		Context("Access Forbidden", func() {
			var accessForbiddenFile string
			var RegularUserContext = XContext
			var isRegularUser = os.Getuid() != 0
			if isRegularUser {
				RegularUserContext = Context
			}

			// Only run permission tests if we are not root
			RegularUserContext("when run without root privileges", func() {
				BeforeEach(func() {
					accessForbiddenFile = utils.TempFileName("access_forbidden-", ".mp3")

					f, err := os.OpenFile(accessForbiddenFile, os.O_WRONLY|os.O_CREATE, 0222)
					Expect(err).ToNot(HaveOccurred())

					DeferCleanup(func() {
						Expect(f.Close()).To(Succeed())
						Expect(os.Remove(accessForbiddenFile)).To(Succeed())
					})
				})

				It("correctly handle unreadable file due to insufficient read permission", func() {
					_, err := e.extractMetadata(accessForbiddenFile)
					Expect(err).To(MatchError(os.ErrPermission))
				})

				It("skips the file if it cannot be read", func() {
					files := []string{
						"tests/fixtures/test.mp3",
						"tests/fixtures/test.ogg",
						accessForbiddenFile,
					}
					mds, err := e.Parse(files...)
					Expect(err).NotTo(HaveOccurred())
					Expect(mds).To(HaveLen(2))
					Expect(mds).ToNot(HaveKey(accessForbiddenFile))
				})
			})
		})

	})

	Describe("Error Checking", func() {
		It("returns a generic ErrPath if file does not exist", func() {
			testFilePath := "tests/fixtures/NON_EXISTENT.ogg"
			_, err := e.extractMetadata(testFilePath)
			Expect(err).To(MatchError(fs.ErrNotExist))
		})
		It("does not throw a SIGSEGV error when reading a file with an invalid frame", func() {
			// File has an empty TDAT frame
			md, err := e.extractMetadata("tests/fixtures/invalid-files/test-invalid-frame.mp3")
			Expect(err).ToNot(HaveOccurred())
			Expect(md.Tags).To(HaveKeyWithValue("albumartist", []string{"Elvis Presley"}))
		})
	})

	Describe("parseTIPL", func() {
		var tags map[string][]string

		BeforeEach(func() {
			tags = make(map[string][]string)
		})

		Context("when the TIPL string is populated", func() {
			It("correctly parses roles and names", func() {
				tags["tipl"] = []string{"arranger Andrew Powell DJ-mix François Kevorkian DJ-mix Jane Doe engineer Chris Blair"}
				parseTIPL(tags)
				Expect(tags["arranger"]).To(ConsistOf("Andrew Powell"))
				Expect(tags["engineer"]).To(ConsistOf("Chris Blair"))
				Expect(tags["djmixer"]).To(ConsistOf("François Kevorkian", "Jane Doe"))
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
