package core

import (
	"context"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TranscodeDecision", func() {
	var (
		ds  *tests.MockDataStore
		svc TranscodeDecision
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		ds = &tests.MockDataStore{
			MockedProperty:    &tests.MockedPropertyRepo{},
			MockedTranscoding: &tests.MockTranscodingRepo{},
		}
		auth.Init(ds)
		svc = NewTranscodeDecision(ds)
	})

	Describe("MakeDecision", func() {
		Context("Direct Play", func() {
			It("allows direct play when profile matches", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 320000, Channels: 2, SampleRate: 44100}
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
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000000, Channels: 2}
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"mp3"}, Protocols: []string{"http"}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
			})

			It("rejects direct play when codec doesn't match", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "m4a", Codec: "ALAC", BitRate: 1000000, Channels: 2}
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"m4a"}, AudioCodecs: []string{"aac"}, Protocols: []string{"http"}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
			})

			It("rejects direct play when channels exceed limit", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000000, Channels: 6}
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{"http"}, MaxAudioChannels: 2},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
			})

			It("handles container aliases (aac -> m4a)", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "m4a", Codec: "AAC", BitRate: 256000, Channels: 2}
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"aac"}, AudioCodecs: []string{"aac"}, Protocols: []string{"http"}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})

			It("allows when protocol list is empty (any protocol)", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000000, Channels: 2}
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, AudioCodecs: []string{"flac"}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})
		})

		Context("MaxAudioBitrate constraint", func() {
			It("revokes direct play when bitrate exceeds maxAudioBitrate", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1500000, Channels: 2}
				ci := &ClientInfo{
					MaxAudioBitrate: 500000,
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{"http"}},
					},
					TranscodingProfiles: []TranscodingProfile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TranscodeReasons).To(ContainElement("bitrate exceeds maxAudioBitrate"))
			})
		})

		Context("Transcoding", func() {
			It("selects transcoding when direct play isn't possible", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000000, Channels: 2, SampleRate: 44100, BitDepth: 16}
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 256000,
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"mp3"}, Protocols: []string{"http"}},
					},
					TranscodingProfiles: []TranscodingProfile{
						{Container: "mp3", AudioCodec: "mp3", Protocol: "http", MaxAudioChannels: 2},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetFormat).To(Equal("mp3"))
				Expect(decision.TargetBitrate).To(Equal(256))
			})

			It("rejects lossy to lossless transcoding", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 320000, Channels: 2}
				ci := &ClientInfo{
					TranscodingProfiles: []TranscodingProfile{
						{Container: "flac", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeFalse())
			})

			It("uses default bitrate when client doesn't specify", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000000, Channels: 2, BitDepth: 16}
				ci := &ClientInfo{
					TranscodingProfiles: []TranscodingProfile{
						{Container: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetBitrate).To(Equal(defaultTranscodeBitrate))
			})

			It("preserves lossy bitrate when under max", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "ogg", BitRate: 192000, Channels: 2}
				ci := &ClientInfo{
					MaxTranscodingAudioBitrate: 256000,
					TranscodingProfiles: []TranscodingProfile{
						{Container: "mp3", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeTrue())
				Expect(decision.TargetBitrate).To(Equal(192))
			})

			It("rejects unsupported transcoding format", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000000, Channels: 2}
				ci := &ClientInfo{
					TranscodingProfiles: []TranscodingProfile{
						{Container: "aac", Protocol: "http"},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanTranscode).To(BeFalse())
			})
		})

		Context("No compatible profile", func() {
			It("returns error when nothing matches", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000000, Channels: 6}
				ci := &ClientInfo{}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
				Expect(decision.CanTranscode).To(BeFalse())
				Expect(decision.ErrorReason).To(Equal("no compatible playback profile found"))
			})
		})

		Context("Codec limitations", func() {
			It("rejects direct play when codec limitation fails (required)", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 512000, Channels: 2, SampleRate: 44100}
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"mp3"}, AudioCodecs: []string{"mp3"}, Protocols: []string{"http"}},
					},
					CodecProfiles: []CodecProfile{
						{
							Type: "AudioCodec",
							Name: "mp3",
							Limitations: []Limitation{
								{Name: "audioBitrate", Comparison: "LessThanEqual", Values: []string{"320000"}, Required: true},
							},
						},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
			})

			It("allows direct play when optional limitation fails", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 512000, Channels: 2, SampleRate: 44100}
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"mp3"}, AudioCodecs: []string{"mp3"}, Protocols: []string{"http"}},
					},
					CodecProfiles: []CodecProfile{
						{
							Type: "AudioCodec",
							Name: "mp3",
							Limitations: []Limitation{
								{Name: "audioBitrate", Comparison: "LessThanEqual", Values: []string{"320000"}, Required: false},
							},
						},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})

			It("handles Equals comparison with multiple values", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000000, Channels: 2, SampleRate: 44100}
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{"http"}},
					},
					CodecProfiles: []CodecProfile{
						{
							Type: "AudioCodec",
							Name: "flac",
							Limitations: []Limitation{
								{Name: "audioChannels", Comparison: "Equals", Values: []string{"1", "2"}, Required: true},
							},
						},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeTrue())
			})

			It("rejects when Equals comparison doesn't match any value", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000000, Channels: 6, SampleRate: 44100}
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{"http"}},
					},
					CodecProfiles: []CodecProfile{
						{
							Type: "AudioCodec",
							Name: "flac",
							Limitations: []Limitation{
								{Name: "audioChannels", Comparison: "Equals", Values: []string{"1", "2"}, Required: true},
							},
						},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.CanDirectPlay).To(BeFalse())
			})
		})

		Context("Source stream details", func() {
			It("populates source stream correctly", func() {
				mf := &model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000000, Channels: 2, SampleRate: 96000, BitDepth: 24, Duration: 300.5, Size: 50000000}
				ci := &ClientInfo{
					DirectPlayProfiles: []DirectPlayProfile{
						{Containers: []string{"flac"}, Protocols: []string{"http"}},
					},
				}
				decision, err := svc.MakeDecision(ctx, mf, ci)
				Expect(err).ToNot(HaveOccurred())
				Expect(decision.SourceStream.Container).To(Equal("flac"))
				Expect(decision.SourceStream.Codec).To(Equal("flac"))
				Expect(decision.SourceStream.Bitrate).To(Equal(1000000))
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
			token, err := svc.CreateToken(decision)
			Expect(err).ToNot(HaveOccurred())
			Expect(token).ToNot(BeEmpty())

			params, err := svc.ParseToken(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.MediaID).To(Equal("media-123"))
			Expect(params.DirectPlay).To(BeTrue())
			Expect(params.TargetFormat).To(BeEmpty())
		})

		It("creates and parses a transcode token", func() {
			decision := &Decision{
				MediaID:        "media-456",
				CanDirectPlay:  false,
				CanTranscode:   true,
				TargetFormat:   "mp3",
				TargetBitrate:  256,
				TargetChannels: 2,
			}
			token, err := svc.CreateToken(decision)
			Expect(err).ToNot(HaveOccurred())

			params, err := svc.ParseToken(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(params.MediaID).To(Equal("media-456"))
			Expect(params.DirectPlay).To(BeFalse())
			Expect(params.TargetFormat).To(Equal("mp3"))
			Expect(params.TargetBitrate).To(Equal(256))
			Expect(params.TargetChannels).To(Equal(2))
		})

		It("rejects an invalid token", func() {
			_, err := svc.ParseToken("invalid-token")
			Expect(err).To(HaveOccurred())
		})
	})
})
