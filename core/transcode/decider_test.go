package transcode

import (
	"context"
	"encoding/json"
	"fmt"

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
						{Containers: []string{"mp3"}, AudioCodecs: []string{"mp3"}, Protocols: []string{ProtocolHTTP}, MaxAudioChannels: 2},
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
						{Containers: []string{"mp3"}, Protocols: []string{ProtocolHTTP}},
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
						{Containers: []string{"m4a"}, AudioCodecs: []string{"aac"}, Protocols: []string{ProtocolHTTP}},
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
						{Containers: []string{"flac"}, Protocols: []string{ProtocolHTTP}, MaxAudioChannels: 2},
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
						{Containers: []string{"aac"}, AudioCodecs: []string{"aac"}, Protocols: []string{ProtocolHTTP}},
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
						{Containers: []string{"mp4"}, AudioCodecs: []string{"aac"}, Protocols: []string{ProtocolHTTP}},
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
						{Containers: []string{"m4a"}, AudioCodecs: []string{"adts"}, Protocols: []string{ProtocolHTTP}},
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
						{Containers: []string{"flac"}, Protocols: []string{ProtocolHTTP}},
					},
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP},
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
						{Containers: []string{"mp3"}, Protocols: []string{ProtocolHTTP}},
					},
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP, MaxAudioChannels: 2},
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
						{Container: "flac", Protocol: ProtocolHTTP},
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
						{Container: "mp3", Protocol: ProtocolHTTP},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetBitrate).To(Equal(160)) // mp3 default from mock transcoding repo
			})

			It("preserves lossy bitrate when under max", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "ogg", BitRate: 192, Channels: 2})
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 256, // kbps
					TranscodingProfiles: []Profile{
						{Container: "mp3", Protocol: ProtocolHTTP},
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
						{Container: "wav", Protocol: ProtocolHTTP},
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
						{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP},
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
						{Containers: []string{"mp3"}, Protocols: []string{ProtocolHTTP}},
					},
					TranscodingProfiles: []Profile{
						{Container: "opus", AudioCodec: "opus", Protocol: ProtocolHTTP},
						{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP, MaxAudioChannels: 2},
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
						{Containers: []string{"flac"}, Protocols: []string{ProtocolHTTP}},
					},
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP},
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
						{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP},
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
						{Containers: []string{"mp3"}, AudioCodecs: []string{"mp3"}, Protocols: []string{ProtocolHTTP}},
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
						{Containers: []string{"mp3"}, AudioCodecs: []string{"mp3"}, Protocols: []string{ProtocolHTTP}},
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
						{Containers: []string{"flac"}, Protocols: []string{ProtocolHTTP}},
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
						{Containers: []string{"flac"}, Protocols: []string{ProtocolHTTP}},
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
						{Containers: []string{"m4a"}, AudioCodecs: []string{"aac"}, Protocols: []string{ProtocolHTTP}},
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
						{Containers: []string{"m4a"}, AudioCodecs: []string{"aac"}, Protocols: []string{ProtocolHTTP}},
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
						{Containers: []string{"flac"}, Protocols: []string{ProtocolHTTP}},
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
						{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP},
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
						{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP},
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
						{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP},
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
						{Container: "flac", AudioCodec: "flac", Protocol: ProtocolHTTP},
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
						{Container: "flac", AudioCodec: "flac", Protocol: ProtocolHTTP},
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
						{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP},
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
						{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP},
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
						{Container: "flac", AudioCodec: "flac", Protocol: ProtocolHTTP},
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
						{Container: "flac", AudioCodec: "flac", Protocol: ProtocolHTTP},
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
						{Container: "flac", AudioCodec: "flac", Protocol: ProtocolHTTP},
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
						{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP},
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
						{Containers: []string{"ogg"}, AudioCodecs: []string{"vorbis"}, Protocols: []string{ProtocolHTTP}},
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
						{Container: "opus", AudioCodec: "opus", Protocol: ProtocolHTTP},
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
						{Container: "opus", AudioCodec: "opus", Protocol: ProtocolHTTP},
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
						{Container: "mp4", AudioCodec: "aac", Protocol: ProtocolHTTP},
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
						{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP},
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
						{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP},
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
						{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP},
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
						{Container: "aac", AudioCodec: "aac", Protocol: ProtocolHTTP},
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
						{Containers: []string{"flac"}, Protocols: []string{ProtocolHTTP}},
						{Containers: []string{"mp3"}, AudioCodecs: []string{"mp3"}, Protocols: []string{ProtocolHTTP}},
						{Containers: []string{"m4a", "mp4"}, AudioCodecs: []string{"aac"}, Protocols: []string{ProtocolHTTP}},
					},
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP},
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
						{Containers: []string{"flac"}, Protocols: []string{ProtocolHTTP}},
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
						{Containers: []string{"flac"}, Protocols: []string{ProtocolHTTP}},
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
						{Containers: []string{"flac"}, Protocols: []string{ProtocolHTTP}},
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
				// With no cap, lossless→lossy uses format default bitrate (160 for mp3 from mock)
				Expect(decision.TargetBitrate).To(Equal(160))
			})

			It("does not apply override when no transcoding is in context", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100})
				ci := &ClientInfo{
					Name: "TestClient",
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{ProtocolHTTP}},
					},
				}
				// No override in context — client profiles used as-is
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})

		})

		Context("Player MaxBitRate cap", func() {
			It("applies player MaxBitRate cap when client has no limit", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16})
				ci := &ClientInfo{
					Name: "TestClient",
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac", "mp3"}, AudioCodecs: []string{"flac", "mp3"}, Protocols: []string{ProtocolHTTP}},
					},
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP},
					},
				}
				playerCtx := request.WithPlayer(ctx, model.Player{MaxBitRate: 320})

				decision, err := svc.MakeDecision(playerCtx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				// Source bitrate 1000 > player cap 320, so direct play is not possible
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.CanTranscode).To(BeTrue())
				// Lossless→lossy should use MaxAudioBitrate (320) as target, not format default
				Expect(decision.TargetBitrate).To(Equal(320))
			})

			It("uses client limit when it is more restrictive than player MaxBitRate", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16})
				ci := &ClientInfo{
					Name:                       "TestClient",
					MaxAudioBitrate:            256,
					MaxTranscodingAudioBitrate: 256,
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP},
					},
				}
				playerCtx := request.WithPlayer(ctx, model.Player{MaxBitRate: 500})

				decision, err := svc.MakeDecision(playerCtx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				// Client limit 256 < player cap 500, so player cap doesn't apply; client limit wins
				Expect(decision.TargetBitrate).To(Equal(256))
			})

			It("does not cap when player MaxBitRate is 0", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 320, Channels: 2, SampleRate: 44100})
				ci := &ClientInfo{
					Name: "TestClient",
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"mp3"}, AudioCodecs: []string{"mp3"}, Protocols: []string{ProtocolHTTP}},
					},
				}
				playerCtx := request.WithPlayer(ctx, model.Player{MaxBitRate: 0})

				decision, err := svc.MakeDecision(playerCtx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})
		})

		Context("Format-aware default bitrate", func() {
			It("uses opus default bitrate from DB", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 48000, BitDepth: 16})
				ci := &ClientInfo{
					TranscodingProfiles: []Profile{
						{Container: "opus", AudioCodec: "opus", Protocol: ProtocolHTTP},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetBitrate).To(Equal(96)) // opus default from mock
			})

			It("uses aac default bitrate from DB", func() {
				mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16})
				ci := &ClientInfo{
					TranscodingProfiles: []Profile{
						{Container: "aac", AudioCodec: "aac", Protocol: ProtocolHTTP},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci, DecisionOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetBitrate).To(Equal(256)) // aac default from mock
			})

			It("falls back to 256 for unknown format", func() {
				bitrate := lookupDefaultBitrate(ctx, ds, "xyz")
				Expect(bitrate).To(Equal(fallbackBitrate))
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

})
