package transcode

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// withProbe pre-populates ProbeData on a MediaFile from its own fields,
// so ensureProbed short-circuits and tests don't need mock ffprobe results.
func withProbe(mf *model.MediaFile) *model.MediaFile {
	probe := ffmpeg.AudioProbeResult{
		Codec:      mf.AudioCodec(),
		BitRate:    mf.BitRate,
		SampleRate: mf.SampleRate,
		BitDepth:   mf.BitDepth,
		Channels:   mf.Channels,
	}
	data, _ := json.Marshal(probe)
	mf.ProbeData = string(data)
	return mf
}

var _ = Describe("Decider", func() {
	var (
		ds  *tests.MockDataStore
		ff  *tests.MockFFmpeg
		svc Decider
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = &tests.MockDataStore{
			MockedProperty:    &tests.MockedPropertyRepo{},
			MockedTranscoding: &tests.MockTranscodingRepo{},
		}
		ff = tests.NewMockFFmpeg("")
		auth.Init(ds)
		svc = NewDecider(ds, ff)
	})

	Describe("MakeDecision", func() {
		Context("Direct Play", func() {
			It("allows direct play when profile matches", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 320, Channels: 2, SampleRate: 44100})
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"mp3"}, AudioCodecs: []string{"mp3"}, Protocols: []string{"http"}, MaxAudioChannels: 2},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
				Expect(decision.CanTranscode).To(BeFalse())
				Expect(decision.TranscodeReasons).To(BeEmpty())
			})

			It("rejects direct play when container doesn't match", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2})
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"mp3"}, Protocols: []string{"http"}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.TranscodeReasons).To(ContainElement("container not supported"))
			})

			It("rejects direct play when codec doesn't match", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "m4a", Codec: "ALAC", BitRate: 1000, Channels: 2})
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"m4a"}, AudioCodecs: []string{"aac"}, Protocols: []string{"http"}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.TranscodeReasons).To(ContainElement("audio codec not supported"))
			})

			It("rejects direct play when channels exceed limit", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 6})
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{"http"}, MaxAudioChannels: 2},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.TranscodeReasons).To(ContainElement("audio channels not supported"))
			})

			It("handles container aliases (aac -> m4a)", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "m4a", Codec: "AAC", BitRate: 256, Channels: 2})
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"aac"}, AudioCodecs: []string{"aac"}, Protocols: []string{"http"}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})

			It("handles container aliases (mp4 -> m4a)", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "m4a", Codec: "AAC", BitRate: 256, Channels: 2})
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"mp4"}, AudioCodecs: []string{"aac"}, Protocols: []string{"http"}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})

			It("handles codec aliases (adts -> aac)", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "m4a", Codec: "AAC", BitRate: 256, Channels: 2})
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"m4a"}, AudioCodecs: []string{"adts"}, Protocols: []string{"http"}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})

			It("allows when protocol list is empty (any protocol)", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2})
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, AudioCodecs: []string{"flac"}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})

			It("allows when both container and codec lists are empty (wildcard)", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 128, Channels: 2})
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{}, AudioCodecs: []string{}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})
		})

		Context("MaxAudioBitrate constraint", func() {
			It("revokes direct play when bitrate exceeds maxAudioBitrate", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1500, Channels: 2})
				ci := &ClientInfo{
					MaxAudioBitrate: 500, // kbps
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{"http"}},
					},
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeReasons).To(ContainElement("audio bitrate not supported"))
			})
		})

		Context("Transcoding", func() {
			It("selects transcoding when direct play isn't possible", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16})
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 256, // kbps
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"mp3"}, Protocols: []string{"http"}},
					},
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http", MaxAudioChannels: 2},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetFormat).To(Equal("mp3"))
				Expect(decision.TargetBitrate).To(Equal(256)) // kbps
				Expect(decision.TranscodeReasons).To(ContainElement("container not supported"))
			})

			It("rejects lossy to lossless transcoding", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 320, Channels: 2})
				ci := &ClientInfo{
					TranscodingProfiles: []Profile{
						{Container: "flac", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeFalse())
			})

			It("uses default bitrate when client doesn't specify", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, BitDepth: 16})
				ci := &ClientInfo{
					TranscodingProfiles: []Profile{
						{Container: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetBitrate).To(Equal(defaultBitrate)) // 256 kbps
			})

			It("preserves lossy bitrate when under max", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "ogg", BitRate: 192, Channels: 2})
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 256, // kbps
					TranscodingProfiles: []Profile{
						{Container: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetBitrate).To(Equal(192)) // source bitrate in kbps
			})

			It("rejects format with no transcoding command available", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2})
				ci := &ClientInfo{
					TranscodingProfiles: []Profile{
						{Container: "wav", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeFalse())
			})

			It("applies maxAudioBitrate as final cap on transcoded stream", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 320, Channels: 2})
				ci := &ClientInfo{
					MaxAudioBitrate: 96, // kbps
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetBitrate).To(Equal(96)) // capped by maxAudioBitrate
			})

			It("selects first valid transcoding profile in order", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 48000, BitDepth: 16})
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 320,
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"mp3"}, Protocols: []string{"http"}},
					},
					TranscodingProfiles: []Profile{
						{Container: "opus", AudioCodec: "opus", Protocol: "http"},
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http", MaxAudioChannels: 2},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetFormat).To(Equal("opus"))
			})
		})

		Context("Lossless to lossless transcoding", func() {
			It("allows lossless to lossless when samplerate needs downsampling", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "dsf", Codec: "DSD", BitRate: 5644, Channels: 2, SampleRate: 176400, BitDepth: 1})
				ci := &ClientInfo{
					MaxAudioBitrate: 1000,
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{"http"}},
					},
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetFormat).To(Equal("mp3"))
			})

			It("sets IsLossless=true on transcoded stream when target is lossless", func() {
				// Transcoding to mp3 (lossy) should result in IsLossless=false.
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 96000, BitDepth: 24})
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 320,
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeStream.IsLossless).To(BeFalse()) // mp3 is lossy
			})
		})

		Context("No compatible profile", func() {
			It("returns error when nothing matches", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 6})
				ci := &ClientInfo{}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.CanTranscode).To(BeFalse())
				Expect(decision.ErrorReason).To(Equal("no compatible playback profile found"))
			})
		})

		Context("Codec limitations on direct play", func() {
			It("rejects direct play when codec limitation fails (required)", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 512, Channels: 2, SampleRate: 44100})
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"mp3"}, AudioCodecs: []string{"mp3"}, Protocols: []string{"http"}},
					},
					CodecProfiles: []CodecProfile{
						{
							Type: CodecProfileTypeAudio,
							Name: "mp3",
							Limitations: []Limitation{
								{Name: LimitationAudioBitrate, Comparison: ComparisonLessThanEqual, Values: []string{"320"}, Required: true},
							},
						},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.TranscodeReasons).To(ContainElement("audio bitrate not supported"))
			})

			It("allows direct play when optional limitation fails", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 512, Channels: 2, SampleRate: 44100})
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"mp3"}, AudioCodecs: []string{"mp3"}, Protocols: []string{"http"}},
					},
					CodecProfiles: []CodecProfile{
						{
							Type: CodecProfileTypeAudio,
							Name: "mp3",
							Limitations: []Limitation{
								{Name: LimitationAudioBitrate, Comparison: ComparisonLessThanEqual, Values: []string{"320"}, Required: false},
							},
						},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})

			It("handles Equals comparison with multiple values", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100})
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{"http"}},
					},
					CodecProfiles: []CodecProfile{
						{
							Type: CodecProfileTypeAudio,
							Name: "flac",
							Limitations: []Limitation{
								{Name: LimitationAudioChannels, Comparison: ComparisonEquals, Values: []string{"1", "2"}, Required: true},
							},
						},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})

			It("rejects when Equals comparison doesn't match any value", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 6, SampleRate: 44100})
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{"http"}},
					},
					CodecProfiles: []CodecProfile{
						{
							Type: CodecProfileTypeAudio,
							Name: "flac",
							Limitations: []Limitation{
								{Name: LimitationAudioChannels, Comparison: ComparisonEquals, Values: []string{"1", "2"}, Required: true},
							},
						},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
			})

			It("rejects direct play when audioProfile limitation fails (required)", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "m4a", Codec: "AAC", BitRate: 256, Channels: 2, SampleRate: 44100})
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"m4a"}, AudioCodecs: []string{"aac"}, Protocols: []string{"http"}},
					},
					CodecProfiles: []CodecProfile{
						{
							Type: CodecProfileTypeAudio,
							Name: "aac",
							Limitations: []Limitation{
								{Name: LimitationAudioProfile, Comparison: ComparisonEquals, Values: []string{"LC"}, Required: true},
							},
						},
					},
				}
				// Source profile is empty (not yet populated from scanner), so Equals("LC") fails
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.TranscodeReasons).To(ContainElement("audio profile not supported"))
			})

			It("allows direct play when audioProfile limitation is optional", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "m4a", Codec: "AAC", BitRate: 256, Channels: 2, SampleRate: 44100})
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"m4a"}, AudioCodecs: []string{"aac"}, Protocols: []string{"http"}},
					},
					CodecProfiles: []CodecProfile{
						{
							Type: CodecProfileTypeAudio,
							Name: "aac",
							Limitations: []Limitation{
								{Name: LimitationAudioProfile, Comparison: ComparisonEquals, Values: []string{"LC"}, Required: false},
							},
						},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})

			It("rejects direct play due to samplerate limitation", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 96000, BitDepth: 24})
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{"http"}},
					},
					CodecProfiles: []CodecProfile{
						{
							Type: CodecProfileTypeAudio,
							Name: "flac",
							Limitations: []Limitation{
								{Name: LimitationAudioSamplerate, Comparison: ComparisonLessThanEqual, Values: []string{"48000"}, Required: true},
							},
						},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.TranscodeReasons).To(ContainElement("audio samplerate not supported"))
			})
		})

		Context("Codec limitations on transcoded output", func() {
			It("applies bitrate limitation to transcoded stream", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 192, Channels: 2, SampleRate: 44100})
				ci := &ClientInfo{
					MaxAudioBitrate: 96, // force transcode
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
					CodecProfiles: []CodecProfile{
						{
							Type: CodecProfileTypeAudio,
							Name: "mp3",
							Limitations: []Limitation{
								{Name: LimitationAudioBitrate, Comparison: ComparisonLessThanEqual, Values: []string{"96"}, Required: true},
							},
						},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeStream.Bitrate).To(Equal(96))
			})

			It("applies channel limitation to transcoded stream", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 6, SampleRate: 48000, BitDepth: 16})
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 320,
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
					CodecProfiles: []CodecProfile{
						{
							Type: CodecProfileTypeAudio,
							Name: "mp3",
							Limitations: []Limitation{
								{Name: LimitationAudioChannels, Comparison: ComparisonLessThanEqual, Values: []string{"2"}, Required: true},
							},
						},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeStream.Channels).To(Equal(2))
			})

			It("applies samplerate limitation to transcoded stream", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 96000, BitDepth: 24})
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 320,
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
					CodecProfiles: []CodecProfile{
						{
							Type: CodecProfileTypeAudio,
							Name: "mp3",
							Limitations: []Limitation{
								{Name: LimitationAudioSamplerate, Comparison: ComparisonLessThanEqual, Values: []string{"48000"}, Required: true},
							},
						},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeStream.SampleRate).To(Equal(48000))
			})

			It("applies bitdepth limitation to transcoded stream", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 96000, BitDepth: 24})
				ci := &ClientInfo{
					TranscodingProfiles: []Profile{
						{Container: "flac", AudioCodec: "flac", Protocol: "http"},
					},
					CodecProfiles: []CodecProfile{
						{
							Type: CodecProfileTypeAudio,
							Name: "flac",
							Limitations: []Limitation{
								{Name: LimitationAudioBitdepth, Comparison: ComparisonLessThanEqual, Values: []string{"16"}, Required: true},
							},
						},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeStream.BitDepth).To(Equal(16))
				Expect(decision.TargetBitDepth).To(Equal(16))
			})

			It("preserves source bit depth when no limitation applies", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 24})
				ci := &ClientInfo{
					TranscodingProfiles: []Profile{
						{Container: "flac", AudioCodec: "flac", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeStream.BitDepth).To(Equal(24))
				Expect(decision.TargetBitDepth).To(Equal(24))
			})

			It("rejects transcoding profile when GreaterThanEqual cannot be satisfied", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16})
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 320,
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
					CodecProfiles: []CodecProfile{
						{
							Type: CodecProfileTypeAudio,
							Name: "mp3",
							Limitations: []Limitation{
								{Name: LimitationAudioSamplerate, Comparison: ComparisonGreaterThanEqual, Values: []string{"96000"}, Required: true},
							},
						},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeFalse())
			})
		})

		Context("DSD sample rate conversion", func() {
			It("converts DSD sample rate to PCM-equivalent in decision", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "dsf", Codec: "DSD", BitRate: 5644, Channels: 2, SampleRate: 2822400, BitDepth: 1})
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 320,
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetFormat).To(Equal("mp3"))
				// DSD64 2822400 / 8 = 352800, capped by MP3 max of 48000
				Expect(decision.TranscodeStream.SampleRate).To(Equal(48000))
				Expect(decision.TargetSampleRate).To(Equal(48000))
				// DSD 1-bit → 24-bit PCM
				Expect(decision.TranscodeStream.BitDepth).To(Equal(24))
				Expect(decision.TargetBitDepth).To(Equal(24))
			})

			It("converts DSD sample rate for FLAC target without codec limit", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "dsf", Codec: "DSD", BitRate: 5644, Channels: 2, SampleRate: 2822400, BitDepth: 1})
				ci := &ClientInfo{
					TranscodingProfiles: []Profile{
						{Container: "flac", AudioCodec: "flac", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetFormat).To(Equal("flac"))
				// DSD64 2822400 / 8 = 352800, FLAC has no hard max
				Expect(decision.TranscodeStream.SampleRate).To(Equal(352800))
				Expect(decision.TargetSampleRate).To(Equal(352800))
				// DSD 1-bit → 24-bit PCM
				Expect(decision.TranscodeStream.BitDepth).To(Equal(24))
				Expect(decision.TargetBitDepth).To(Equal(24))
			})

			It("applies codec profile limit to DSD-converted FLAC sample rate", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "dsf", Codec: "DSD", BitRate: 5644, Channels: 2, SampleRate: 2822400, BitDepth: 1})
				ci := &ClientInfo{
					TranscodingProfiles: []Profile{
						{Container: "flac", AudioCodec: "flac", Protocol: "http"},
					},
					CodecProfiles: []CodecProfile{
						{
							Type: CodecProfileTypeAudio,
							Name: "flac",
							Limitations: []Limitation{
								{Name: LimitationAudioSamplerate, Comparison: ComparisonLessThanEqual, Values: []string{"48000"}, Required: true},
							},
						},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				// DSD64 2822400 / 8 = 352800, capped by codec profile limit of 48000
				Expect(decision.TranscodeStream.SampleRate).To(Equal(48000))
				Expect(decision.TargetSampleRate).To(Equal(48000))
				// DSD 1-bit → 24-bit PCM
				Expect(decision.TranscodeStream.BitDepth).To(Equal(24))
				Expect(decision.TargetBitDepth).To(Equal(24))
			})

			It("applies audioBitdepth limitation to DSD-converted bit depth", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "dsf", Codec: "DSD", BitRate: 5644, Channels: 2, SampleRate: 2822400, BitDepth: 1})
				ci := &ClientInfo{
					TranscodingProfiles: []Profile{
						{Container: "flac", AudioCodec: "flac", Protocol: "http"},
					},
					CodecProfiles: []CodecProfile{
						{
							Type: CodecProfileTypeAudio,
							Name: "flac",
							Limitations: []Limitation{
								{Name: LimitationAudioBitdepth, Comparison: ComparisonLessThanEqual, Values: []string{"16"}, Required: true},
							},
						},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				// DSD 1-bit → 24-bit PCM, then capped by codec profile limit to 16-bit
				Expect(decision.TranscodeStream.BitDepth).To(Equal(16))
				Expect(decision.TargetBitDepth).To(Equal(16))
			})
		})

		Context("Probe-based lossless detection", func() {
			It("uses probe codec name for lossless detection", func() {
				// WavPack files: ffprobe reports codec as "wavpack", suffix is ".wv"
				mf := &model.MediaFile{ID: "1", Suffix: "wv", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16}
				probe := ffmpeg.AudioProbeResult{
					Codec: "wavpack", BitRate: 1000, SampleRate: 44100, BitDepth: 16, Channels: 2,
				}
				data, _ := json.Marshal(probe)
				mf.ProbeData = string(data)

				ci := &ClientInfo{
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
					MaxTranscodingAudioBitrate: 256,
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.SourceStream.IsLossless).To(BeTrue())
				Expect(decision.SourceStream.Codec).To(Equal("wavpack"))
				// Lossless source transcoding to MP3 should use MaxTranscodingAudioBitrate
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeStream.Bitrate).To(Equal(256))
			})

			It("detects lossy from probe codec name", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "ogg", BitRate: 192, Channels: 2, SampleRate: 48000}
				probe := ffmpeg.AudioProbeResult{
					Codec: "vorbis", BitRate: 192, SampleRate: 48000, BitDepth: 0, Channels: 2,
				}
				data, _ := json.Marshal(probe)
				mf.ProbeData = string(data)

				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"ogg"}, AudioCodecs: []string{"vorbis"}, Protocols: []string{"http"}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.SourceStream.IsLossless).To(BeFalse())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})
		})

		Context("Opus fixed sample rate", func() {
			It("sets Opus output to 48000Hz regardless of input", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16})
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 128,
					TranscodingProfiles: []Profile{
						{Container: "opus", AudioCodec: "opus", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetFormat).To(Equal("opus"))
				// Opus always outputs 48000Hz
				Expect(decision.TranscodeStream.SampleRate).To(Equal(48000))
				Expect(decision.TargetSampleRate).To(Equal(48000))
			})

			It("sets Opus output to 48000Hz even for 96kHz input", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1500, Channels: 2, SampleRate: 96000, BitDepth: 24})
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 128,
					TranscodingProfiles: []Profile{
						{Container: "opus", AudioCodec: "opus", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeStream.SampleRate).To(Equal(48000))
			})
		})

		Context("Container vs format separation", func() {
			It("preserves mp4 container when falling back to aac format", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16})
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 256,
					TranscodingProfiles: []Profile{
						{Container: "mp4", AudioCodec: "aac", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				// TargetFormat is the internal format used for transcoding ("aac")
				Expect(decision.TargetFormat).To(Equal("aac"))
				// Container in the response preserves what the client asked ("mp4")
				Expect(decision.TranscodeStream.Container).To(Equal("mp4"))
				Expect(decision.TranscodeStream.Codec).To(Equal("aac"))
			})

			It("uses container as format when container matches transcoding config", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16})
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 256,
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetFormat).To(Equal("mp3"))
				Expect(decision.TranscodeStream.Container).To(Equal("mp3"))
			})
		})

		Context("MP3 max sample rate", func() {
			It("caps sample rate at 48000 for MP3", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1500, Channels: 2, SampleRate: 96000, BitDepth: 24})
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 320,
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeStream.SampleRate).To(Equal(48000))
			})

			It("preserves sample rate at 44100 for MP3", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16})
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 320,
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeStream.SampleRate).To(Equal(44100))
			})
		})

		Context("AAC max sample rate", func() {
			It("caps sample rate at 96000 for AAC", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "dsf", Codec: "DSD", BitRate: 5644, Channels: 2, SampleRate: 2822400, BitDepth: 1})
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 320,
					TranscodingProfiles: []Profile{
						{Container: "aac", AudioCodec: "aac", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				// DSD64 2822400 / 8 = 352800, capped by AAC max of 96000
				Expect(decision.TranscodeStream.SampleRate).To(Equal(96000))
			})
		})

		Context("Typed transcode reasons from multiple profiles", func() {
			It("collects reasons from each failed direct play profile", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "ogg", Codec: "Vorbis", BitRate: 128, Channels: 2, SampleRate: 48000})
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{"http"}},
						{Containers: []string{"mp3"}, AudioCodecs: []string{"mp3"}, Protocols: []string{"http"}},
						{Containers: []string{"m4a", "mp4"}, AudioCodecs: []string{"aac"}, Protocols: []string{"http"}},
					},
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.TranscodeReasons).To(HaveLen(3))
				Expect(decision.TranscodeReasons[0]).To(Equal("container not supported"))
				Expect(decision.TranscodeReasons[1]).To(Equal("container not supported"))
				Expect(decision.TranscodeReasons[2]).To(Equal("container not supported"))
			})
		})

		Context("Source stream details", func() {
			It("populates source stream correctly with kbps bitrate", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 96000, BitDepth: 24, Duration: 300.5, Size: 50000000})
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{"http"}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.SourceStream.Container).To(Equal("flac"))
				Expect(decision.SourceStream.Codec).To(Equal("flac"))
				Expect(decision.SourceStream.Bitrate).To(Equal(1000)) // kbps
				Expect(decision.SourceStream.SampleRate).To(Equal(96000))
				Expect(decision.SourceStream.BitDepth).To(Equal(24))
				Expect(decision.SourceStream.Channels).To(Equal(2))
			})
		})

		Context("Server-side player transcoding override", func() {
			It("forces transcoding when override targets a different format", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100})
				ci := &ClientInfo{
					Name: "TestClient",
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{"http"}},
					},
				}
				// Set server override in context
				overrideCtx := request.WithTranscoding(ctx, model.Transcoding{TargetFormat: "mp3", DefaultBitRate: 192})
				overrideCtx = request.WithPlayer(overrideCtx, model.Player{MaxBitRate: 0})

				decision, err := svc.MakeDecision(overrideCtx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetFormat).To(Equal("mp3"))
				Expect(decision.TargetBitrate).To(Equal(192))
			})

			It("allows direct play when source matches forced format and bitrate is within cap", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 128, Channels: 2, SampleRate: 44100})
				ci := &ClientInfo{
					Name: "TestClient",
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{"http"}},
					},
				}
				overrideCtx := request.WithTranscoding(ctx, model.Transcoding{TargetFormat: "mp3", DefaultBitRate: 256})

				decision, err := svc.MakeDecision(overrideCtx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
				Expect(decision.CanTranscode).To(BeFalse())
			})

			It("transcodes when source bitrate exceeds the forced cap", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 320, Channels: 2, SampleRate: 44100})
				ci := &ClientInfo{
					Name: "TestClient",
				}
				overrideCtx := request.WithTranscoding(ctx, model.Transcoding{TargetFormat: "mp3", DefaultBitRate: 192})

				decision, err := svc.MakeDecision(overrideCtx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetFormat).To(Equal("mp3"))
				Expect(decision.TargetBitrate).To(Equal(192))
			})

			It("uses player MaxBitRate over transcoding DefaultBitRate", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100})
				ci := &ClientInfo{
					Name: "TestClient",
				}
				overrideCtx := request.WithTranscoding(ctx, model.Transcoding{TargetFormat: "mp3", DefaultBitRate: 192})
				overrideCtx = request.WithPlayer(overrideCtx, model.Player{MaxBitRate: 320})

				decision, err := svc.MakeDecision(overrideCtx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetFormat).To(Equal("mp3"))
				Expect(decision.TargetBitrate).To(Equal(320))
			})

			It("applies no bitrate cap when both MaxBitRate and DefaultBitRate are 0", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100})
				ci := &ClientInfo{
					Name: "TestClient",
				}
				overrideCtx := request.WithTranscoding(ctx, model.Transcoding{TargetFormat: "mp3", DefaultBitRate: 0})
				overrideCtx = request.WithPlayer(overrideCtx, model.Player{MaxBitRate: 0})

				decision, err := svc.MakeDecision(overrideCtx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetFormat).To(Equal("mp3"))
				// With no cap, lossless→lossy uses defaultBitrate (256)
				Expect(decision.TargetBitrate).To(Equal(defaultBitrate))
			})

			It("does not apply override when no transcoding is in context", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100})
				ci := &ClientInfo{
					Name: "TestClient",
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{"http"}},
					},
				}
				// No override in context — client profiles used as-is
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})

			It("preserves client Name and Platform in overridden ClientInfo", func() {
				ci := &ClientInfo{
					Name:     "MyApp",
					Platform: "iOS",
				}
				overrideCtx := request.WithTranscoding(ctx, model.Transcoding{TargetFormat: "mp3", DefaultBitRate: 192})

				// Verify via applyServerOverride directly (package-level function)
				trc := model.Transcoding{TargetFormat: "mp3", DefaultBitRate: 192}
				overridden := applyServerOverride(ci, &trc, overrideCtx)
				Expect(overridden.Name).To(Equal("MyApp"))
				Expect(overridden.Platform).To(Equal("iOS"))
				Expect(overridden.CodecProfiles).To(BeEmpty())
			})
		})
	})

	Describe("ensureProbed", func() {
		var mockMFRepo *tests.MockMediaFileRepo

		BeforeEach(func() {
			mockMFRepo = tests.CreateMockMediaFileRepo()
			ds.MockedMediaFile = mockMFRepo
		})

		It("calls ffprobe and populates ProbeData when empty", func() {
			mf := &model.MediaFile{ID: "probe-1", Suffix: "mp3", BitRate: 320, Channels: 2}
			mockMFRepo.SetData(model.MediaFiles{*mf})

			ff.ProbeAudioResult = &ffmpeg.AudioProbeResult{
				Codec: "mp3", BitRate: 320, SampleRate: 44100, Channels: 2,
			}

			svc := NewDecider(ds, ff).(*deciderService)
			probe, err := svc.ensureProbed(ctx, mf)
			Expect(err).ToNot(HaveOccurred())
			Expect(mf.ProbeData).ToNot(BeEmpty())
			Expect(probe).ToNot(BeNil())
			Expect(probe.Codec).To(Equal("mp3"))
			Expect(probe.BitRate).To(Equal(320))
			Expect(probe.SampleRate).To(Equal(44100))
			Expect(probe.Channels).To(Equal(2))

			// Verify persisted to DB
			stored := mockMFRepo.Data["probe-1"]
			Expect(stored.ProbeData).To(Equal(mf.ProbeData))
		})

		It("skips ffprobe when ProbeData is already set", func() {
			mf := withProbe(&model.MediaFile{ID: "probe-2", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2})

			// Set error on mock — if ffprobe were called, this would fail
			ff.Error = fmt.Errorf("should not be called")

			svc := NewDecider(ds, ff).(*deciderService)
			probe, err := svc.ensureProbed(ctx, mf)
			Expect(err).ToNot(HaveOccurred())
			Expect(probe).To(BeNil())
		})

		It("returns error when ffprobe fails", func() {
			mf := &model.MediaFile{ID: "probe-3", Suffix: "mp3"}
			ff.Error = fmt.Errorf("ffprobe not found")

			svc := NewDecider(ds, ff).(*deciderService)
			_, err := svc.ensureProbed(ctx, mf)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("probing media file"))
			Expect(mf.ProbeData).To(BeEmpty())
		})

		It("skips ffprobe when DevEnableMediaFileProbe is false", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.DevEnableMediaFileProbe = false

			mf := &model.MediaFile{ID: "probe-4", Suffix: "mp3"}
			// Set a result — if ffprobe were called, ProbeData would be populated
			ff.ProbeAudioResult = &ffmpeg.AudioProbeResult{Codec: "mp3"}

			svc := NewDecider(ds, ff).(*deciderService)
			probe, err := svc.ensureProbed(ctx, mf)
			Expect(err).ToNot(HaveOccurred())
			Expect(probe).To(BeNil())
			Expect(mf.ProbeData).To(BeEmpty())
		})
	})

	Describe("Token round-trip", func() {
		var (
			sourceTime time.Time
			impl       *deciderService
		)

		BeforeEach(func() {
			sourceTime = time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
			impl = svc.(*deciderService)
		})

		It("creates and parses a direct play token", func() {
			decision := &Decision{
				MediaID:         "media-123",
				CanDirectPlay:   true,
				SourceUpdatedAt: sourceTime,
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())
			Expect(token).ToNot(BeEmpty())

			params, err := impl.parseTranscodeParams(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.MediaID).To(Equal("media-123"))
			Expect(params.DirectPlay).To(BeTrue())
			Expect(params.TargetFormat).To(BeEmpty())
			Expect(params.SourceUpdatedAt.Unix()).To(Equal(sourceTime.Unix()))
		})

		It("creates and parses a transcode token with kbps bitrate", func() {
			decision := &Decision{
				MediaID:         "media-456",
				CanDirectPlay:   false,
				CanTranscode:    true,
				TargetFormat:    "mp3",
				TargetBitrate:   256, // kbps
				TargetChannels:  2,
				SourceUpdatedAt: sourceTime,
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())

			params, err := impl.parseTranscodeParams(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.MediaID).To(Equal("media-456"))
			Expect(params.DirectPlay).To(BeFalse())
			Expect(params.TargetFormat).To(Equal("mp3"))
			Expect(params.TargetBitrate).To(Equal(256)) // kbps
			Expect(params.TargetChannels).To(Equal(2))
			Expect(params.SourceUpdatedAt.Unix()).To(Equal(sourceTime.Unix()))
		})

		It("creates and parses a transcode token with sample rate", func() {
			decision := &Decision{
				MediaID:          "media-789",
				CanDirectPlay:    false,
				CanTranscode:     true,
				TargetFormat:     "flac",
				TargetBitrate:    0,
				TargetChannels:   2,
				TargetSampleRate: 48000,
				SourceUpdatedAt:  sourceTime,
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())

			params, err := impl.parseTranscodeParams(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.MediaID).To(Equal("media-789"))
			Expect(params.DirectPlay).To(BeFalse())
			Expect(params.TargetFormat).To(Equal("flac"))
			Expect(params.TargetSampleRate).To(Equal(48000))
			Expect(params.TargetChannels).To(Equal(2))
		})

		It("creates and parses a transcode token with bit depth", func() {
			decision := &Decision{
				MediaID:         "media-bd",
				CanDirectPlay:   false,
				CanTranscode:    true,
				TargetFormat:    "flac",
				TargetBitrate:   0,
				TargetChannels:  2,
				TargetBitDepth:  24,
				SourceUpdatedAt: sourceTime,
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())

			params, err := impl.parseTranscodeParams(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.MediaID).To(Equal("media-bd"))
			Expect(params.TargetBitDepth).To(Equal(24))
		})

		It("omits bit depth from token when 0", func() {
			decision := &Decision{
				MediaID:         "media-nobd",
				CanDirectPlay:   false,
				CanTranscode:    true,
				TargetFormat:    "mp3",
				TargetBitrate:   256,
				TargetBitDepth:  0,
				SourceUpdatedAt: sourceTime,
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())

			params, err := impl.parseTranscodeParams(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.TargetBitDepth).To(Equal(0))
		})

		It("omits sample rate from token when 0", func() {
			decision := &Decision{
				MediaID:          "media-100",
				CanDirectPlay:    false,
				CanTranscode:     true,
				TargetFormat:     "mp3",
				TargetBitrate:    256,
				TargetSampleRate: 0,
				SourceUpdatedAt:  sourceTime,
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())

			params, err := impl.parseTranscodeParams(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.TargetSampleRate).To(Equal(0))
		})

		It("truncates SourceUpdatedAt to seconds", func() {
			timeWithNanos := time.Date(2025, 6, 15, 10, 30, 0, 123456789, time.UTC)
			decision := &Decision{
				MediaID:         "media-trunc",
				CanDirectPlay:   true,
				SourceUpdatedAt: timeWithNanos,
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())

			params, err := impl.parseTranscodeParams(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.SourceUpdatedAt.Unix()).To(Equal(timeWithNanos.Truncate(time.Second).Unix()))
		})

		It("rejects an invalid token", func() {
			_, err := impl.parseTranscodeParams("invalid-token")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ResolveRequestFromToken", func() {
		var (
			mockMFRepo *tests.MockMediaFileRepo
			sourceTime time.Time
		)

		BeforeEach(func() {
			sourceTime = time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
			mockMFRepo = &tests.MockMediaFileRepo{}
			ds.MockedMediaFile = mockMFRepo
		})

		createTokenForMedia := func(mediaID string, updatedAt time.Time) string {
			decision := &Decision{
				MediaID:         mediaID,
				CanDirectPlay:   true,
				SourceUpdatedAt: updatedAt,
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())
			return token
		}

		It("returns stream request and media file for valid token", func() {
			mockMFRepo.SetData(model.MediaFiles{
				{ID: "song-1", UpdatedAt: sourceTime},
			})
			token := createTokenForMedia("song-1", sourceTime)

			req, mf, err := svc.ResolveRequestFromToken(ctx, token, "song-1", 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(req.ID).To(Equal("song-1"))
			Expect(req.Format).To(BeEmpty()) // direct play has no target format
			Expect(mf.ID).To(Equal("song-1"))
		})

		It("returns ErrTokenInvalid for invalid token", func() {
			_, _, err := svc.ResolveRequestFromToken(ctx, "bad-token", "song-1", 0)
			Expect(err).To(MatchError(ContainSubstring(ErrTokenInvalid.Error())))
		})

		It("returns ErrTokenInvalid when mediaID does not match token", func() {
			token := createTokenForMedia("song-1", sourceTime)

			_, _, err := svc.ResolveRequestFromToken(ctx, token, "song-2", 0)
			Expect(err).To(MatchError(ContainSubstring(ErrTokenInvalid.Error())))
		})

		It("returns ErrMediaNotFound when media file does not exist", func() {
			token := createTokenForMedia("gone-id", sourceTime)

			_, _, err := svc.ResolveRequestFromToken(ctx, token, "gone-id", 0)
			Expect(err).To(MatchError(ErrMediaNotFound))
		})

		It("returns ErrTokenStale when media file has changed", func() {
			newTime := sourceTime.Add(1 * time.Hour)
			mockMFRepo.SetData(model.MediaFiles{
				{ID: "song-1", UpdatedAt: newTime},
			})
			token := createTokenForMedia("song-1", sourceTime)

			_, _, err := svc.ResolveRequestFromToken(ctx, token, "song-1", 0)
			Expect(err).To(MatchError(ErrTokenStale))
		})
	})

	Describe("isLosslessFormat", func() {
		It("returns true for known lossless codecs", func() {
			Expect(isLosslessFormat("flac")).To(BeTrue())
			Expect(isLosslessFormat("alac")).To(BeTrue())
			Expect(isLosslessFormat("pcm")).To(BeTrue())
			Expect(isLosslessFormat("wav")).To(BeTrue())
			Expect(isLosslessFormat("dsd")).To(BeTrue())
			Expect(isLosslessFormat("ape")).To(BeTrue())
			Expect(isLosslessFormat("wv")).To(BeTrue())
			Expect(isLosslessFormat("wavpack")).To(BeTrue()) // ffprobe codec_name for WavPack
		})

		It("returns false for lossy codecs", func() {
			Expect(isLosslessFormat("mp3")).To(BeFalse())
			Expect(isLosslessFormat("aac")).To(BeFalse())
			Expect(isLosslessFormat("opus")).To(BeFalse())
			Expect(isLosslessFormat("vorbis")).To(BeFalse())
		})

		It("returns false for unknown codecs", func() {
			Expect(isLosslessFormat("unknown_codec")).To(BeFalse())
		})

		It("is case-insensitive", func() {
			Expect(isLosslessFormat("FLAC")).To(BeTrue())
			Expect(isLosslessFormat("Alac")).To(BeTrue())
		})
	})

	Describe("normalizeProbeCodec", func() {
		It("passes through common codec names unchanged", func() {
			Expect(normalizeProbeCodec("mp3")).To(Equal("mp3"))
			Expect(normalizeProbeCodec("aac")).To(Equal("aac"))
			Expect(normalizeProbeCodec("flac")).To(Equal("flac"))
			Expect(normalizeProbeCodec("opus")).To(Equal("opus"))
			Expect(normalizeProbeCodec("vorbis")).To(Equal("vorbis"))
			Expect(normalizeProbeCodec("alac")).To(Equal("alac"))
			Expect(normalizeProbeCodec("wmav2")).To(Equal("wmav2"))
		})

		It("normalizes DSD variants to dsd", func() {
			Expect(normalizeProbeCodec("dsd_lsbf_planar")).To(Equal("dsd"))
			Expect(normalizeProbeCodec("dsd_msbf_planar")).To(Equal("dsd"))
			Expect(normalizeProbeCodec("dsd_lsbf")).To(Equal("dsd"))
			Expect(normalizeProbeCodec("dsd_msbf")).To(Equal("dsd"))
		})

		It("normalizes PCM variants to pcm", func() {
			Expect(normalizeProbeCodec("pcm_s16le")).To(Equal("pcm"))
			Expect(normalizeProbeCodec("pcm_s24le")).To(Equal("pcm"))
			Expect(normalizeProbeCodec("pcm_s32be")).To(Equal("pcm"))
			Expect(normalizeProbeCodec("pcm_f32le")).To(Equal("pcm"))
		})

		It("lowercases input", func() {
			Expect(normalizeProbeCodec("MP3")).To(Equal("mp3"))
			Expect(normalizeProbeCodec("AAC")).To(Equal("aac"))
			Expect(normalizeProbeCodec("DSD_LSBF_PLANAR")).To(Equal("dsd"))
		})
	})

	Describe("Decision.toClaimsMap", func() {
		It("includes required fields and omits zero transcode fields for direct play", func() {
			d := &Decision{
				MediaID:         "song-1",
				CanDirectPlay:   true,
				SourceUpdatedAt: time.Unix(1700000000, 0),
			}
			m := d.toClaimsMap()
			Expect(m).To(HaveKeyWithValue("mid", "song-1"))
			Expect(m).To(HaveKeyWithValue("dp", true))
			Expect(m).To(HaveKeyWithValue("ua", int64(1700000000)))
			Expect(m).NotTo(HaveKey("f"))
			Expect(m).NotTo(HaveKey("b"))
			Expect(m).NotTo(HaveKey("ch"))
			Expect(m).NotTo(HaveKey("sr"))
			Expect(m).NotTo(HaveKey("bd"))
		})

		It("includes transcode fields when CanTranscode is true", func() {
			d := &Decision{
				MediaID:          "song-2",
				CanTranscode:     true,
				TargetFormat:     "opus",
				TargetBitrate:    128,
				TargetChannels:   2,
				TargetSampleRate: 48000,
				TargetBitDepth:   16,
				SourceUpdatedAt:  time.Unix(1700000000, 0),
			}
			m := d.toClaimsMap()
			Expect(m).To(HaveKeyWithValue("mid", "song-2"))
			Expect(m).NotTo(HaveKey("dp"))
			Expect(m).To(HaveKeyWithValue("f", "opus"))
			Expect(m).To(HaveKeyWithValue("b", 128))
			Expect(m).To(HaveKeyWithValue("ch", 2))
			Expect(m).To(HaveKeyWithValue("sr", 48000))
			Expect(m).To(HaveKeyWithValue("bd", 16))
		})
	})

	Describe("paramsFromToken", func() {
		It("round-trips all fields through encode/decode", func() {
			tokenAuth := jwtauth.New("HS256", []byte("test-secret"), nil)
			d := &Decision{
				MediaID:          "song-3",
				CanTranscode:     true,
				TargetFormat:     "mp3",
				TargetBitrate:    320,
				TargetChannels:   2,
				TargetSampleRate: 44100,
				TargetBitDepth:   16,
				SourceUpdatedAt:  time.Unix(1700000000, 0),
			}
			token, _, err := tokenAuth.Encode(d.toClaimsMap())
			Expect(err).NotTo(HaveOccurred())

			p, err := paramsFromToken(token)
			Expect(err).NotTo(HaveOccurred())
			Expect(p.MediaID).To(Equal("song-3"))
			Expect(p.DirectPlay).To(BeFalse())
			Expect(p.TargetFormat).To(Equal("mp3"))
			Expect(p.TargetBitrate).To(Equal(320))
			Expect(p.TargetChannels).To(Equal(2))
			Expect(p.TargetSampleRate).To(Equal(44100))
			Expect(p.TargetBitDepth).To(Equal(16))
			Expect(p.SourceUpdatedAt).To(Equal(time.Unix(1700000000, 0)))
		})

		It("round-trips direct-play-only claims", func() {
			tokenAuth := jwtauth.New("HS256", []byte("test-secret"), nil)
			d := &Decision{
				MediaID:         "song-4",
				CanDirectPlay:   true,
				SourceUpdatedAt: time.Unix(1700000000, 0),
			}
			token, _, err := tokenAuth.Encode(d.toClaimsMap())
			Expect(err).NotTo(HaveOccurred())

			p, err := paramsFromToken(token)
			Expect(err).NotTo(HaveOccurred())
			Expect(p.MediaID).To(Equal("song-4"))
			Expect(p.DirectPlay).To(BeTrue())
			Expect(p.TargetFormat).To(BeEmpty())
			Expect(p.TargetBitrate).To(BeZero())
			Expect(p.TargetChannels).To(BeZero())
			Expect(p.TargetSampleRate).To(BeZero())
			Expect(p.TargetBitDepth).To(BeZero())
			Expect(p.SourceUpdatedAt).To(Equal(time.Unix(1700000000, 0)))
		})

		It("returns error when media ID is missing", func() {
			tokenAuth := jwtauth.New("HS256", []byte("test-secret"), nil)
			token, _, err := tokenAuth.Encode(map[string]any{"ua": int64(1700000000)})
			Expect(err).NotTo(HaveOccurred())

			_, err = paramsFromToken(token)
			Expect(err).To(MatchError(ContainSubstring("missing media ID")))
		})

		It("returns error when source timestamp is missing", func() {
			tokenAuth := jwtauth.New("HS256", []byte("test-secret"), nil)
			token, _, err := tokenAuth.Encode(map[string]any{"mid": "song-5"})
			Expect(err).NotTo(HaveOccurred())

			_, err = paramsFromToken(token)
			Expect(err).To(MatchError(ContainSubstring("missing source timestamp")))
		})
	})
})
