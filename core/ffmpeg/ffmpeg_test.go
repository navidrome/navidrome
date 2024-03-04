package ffmpeg

import (
	"testing"

	"github.com/navidrome/navidrome/model"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/tests"
)

func TestFFmpeg(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "FFmpeg Suite")
}

var _ = Describe("ffmpeg", func() {
	BeforeEach(func() {
		_, _ = ffmpegCmd()
		ffmpegPath = "ffmpeg"
		ffmpegErr = nil
	})
	Describe("createFFmpegCommand", func() {
		It("creates a valid command line", func() {
			mf := model.MediaFile{
				Path:     "/music library/file.mp3",
				SubTrack: -1,
			}
			args := createFFmpegCommandForMedia("ffmpeg -i %s -b:a %bk -", "mp3", "/music library/file.mp3", "", &mf, 123, 0)
			Expect(args).To(Equal([]string{"ffmpeg", "-i", "/music library/file.mp3", "-b:a", "123k", "-f", "mp3", "-"}))
		})
		Context("when command has time offset param", func() {
			It("creates a valid command line with offset", func() {
				mf := model.MediaFile{
					Path:     "/music library/file.mp3",
					SubTrack: -1,
				}
				args := createFFmpegCommandForMedia("ffmpeg -i %s -b:a %bk -ss %t -f mp3 -", "ogg", "/music library/file.mp3", "", &mf, 123, 456)
				Expect(args).To(Equal([]string{"ffmpeg", "-i", "/music library/file.mp3", "-b:a", "123k", "-ss", "456", "-f", "mp3", "-"}))
			})
		})
		Context("when command does not have time offset param", func() {
			It("adds time offset after the input file name", func() {
				mf := model.MediaFile{
					Path:     "/music library/file.mp3",
					SubTrack: -1,
				}
				args := createFFmpegCommandForMedia("ffmpeg -i %s -b:a %bk -f mp3 -", "ogg", "/music library/file.mp3", "", &mf, 123, 456)
				Expect(args).To(Equal([]string{"ffmpeg", "-i", "/music library/file.mp3", "-ss", "456", "-b:a", "123k", "-f", "mp3", "-"}))
			})
		})
		Context("for subtracks", func() {
			It("adds time offset amd duration before -i", func() {
				mf := model.MediaFile{
					Path:        "/music library/file.ape",
					SubTrack:    1,
					Offset:      100,
					Duration:    5.0,
					Title:       "title",
					Artist:      "Artist",
					Album:       "Album",
					Year:        2019,
					TrackNumber: 5,
					Comment:     "c",
					Genre:       "rock",
				}
				args := createFFmpegCommandForMedia("ffmpeg -i %s -b:a %bk -f mp3 -", "ogg", mf.Path, "", &mf, 123, 0)
				Expect(args).To(Equal([]string{"ffmpeg", "-ss", "00:01:40.000", "-t", "00:00:05.000", "-i", "/music library/file.ape", "-b:a", "123k", "-f", "mp3",
					"-metadata", "title=title",
					"-metadata", "artist=Artist",
					"-metadata", "album=Album",
					"-metadata", "year=2019",
					"-metadata", "track=5",
					"-metadata", "comment=c",
					"-metadata", "genre=rock",
					"-metadata", "cuesheet=", "-"}))
			})
		})
		Context("for subtracks", func() {
			It("adds time offset with addition amd duration before -i", func() {
				mf := model.MediaFile{
					Path:        "/music library/file.ape",
					SubTrack:    1,
					Offset:      100,
					Duration:    5.0,
					Title:       "title",
					Artist:      "Artist",
					Album:       "Album",
					Year:        2019,
					TrackNumber: 5,
					Comment:     "c",
					Genre:       "rock",
				}
				args := createFFmpegCommandForMedia("ffmpeg -i %s -b:a %bk -f mp3 -", "ogg", mf.Path, "", &mf, 123, 456)
				Expect(args).To(Equal([]string{"ffmpeg", "-ss", "00:09:16.000", "-t", "00:00:05.000", "-i", "/music library/file.ape", "-b:a", "123k", "-f", "mp3",
					"-metadata", "title=title",
					"-metadata", "artist=Artist",
					"-metadata", "album=Album",
					"-metadata", "year=2019",
					"-metadata", "track=5",
					"-metadata", "comment=c",
					"-metadata", "genre=rock",
					"-metadata", "cuesheet=", "-"}))
			})
		})
		Context("for subtracks", func() {
			It("adds time only duration before -i", func() {
				mf := model.MediaFile{
					Path:        "/music library/file.ape",
					SubTrack:    1,
					Offset:      0,
					Duration:    5.0,
					Title:       "title",
					Artist:      "Artist",
					Album:       "Album",
					Year:        2019,
					TrackNumber: 5,
					Comment:     "c",
					Genre:       "rock",
				}
				args := createFFmpegCommandForMedia("ffmpeg -i %s -b:a %bk -f mp3 -", "ogg", mf.Path, "", &mf, 123, 0)
				Expect(args).To(Equal([]string{"ffmpeg", "-t", "00:00:05.000", "-i", "/music library/file.ape", "-b:a", "123k", "-f", "mp3",
					"-metadata", "title=title",
					"-metadata", "artist=Artist",
					"-metadata", "album=Album",
					"-metadata", "year=2019",
					"-metadata", "track=5",
					"-metadata", "comment=c",
					"-metadata", "genre=rock",
					"-metadata", "cuesheet=", "-"}))
			})
		})
		Context("for subtracks", func() {
			It("use source path instead of mediafile path", func() {
				mf := model.MediaFile{
					Path:        "/music library/file.wv",
					Suffix:      "wv",
					SubTrack:    1,
					Offset:      0,
					Duration:    5.0,
					Title:       "title",
					Artist:      "Artist",
					Album:       "Album",
					Year:        2019,
					TrackNumber: 5,
					Comment:     "c",
					Genre:       "rock",
				}
				args := createFFmpegCommandForMedia("ffmpeg -i %s -b:a %bk -", "flac", "-", "", &mf, 123, 0)
				Expect(args).To(Equal([]string{"ffmpeg", "-t", "00:00:05.000", "-i", "-", "-b:a", "123k", "-f", "flac",
					"-metadata", "title=title",
					"-metadata", "artist=Artist",
					"-metadata", "album=Album",
					"-metadata", "year=2019",
					"-metadata", "track=5",
					"-metadata", "comment=c",
					"-metadata", "genre=rock",
					"-metadata", "cuesheet=", "-"}))
			})
		})
		Context("for subtracks", func() {
			It("flac to flac use intermediate path and no copy stream", func() {
				mf := model.MediaFile{
					Path:        "/music library/file.flac",
					Suffix:      "flac",
					SubTrack:    1,
					Offset:      0,
					Duration:    5.0,
					Title:       "title",
					Artist:      "Artist",
					Album:       "Album",
					Year:        2019,
					TrackNumber: 5,
					Comment:     "c",
					Genre:       "rock",
				}
				args := createFFmpegCommandForMedia("ffmpeg -i %s -b:a %bk -", "flac", mf.Path, "intermediate", &mf, 123, 0)
				Expect(args).To(Equal([]string{"ffmpeg", "-t", "00:00:05.000", "-i", "/music library/file.flac", "-b:a", "123k", "-f", "flac",
					"-metadata", "title=title",
					"-metadata", "artist=Artist",
					"-metadata", "album=Album",
					"-metadata", "year=2019",
					"-metadata", "track=5",
					"-metadata", "comment=c",
					"-metadata", "genre=rock",
					"-metadata", "cuesheet=", "intermediate"}))
			})
		})
	})

	Describe("createProbeCommand", func() {
		It("creates a valid command line", func() {
			args := createProbeCommand(probeCmd, []string{"/music library/one.mp3", "/music library/two.mp3"})
			Expect(args).To(Equal([]string{"ffmpeg", "-i", "/music library/one.mp3", "-i", "/music library/two.mp3", "-f", "ffmetadata"}))
		})
	})
})
