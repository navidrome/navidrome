package stream

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Codec", func() {
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

	Describe("codecMaxChannels", func() {
		It("returns 2 for mp3", func() {
			Expect(codecMaxChannels("mp3")).To(Equal(2))
		})

		It("returns 8 for opus", func() {
			Expect(codecMaxChannels("opus")).To(Equal(8))
		})

		It("is case-insensitive", func() {
			Expect(codecMaxChannels("MP3")).To(Equal(2))
			Expect(codecMaxChannels("Opus")).To(Equal(8))
		})

		It("returns 0 for codecs with no hard limit", func() {
			Expect(codecMaxChannels("aac")).To(Equal(0))
			Expect(codecMaxChannels("flac")).To(Equal(0))
			Expect(codecMaxChannels("vorbis")).To(Equal(0))
			Expect(codecMaxChannels("")).To(Equal(0))
		})
	})
})
