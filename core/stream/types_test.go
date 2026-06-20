package stream

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ClientInfo", func() {
	Describe("CapBitrate", func() {
		It("is a no-op when maxKbps is zero", func() {
			ci := &ClientInfo{MaxAudioBitrate: 320, MaxTranscodingAudioBitrate: 320}
			Expect(ci.CapBitrate(0)).To(BeFalse())
			Expect(ci.MaxAudioBitrate).To(Equal(320))
			Expect(ci.MaxTranscodingAudioBitrate).To(Equal(320))
		})

		It("is a no-op when maxKbps is negative", func() {
			ci := &ClientInfo{MaxAudioBitrate: 320, MaxTranscodingAudioBitrate: 320}
			Expect(ci.CapBitrate(-1)).To(BeFalse())
			Expect(ci.MaxAudioBitrate).To(Equal(320))
			Expect(ci.MaxTranscodingAudioBitrate).To(Equal(320))
		})

		It("sets both limits when both are zero (unlimited)", func() {
			ci := &ClientInfo{}
			Expect(ci.CapBitrate(256)).To(BeTrue())
			Expect(ci.MaxAudioBitrate).To(Equal(256))
			Expect(ci.MaxTranscodingAudioBitrate).To(Equal(256))
		})

		It("lowers limits higher than maxKbps", func() {
			ci := &ClientInfo{MaxAudioBitrate: 320, MaxTranscodingAudioBitrate: 500}
			Expect(ci.CapBitrate(192)).To(BeTrue())
			Expect(ci.MaxAudioBitrate).To(Equal(192))
			Expect(ci.MaxTranscodingAudioBitrate).To(Equal(192))
		})

		It("does not raise limits lower than maxKbps", func() {
			ci := &ClientInfo{MaxAudioBitrate: 128, MaxTranscodingAudioBitrate: 96}
			Expect(ci.CapBitrate(320)).To(BeFalse())
			Expect(ci.MaxAudioBitrate).To(Equal(128))
			Expect(ci.MaxTranscodingAudioBitrate).To(Equal(96))
		})

		It("reports changed when only one limit is lowered", func() {
			ci := &ClientInfo{MaxAudioBitrate: 320, MaxTranscodingAudioBitrate: 128}
			Expect(ci.CapBitrate(192)).To(BeTrue())
			Expect(ci.MaxAudioBitrate).To(Equal(192))
			Expect(ci.MaxTranscodingAudioBitrate).To(Equal(128))
		})

		It("caps only the zero (unlimited) limit", func() {
			ci := &ClientInfo{MaxAudioBitrate: 128, MaxTranscodingAudioBitrate: 0}
			Expect(ci.CapBitrate(192)).To(BeTrue())
			Expect(ci.MaxAudioBitrate).To(Equal(128))
			Expect(ci.MaxTranscodingAudioBitrate).To(Equal(192))
		})
	})

	Describe("ForceFormat", func() {
		It("restricts to the forced format and clears direct play when supported", func() {
			ci := &ClientInfo{
				DirectPlayProfiles: []DirectPlayProfile{{Containers: []string{"flac"}, AudioCodecs: []string{"flac"}}},
				TranscodingProfiles: []Profile{
					{Container: "flac", AudioCodec: "flac", Protocol: ProtocolHTTP},
					{Container: "ogg", AudioCodec: "opus", Protocol: ProtocolHTTP},
					{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP},
				},
			}
			ok := ci.ForceFormat("opus")
			Expect(ok).To(BeTrue())
			Expect(ci.TranscodingProfiles).To(HaveLen(1))
			Expect(ci.TranscodingProfiles[0].AudioCodec).To(Equal("opus"))
			Expect(ci.DirectPlayProfiles).To(BeEmpty())
		})

		It("matches a container-only forced format (mp3)", func() {
			ci := &ClientInfo{
				TranscodingProfiles: []Profile{
					{Container: "ogg", AudioCodec: "opus", Protocol: ProtocolHTTP},
					{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP},
				},
			}
			ok := ci.ForceFormat("mp3")
			Expect(ok).To(BeTrue())
			Expect(ci.TranscodingProfiles).To(HaveLen(1))
			Expect(ci.TranscodingProfiles[0].Container).To(Equal("mp3"))
		})

		It("matches the forced format against codec aliases (oga/opus)", func() {
			ci := &ClientInfo{
				TranscodingProfiles: []Profile{
					{Container: "ogg", AudioCodec: "opus", Protocol: ProtocolHTTP},
					{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP},
				},
			}
			// Legacy DBs may store the Opus transcoding as target_format "oga".
			ok := ci.ForceFormat("oga")
			Expect(ok).To(BeTrue())
			Expect(ci.TranscodingProfiles).To(HaveLen(1))
			Expect(ci.TranscodingProfiles[0].AudioCodec).To(Equal("opus"))
		})

		It("is a no-op when the forced format is not supported by the client", func() {
			original := []Profile{{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP}}
			ci := &ClientInfo{
				DirectPlayProfiles:  []DirectPlayProfile{{Containers: []string{"flac"}}},
				TranscodingProfiles: original,
			}
			ok := ci.ForceFormat("opus")
			Expect(ok).To(BeFalse())
			Expect(ci.TranscodingProfiles).To(Equal(original))
			Expect(ci.DirectPlayProfiles).To(HaveLen(1))
		})

		It("is a no-op for an empty target format", func() {
			ci := &ClientInfo{TranscodingProfiles: []Profile{{Container: "mp3", AudioCodec: "mp3"}}}
			Expect(ci.ForceFormat("")).To(BeFalse())
		})

		It("keeps all matching profiles when multiple resolve to the forced format", func() {
			first := Profile{Container: "ogg", AudioCodec: "opus", Protocol: ProtocolHTTP}
			second := Profile{Container: "ogg", AudioCodec: "opus", Protocol: ProtocolHTTP, MaxAudioChannels: 2}
			other := Profile{Container: "mp3", AudioCodec: "mp3", Protocol: ProtocolHTTP}
			ci := &ClientInfo{TranscodingProfiles: []Profile{first, other, second}}

			ok := ci.ForceFormat("opus")

			Expect(ok).To(BeTrue())
			Expect(ci.TranscodingProfiles).To(ConsistOf(first, second))
		})
	})
})
