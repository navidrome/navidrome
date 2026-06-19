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

var _ = Describe("NormalizeCueEnds", func() {
	// p returns a fresh pointer so cases don't share *int64 state.
	p := func(v int64) *int64 { return &v }

	// endsOf extracts the resolved end times (nil-safe) for compact assertions.
	endsOf := func(cues []Cue) []*int64 {
		out := make([]*int64, len(cues))
		for i := range cues {
			out[i] = cues[i].End
		}
		return out
	}

	It("returns the input as-is when empty", func() {
		Expect(NormalizeCueEnds(nil, p(1000))).To(BeNil())
		Expect(NormalizeCueEnds([]Cue{}, p(1000))).To(BeEmpty())
	})

	It("fills a missing end from the next cue's start", func() {
		cues := []Cue{
			{Start: p(1000)},
			{Start: p(1500)},
		}

		out := NormalizeCueEnds(cues, p(3000))

		Expect(endsOf(out)).To(Equal([]*int64{p(1500), p(3000)}))
	})

	It("fills the last cue's missing end from fallbackEnd", func() {
		cues := []Cue{
			{Start: p(1000), End: p(1200)},
			{Start: p(1500)},
		}

		out := NormalizeCueEnds(cues, p(3000))

		Expect(endsOf(out)).To(Equal([]*int64{p(1200), p(3000)}))
	})

	It("clamps an end that overruns the next cue's start", func() {
		cues := []Cue{
			{Start: p(1000), End: p(9999)},
			{Start: p(1500), End: p(2000)},
		}

		out := NormalizeCueEnds(cues, p(3000))

		Expect(endsOf(out)).To(Equal([]*int64{p(1500), p(2000)}))
	})

	It("clamps an end that precedes the cue's own start", func() {
		cues := []Cue{
			{Start: p(1000), End: p(500)},
		}

		out := NormalizeCueEnds(cues, p(3000))

		Expect(endsOf(out)).To(Equal([]*int64{p(1000)}))
	})

	It("clears all ends when any cue still lacks one (all-or-none)", func() {
		// The last cue has no end and there is no fallback, so it stays nil and
		// every end in the group is cleared.
		cues := []Cue{
			{Start: p(1000), End: p(1200)},
			{Start: p(1500)},
		}

		out := NormalizeCueEnds(cues, nil)

		Expect(endsOf(out)).To(Equal([]*int64{nil, nil}))
	})

	It("does not mutate the input slice", func() {
		cues := []Cue{
			{Start: p(1000)},
			{Start: p(1500)},
		}

		_ = NormalizeCueEnds(cues, p(3000))

		Expect(cues[0].End).To(BeNil())
		Expect(cues[1].End).To(BeNil())
	})
})
