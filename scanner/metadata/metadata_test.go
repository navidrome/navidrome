package metadata_test

import (
	"cmp"
	"encoding/json"
	"slices"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/scanner/metadata"
	_ "github.com/navidrome/navidrome/scanner/metadata/ffmpeg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
		slices.SortFunc(lines, func(a, b model.Lyrics) int {
			langDiff := cmp.Compare(a.Lang, b.Lang)
			if langDiff != 0 {
				return langDiff
			}
			return cmp.Compare(a.Line[1].Value, b.Line[1].Value)
		})

		return lines
	}

	compareLyrics := func(m metadata.Tags, expected model.LyricList) {
		lyrics := model.LyricList{}
		Expect(json.Unmarshal([]byte(m.Lyrics()), &lyrics)).To(BeNil())
		Expect(sortLyrics(lyrics)).To(Equal(sortLyrics(expected)))
	}

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
