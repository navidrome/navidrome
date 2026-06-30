package types_test

import (
	"github.com/navidrome/navidrome/plugins/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SongRef", func() {
	Describe("DurationInMs", func() {
		It("returns DurationMs when set", func() {
			s := types.SongRef{DurationMs: 247333, Duration: 247.5}
			Expect(s.DurationInMs()).To(Equal(uint32(247333)))
		})

		It("falls back to Duration (seconds) when DurationMs is zero", func() {
			s := types.SongRef{Duration: 247.5}
			Expect(s.DurationInMs()).To(Equal(uint32(247500)))
		})

		It("returns 0 when neither is set", func() {
			Expect(types.SongRef{}.DurationInMs()).To(BeZero())
		})
	})

	Describe("SetDurationMs", func() {
		It("populates both DurationMs and the deprecated Duration", func() {
			var s types.SongRef
			s.SetDurationMs(247333)
			Expect(s.DurationMs).To(Equal(uint32(247333)))
			Expect(s.Duration).To(BeNumerically("~", 247.333, 0.001))
		})

		It("keeps DurationInMs consistent with what was set", func() {
			var s types.SongRef
			s.SetDurationMs(60000)
			Expect(s.DurationInMs()).To(Equal(uint32(60000)))
		})
	})
})
