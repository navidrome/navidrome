package metadata_test

import (
	"encoding/json"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/scanner/metadata"
	_ "github.com/navidrome/navidrome/scanner/metadata/ffmpeg"
	_ "github.com/navidrome/navidrome/scanner/metadata/taglib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/exp/slices"
)

var _ = Describe("Tags", func() {
	var zero int64 = 0
	var secondTs int64 = 2500

	makeLyrics := func(synced bool, lang, secondLine string) model.Lyrics {
		lines := []model.Line{
			{Value: "This is"},
			{Value: secondLine},
		}

		if synced {
			lines[0].Start = &zero
			lines[1].Start = &secondTs
		}

		lyrics := model.Lyrics{
			Lang:   lang,
			Line:   lines,
			Synced: synced,
		}

		return lyrics
	}

	sortLyrics := func(lines model.LyricList) model.LyricList {
		slices.SortFunc(lines, func(a, b model.Lyrics) bool {
			langDiff := strings.Compare(a.Lang, b.Lang)
			if langDiff == 0 {
				return strings.Compare(a.Line[1].Value, b.Line[1].Value) < 0
			} else {
				return langDiff < 0
			}
		})

		return lines
	}

	compareLyrics := func(m metadata.Tags, expected model.LyricList) {
		lyrics := model.LyricList{}
		Expect(json.Unmarshal([]byte(m.Lyrics()), &lyrics)).To(BeNil())
		Expect(sortLyrics(lyrics)).To(Equal(sortLyrics(expected)))
	}

	Context("Extract", func() {
		BeforeEach(func() {
			conf.Server.Scanner.Extractor = "taglib"
		})

		It("correctly parses metadata from all files in folder", func() {
			mds, err := metadata.Extract("tests/fixtures/test.mp3", "tests/fixtures/test.ogg", "tests/fixtures/test.wma")
			Expect(err).NotTo(HaveOccurred())
			Expect(mds).To(HaveLen(3))

			m := mds["tests/fixtures/test.mp3"]
			Expect(m.Title()).To(Equal("Song"))
			Expect(m.Album()).To(Equal("Album"))
			Expect(m.Artist()).To(Equal("Artist"))
			Expect(m.AlbumArtist()).To(Equal("Album Artist"))
			Expect(m.Compilation()).To(BeTrue())
			Expect(m.Genres()).To(Equal([]string{"Rock"}))
			y, d := m.Date()
			Expect(y).To(Equal(2014))
			Expect(d).To(Equal("2014-05-21"))
			y, d = m.OriginalDate()
			Expect(y).To(Equal(1996))
			Expect(d).To(Equal("1996-11-21"))
			y, d = m.ReleaseDate()
			Expect(y).To(Equal(2020))
			Expect(d).To(Equal("2020-12-31"))
			n, t := m.TrackNumber()
			Expect(n).To(Equal(2))
			Expect(t).To(Equal(10))
			n, t = m.DiscNumber()
			Expect(n).To(Equal(1))
			Expect(t).To(Equal(2))
			Expect(m.HasPicture()).To(BeTrue())
			Expect(m.Duration()).To(BeNumerically("~", 1.02, 0.01))
			Expect(m.BitRate()).To(Equal(192))
			Expect(m.Channels()).To(Equal(2))
			Expect(m.FilePath()).To(Equal("tests/fixtures/test.mp3"))
			Expect(m.Suffix()).To(Equal("mp3"))
			Expect(m.Size()).To(Equal(int64(51876)))
			Expect(m.RGAlbumGain()).To(Equal(3.21518))
			Expect(m.RGAlbumPeak()).To(Equal(0.9125))
			Expect(m.RGTrackGain()).To(Equal(-1.48))
			Expect(m.RGTrackPeak()).To(Equal(0.4512))

			m = mds["tests/fixtures/test.ogg"]
			Expect(err).To(BeNil())
			Expect(m.Title()).To(Equal("Title"))
			Expect(m.HasPicture()).To(BeFalse())
			Expect(m.Duration()).To(BeNumerically("~", 1.04, 0.01))
			Expect(m.Suffix()).To(Equal("ogg"))
			Expect(m.FilePath()).To(Equal("tests/fixtures/test.ogg"))
			Expect(m.Size()).To(Equal(int64(5534)))
			// TabLib 1.12 returns 18, previous versions return 39.
			// See https://github.com/taglib/taglib/commit/2f238921824741b2cfe6fbfbfc9701d9827ab06b
			Expect(m.BitRate()).To(BeElementOf(18, 39, 40, 43, 49))

			m = mds["tests/fixtures/test.wma"]
			Expect(err).To(BeNil())
			Expect(m.Compilation()).To(BeTrue())
			Expect(m.Title()).To(Equal("Title"))
			Expect(m.HasPicture()).To(BeFalse())
			Expect(m.Duration()).To(BeNumerically("~", 1.02, 0.01))
			Expect(m.Suffix()).To(Equal("wma"))
			Expect(m.FilePath()).To(Equal("tests/fixtures/test.wma"))
			Expect(m.Size()).To(Equal(int64(21581)))
			Expect(m.BitRate()).To(BeElementOf(128))
		})

		DescribeTable("Lyrics test",
			func(file string, langEncoded bool) {
				path := "tests/fixtures/" + file
				mds, err := metadata.Extract(path)
				Expect(err).ToNot(HaveOccurred())
				Expect(mds).To(HaveLen(1))

				m := mds[path]
				lyrics := model.LyricList{
					makeLyrics(true, "xxx", "English"),
					makeLyrics(true, "xxx", "unspecified"),
				}
				if langEncoded {
					lyrics[0].Lang = "eng"
				}
				compareLyrics(m, lyrics)
			},

			Entry("Parses AIFF file", "test.aiff", true),
			Entry("Parses FLAC files", "test.flac", false),
			Entry("Parses M4A files", "01 Invisible (RED) Edit Version.m4a", false),
			Entry("Parses OGG Vorbis files", "test.ogg", false),
			Entry("Parses WAV files", "test.wav", true),
			Entry("Parses WMA files", "test.wma", false),
			Entry("Parses WV files", "test.wv", false),
		)

		It("Should parse mp3 with USLT and SYLT", func() {
			path := "tests/fixtures/test.mp3"
			mds, err := metadata.Extract(path)
			Expect(err).ToNot(HaveOccurred())
			Expect(mds).To(HaveLen(1))

			m := mds[path]
			compareLyrics(m, model.LyricList{
				makeLyrics(true, "eng", "English SYLT"),
				makeLyrics(true, "eng", "English"),
				makeLyrics(true, "xxx", "unspecified SYLT"),
				makeLyrics(true, "xxx", "unspecified"),
			})
		})
	})

	// Only run these tests if FFmpeg is available
	FFmpegContext := XContext
	if ffmpeg.New().IsAvailable() {
		FFmpegContext = Context
	}
	FFmpegContext("Extract with FFmpeg", func() {
		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Scanner.Extractor = "ffmpeg"
		})

		DescribeTable("Lyrics test",
			func(file string) {
				path := "tests/fixtures/" + file
				mds, err := metadata.Extract(path)
				Expect(err).ToNot(HaveOccurred())
				Expect(mds).To(HaveLen(1))

				m := mds[path]
				compareLyrics(m, model.LyricList{
					makeLyrics(true, "eng", "English"),
					makeLyrics(true, "xxx", "unspecified"),
				})
			},

			Entry("Parses AIFF file", "test.aiff"),
			Entry("Parses MP3 files", "test.mp3"),
			// Disabled, because it fails in pipeline
			// Entry("Parses WAV files", "test.wav"),

			// FFMPEG behaves very weirdly for multivalued tags for non-ID3
			// Specifically, they are separated by ";, which is indistinguishable
			// from other fields
		)
	})
})
