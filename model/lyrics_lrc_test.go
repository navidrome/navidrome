package model

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("parseLRC", func() {
	It("should parse tags with spaces", func() {
		lyrics, err := parseLRC("xxx", "[lang:  eng  ]\n[offset: 1551 ]\n[ti: A title ]\n[ar: An artist ]\n[00:00.00]Hi there")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Lang).To(Equal("eng"))
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.DisplayArtist).To(Equal("An artist"))
		Expect(lyrics.DisplayTitle).To(Equal("A title"))
		Expect(lyrics.Offset).To(Equal(new(int64(1551))))
	})

	It("Should ignore bad offset", func() {
		lyrics, err := parseLRC("xxx", "[offset: NotANumber ]\n[00:00.00]Hi there")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Offset).To(BeNil())
	})

	It("should accept lines with no text and weird times", func() {
		lyrics, err := parseLRC("xxx", "[00:00.00]Hi there\n\n\n[00:10.040]\n[00:40]Test\n[01:00:00]late")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: new(int64(0)), Value: "Hi there"},
			{Start: new(int64(10040)), Value: ""},
			{Start: new(int64(40000)), Value: "Test"},
			{Start: new(int64(1000 * 60 * 60)), Value: "late"},
		}))
	})

	It("Should support multiple timestamps per line", func() {
		lyrics, err := parseLRC("xxx", "[00:00.00]  [00:10.00]Repeated\n[13:00][51:00:00.00]")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: new(int64(0)), Value: "Repeated"},
			{Start: new(int64(10000)), Value: "Repeated"},
			{Start: new(int64(13 * 60 * 1000)), Value: ""},
			{Start: new(int64(1000 * 60 * 60 * 51)), Value: ""},
		}))
	})

	It("Should support parsing multiline string", func() {
		lyrics, err := parseLRC("xxx", "[00:00.00]This is\na multiline  \n\n  [:0] string\n[10:00.001]This is\nalso one")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: new(int64(0)), Value: "This is\na multiline\n\n[:0] string"},
			{Start: new(int64(10*60*1000 + 1)), Value: "This is\nalso one"},
		}))
	})

	It("Does not match timestamp in middle of line", func() {
		lyrics, err := parseLRC("xxx", "This could [00:00:00] be a synced file")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeFalse())
		Expect(lyrics.Line).To(Equal([]Line{
			{Value: "This could [00:00:00] be a synced file"},
		}))
	})

	It("Allows timestamp in middle of line if also at beginning", func() {
		lyrics, err := parseLRC("xxx", "  [00:00] This is [00:00:00] be a synced file\n		[00:01]Line 2")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: new(int64(0)), Value: "This is [00:00:00] be a synced file"},
			{Start: new(int64(1000)), Value: "Line 2"},
		}))
	})

	It("Ignores lines in synchronized lyric prior to first timestamp", func() {
		lyrics, err := parseLRC("xxx", "This is some prelude\nThat doesn't\nmatter\n[00:00]Text")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: new(int64(0)), Value: "Text"},
		}))
	})

	It("Handles all possible ms cases", func() {
		lyrics, err := parseLRC("xxx", "[00:00.001]a\n[00:00.01]b\n[00:00.1]c")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: new(int64(1)), Value: "a"},
			{Start: new(int64(10)), Value: "b"},
			{Start: new(int64(100)), Value: "c"},
		}))
	})

	It("Properly sorts repeated lyrics out of order", func() {
		lyrics, err := parseLRC("xxx", "[00:00.00]  [13:00]Repeated\n[00:10.00][51:00:00.00]Test\n[00:40.00]Not repeated")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: new(int64(0)), Value: "Repeated"},
			{Start: new(int64(10000)), Value: "Test"},
			{Start: new(int64(40000)), Value: "Not repeated"},
			{Start: new(int64(13 * 60 * 1000)), Value: "Repeated"},
			{Start: new(int64(1000 * 60 * 60 * 51)), Value: "Test"},
		}))
	})

	It("should parse Enhanced LRC with word-level timing", func() {
		lyrics, err := parseLRC("xxx", "[00:01.00]<00:01.00>Some <00:01.50>lyrics <00:02.00>here\n[00:03.00]<00:03.00>More <00:03.50>words")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(HaveLen(2))

		t1000, t1500, t2000, t3000, t3500 := int64(1000), int64(1500), int64(2000), int64(3000), int64(3500)

		line0 := lyrics.Line[0]
		Expect(line0.Start).To(Equal(&t1000))
		Expect(line0.End).To(Equal(&t3000))
		Expect(line0.Value).To(Equal("Some lyrics here"))
		Expect(line0.Cue).To(Equal([]Cue{
			{Start: &t1000, End: &t1500, Value: "Some ", ByteStart: 0, ByteEnd: 4},
			{Start: &t1500, End: &t2000, Value: "lyrics ", ByteStart: 5, ByteEnd: 11},
			{Start: &t2000, End: &t3000, Value: "here", ByteStart: 12, ByteEnd: 15},
		}))

		line1 := lyrics.Line[1]
		Expect(line1.Start).To(Equal(&t3000))
		Expect(line1.End).To(Equal(&t3500))
		Expect(line1.Value).To(Equal("More words"))
		Expect(line1.Cue).To(Equal([]Cue{
			{Start: &t3000, Value: "More ", ByteStart: 0, ByteEnd: 4},
			{Start: &t3500, Value: "words", ByteStart: 5, ByteEnd: 9},
		}))

		Expect(line1.Cue[1].End).To(BeNil())
	})

	It("should not parse malformed Enhanced LRC timing markers", func() {
		lyrics, err := parseLRC("xxx", "[00:01.00]<00:01a50>Not a marker")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: new(int64(1000)), Value: "<00:01a50>Not a marker"},
		}))
	})

	It("should handle mixed Enhanced and plain LRC lines", func() {
		lyrics, err := parseLRC("xxx", "[00:01.00]<00:01.00>Some <00:01.50>lyrics\n[00:03.00]Plain line\n[00:05.00]<00:05.00>More <00:05.50>words")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Line).To(HaveLen(3))

		t1000, t1500, t5000, t5500 := int64(1000), int64(1500), int64(5000), int64(5500)
		t3000 := int64(3000)

		Expect(lyrics.Line[0].Cue).To(Equal([]Cue{
			{Start: &t1000, End: &t1500, Value: "Some ", ByteStart: 0, ByteEnd: 4},
			{Start: &t1500, End: &t3000, Value: "lyrics", ByteStart: 5, ByteEnd: 10},
		}))
		Expect(lyrics.Line[0].Value).To(Equal("Some lyrics"))
		Expect(lyrics.Line[0].End).To(Equal(&t3000))

		Expect(lyrics.Line[1].Cue).To(BeNil())
		Expect(lyrics.Line[1].Value).To(Equal("Plain line"))

		Expect(lyrics.Line[2].Cue).To(Equal([]Cue{
			{Start: &t5000, Value: "More ", ByteStart: 0, ByteEnd: 4},
			{Start: &t5500, Value: "words", ByteStart: 5, ByteEnd: 9},
		}))
		Expect(lyrics.Line[2].Value).To(Equal("More words"))
	})

	It("should preserve byte offsets for Enhanced LRC cues", func() {
		lyrics, err := parseLRC("xxx", "[00:00.00]<00:00.00>Oh <00:00.90>love<00:01.30> me <00:01.60>tonight")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Line).To(HaveLen(1))

		t0, t900, t1300, t1600 := int64(0), int64(900), int64(1300), int64(1600)
		line := lyrics.Line[0]
		Expect(line.Value).To(Equal("Oh love me tonight"))
		Expect(line.Cue).To(Equal([]Cue{
			{Start: &t0, Value: "Oh ", ByteStart: 0, ByteEnd: 2},
			{Start: &t900, Value: "love", ByteStart: 3, ByteEnd: 6},
			{Start: &t1300, Value: " me ", ByteStart: 7, ByteEnd: 10},
			{Start: &t1600, Value: "tonight", ByteStart: 11, ByteEnd: 17},
		}))
	})

	It("should use a trailing Enhanced LRC marker as the end of the last word", func() {
		lyrics, err := parseLRC("xxx", "[00:01.00]<00:01.00>Some <00:01.50>lyrics<00:02.00>\n[00:30.00]Instrumental over")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Line).To(HaveLen(2))

		t1000, t1500, t2000 := int64(1000), int64(1500), int64(2000)
		line := lyrics.Line[0]
		Expect(line.Value).To(Equal("Some lyrics"))
		Expect(line.End).To(Equal(&t2000))
		Expect(line.Cue).To(Equal([]Cue{
			{Start: &t1000, End: &t1500, Value: "Some ", ByteStart: 0, ByteEnd: 4},
			{Start: &t1500, End: &t2000, Value: "lyrics", ByteStart: 5, ByteEnd: 10},
		}))
	})

	It("should shift a trailing Enhanced LRC marker for repeated line occurrences", func() {
		lyrics, err := parseLRC("xxx", "[00:10.00][00:30.00]<00:10.10>Hello <00:10.50>world<00:10.90>")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Line).To(HaveLen(2))

		t10100, t10500, t10900 := int64(10100), int64(10500), int64(10900)
		t30100, t30500, t30900 := int64(30100), int64(30500), int64(30900)

		Expect(lyrics.Line[0].End).To(Equal(&t10900))
		Expect(lyrics.Line[0].Cue).To(Equal([]Cue{
			{Start: &t10100, End: &t10500, Value: "Hello ", ByteStart: 0, ByteEnd: 5},
			{Start: &t10500, End: &t10900, Value: "world", ByteStart: 6, ByteEnd: 10},
		}))

		Expect(lyrics.Line[1].End).To(Equal(&t30900))
		Expect(lyrics.Line[1].Cue).To(Equal([]Cue{
			{Start: &t30100, End: &t30500, Value: "Hello ", ByteStart: 0, ByteEnd: 5},
			{Start: &t30500, End: &t30900, Value: "world", ByteStart: 6, ByteEnd: 10},
		}))
	})

	It("should shift inline ELRC word timestamps for each repeated line occurrence", func() {
		lyrics, err := parseLRC("xxx", "[00:10.00][00:30.00]<00:10.10>Hello <00:10.50>world")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Line).To(HaveLen(2))

		t10000 := int64(10000)
		t10100 := int64(10100)
		t10500 := int64(10500)
		t30000 := int64(30000)
		t30100 := int64(30100)
		t30500 := int64(30500)

		Expect(lyrics.Line[0].Start).To(Equal(&t10000))
		Expect(lyrics.Line[0].End).To(Equal(&t30000))
		Expect(lyrics.Line[0].Value).To(Equal("Hello world"))
		Expect(lyrics.Line[0].Cue).To(Equal([]Cue{
			{Start: &t10100, End: &t10500, Value: "Hello ", ByteStart: 0, ByteEnd: 5},
			{Start: &t10500, End: &t30000, Value: "world", ByteStart: 6, ByteEnd: 10},
		}))

		Expect(lyrics.Line[1].Start).To(Equal(&t30000))
		Expect(lyrics.Line[1].End).To(Equal(&t30500))
		Expect(lyrics.Line[1].Value).To(Equal("Hello world"))
		Expect(lyrics.Line[1].Cue).To(Equal([]Cue{
			{Start: &t30100, Value: "Hello ", ByteStart: 0, ByteEnd: 5},
			{Start: &t30500, Value: "world", ByteStart: 6, ByteEnd: 10},
		}))
	})
})
