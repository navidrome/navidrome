package model_test

import (
	. "github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ToLyrics", func() {
	It("should parse tags with spaces", func() {
		num := int64(1551)
		lyrics, err := ToLyrics("xxx", "[lang:  eng  ]\n[offset: 1551 ]\n[ti: A title ]\n[ar: An artist ]\n[00:00.00]Hi there")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Lang).To(Equal("eng"))
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.DisplayArtist).To(Equal("An artist"))
		Expect(lyrics.DisplayTitle).To(Equal("A title"))
		Expect(lyrics.Offset).To(Equal(&num))
	})

	It("Should ignore bad offset", func() {
		lyrics, err := ToLyrics("xxx", "[offset: NotANumber ]\n[00:00.00]Hi there")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Offset).To(BeNil())
	})

	It("should accept lines with no text and weird times", func() {
		a, b, c, d := int64(0), int64(10040), int64(40000), int64(1000*60*60)
		lyrics, err := ToLyrics("xxx", "[00:00.00]Hi there\n\n\n[00:10.040]\n[00:40]Test\n[01:00:00]late")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: &a, Value: "Hi there"},
			{Start: &b, Value: ""},
			{Start: &c, Value: "Test"},
			{Start: &d, Value: "late"},
		}))
	})

	It("Should support multiple timestamps per line", func() {
		a, b, c, d := int64(0), int64(10000), int64(13*60*1000), int64(1000*60*60*51)
		lyrics, err := ToLyrics("xxx", "[00:00.00]  [00:10.00]Repeated\n[13:00][51:00:00.00]")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: &a, Value: "Repeated"},
			{Start: &b, Value: "Repeated"},
			{Start: &c, Value: ""},
			{Start: &d, Value: ""},
		}))
	})

	It("Should support parsing multiline string", func() {
		a, b := int64(0), int64(10*60*1000+1)
		lyrics, err := ToLyrics("xxx", "[00:00.00]This is\na multiline  \n\n  [:0] string\n[10:00.001]This is\nalso one")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: &a, Value: "This is\na multiline\n\n[:0] string"},
			{Start: &b, Value: "This is\nalso one"},
		}))
	})

	It("Does not match timestamp in middle of line", func() {
		lyrics, err := ToLyrics("xxx", "This could [00:00:00] be a synced file")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeFalse())
		Expect(lyrics.Line).To(Equal([]Line{
			{Value: "This could [00:00:00] be a synced file"},
		}))
	})

	It("Allows timestamp in middle of line if also at beginning", func() {
		a, b := int64(0), int64(1000)
		lyrics, err := ToLyrics("xxx", "  [00:00] This is [00:00:00] be a synced file\n		[00:01]Line 2")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: &a, Value: "This is [00:00:00] be a synced file"},
			{Start: &b, Value: "Line 2"},
		}))
	})

	It("Ignores lines in synchronized lyric prior to first timestamp", func() {
		a := int64(0)
		lyrics, err := ToLyrics("xxx", "This is some prelude\nThat doesn't\nmatter\n[00:00]Text")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: &a, Value: "Text"},
		}))
	})

	It("Handles all possible ms cases", func() {
		a, b, c := int64(1), int64(10), int64(100)
		lyrics, err := ToLyrics("xxx", "[00:00.001]a\n[00:00.01]b\n[00:00.1]c")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: &a, Value: "a"},
			{Start: &b, Value: "b"},
			{Start: &c, Value: "c"},
		}))
	})

	It("Properly sorts repeated lyrics out of order", func() {
		a, b, c, d, e := int64(0), int64(10000), int64(40000), int64(13*60*1000), int64(1000*60*60*51)
		lyrics, err := ToLyrics("xxx", "[00:00.00]  [13:00]Repeated\n[00:10.00][51:00:00.00]Test\n[00:40.00]Not repeated")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: &a, Value: "Repeated"},
			{Start: &b, Value: "Test"},
			{Start: &c, Value: "Not repeated"},
			{Start: &d, Value: "Repeated"},
			{Start: &e, Value: "Test"},
		}))
	})

	It("should parse Enhanced LRC with word-level timing", func() {
		lyrics, err := ToLyrics("xxx", "[00:01.00]<00:01.00>Some <00:01.50>lyrics <00:02.00>here\n[00:03.00]<00:03.00>More <00:03.50>words")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(HaveLen(2))

		t1000, t1500, t2000, t3000, t3500 := int64(1000), int64(1500), int64(2000), int64(3000), int64(3500)

		line0 := lyrics.Line[0]
		Expect(line0.Start).To(Equal(&t1000))
		Expect(line0.End).To(Equal(&t3000))
		Expect(line0.Value).To(Equal("Some lyrics here"))
		Expect(line0.Cue).To(Equal([]Cue{
			{Start: &t1000, End: &t1500, Value: "Some "},
			{Start: &t1500, End: &t2000, Value: "lyrics "},
			{Start: &t2000, End: &t3000, Value: "here"},
		}))

		line1 := lyrics.Line[1]
		Expect(line1.Start).To(Equal(&t3000))
		Expect(line1.End).To(Equal(&t3500))
		Expect(line1.Value).To(Equal("More words"))
		Expect(line1.Cue).To(Equal([]Cue{
			{Start: &t3000, Value: "More "},
			{Start: &t3500, Value: "words"},
		}))

		Expect(line1.Cue[1].End).To(BeNil())
	})

	It("should ignore Enhanced LRC markers and return plain lines when no markers present", func() {
		a, b := int64(1000), int64(3000)
		lyrics, err := ToLyrics("xxx", "[00:01.00]Plain line\n[00:03.00]Another plain line")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: &a, Value: "Plain line"},
			{Start: &b, Value: "Another plain line"},
		}))
	})

	It("should handle mixed Enhanced and plain LRC lines", func() {
		lyrics, err := ToLyrics("xxx", "[00:01.00]<00:01.00>Some <00:01.50>lyrics\n[00:03.00]Plain line\n[00:05.00]<00:05.00>More <00:05.50>words")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Line).To(HaveLen(3))

		t1000, t1500, t5000, t5500 := int64(1000), int64(1500), int64(5000), int64(5500)
		t3000 := int64(3000)

		Expect(lyrics.Line[0].Cue).To(Equal([]Cue{
			{Start: &t1000, End: &t1500, Value: "Some "},
			{Start: &t1500, End: &t3000, Value: "lyrics"},
		}))
		Expect(lyrics.Line[0].Value).To(Equal("Some lyrics"))
		Expect(lyrics.Line[0].End).To(Equal(&t3000))

		Expect(lyrics.Line[1].Cue).To(BeNil())
		Expect(lyrics.Line[1].Value).To(Equal("Plain line"))

		Expect(lyrics.Line[2].Cue).To(Equal([]Cue{
			{Start: &t5000, Value: "More "},
			{Start: &t5500, Value: "words"},
		}))
		Expect(lyrics.Line[2].Value).To(Equal("More words"))
	})
})
