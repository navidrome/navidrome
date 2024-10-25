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
			args := createFFmpegCommand("ffmpeg -i %s -b:a %bk mp3 -", "/music library/file.mp3", 123, 0)
			Expect(args).To(Equal([]string{"ffmpeg", "-i", "/music library/file.mp3", "-b:a", "123k", "mp3", "-"}))
		})
		It("handles extra spaces in the command string", func() {
			args := createFFmpegCommand("ffmpeg    -i %s -b:a    %bk      mp3 -", "/music library/file.mp3", 123, 0)
			Expect(args).To(Equal([]string{"ffmpeg", "-i", "/music library/file.mp3", "-b:a", "123k", "mp3", "-"}))
		})
		Context("when command has time offset param", func() {
			It("creates a valid command line with offset", func() {
				args := createFFmpegCommand("ffmpeg -i %s -b:a %bk -ss %t mp3 -", "/music library/file.mp3", 123, 456)
				Expect(args).To(Equal([]string{"ffmpeg", "-i", "/music library/file.mp3", "-b:a", "123k", "-ss", "456", "mp3", "-"}))
			})

		})
		Context("when command does not have time offset param", func() {
			It("adds time offset after the input file name", func() {
				args := createFFmpegCommand("ffmpeg -i %s -b:a %bk mp3 -", "/music library/file.mp3", 123, 456)
				Expect(args).To(Equal([]string{"ffmpeg", "-i", "/music library/file.mp3", "-ss", "456", "-b:a", "123k", "mp3", "-"}))
			})
		})
	})

	Describe("createProbeCommand", func() {
		It("creates a valid command line", func() {
			args := createProbeCommand(probeCmd, []string{"/music library/one.mp3", "/music library/two.mp3"})
			Expect(args).To(Equal([]string{"ffmpeg", "-i", "/music library/one.mp3", "-i", "/music library/two.mp3", "-f", "ffmetadata"}))
		})
	})

	When("ffmpegPath is set", func() {
		It("returns the correct ffmpeg path", func() {
			ffmpegPath = "/usr/bin/ffmpeg"
			args := createProbeCommand(probeCmd, []string{"one.mp3"})
			Expect(args).To(Equal([]string{"/usr/bin/ffmpeg", "-i", "one.mp3", "-f", "ffmetadata"}))
		})
		It("returns the correct ffmpeg path with spaces", func() {
			ffmpegPath = "/usr/bin/with spaces/ffmpeg.exe"
			args := createProbeCommand(probeCmd, []string{"one.mp3"})
			Expect(args).To(Equal([]string{"/usr/bin/with spaces/ffmpeg.exe", "-i", "one.mp3", "-f", "ffmetadata"}))
		})
	})
})
