package model

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NormalizeCueLines", func() {
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

		normalized := NormalizeCueLines(lines)

		Expect(normalized[0].Cue[0].End).To(Equal(&start1))
		Expect(normalized[0].Cue[1].End).To(Equal(&nextLineStart))
		Expect(lines[0].Cue[0].End).To(BeNil())
		Expect(lines[0].Cue[1].End).To(BeNil())
	})
})

var _ = Describe("Lyrics.EffectiveKind", func() {
	It("defaults a blank kind to main", func() {
		Expect(Lyrics{}.EffectiveKind()).To(Equal(LyricKindMain))
		Expect(Lyrics{Kind: "  "}.EffectiveKind()).To(Equal(LyricKindMain))
	})

	It("returns the kind as-is when set", func() {
		Expect(Lyrics{Kind: LyricKindTranslation}.EffectiveKind()).To(Equal(LyricKindTranslation))
	})
})

var _ = Describe("Lyrics.IsMainKind", func() {
	It("is true for a blank (untyped) kind", func() {
		Expect(Lyrics{}.IsMainKind()).To(BeTrue())
	})

	It("is true for the main kind", func() {
		Expect(Lyrics{Kind: LyricKindMain}.IsMainKind()).To(BeTrue())
	})

	It("is false for translation and pronunciation kinds", func() {
		Expect(Lyrics{Kind: LyricKindTranslation}.IsMainKind()).To(BeFalse())
		Expect(Lyrics{Kind: LyricKindPronunciation}.IsMainKind()).To(BeFalse())
	})
})

var _ = Describe("LyricList.Main", func() {
	It("returns false when the list is empty", func() {
		_, ok := LyricList{}.Main()
		Expect(ok).To(BeFalse())
	})

	It("returns the main-kind entry when present", func() {
		list := LyricList{
			{Kind: LyricKindTranslation, Lang: "en"},
			{Kind: LyricKindMain, Lang: "xxx"},
		}
		main, ok := list.Main()
		Expect(ok).To(BeTrue())
		Expect(main.Kind).To(Equal(LyricKindMain))
	})

	It("falls back to the first entry when no main kind exists", func() {
		list := LyricList{
			{Kind: LyricKindTranslation, Lang: "en"},
			{Kind: LyricKindPronunciation, Lang: "ja"},
		}
		main, ok := list.Main()
		Expect(ok).To(BeTrue())
		Expect(main.Lang).To(Equal("en"))
	})

	It("treats a blank kind as main", func() {
		list := LyricList{
			{Kind: LyricKindTranslation, Lang: "en"},
			{Lang: "xxx"},
		}
		main, ok := list.Main()
		Expect(ok).To(BeTrue())
		Expect(main.Lang).To(Equal("xxx"))
	})
})
