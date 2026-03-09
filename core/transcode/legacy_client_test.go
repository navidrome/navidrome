package transcode

import (
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
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
		Expect(ci.DirectPlayProfiles).To(HaveLen(1))
		Expect(ci.DirectPlayProfiles[0].Containers).To(Equal([]string{"flac"}))
		Expect(ci.DirectPlayProfiles[0].AudioCodecs).To(Equal([]string{mf.AudioCodec()}))
		Expect(ci.DirectPlayProfiles[0].Protocols).To(Equal([]string{ProtocolHTTP}))
	})

	It("sets transcoding profile and bitrate for explicit format with bitrate", func() {
		ci := buildLegacyClientInfo(mf, "mp3", 192)

		Expect(ci.TranscodingProfiles).To(HaveLen(1))
		Expect(ci.TranscodingProfiles[0].Container).To(Equal("mp3"))
		Expect(ci.TranscodingProfiles[0].AudioCodec).To(Equal("mp3"))
		Expect(ci.MaxAudioBitrate).To(Equal(192))
		Expect(ci.MaxTranscodingAudioBitrate).To(Equal(192))
		Expect(ci.DirectPlayProfiles).To(HaveLen(1))
		Expect(ci.DirectPlayProfiles[0].Containers).To(Equal([]string{"flac"}))
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
