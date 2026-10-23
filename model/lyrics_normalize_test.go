package model

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NormalizeLyrics", func() {
	p := func(v int64) *int64 { return &v }

	It("is immutable and idempotent", func() {
		input := Lyrics{
			Offset: p(25),
			Line: []Line{
				{
					Start: p(1000), Value: "Some lyrics",
					Cue: []Cue{
						{Start: p(1000), End: p(1200), Value: "Some ", ByteStart: 0, ByteEnd: 4},
						{Start: p(1500), Value: "lyrics", ByteStart: 5, ByteEnd: 10},
					},
				},
				{Start: p(3000), Value: "Next line"},
			},
		}
		before := cloneLyrics(input)

		first := NormalizeLyrics(input)
		second := NormalizeLyrics(first)

		Expect(first).To(Equal(second))
		Expect(input).To(Equal(before))
		Expect(first.Line[0].Cue[0].End).To(Equal(p(1200)))
		Expect(first.Line[0].Cue[1].End).To(Equal(p(3000)))
		Expect(first.Line[0].End).To(BeNil(), "next-line fallback must not become an exact line end")
	})

	It("preserves fully start-only cue groups", func() {
		lyrics := Lyrics{Line: []Line{{
			Value: "hello world",
			Cue: []Cue{
				{Start: p(1000), Value: "hello", ByteStart: 0, ByteEnd: 4},
				{Start: p(1500), Value: "world", ByteStart: 6, ByteEnd: 10},
			},
		}}}

		out := NormalizeLyrics(lyrics)

		Expect(out.Line[0].Start).To(Equal(p(1000)))
		Expect(out.Line[0].End).To(BeNil())
		Expect(out.Line[0].Cue[0].End).To(BeNil())
		Expect(out.Line[0].Cue[1].End).To(BeNil())
	})

	It("scans past untimed lines for a partial cue group's fallback", func() {
		lyrics := Lyrics{Line: []Line{
			{
				Start: p(1000), Value: "a b",
				Cue: []Cue{
					{Start: p(1000), End: p(1200), Value: "a", ByteStart: 0, ByteEnd: 0},
					{Start: p(1500), Value: "b", ByteStart: 2, ByteEnd: 2},
				},
			},
			{Value: "untimed"},
			{Start: p(4000), Value: "timed"},
		}}

		out := NormalizeLyrics(lyrics)

		Expect(out.Line[0].Cue[1].End).To(Equal(p(4000)))
		Expect(out.Line[0].End).To(BeNil())
	})

	It("clears a partial group when it cannot complete consistently", func() {
		lyrics := Lyrics{Line: []Line{{
			Value: "a b",
			Cue: []Cue{
				{Start: p(1000), End: p(1200), Value: "a", ByteStart: 0, ByteEnd: 0},
				{Start: p(1500), Value: "b", ByteStart: 2, ByteEnd: 2},
			},
		}}}

		out := NormalizeLyrics(lyrics)

		Expect(out.Line[0].Cue[0].End).To(BeNil())
		Expect(out.Line[0].Cue[1].End).To(BeNil())
	})

	It("clamps overlaps only inside the same agent", func() {
		lyrics := Lyrics{
			Agents: []Agent{{ID: "lead"}, {ID: "backing"}},
			Line: []Line{{
				Value: "lead back next",
				Cue: []Cue{
					{Start: p(1000), End: p(2500), Value: "lead", ByteStart: 0, ByteEnd: 3, AgentID: "lead"},
					{Start: p(1200), End: p(2200), Value: "back", ByteStart: 5, ByteEnd: 8, AgentID: "backing"},
					{Start: p(2000), End: p(3000), Value: "next", ByteStart: 10, ByteEnd: 13, AgentID: "lead"},
				},
			}},
		}

		out := NormalizeLyrics(lyrics)

		Expect(out.Line[0].Cue[0].End).To(Equal(p(2000)))
		Expect(out.Line[0].Cue[1].End).To(Equal(p(2200)), "cross-agent overlap must remain")
		Expect(out.Line[0].Cue[2].End).To(Equal(p(3000)))
	})

	It("allows different agents to reference the same byte range", func() {
		lyrics := Lyrics{
			Agents: []Agent{{ID: "lead"}, {ID: "backing"}},
			Line: []Line{{
				Value: "hello",
				Cue: []Cue{
					{Start: p(1000), End: p(2000), Value: "hello", ByteStart: 0, ByteEnd: 4, AgentID: "lead"},
					{Start: p(1200), End: p(2200), Value: "hello", ByteStart: 0, ByteEnd: 4, AgentID: "backing"},
				},
			}},
		}

		out := NormalizeLyrics(lyrics)

		Expect(out.Line[0].Cue).To(HaveLen(2))
		Expect(out.Line[0].Cue[0].AgentID).To(Equal("lead"))
		Expect(out.Line[0].Cue[1].AgentID).To(Equal("backing"))
	})

	It("ignores missing cue starts while widening partially assembled lines", func() {
		line := Line{
			Start: p(1000),
			Value: "partial",
			Cue:   []Cue{{End: p(2000), Value: "partial", ByteStart: 0, ByteEnd: 6}},
		}

		var out Line
		Expect(func() { out = normalizeLineTiming(line) }).ToNot(Panic())
		Expect(out.Start).To(Equal(p(1000)))
		Expect(out.End).To(BeNil())
	})

	It("preserves zero-duration markers and repairs reverse ends", func() {
		lyrics := Lyrics{Line: []Line{
			{Start: p(1000), End: p(500), Value: "marker", Cue: []Cue{{Start: p(1000), End: p(1000), Value: "marker", ByteStart: 0, ByteEnd: 5}}},
			{Start: p(1500), End: p(1400), Value: "reverse"},
		}}

		out := NormalizeLyrics(lyrics)

		Expect(out.Line[0].Cue[0].End).To(Equal(p(1000)))
		Expect(out.Line[0].End).To(Equal(p(1000)))
		Expect(out.Line[1].End).To(BeNil())
	})

	It("uses an explicit terminal cue end as an exact line end", func() {
		lyrics := Lyrics{Line: []Line{{
			Start: p(2000), Value: "hello",
			Cue: []Cue{{Start: p(1000), End: p(2500), Value: "hello", ByteStart: 0, ByteEnd: 4}},
		}}}

		out := NormalizeLyrics(lyrics)

		Expect(out.Line[0].Start).To(Equal(p(1000)))
		Expect(out.Line[0].End).To(Equal(p(2500)))
	})

	DescribeTable("drops irreparable cue geometry but retains line data",
		func(value string, cues []Cue, agents []Agent) {
			lineStart, lineEnd := int64(1000), int64(2000)
			out := NormalizeLyrics(Lyrics{
				Agents: agents,
				Line:   []Line{{Start: &lineStart, End: &lineEnd, Value: value, Cue: cues}},
			})

			Expect(out.Line).To(HaveLen(1))
			Expect(out.Line[0].Value).To(Equal(value))
			Expect(out.Line[0].Start).To(Equal(&lineStart))
			Expect(out.Line[0].End).To(Equal(&lineEnd))
			Expect(out.Line[0].Cue).To(BeNil())
			Expect(out.Agents).To(BeNil())
		},
		Entry("invalid UTF-8", string([]byte{0xff}), []Cue{{Start: p(1000), Value: string([]byte{0xff}), ByteStart: 0, ByteEnd: 0}}, nil),
		Entry("range outside the line", "hi", []Cue{{Start: p(1000), Value: "hi", ByteStart: 0, ByteEnd: 2}}, nil),
		Entry("range splitting a CJK rune", "한", []Cue{{Start: p(1000), Value: string([]byte("한")[:2]), ByteStart: 0, ByteEnd: 1}}, nil),
		Entry("cue value mismatch", "hello", []Cue{{Start: p(1000), Value: "world", ByteStart: 0, ByteEnd: 4}}, nil),
		Entry("overlapping byte ranges", "hello", []Cue{
			{Start: p(1000), Value: "hel", ByteStart: 0, ByteEnd: 2},
			{Start: p(1100), Value: "llo", ByteStart: 2, ByteEnd: 4},
		}, nil),
		Entry("out-of-order same-agent cues", "a b", []Cue{
			{Start: p(1500), Value: "a", ByteStart: 0, ByteEnd: 0},
			{Start: p(1000), Value: "b", ByteStart: 2, ByteEnd: 2},
		}, nil),
		Entry("unknown agent attribution", "hi", []Cue{{Start: p(1000), Value: "hi", ByteStart: 0, ByteEnd: 1, AgentID: "missing"}}, []Agent{{ID: "known"}}),
	)

	It("validates CJK, emoji, and combining-mark byte boundaries", func() {
		value := "한🙂é"
		lyrics := Lyrics{Line: []Line{{
			Value: value,
			Cue: []Cue{
				{Start: p(1000), Value: "한", ByteStart: 0, ByteEnd: 2},
				{Start: p(1100), Value: "🙂", ByteStart: 3, ByteEnd: 6},
				{Start: p(1200), Value: "é", ByteStart: 7, ByteEnd: 9},
			},
		}}}

		out := NormalizeLyrics(lyrics)

		Expect(out.Line[0].Cue).To(HaveLen(3))
		Expect([]string{out.Line[0].Cue[0].Value, out.Line[0].Cue[1].Value, out.Line[0].Cue[2].Value}).To(Equal([]string{"한", "🙂", "é"}))
	})

	It("prunes unused and duplicate agents", func() {
		lyrics := Lyrics{
			Agents: []Agent{{ID: "lead"}, {ID: "unused"}, {ID: "lead"}},
			Line:   []Line{{Value: "hi", Cue: []Cue{{Start: p(1000), Value: "hi", ByteStart: 0, ByteEnd: 1, AgentID: "lead"}}}},
		}

		out := NormalizeLyrics(lyrics)

		Expect(out.Agents).To(Equal([]Agent{{ID: "lead"}}))
	})
})

var _ = Describe("NormalizeCueEnds", func() {
	p := func(v int64) *int64 { return &v }

	It("retains its response compatibility contract without mutating input", func() {
		cues := []Cue{{Start: p(1000)}, {Start: p(1500)}}
		out := NormalizeCueEnds(cues, p(3000))

		Expect(out[0].End).To(Equal(p(1500)))
		Expect(out[1].End).To(Equal(p(3000)))
		Expect(cues[0].End).To(BeNil())
		Expect(cues[1].End).To(BeNil())
	})
})
