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

		It("builds aac args with ADTS output", func() {
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

		It("builds flac args with bit depth", func() {
			args := buildDynamicArgs(TranscodeOptions{
				Format:   "flac",
				FilePath: "/music/file.dsf",
				BitDepth: 24,
			})
			Expect(args).To(Equal([]string{
				"ffmpeg", "-i", "/music/file.dsf",
				"-map", "0:a:0",
				"-c:a", "flac",
				"-sample_fmt", "s32",
				"-v", "0",
				"-f", "flac",
				"-",
			}))
		})

		It("omits -sample_fmt when bit depth is 0", func() {
			args := buildDynamicArgs(TranscodeOptions{
				Format:   "flac",
				FilePath: "/music/file.flac",
				BitDepth: 0,
			})
			Expect(args).ToNot(ContainElement("-sample_fmt"))
		})

		It("omits -sample_fmt when bit depth is too low (DSD)", func() {
			args := buildDynamicArgs(TranscodeOptions{
				Format:   "flac",
				FilePath: "/music/file.dsf",
				BitDepth: 1,
			})
			Expect(args).ToNot(ContainElement("-sample_fmt"))
		})

		DescribeTable("omits -sample_fmt for lossy formats even when bit depth >= 16",
			func(format string, bitRate int) {
				args := buildDynamicArgs(TranscodeOptions{
					Format:   format,
					FilePath: "/music/file.flac",
					BitRate:  bitRate,
					BitDepth: 16,
				})
				Expect(args).ToNot(ContainElement("-sample_fmt"))
			},
			Entry("mp3", "mp3", 256),
			Entry("aac", "aac", 256),
			Entry("opus", "opus", 128),
		)
	})

	Describe("bitDepthToSampleFmt", func() {
		It("converts 16-bit", func() {
			Expect(bitDepthToSampleFmt(16)).To(Equal("s16"))
		})
		It("converts 24-bit to s32 (FLAC only supports s16/s32)", func() {
			Expect(bitDepthToSampleFmt(24)).To(Equal("s32"))
		})
		It("converts 32-bit", func() {
			Expect(bitDepthToSampleFmt(32)).To(Equal("s32"))
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

		It("injects -sample_fmt for lossless output format with bit depth", func() {
			args := buildTemplateArgs(TranscodeOptions{
				Command:  "ffmpeg -i %s -v 0 -c:a flac -f flac -",
				Format:   "flac",
				FilePath: "/music/file.dsf",
				BitDepth: 24,
			})
			Expect(args).To(Equal([]string{
				"ffmpeg", "-i", "/music/file.dsf",
				"-v", "0", "-c:a", "flac", "-f", "flac",
				"-sample_fmt", "s32",
				"-",
			}))
		})

		It("does not inject -sample_fmt for lossy output format even with bit depth", func() {
			args := buildTemplateArgs(TranscodeOptions{
				Command:  "ffmpeg -i %s -b:a %bk -v 0 -f mp3 -",
				Format:   "mp3",
				FilePath: "/music/file.flac",
				BitRate:  192,
				BitDepth: 16,
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

	Describe("parseProbeOutput", func() {
		It("parses MP3 with embedded artwork (real ffprobe output)", func() {
			// Real: MP3 file with mjpeg artwork stream after audio
			data := []byte(`{"streams":[` +
				`{"index":0,"codec_name":"mp3","codec_long_name":"MP3 (MPEG audio layer 3)","codec_type":"audio",` +
				`"sample_fmt":"fltp","sample_rate":"44100","channels":2,"channel_layout":"stereo",` +
				`"bits_per_sample":0,"bit_rate":"198314","tags":{"encoder":"LAME3.99r"}},` +
				`{"index":1,"codec_name":"mjpeg","codec_type":"video","profile":"Baseline","width":400,"height":400}]}`)
			result, err := parseProbeOutput(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Codec).To(Equal("mp3"))
			Expect(result.Profile).To(BeEmpty()) // MP3 has no profile field
			Expect(result.SampleRate).To(Equal(44100))
			Expect(result.Channels).To(Equal(2))
			Expect(result.BitRate).To(Equal(198)) // 198314 bps -> 198 kbps
			Expect(result.BitDepth).To(Equal(0))  // lossy codec
		})

		It("parses AAC-LC in m4a container (real ffprobe output)", func() {
			// Real: AAC LC file with profile and artwork
			data := []byte(`{"streams":[` +
				`{"index":0,"codec_name":"aac","codec_long_name":"AAC (Advanced Audio Coding)",` +
				`"profile":"LC","codec_type":"audio","sample_fmt":"fltp","sample_rate":"44100",` +
				`"channels":2,"channel_layout":"stereo","bits_per_sample":0,"bit_rate":"279958"},` +
				`{"index":1,"codec_name":"mjpeg","codec_type":"video","profile":"Baseline"}]}`)
			result, err := parseProbeOutput(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Codec).To(Equal("aac"))
			Expect(result.Profile).To(Equal("LC"))
			Expect(result.SampleRate).To(Equal(44100))
			Expect(result.Channels).To(Equal(2))
			Expect(result.BitRate).To(Equal(279)) // 279958 bps -> 279 kbps
		})

		It("parses HE-AACv2 in mp4 container with video stream (real ffprobe output)", func() {
			// Real: Fraunhofer HE-AACv2 sample (LFE-SBRstereo.mp4), video stream before audio
			data := []byte(`{"streams":[` +
				`{"index":0,"codec_name":"h264","codec_type":"video","profile":"Main"},` +
				`{"index":1,"codec_name":"aac","codec_long_name":"AAC (Advanced Audio Coding)",` +
				`"profile":"HE-AACv2","codec_type":"audio","sample_fmt":"fltp",` +
				`"sample_rate":"48000","channels":2,"channel_layout":"stereo",` +
				`"bits_per_sample":0,"bit_rate":"55999"}]}`)
			result, err := parseProbeOutput(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Codec).To(Equal("aac"))
			Expect(result.Profile).To(Equal("HE-AACv2"))
			Expect(result.SampleRate).To(Equal(48000))
			Expect(result.Channels).To(Equal(2))
			Expect(result.BitRate).To(Equal(55)) // 55999 bps -> 55 kbps
		})

		It("parses FLAC using bits_per_raw_sample and format-level bit_rate (real ffprobe output)", func() {
			// Real: FLAC reports bit depth in bits_per_raw_sample, not bits_per_sample.
			// Stream-level bit_rate is absent; format-level bit_rate is used as fallback.
			data := []byte(`{"streams":[` +
				`{"index":0,"codec_name":"flac","codec_long_name":"FLAC (Free Lossless Audio Codec)",` +
				`"codec_type":"audio","sample_fmt":"s16","sample_rate":"44100","channels":2,` +
				`"channel_layout":"stereo","bits_per_sample":0,"bits_per_raw_sample":"16"},` +
				`{"index":1,"codec_name":"mjpeg","codec_type":"video","profile":"Baseline"}],` +
				`"format":{"bit_rate":"906900"}}`)
			result, err := parseProbeOutput(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Codec).To(Equal("flac"))
			Expect(result.SampleRate).To(Equal(44100))
			Expect(result.BitDepth).To(Equal(16)) // from bits_per_raw_sample
			Expect(result.BitRate).To(Equal(906)) // format-level: 906900 bps -> 906 kbps
			Expect(result.Profile).To(BeEmpty())  // no profile field in real output
		})

		It("parses Opus with format-level bit_rate fallback (real ffprobe output)", func() {
			// Real: Opus stream-level bit_rate is absent; format-level is used as fallback.
			data := []byte(`{"streams":[` +
				`{"index":0,"codec_name":"opus","codec_long_name":"Opus (Opus Interactive Audio Codec)",` +
				`"codec_type":"audio","sample_fmt":"fltp","sample_rate":"48000","channels":2,` +
				`"channel_layout":"stereo","bits_per_sample":0}],` +
				`"format":{"bit_rate":"128000"}}`)
			result, err := parseProbeOutput(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Codec).To(Equal("opus"))
			Expect(result.SampleRate).To(Equal(48000))
			Expect(result.Channels).To(Equal(2))
			Expect(result.BitRate).To(Equal(128)) // format-level: 128000 bps -> 128 kbps
			Expect(result.BitDepth).To(Equal(0))
		})

		It("parses WAV/PCM with bits_per_sample (real ffprobe output)", func() {
			// Real: WAV uses bits_per_sample directly
			data := []byte(`{"streams":[` +
				`{"index":0,"codec_name":"pcm_s16le","codec_long_name":"PCM signed 16-bit little-endian",` +
				`"codec_type":"audio","sample_fmt":"s16","sample_rate":"44100","channels":2,` +
				`"bits_per_sample":16,"bit_rate":"1411200"}]}`)
			result, err := parseProbeOutput(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Codec).To(Equal("pcm_s16le"))
			Expect(result.SampleRate).To(Equal(44100))
			Expect(result.Channels).To(Equal(2))
			Expect(result.BitDepth).To(Equal(16))
			Expect(result.BitRate).To(Equal(1411))
		})

		It("parses ALAC in m4a container (real ffprobe output)", func() {
			// Real: Beatles - You Can't Do That (2023 Mix), ALAC 16-bit
			data := []byte(`{"streams":[` +
				`{"index":0,"codec_name":"alac","codec_long_name":"ALAC (Apple Lossless Audio Codec)",` +
				`"codec_type":"audio","sample_fmt":"s16p","sample_rate":"44100","channels":2,` +
				`"channel_layout":"stereo","bits_per_sample":0,"bit_rate":"1011003",` +
				`"bits_per_raw_sample":"16"},` +
				`{"index":1,"codec_name":"mjpeg","codec_type":"video","profile":"Baseline"}]}`)
			result, err := parseProbeOutput(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Codec).To(Equal("alac"))
			Expect(result.BitDepth).To(Equal(16)) // from bits_per_raw_sample
			Expect(result.SampleRate).To(Equal(44100))
			Expect(result.Channels).To(Equal(2))
			Expect(result.BitRate).To(Equal(1011)) // 1011003 bps -> 1011 kbps
		})

		It("skips video-only streams", func() {
			data := []byte(`{"streams":[{"index":0,"codec_name":"mjpeg","codec_type":"video","profile":"Baseline"}]}`)
			_, err := parseProbeOutput(data)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no audio stream"))
		})

		It("returns error for empty streams array", func() {
			data := []byte(`{"streams":[]}`)
			_, err := parseProbeOutput(data)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for invalid JSON", func() {
			data := []byte(`not json`)
			_, err := parseProbeOutput(data)
			Expect(err).To(HaveOccurred())
		})

		It("parses HiRes multichannel FLAC with format-level bit_rate (real ffprobe output)", func() {
			// Real: Pink Floyd - 192kHz/24-bit/7.1 surround FLAC
			data := []byte(`{"streams":[` +
				`{"index":0,"codec_name":"flac","codec_long_name":"FLAC (Free Lossless Audio Codec)",` +
				`"codec_type":"audio","sample_fmt":"s32","sample_rate":"192000","channels":8,` +
				`"channel_layout":"7.1","bits_per_sample":0,"bits_per_raw_sample":"24"},` +
				`{"index":1,"codec_name":"mjpeg","codec_type":"video","profile":"Progressive"}],` +
				`"format":{"bit_rate":"18432000"}}`)
			result, err := parseProbeOutput(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Codec).To(Equal("flac"))
			Expect(result.SampleRate).To(Equal(192000))
			Expect(result.BitDepth).To(Equal(24))
			Expect(result.Channels).To(Equal(8))
			Expect(result.BitRate).To(Equal(18432)) // format-level: 18432000 bps -> 18432 kbps
		})

		It("parses DSD/DSF file (real ffprobe output)", func() {
			// Real: Yes - Owner of a Lonely Heart, DSD64 DSF
			data := []byte(`{"streams":[` +
				`{"index":0,"codec_name":"dsd_lsbf_planar",` +
				`"codec_long_name":"DSD (Direct Stream Digital), least significant bit first, planar",` +
				`"codec_type":"audio","sample_fmt":"fltp","sample_rate":"352800","channels":2,` +
				`"channel_layout":"stereo","bits_per_sample":8,"bit_rate":"5644800"},` +
				`{"index":1,"codec_name":"mjpeg","codec_type":"video","profile":"Baseline"}]}`)
			result, err := parseProbeOutput(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Codec).To(Equal("dsd_lsbf_planar"))
			Expect(result.BitDepth).To(Equal(8))        // DSD reports 8 bits_per_sample
			Expect(result.SampleRate).To(Equal(352800)) // DSD64 sample rate
			Expect(result.Channels).To(Equal(2))
			Expect(result.BitRate).To(Equal(5644)) // 5644800 bps -> 5644 kbps
		})

		It("prefers stream-level bit_rate over format-level when both are present", func() {
			// ALAC/DSD: stream has bit_rate, format also has bit_rate — stream wins
			data := []byte(`{"streams":[` +
				`{"index":0,"codec_name":"alac","codec_type":"audio","sample_fmt":"s16p",` +
				`"sample_rate":"44100","channels":2,"bits_per_sample":0,` +
				`"bit_rate":"1011003","bits_per_raw_sample":"16"}],` +
				`"format":{"bit_rate":"1050000"}}`)
			result, err := parseProbeOutput(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.BitRate).To(Equal(1011)) // stream-level: 1011003 bps -> 1011 kbps (not format's 1050)
		})

		It("returns BitRate 0 when neither stream nor format has bit_rate", func() {
			data := []byte(`{"streams":[` +
				`{"index":0,"codec_name":"flac","codec_type":"audio","sample_fmt":"s16",` +
				`"sample_rate":"44100","channels":2,"bits_per_sample":0,"bits_per_raw_sample":"16"}],` +
				`"format":{}}`)
			result, err := parseProbeOutput(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.BitRate).To(Equal(0))
		})

		It("clears 'unknown' profile to empty string", func() {
			data := []byte(`{"streams":[{"index":0,"codec_name":"flac",` +
				`"codec_type":"audio","profile":"unknown","sample_rate":"44100",` +
				`"channels":2,"bits_per_sample":0}]}`)
			result, err := parseProbeOutput(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Profile).To(BeEmpty())
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

		Context("stderr capture", func() {
			BeforeEach(func() {
				if runtime.GOOS == "windows" {
					Skip("stderr capture tests use /bin/sh, skipping on Windows")
				}
			})

			It("should include stderr in error when process fails", func() {
				ff := &ffmpeg{}
				ctx := GinkgoT().Context()

				// Directly call start() with a bash command that writes to stderr and fails
				args := []string{"/bin/sh", "-c", "echo 'codec not found: libopus' >&2; exit 1"}
				stream, err := ff.start(ctx, args)
				Expect(err).ToNot(HaveOccurred())
				defer stream.Close()

				buf := make([]byte, 1024)
				_, err = stream.Read(buf)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("codec not found: libopus"))
			})

			It("should not include stderr in error when process succeeds", func() {
				ff := &ffmpeg{}
				ctx := GinkgoT().Context()

				// Command that writes to stderr but exits successfully
				args := []string{"/bin/sh", "-c", "echo 'warning: something' >&2; printf 'output'"}
				stream, err := ff.start(ctx, args)
				Expect(err).ToNot(HaveOccurred())
				defer stream.Close()

				buf := make([]byte, 1024)
				n, err := stream.Read(buf)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(buf[:n])).To(Equal("output"))
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
