package ffmpeg

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	sync "sync"
	"testing"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFFmpeg(t *testing.T) {
	// Inline test init to avoid import cycle with tests package
	//nolint:dogsled
	_, file, _, _ := runtime.Caller(0)
	appPath, _ := filepath.Abs(filepath.Join(filepath.Dir(file), "..", ".."))
	confPath := filepath.Join(appPath, "tests", "navidrome-test.toml")
	_ = os.Chdir(appPath)
	conf.LoadFromFile(confPath)
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

	Describe("isDefaultCommand", func() {
		It("returns true for known default mp3 command", func() {
			Expect(isDefaultCommand("mp3", "ffmpeg -i %s -ss %t -map 0:a:0 -b:a %bk -v 0 -f mp3 -")).To(BeTrue())
		})
		It("returns true for known default opus command", func() {
			Expect(isDefaultCommand("opus", "ffmpeg -i %s -ss %t -map 0:a:0 -b:a %bk -v 0 -c:a libopus -f opus -")).To(BeTrue())
		})
		It("returns true for known default aac command", func() {
			Expect(isDefaultCommand("aac", "ffmpeg -i %s -ss %t -map 0:a:0 -b:a %bk -v 0 -c:a aac -f adts -")).To(BeTrue())
		})
		It("returns true for known default flac command", func() {
			Expect(isDefaultCommand("flac", "ffmpeg -i %s -ss %t -map 0:a:0 -v 0 -c:a flac -f flac -")).To(BeTrue())
		})
		It("returns false for a custom command", func() {
			Expect(isDefaultCommand("mp3", "ffmpeg -i %s -b:a %bk -custom-flag -f mp3 -")).To(BeFalse())
		})
		It("returns false for unknown format", func() {
			Expect(isDefaultCommand("wav", "ffmpeg -i %s -f wav -")).To(BeFalse())
		})
	})

	Describe("buildDynamicArgs", func() {
		It("builds mp3 args with bitrate, samplerate, and channels", func() {
			args := buildDynamicArgs(TranscodeOptions{
				Format:     "mp3",
				FilePath:   "/music/file.flac",
				BitRate:    256,
				SampleRate: 48000,
				Channels:   2,
			})
			Expect(args).To(Equal([]string{
				"ffmpeg", "-i", "/music/file.flac",
				"-map", "0:a:0",
				"-c:a", "libmp3lame",
				"-b:a", "256k",
				"-ar", "48000",
				"-ac", "2",
				"-v", "0",
				"-f", "mp3",
				"-",
			}))
		})

		It("builds flac args without bitrate", func() {
			args := buildDynamicArgs(TranscodeOptions{
				Format:     "flac",
				FilePath:   "/music/file.dsf",
				SampleRate: 48000,
			})
			Expect(args).To(Equal([]string{
				"ffmpeg", "-i", "/music/file.dsf",
				"-map", "0:a:0",
				"-c:a", "flac",
				"-ar", "48000",
				"-v", "0",
				"-f", "flac",
				"-",
			}))
		})

		It("builds opus args with bitrate only", func() {
			args := buildDynamicArgs(TranscodeOptions{
				Format:   "opus",
				FilePath: "/music/file.flac",
				BitRate:  128,
			})
			Expect(args).To(Equal([]string{
				"ffmpeg", "-i", "/music/file.flac",
				"-map", "0:a:0",
				"-c:a", "libopus",
				"-b:a", "128k",
				"-v", "0",
				"-f", "opus",
				"-",
			}))
		})

		It("includes offset when specified", func() {
			args := buildDynamicArgs(TranscodeOptions{
				Format:   "mp3",
				FilePath: "/music/file.mp3",
				BitRate:  192,
				Offset:   30,
			})
			Expect(args).To(Equal([]string{
				"ffmpeg", "-i", "/music/file.mp3",
				"-ss", "30",
				"-map", "0:a:0",
				"-c:a", "libmp3lame",
				"-b:a", "192k",
				"-v", "0",
				"-f", "mp3",
				"-",
			}))
		})

		It("builds aac args correctly", func() {
			args := buildDynamicArgs(TranscodeOptions{
				Format:   "aac",
				FilePath: "/music/file.flac",
				BitRate:  256,
			})
			Expect(args).To(Equal([]string{
				"ffmpeg", "-i", "/music/file.flac",
				"-map", "0:a:0",
				"-c:a", "aac",
				"-b:a", "256k",
				"-v", "0",
				"-f", "adts",
				"-",
			}))
		})
	})

	Describe("buildTemplateArgs", func() {
		It("injects -ar and -ac into custom template", func() {
			args := buildTemplateArgs(TranscodeOptions{
				Command:    "ffmpeg -i %s -b:a %bk -v 0 -f mp3 -",
				FilePath:   "/music/file.flac",
				BitRate:    192,
				SampleRate: 44100,
				Channels:   2,
			})
			Expect(args).To(Equal([]string{
				"ffmpeg", "-i", "/music/file.flac",
				"-b:a", "192k", "-v", "0", "-f", "mp3",
				"-ar", "44100", "-ac", "2",
				"-",
			}))
		})

		It("injects only -ar when channels is 0", func() {
			args := buildTemplateArgs(TranscodeOptions{
				Command:    "ffmpeg -i %s -b:a %bk -v 0 -f mp3 -",
				FilePath:   "/music/file.flac",
				BitRate:    192,
				SampleRate: 48000,
			})
			Expect(args).To(Equal([]string{
				"ffmpeg", "-i", "/music/file.flac",
				"-b:a", "192k", "-v", "0", "-f", "mp3",
				"-ar", "48000",
				"-",
			}))
		})

		It("does not inject anything when sample rate and channels are 0", func() {
			args := buildTemplateArgs(TranscodeOptions{
				Command:  "ffmpeg -i %s -b:a %bk -v 0 -f mp3 -",
				FilePath: "/music/file.flac",
				BitRate:  192,
			})
			Expect(args).To(Equal([]string{
				"ffmpeg", "-i", "/music/file.flac",
				"-b:a", "192k", "-v", "0", "-f", "mp3",
				"-",
			}))
		})
	})

	Describe("injectBeforeOutput", func() {
		It("inserts flag before trailing dash", func() {
			args := injectBeforeOutput([]string{"ffmpeg", "-i", "file.mp3", "-f", "mp3", "-"}, "-ar", "48000")
			Expect(args).To(Equal([]string{"ffmpeg", "-i", "file.mp3", "-f", "mp3", "-ar", "48000", "-"}))
		})

		It("appends when no trailing dash", func() {
			args := injectBeforeOutput([]string{"ffmpeg", "-i", "file.mp3"}, "-ar", "48000")
			Expect(args).To(Equal([]string{"ffmpeg", "-i", "file.mp3", "-ar", "48000"}))
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
				stream, err := ff.Transcode(ctx, TranscodeOptions{
					Command:  command,
					Format:   "mp3",
					FilePath: "tests/fixtures/test.mp3",
					BitRate:  128,
				})
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
				_, err := ff.Transcode(ctx, TranscodeOptions{
					Command:  "ffmpeg -i %s -f mp3 -",
					Format:   "mp3",
					FilePath: "tests/fixtures/test.mp3",
					BitRate:  128,
				})
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
				stream, err := ff.Transcode(ctx, TranscodeOptions{
					Command:  longRunningCmd,
					FilePath: "tests/fixtures/test.mp3",
				})
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
