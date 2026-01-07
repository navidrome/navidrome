package ffmpeg

import (
	"context"
	"runtime"
	sync "sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
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
	Describe("createFFmpegCommandForMedia", func() {
		It("creates a valid command line", func() {
			mf := model.MediaFile{
				Path:     "/music library/file.mp3",
				SubTrack: -1,
			}
			args := createFFmpegCommandForMedia("ffmpeg -i %s -b:a %bk -", "mp3", mf.Path, "", &mf, 123, 0)
			Expect(args).To(Equal([]string{"ffmpeg", "-i", "/music library/file.mp3", "-b:a", "123k", "-f", "mp3", "-"}))
		})
		It("handles extra spaces in the command string", func() {
			mf := model.MediaFile{
				Path:     "/music library/file.mp3",
				SubTrack: -1,
			}
			args := createFFmpegCommandForMedia("ffmpeg    -i %s -b:a    %bk      -", "mp3", mf.Path, "", &mf, 123, 0)
			Expect(args).To(Equal([]string{"ffmpeg", "-i", "/music library/file.mp3", "-b:a", "123k", "-f", "mp3", "-"}))
		})
		Context("when command has time offset param", func() {
			It("creates a valid command line with offset", func() {
				mf := model.MediaFile{
					Path:     "/music library/file.mp3",
					SubTrack: -1,
				}
				args := createFFmpegCommandForMedia("ffmpeg -i %s -b:a %bk -ss %t -", "mp3", mf.Path, "", &mf, 123, 456)
				Expect(args).To(Equal([]string{"ffmpeg", "-i", "/music library/file.mp3", "-b:a", "123k", "-ss", "456", "-f", "mp3", "-"}))
			})
		})
		Context("when command does not have time offset param", func() {
			It("adds time offset after the input file name", func() {
				mf := model.MediaFile{
					Path:     "/music library/file.mp3",
					SubTrack: -1,
				}
				args := createFFmpegCommandForMedia("ffmpeg -i %s -b:a %bk -", "mp3", mf.Path, "", &mf, 123, 456)
				Expect(args).To(Equal([]string{"ffmpeg", "-i", "/music library/file.mp3", "-ss", "456", "-b:a", "123k", "-f", "mp3", "-"}))
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
				command := "ffmpeg -f lavfi -i sine=frequency=1000:duration=0 -"

				mf := model.MediaFile{
					Path:     "tests/fixtures/test.mp3",
					SubTrack: -1,
				}

				// The input file is not used here, but we need to provide a valid path to the Transcode function
				stream, err := ff.Transcode(ctx, command, "opus", &mf, 128, 0)
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

				mf := model.MediaFile{
					Path:     "tests/fixtures/test.mp3",
					SubTrack: -1,
				}

				// This should fail immediately
				_, err := ff.Transcode(ctx, "ffmpeg -i %s -", "opus", &mf, 128, 0)
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

				mf := model.MediaFile{
					Path:     "tests/fixtures/test.mp3",
					SubTrack: -1,
				}

				// Start a process that will run for a while
				stream, err := ff.Transcode(ctx, longRunningCmd, "opus", &mf, 0, 0)
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
