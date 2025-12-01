package ffmpeg

import (
	"context"
	"runtime"
	sync "sync"
	"testing"
	"time"

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

	Describe("FFmpeg", func() {
		Context("when FFmpeg is available", func() {
			var ff FFmpeg

			BeforeEach(func() {
				ffOnce = sync.Once{}
				ff = New()
				// Skip if FFmpeg is not available
				if !ff.IsAvailable() {
					Skip("FFmpeg not available on this system")
				}
			})

			It("should interrupt transcoding when context is cancelled", func() {
				ctx, cancel := context.WithTimeout(GinkgoT().Context(), 5*time.Second)
				defer cancel()

				// Use a command that generates audio indefinitely
				// -f lavfi uses FFmpeg's built-in audio source
				// -t 0 means no time limit (runs forever)
				command := "ffmpeg -f lavfi -i sine=frequency=1000:duration=0 -f mp3 -"

				// The input file is not used here, but we need to provide a valid path to the Transcode function
				stream, err := ff.Transcode(ctx, command, "tests/fixtures/test.mp3", 128, 0)
				Expect(err).ToNot(HaveOccurred())
				defer stream.Close()

				// Read some data first to ensure FFmpeg is running
				buf := make([]byte, 1024)
				_, err = stream.Read(buf)
				Expect(err).ToNot(HaveOccurred())

				// Cancel the context
				cancel()

				// Next read should fail due to cancelled context
				_, err = stream.Read(buf)
				Expect(err).To(HaveOccurred())
			})

			It("should handle immediate context cancellation", func() {
				ctx, cancel := context.WithCancel(GinkgoT().Context())
				cancel() // Cancel immediately

				// This should fail immediately
				_, err := ff.Transcode(ctx, "ffmpeg -i %s -f mp3 -", "tests/fixtures/test.mp3", 128, 0)
				Expect(err).To(MatchError(context.Canceled))
			})
		})

		Context("with mock process behavior", func() {
			var longRunningCmd string
			BeforeEach(func() {
				// Use a long-running command for testing cancellation
				switch runtime.GOOS {
				case "windows":
					// Use PowerShell's Start-Sleep
					ffmpegPath = "powershell"
					longRunningCmd = "powershell -Command Start-Sleep -Seconds 10"
				default:
					// Use sleep on Unix-like systems
					ffmpegPath = "sleep"
					longRunningCmd = "sleep 10"
				}
			})

			It("should terminate the underlying process when context is cancelled", func() {
				ff := New()
				ctx, cancel := context.WithTimeout(GinkgoT().Context(), 5*time.Second)
				defer cancel()

				// Start a process that will run for a while
				stream, err := ff.Transcode(ctx, longRunningCmd, "tests/fixtures/test.mp3", 0, 0)
				Expect(err).ToNot(HaveOccurred())
				defer stream.Close()

				// Give the process time to start
				time.Sleep(50 * time.Millisecond)

				// Cancel the context
				cancel()

				// Try to read from the stream, which should fail
				buf := make([]byte, 100)
				_, err = stream.Read(buf)
				Expect(err).To(HaveOccurred(), "Expected stream to be closed due to process termination")

				// Verify the stream is closed by attempting another read
				_, err = stream.Read(buf)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
