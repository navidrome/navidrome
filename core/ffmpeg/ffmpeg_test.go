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

	Describe("detectKnownImageHeader", func() {
		It("should recognize png header", func() {
			imageType := detectKnownImageHeader([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52})
			Expect(imageType).To(Equal("png"))
		})
		It("should recognize jpg header", func() {
			imageType := detectKnownImageHeader([]byte{0xFF, 0xD8, 0xFF, 0xE1, 0x01, 0x2C, 0x45, 0x78, 0x69, 0x66, 0x00, 0x00, 0x4D, 0x4D, 0x00, 0x2A})
			Expect(imageType).To(Equal("jpg"))
		})
		It("should recognize webp header", func() {
			imageType := detectKnownImageHeader([]byte{0x52, 0x49, 0x46, 0x46, 0x32, 0x07, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50, 0x56, 0x50, 0x38, 0x20})
			Expect(imageType).To(Equal("webp"))
		})
		It("should not recognize nonsense header", func() {
			imageType := detectKnownImageHeader([]byte{0x3c, 0x21, 0x44, 0x4f, 0x43, 0x54, 0x59, 0x50, 0x45, 0x20, 0x68, 0x74, 0x6d, 0x6c, 0x3e, 0x3c})
			Expect(imageType).To(Equal(""))
		})
		It("should reject when buffer is too small", func() {
			imageType := detectKnownImageHeader([]byte{'h', 'e', 'l', 'l', 'o'})
			Expect(imageType).To(Equal(""))
		})
	})
})
