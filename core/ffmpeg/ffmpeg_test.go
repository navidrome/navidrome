package ffmpeg

import (
	"testing"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
			args := createFFmpegCommand("ffmpeg -i %s -b:a %bk mp3 -", "/music library/file.mp3", 123)
			Expect(args).To(Equal([]string{"ffmpeg", "-i", "/music library/file.mp3", "-b:a", "123k", "mp3", "-"}))
		})
	})

	Describe("createProbeCommand", func() {
		It("creates a valid command line", func() {
			args := createProbeCommand(probeCmd, []string{"/music library/one.mp3", "/music library/two.mp3"})
			Expect(args).To(Equal([]string{"ffmpeg", "-i", "/music library/one.mp3", "-i", "/music library/two.mp3", "-f", "ffmetadata"}))
		})
	})

	Describe("matchProbedVideoStreamType", func() {
		It("should be able to detect cover format", func() {
			Expect(matchProbedVideoStreamType(`
  Stream #0:0: Audio: ... whatever ...
  Stream #0:0: Video: mjpeg (Baseline), yuvj444p(pc, bt470bg/unknown/unknown), 300x300 [SAR 72:72 DAR 1:1], 25 fps, 25 tbr, 25 tbn, 25 tbc
  			`)).To(Equal("mjpeg"))

			Expect(matchProbedVideoStreamType(`
  Stream #0:0: Audio: ... whatever ...
  Stream #0:1: Video: png, rgba(pc), 16x16 [SAR 5669:5669 DAR 1:1], 90k tbr, 90k tbn, 90k tbc (attached pic)
			`)).To(Equal("png"))
			
			Expect(matchProbedVideoStreamType(`
  Duration: 00:00:03.66, start: 0.000000, bitrate: 22 kb/s
  Stream #0:0: Video: theora, yuv444p, 214x152 [SAR 1:1 DAR 107:76], 25 tbr, 25 tbn, 25 tbc
    Metadata:
      encoder         : Lavc59.37.100 libtheora
	  		`)).To(Equal("theora"))

			Expect(matchProbedVideoStreamType(`
  Duration: 00:00:03.71, start: 0.000000, bitrate: 9 kb/s
  Stream #0:0(und): Video: h264 (High 4:4:4 Predictive) (avc1 / 0x31637661), yuv444p, 214x152 [SAR 1:1 DAR 107:76], 308 kb/s, 25 fps, 25 tbr, 12800 tbn, 50 tbc (default)
    Metadata:
      handler_name    : VideoHandler
      vendor_id       : [0][0][0][0]
  Stream #0:1(und): Audio: aac (LC) (mp4a / 0x6134706D), 44100 Hz, stereo, fltp, 2 kb/s (default)
	  		`)).To(Equal("h264"))
		})
	})

	Describe("createCoverExtractCommand", func() {
		It("should be able to create correct set of commands to extract covers", func() {
			Expect(createCoverExtractCommand("mjpeg", "/tmp/audio.mp3")).To(Equal([]string{
				"ffmpeg",
				"-i", "/tmp/audio.mp3",
				"-an",
				"-c:v", "copy",
				"-f", "image2pipe",
				"-",
			}))

			Expect(createCoverExtractCommand("theora", "/tmp/audio.ogg")).To(Equal([]string{
				"ffmpeg",
				"-i", "/tmp/audio.ogg",
				"-an",
				"-c:v", "mjpeg",
				"-f", "image2pipe",
				"-",
			}))

			Expect(createCoverExtractCommand("h264", "/tmp/audio.m4a")).To(Equal([]string{
				"ffmpeg",
				"-i", "/tmp/audio.m4a",
				"-an",
				"-c:v", "mjpeg",
				"-f", "image2pipe",
				"-",
			}))
		})
	})
})
