package ffmpeg

import (
	"context"
	"io"
	"os/exec"
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

	Describe("Context Cancellation", func() {
		Context("when FFmpeg is available", func() {
			var ff FFmpeg

			BeforeEach(func() {
				ff = New()
				// Skip if FFmpeg is not available
				if !ff.IsAvailable() {
					Skip("FFmpeg not available on this system")
				}
			})

			It("should interrupt transcoding when context is cancelled", func() {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				// Start a long-running FFmpeg process (slow transcode to give time for cancellation)
				// Use a command that will take some time to complete
				command := "ffmpeg -f lavfi -i sine=frequency=1000:duration=30 -acodec pcm_s16le -"

				errChan := make(chan error, 1)
				var stream io.ReadCloser

				go func() {
					var err error
					// The input file is not used here, but we need to provide a valid path to the Transcode function
					stream, err = ff.Transcode(ctx, command, "tests/fixtures/test.mp3", 128, 0)
					errChan <- err
				}()

				// Give FFmpeg a moment to start
				time.Sleep(100 * time.Millisecond)

				// Cancel the context
				cancel()

				// The operation should fail due to cancellation
				select {
				case err := <-errChan:
					if err == nil {
						// If no error during start, the stream should be cancelled when we try to read
						if stream != nil {
							defer stream.Close()
							buf := make([]byte, 1024)
							_, readErr := stream.Read(buf)
							Expect(readErr).To(HaveOccurred(), "Expected read to fail due to cancelled context")
						}
					} else {
						// Starting should fail due to cancelled context
						Expect(err).To(HaveOccurred())
					}
				case <-time.After(5 * time.Second):
					Fail("Expected FFmpeg to be cancelled within 5 seconds")
				}
			})

			It("should handle immediate context cancellation", func() {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately

				// This should fail immediately
				_, err := ff.Transcode(ctx, "ffmpeg -i %s -f mp3 -", "tests/fixtures/test.mp3", 128, 0)
				Expect(err).To(MatchError(context.Canceled))
			})
		})

		Context("with mock process behavior", func() {
			var originalFfmpegPath string

			BeforeEach(func() {
				originalFfmpegPath = ffmpegPath
				// Use a long-running command for testing cancellation
				ffmpegPath = "sleep"
			})

			AfterEach(func() {
				ffmpegPath = originalFfmpegPath
			})

			It("should terminate the underlying process when context is cancelled", func() {
				ff := New()
				ctx, cancel := context.WithCancel(context.Background())

				// Start a process that will run for a while
				stream, err := ff.Transcode(ctx, "sleep 10", "tests/fixtures/test.mp3", 0, 0)
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

				// Give some time for cleanup
				time.Sleep(100 * time.Millisecond)

				// Verify no sleep processes are left running
				checkCmd := exec.Command("pgrep", "-f", "sleep 10")
				err = checkCmd.Run()
				Expect(err).To(HaveOccurred(), "Expected no 'sleep 10' processes to be running")
			})
		})
	})
})
