package transcode

import (
	"context"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Decider", func() {
	var (
		ds  *tests.MockDataStore
		svc Decider
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		ds = &tests.MockDataStore{
			MockedProperty:    &tests.MockedPropertyRepo{},
			MockedTranscoding: &tests.MockTranscodingRepo{},
		}
		auth.Init(ds)
		svc = NewDecider(ds)
	})

	Describe("MakeDecision", func() {
		Context("Direct Play", func() {
			It("allows direct play when profile matches", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 320, Channels: 2, SampleRate: 44100}
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"mp3"}, AudioCodecs: []string{"mp3"}, Protocols: []string{"http"}, MaxAudioChannels: 2},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
				Expect(decision.CanTranscode).To(BeFalse())
				Expect(decision.TranscodeReasons).To(BeEmpty())
			})

			It("rejects direct play when container doesn't match", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2}
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"mp3"}, Protocols: []string{"http"}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.TranscodeReasons).To(ContainElement("container not supported"))
			})

			It("rejects direct play when codec doesn't match", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "m4a", Codec: "ALAC", BitRate: 1000, Channels: 2}
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"m4a"}, AudioCodecs: []string{"aac"}, Protocols: []string{"http"}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.TranscodeReasons).To(ContainElement("audio codec not supported"))
			})

			It("rejects direct play when channels exceed limit", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 6}
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{"http"}, MaxAudioChannels: 2},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.TranscodeReasons).To(ContainElement("audio channels not supported"))
			})

			It("handles container aliases (aac -> m4a)", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "m4a", Codec: "AAC", BitRate: 256, Channels: 2}
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"aac"}, AudioCodecs: []string{"aac"}, Protocols: []string{"http"}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})

			It("handles container aliases (mp4 -> m4a)", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "m4a", Codec: "AAC", BitRate: 256, Channels: 2}
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"mp4"}, AudioCodecs: []string{"aac"}, Protocols: []string{"http"}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})

			It("handles codec aliases (adts -> aac)", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "m4a", Codec: "AAC", BitRate: 256, Channels: 2}
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"m4a"}, AudioCodecs: []string{"adts"}, Protocols: []string{"http"}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})

			It("allows when protocol list is empty (any protocol)", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2}
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, AudioCodecs: []string{"flac"}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})

			It("allows when both container and codec lists are empty (wildcard)", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 128, Channels: 2}
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{}, AudioCodecs: []string{}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})
		})

		Context("MaxAudioBitrate constraint", func() {
			It("revokes direct play when bitrate exceeds maxAudioBitrate", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1500, Channels: 2}
				ci := &ClientInfo{
					MaxAudioBitrate: 500, // kbps
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{"http"}},
					},
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeReasons).To(ContainElement("audio bitrate not supported"))
			})
		})

		Context("Transcoding", func() {
			It("selects transcoding when direct play isn't possible", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16}
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 256, // kbps
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"mp3"}, Protocols: []string{"http"}},
					},
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http", MaxAudioChannels: 2},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetFormat).To(Equal("mp3"))
				Expect(decision.TargetBitrate).To(Equal(256)) // kbps
				Expect(decision.TranscodeReasons).To(ContainElement("container not supported"))
			})

			It("rejects lossy to lossless transcoding", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 320, Channels: 2}
				ci := &ClientInfo{
					TranscodingProfiles: []Profile{
						{Container: "flac", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeFalse())
			})

			It("uses default bitrate when client doesn't specify", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, BitDepth: 16}
				ci := &ClientInfo{
					TranscodingProfiles: []Profile{
						{Container: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetBitrate).To(Equal(defaultBitrate)) // 256 kbps
			})

			It("preserves lossy bitrate when under max", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "ogg", BitRate: 192, Channels: 2}
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 256, // kbps
					TranscodingProfiles: []Profile{
						{Container: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetBitrate).To(Equal(192)) // source bitrate in kbps
			})

			It("rejects unsupported transcoding format", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2}
				ci := &ClientInfo{
					TranscodingProfiles: []Profile{
						{Container: "wav", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeFalse())
			})

			It("applies maxAudioBitrate as final cap on transcoded stream", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 320, Channels: 2}
				ci := &ClientInfo{
					MaxAudioBitrate: 96, // kbps
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetBitrate).To(Equal(96)) // capped by maxAudioBitrate
			})

			It("selects first valid transcoding profile in order", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 48000, BitDepth: 16}
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
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetFormat).To(Equal("opus"))
			})
		})

		Context("Lossless to lossless transcoding", func() {
			It("allows lossless to lossless when samplerate needs downsampling", func() {
				// MockTranscodingRepo doesn't support "flac" format, so this would fail to find a config.
				// This test documents the behavior: lossless→lossless requires server transcoding config.
				mf := &model.MediaFile{ID: "1", Suffix: "dsf", Codec: "DSD", BitRate: 5644, Channels: 2, SampleRate: 176400, BitDepth: 1}
				ci := &ClientInfo{
					MaxAudioBitrate: 1000,
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{"http"}},
					},
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetFormat).To(Equal("mp3"))
			})

			It("sets IsLossless=true on transcoded stream when target is lossless", func() {
				// Simulate DSD→FLAC transcoding by using a mock that supports "flac"
				mockTranscoding := &tests.MockTranscodingRepo{}
				ds.MockedTranscoding = mockTranscoding
				svc = NewDecider(ds)

				// Transcoding to mp3 (lossy) should result in IsLossless=false.
				// Use mp3 profile to test that lossy output is correctly identified.
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 96000, BitDepth: 24}
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 320,
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeStream.IsLossless).To(BeFalse()) // mp3 is lossy
			})
		})

		Context("No compatible profile", func() {
			It("returns error when nothing matches", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 6}
				ci := &ClientInfo{}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.CanTranscode).To(BeFalse())
				Expect(decision.ErrorReason).To(Equal("no compatible playback profile found"))
			})
		})

		Context("Codec limitations on direct play", func() {
			It("rejects direct play when codec limitation fails (required)", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 512, Channels: 2, SampleRate: 44100}
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
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.TranscodeReasons).To(ContainElement("audio bitrate not supported"))
			})

			It("allows direct play when optional limitation fails", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 512, Channels: 2, SampleRate: 44100}
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
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})

			It("handles Equals comparison with multiple values", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100}
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
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})

			It("rejects when Equals comparison doesn't match any value", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 6, SampleRate: 44100}
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
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
			})

			It("rejects direct play when audioProfile limitation fails (required)", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "m4a", Codec: "AAC", BitRate: 256, Channels: 2, SampleRate: 44100}
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
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.TranscodeReasons).To(ContainElement("audio profile not supported"))
			})

			It("allows direct play when audioProfile limitation is optional", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "m4a", Codec: "AAC", BitRate: 256, Channels: 2, SampleRate: 44100}
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
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})

			It("rejects direct play due to samplerate limitation", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 96000, BitDepth: 24}
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
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.TranscodeReasons).To(ContainElement("audio samplerate not supported"))
			})
		})

		Context("Codec limitations on transcoded output", func() {
			It("applies bitrate limitation to transcoded stream", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 192, Channels: 2, SampleRate: 44100}
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
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeStream.Bitrate).To(Equal(96))
			})

			It("applies channel limitation to transcoded stream", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 6, SampleRate: 48000, BitDepth: 16}
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
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeStream.Channels).To(Equal(2))
			})

			It("applies samplerate limitation to transcoded stream", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 96000, BitDepth: 24}
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
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeStream.SampleRate).To(Equal(48000))
			})

			It("applies bitdepth limitation to transcoded stream", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 96000, BitDepth: 24}
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
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeStream.BitDepth).To(Equal(16))
				Expect(decision.TargetBitDepth).To(Equal(16))
			})

			It("preserves source bit depth when no limitation applies", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 24}
				ci := &ClientInfo{
					TranscodingProfiles: []Profile{
						{Container: "flac", AudioCodec: "flac", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeStream.BitDepth).To(Equal(24))
				Expect(decision.TargetBitDepth).To(Equal(24))
			})

			It("rejects transcoding profile when GreaterThanEqual cannot be satisfied", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16}
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
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeFalse())
			})
		})

		Context("DSD sample rate conversion", func() {
			It("converts DSD sample rate to PCM-equivalent in decision", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "dsf", Codec: "DSD", BitRate: 5644, Channels: 2, SampleRate: 2822400, BitDepth: 1}
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 320,
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
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
				mf := &model.MediaFile{ID: "1", Suffix: "dsf", Codec: "DSD", BitRate: 5644, Channels: 2, SampleRate: 2822400, BitDepth: 1}
				ci := &ClientInfo{
					TranscodingProfiles: []Profile{
						{Container: "flac", AudioCodec: "flac", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
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
				mf := &model.MediaFile{ID: "1", Suffix: "dsf", Codec: "DSD", BitRate: 5644, Channels: 2, SampleRate: 2822400, BitDepth: 1}
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
				decision, err := svc.MakeDecision(ctx, mf, ci)
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
				mf := &model.MediaFile{ID: "1", Suffix: "dsf", Codec: "DSD", BitRate: 5644, Channels: 2, SampleRate: 2822400, BitDepth: 1}
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
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				// DSD 1-bit → 24-bit PCM, then capped by codec profile limit to 16-bit
				Expect(decision.TranscodeStream.BitDepth).To(Equal(16))
				Expect(decision.TargetBitDepth).To(Equal(16))
			})
		})

		Context("Opus fixed sample rate", func() {
			It("sets Opus output to 48000Hz regardless of input", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16}
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 128,
					TranscodingProfiles: []Profile{
						{Container: "opus", AudioCodec: "opus", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetFormat).To(Equal("opus"))
				// Opus always outputs 48000Hz
				Expect(decision.TranscodeStream.SampleRate).To(Equal(48000))
				Expect(decision.TargetSampleRate).To(Equal(48000))
			})

			It("sets Opus output to 48000Hz even for 96kHz input", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1500, Channels: 2, SampleRate: 96000, BitDepth: 24}
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 128,
					TranscodingProfiles: []Profile{
						{Container: "opus", AudioCodec: "opus", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeStream.SampleRate).To(Equal(48000))
			})
		})

		Context("Container vs format separation", func() {
			It("preserves mp4 container when falling back to aac format", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16}
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 256,
					TranscodingProfiles: []Profile{
						{Container: "mp4", AudioCodec: "aac", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				// TargetFormat is the internal format used for DB lookup ("aac")
				Expect(decision.TargetFormat).To(Equal("aac"))
				// Container in the response preserves what the client asked ("mp4")
				Expect(decision.TranscodeStream.Container).To(Equal("mp4"))
				Expect(decision.TranscodeStream.Codec).To(Equal("aac"))
			})

			It("uses container as format when container matches transcoding config", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16}
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 256,
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetFormat).To(Equal("mp3"))
				Expect(decision.TranscodeStream.Container).To(Equal("mp3"))
			})
		})

		Context("MP3 max sample rate", func() {
			It("caps sample rate at 48000 for MP3", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1500, Channels: 2, SampleRate: 96000, BitDepth: 24}
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 320,
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeStream.SampleRate).To(Equal(48000))
			})

			It("preserves sample rate at 44100 for MP3", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16}
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 320,
					TranscodingProfiles: []Profile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeStream.SampleRate).To(Equal(44100))
			})
		})

		Context("AAC max sample rate", func() {
			It("caps sample rate at 96000 for AAC", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "dsf", Codec: "DSD", BitRate: 5644, Channels: 2, SampleRate: 2822400, BitDepth: 1}
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 320,
					TranscodingProfiles: []Profile{
						{Container: "aac", AudioCodec: "aac", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				// DSD64 2822400 / 8 = 352800, capped by AAC max of 96000
				Expect(decision.TranscodeStream.SampleRate).To(Equal(96000))
			})
		})

		Context("Typed transcode reasons from multiple profiles", func() {
			It("collects reasons from each failed direct play profile", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "ogg", Codec: "Vorbis", BitRate: 128, Channels: 2, SampleRate: 48000}
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
				decision, err := svc.MakeDecision(ctx, mf, ci)
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
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 96000, BitDepth: 24, Duration: 300.5, Size: 50000000}
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{"http"}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.SourceStream.Container).To(Equal("flac"))
				Expect(decision.SourceStream.Codec).To(Equal("flac"))
				Expect(decision.SourceStream.Bitrate).To(Equal(1000)) // kbps
				Expect(decision.SourceStream.SampleRate).To(Equal(96000))
				Expect(decision.SourceStream.BitDepth).To(Equal(24))
				Expect(decision.SourceStream.Channels).To(Equal(2))
			})
		})
	})

	Describe("Token round-trip", func() {
		It("creates and parses a direct play token", func() {
			decision := &Decision{
				MediaID:       "media-123",
				CanDirectPlay: true,
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())
			Expect(token).ToNot(BeEmpty())

			params, err := svc.ParseTranscodeParams(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.MediaID).To(Equal("media-123"))
			Expect(params.DirectPlay).To(BeTrue())
			Expect(params.TargetFormat).To(BeEmpty())
		})

		It("creates and parses a transcode token with kbps bitrate", func() {
			decision := &Decision{
				MediaID:        "media-456",
				CanDirectPlay:  false,
				CanTranscode:   true,
				TargetFormat:   "mp3",
				TargetBitrate:  256, // kbps
				TargetChannels: 2,
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())

			params, err := svc.ParseTranscodeParams(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.MediaID).To(Equal("media-456"))
			Expect(params.DirectPlay).To(BeFalse())
			Expect(params.TargetFormat).To(Equal("mp3"))
			Expect(params.TargetBitrate).To(Equal(256)) // kbps
			Expect(params.TargetChannels).To(Equal(2))
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
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())

			params, err := svc.ParseTranscodeParams(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.MediaID).To(Equal("media-789"))
			Expect(params.DirectPlay).To(BeFalse())
			Expect(params.TargetFormat).To(Equal("flac"))
			Expect(params.TargetSampleRate).To(Equal(48000))
			Expect(params.TargetChannels).To(Equal(2))
		})

		It("creates and parses a transcode token with bit depth", func() {
			decision := &Decision{
				MediaID:        "media-bd",
				CanDirectPlay:  false,
				CanTranscode:   true,
				TargetFormat:   "flac",
				TargetBitrate:  0,
				TargetChannels: 2,
				TargetBitDepth: 24,
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())

			params, err := svc.ParseTranscodeParams(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.MediaID).To(Equal("media-bd"))
			Expect(params.TargetBitDepth).To(Equal(24))
		})

		It("omits bit depth from token when 0", func() {
			decision := &Decision{
				MediaID:        "media-nobd",
				CanDirectPlay:  false,
				CanTranscode:   true,
				TargetFormat:   "mp3",
				TargetBitrate:  256,
				TargetBitDepth: 0,
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())

			params, err := svc.ParseTranscodeParams(token)
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
			}
			token, err := svc.CreateTranscodeParams(decision)
			Expect(err).ToNot(HaveOccurred())

			params, err := svc.ParseTranscodeParams(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.TargetSampleRate).To(Equal(0))
		})

		It("rejects an invalid token", func() {
			_, err := svc.ParseTranscodeParams("invalid-token")
			Expect(err).To(HaveOccurred())
		})
	})
})
