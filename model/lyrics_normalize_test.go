package model

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("normalizeCueLines", func() {
	It("should not mutate caller cue slices when filling missing cue end times", func() {
		start0, start1, nextLineStart := int64(1000), int64(1500), int64(3000)
		lines := []Line{
			{
				Start: &start0,
				Value: "Some lyrics",
				Cue: []Cue{
					{Start: &start0, Value: "Some ", ByteStart: 0, ByteEnd: 4},
					{Start: &start1, Value: "lyrics", ByteStart: 5, ByteEnd: 10},
				},
			},
			{
				Start: &nextLineStart,
				Value: "Next line",
			},
		}

		normalized := normalizeCueLines(lines)

		Expect(normalized[0].Cue[0].End).To(Equal(&start1))
		Expect(normalized[0].Cue[1].End).To(Equal(&nextLineStart))
		Expect(lines[0].Cue[0].End).To(BeNil())
		Expect(lines[0].Cue[1].End).To(BeNil())
	})
})
