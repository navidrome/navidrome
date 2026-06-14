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
})
