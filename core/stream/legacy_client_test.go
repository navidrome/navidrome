package stream

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("buildLegacyClientInfo", func() {
	var mf *model.MediaFile

	BeforeEach(func() {
		mf = &model.MediaFile{Suffix: "flac", BitRate: 960}
	})

	It("sets transcoding profile for explicit format without bitrate", func() {
		ci := buildLegacyClientInfo(mf, "mp3", 0)

		Expect(ci.Name).To(Equal("legacy"))
		Expect(ci.TranscodingProfiles).To(HaveLen(1))
		Expect(ci.TranscodingProfiles[0].Container).To(Equal("mp3"))
		Expect(ci.TranscodingProfiles[0].AudioCodec).To(Equal("mp3"))
		Expect(ci.TranscodingProfiles[0].Protocol).To(Equal(ProtocolHTTP))
		Expect(ci.MaxAudioBitrate).To(BeZero())
		Expect(ci.MaxTranscodingAudioBitrate).To(BeZero())
		Expect(ci.DirectPlayProfiles).To(BeEmpty())
	})

	It("does not add direct play profile when explicit format differs from source (no bitrate)", func() {
		ci := buildLegacyClientInfo(mf, "opus", 0)

		Expect(ci.TranscodingProfiles).To(HaveLen(1))
		Expect(ci.TranscodingProfiles[0].Container).To(Equal("opus"))
		Expect(ci.DirectPlayProfiles).To(BeEmpty())
	})

	It("adds direct play profile when explicit format matches source format", func() {
		ci := buildLegacyClientInfo(mf, "flac", 0)

		Expect(ci.TranscodingProfiles).To(HaveLen(1))
		Expect(ci.TranscodingProfiles[0].Container).To(Equal("flac"))
		Expect(ci.DirectPlayProfiles).To(HaveLen(1))
		Expect(ci.DirectPlayProfiles[0].Containers).To(Equal([]string{"flac"}))
		Expect(ci.DirectPlayProfiles[0].AudioCodecs).To(Equal([]string{mf.AudioCodec()}))
	})

	It("sets transcoding profile and bitrate for explicit format with bitrate", func() {
		ci := buildLegacyClientInfo(mf, "mp3", 192)

		Expect(ci.TranscodingProfiles).To(HaveLen(1))
		Expect(ci.TranscodingProfiles[0].Container).To(Equal("mp3"))
		Expect(ci.TranscodingProfiles[0].AudioCodec).To(Equal("mp3"))
		Expect(ci.MaxAudioBitrate).To(Equal(192))
		Expect(ci.MaxTranscodingAudioBitrate).To(Equal(192))
		Expect(ci.DirectPlayProfiles).To(BeEmpty())
	})

	It("returns direct play profile when no format and no bitrate", func() {
		ci := buildLegacyClientInfo(mf, "", 0)

		Expect(ci.DirectPlayProfiles).To(HaveLen(1))
		Expect(ci.DirectPlayProfiles[0].Containers).To(BeEmpty())
		Expect(ci.DirectPlayProfiles[0].AudioCodecs).To(BeEmpty())
		Expect(ci.DirectPlayProfiles[0].Protocols).To(Equal([]string{ProtocolHTTP}))
		Expect(ci.TranscodingProfiles).To(BeEmpty())
		Expect(ci.MaxAudioBitrate).To(BeZero())
	})

	It("uses default downsampling format for bitrate-only downsampling", func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DefaultDownsamplingFormat = "opus"

		ci := buildLegacyClientInfo(mf, "", 128)

		Expect(ci.TranscodingProfiles).To(HaveLen(1))
		Expect(ci.TranscodingProfiles[0].Container).To(Equal("opus"))
		Expect(ci.TranscodingProfiles[0].AudioCodec).To(Equal("opus"))
		Expect(ci.TranscodingProfiles[0].Protocol).To(Equal(ProtocolHTTP))
		Expect(ci.MaxAudioBitrate).To(Equal(128))
		Expect(ci.MaxTranscodingAudioBitrate).To(Equal(128))
		Expect(ci.DirectPlayProfiles).To(HaveLen(1))
		Expect(ci.DirectPlayProfiles[0].Containers).To(Equal([]string{"flac"}))
		Expect(ci.DirectPlayProfiles[0].AudioCodecs).To(Equal([]string{mf.AudioCodec()}))
	})

	It("returns direct play when bitrate >= source bitrate", func() {
		ci := buildLegacyClientInfo(mf, "", 960)

		Expect(ci.DirectPlayProfiles).To(HaveLen(1))
		Expect(ci.DirectPlayProfiles[0].Containers).To(BeEmpty())
		Expect(ci.DirectPlayProfiles[0].AudioCodecs).To(BeEmpty())
		Expect(ci.DirectPlayProfiles[0].Protocols).To(Equal([]string{ProtocolHTTP}))
		Expect(ci.TranscodingProfiles).To(BeEmpty())
		Expect(ci.MaxAudioBitrate).To(BeZero())
	})
})

var _ = Describe("ResolveRequest", func() {
	var (
		svc TranscodeDecider
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds := &tests.MockDataStore{
			MockedProperty:    &tests.MockedPropertyRepo{},
			MockedTranscoding: &tests.MockTranscodingRepo{},
		}
		ff := tests.NewMockFFmpeg("")
		auth.Init(ds)
		svc = NewTranscodeDecider(ds, ff)
	})

	It("returns raw when format is 'raw'", func() {
		mf := withProbe(&model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 320, Channels: 2, SampleRate: 44100})

		decider := svc.(*deciderService)
		req := decider.ResolveRequest(ctx, mf, "raw", 0, 0)

		Expect(req.Format).To(Equal("raw"))
	})

	It("returns raw (direct play) when no format or bitrate specified", func() {
		mf := withProbe(&model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 320, Channels: 2, SampleRate: 44100})

		decider := svc.(*deciderService)
		req := decider.ResolveRequest(ctx, mf, "", 0, 0)

		Expect(req.Format).To(Equal("raw"))
	})

	It("transcodes to requested format", func() {
		mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16})

		decider := svc.(*deciderService)
		req := decider.ResolveRequest(ctx, mf, "opus", 0, 0)

		Expect(req.Format).To(Equal("opus"))
	})

	It("transcodes to requested format with bitrate limit", func() {
		mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16})

		decider := svc.(*deciderService)
		req := decider.ResolveRequest(ctx, mf, "mp3", 128, 0)

		Expect(req.Format).To(Equal("mp3"))
		Expect(req.BitRate).To(Equal(128))
	})

	It("returns raw when requested format matches source and no bitrate reduction", func() {
		mf := withProbe(&model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 320, Channels: 2, SampleRate: 44100})

		decider := svc.(*deciderService)
		req := decider.ResolveRequest(ctx, mf, "mp3", 320, 0)

		Expect(req.Format).To(Equal("raw"))
	})

	It("downsamples when only bitrate is specified below source", func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DefaultDownsamplingFormat = "opus"

		mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16})

		decider := svc.(*deciderService)
		req := decider.ResolveRequest(ctx, mf, "", 128, 0)

		Expect(req.Format).To(Equal("opus"))
		Expect(req.BitRate).To(Equal(128))
	})

	It("passes offset through", func() {
		mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16})

		decider := svc.(*deciderService)
		req := decider.ResolveRequest(ctx, mf, "opus", 128, 30)

		Expect(req.Format).To(Equal("opus"))
		Expect(req.Offset).To(Equal(30))
	})

	Context("fallback for unknown format", func() {
		It("falls back to DefaultDownsamplingFormat", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.DefaultDownsamplingFormat = "opus"

			mf := withProbe(&model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 320, Channels: 2, SampleRate: 44100})

			decider := svc.(*deciderService)
			req := decider.ResolveRequest(ctx, mf, "xyz", 0, 0)

			Expect(req.Format).To(Equal("opus"))
		})

		It("falls back to raw when DefaultDownsamplingFormat is empty", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.DefaultDownsamplingFormat = ""

			mf := withProbe(&model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 320, Channels: 2, SampleRate: 44100})

			decider := svc.(*deciderService)
			req := decider.ResolveRequest(ctx, mf, "xyz", 0, 0)

			Expect(req.Format).To(Equal("raw"))
		})

		It("falls back to raw when DefaultDownsamplingFormat is also invalid", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.DefaultDownsamplingFormat = "xyz"

			mf := withProbe(&model.MediaFile{ID: "1", Suffix: "mp3", Codec: "MP3", BitRate: 320, Channels: 2, SampleRate: 44100})

			decider := svc.(*deciderService)
			req := decider.ResolveRequest(ctx, mf, "xyz", 0, 0)

			Expect(req.Format).To(Equal("raw"))
		})

		It("preserves bitrate when falling back to DefaultDownsamplingFormat", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.DefaultDownsamplingFormat = "opus"

			mf := withProbe(&model.MediaFile{ID: "1", Suffix: "flac", Codec: "FLAC", BitRate: 1000, Channels: 2, SampleRate: 44100, BitDepth: 16})

			decider := svc.(*deciderService)
			req := decider.ResolveRequest(ctx, mf, "xyz", 128, 0)

			Expect(req.Format).To(Equal("opus"))
			Expect(req.BitRate).To(Equal(128))
		})
	})
})
