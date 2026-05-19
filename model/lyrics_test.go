package model_test

import (
	. "github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ToLyrics", func() {
	It("should parse tags with spaces", func() {
		lyrics, err := ToLyrics("xxx", "[lang:  eng  ]\n[offset: 1551 ]\n[ti: A title ]\n[ar: An artist ]\n[00:00.00]Hi there")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Lang).To(Equal("eng"))
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.DisplayArtist).To(Equal("An artist"))
		Expect(lyrics.DisplayTitle).To(Equal("A title"))
		Expect(lyrics.Offset).To(Equal(new(int64(1551))))
	})

	It("Should ignore bad offset", func() {
		lyrics, err := ToLyrics("xxx", "[offset: NotANumber ]\n[00:00.00]Hi there")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Offset).To(BeNil())
	})

	It("should accept lines with no text and weird times", func() {
		lyrics, err := ToLyrics("xxx", "[00:00.00]Hi there\n\n\n[00:10.040]\n[00:40]Test\n[01:00:00]late")
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
		lyrics, err := ToLyrics("xxx", "[00:00.00]  [00:10.00]Repeated\n[13:00][51:00:00.00]")
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
		lyrics, err := ToLyrics("xxx", "[00:00.00]This is\na multiline  \n\n  [:0] string\n[10:00.001]This is\nalso one")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: new(int64(0)), Value: "This is\na multiline\n\n[:0] string"},
			{Start: new(int64(10*60*1000 + 1)), Value: "This is\nalso one"},
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
		lyrics, err := ToLyrics("xxx", "  [00:00] This is [00:00:00] be a synced file\n		[00:01]Line 2")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: new(int64(0)), Value: "This is [00:00:00] be a synced file"},
			{Start: new(int64(1000)), Value: "Line 2"},
		}))
	})

	It("Ignores lines in synchronized lyric prior to first timestamp", func() {
		lyrics, err := ToLyrics("xxx", "This is some prelude\nThat doesn't\nmatter\n[00:00]Text")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: new(int64(0)), Value: "Text"},
		}))
	})

	It("Handles all possible ms cases", func() {
		lyrics, err := ToLyrics("xxx", "[00:00.001]a\n[00:00.01]b\n[00:00.1]c")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: new(int64(1)), Value: "a"},
			{Start: new(int64(10)), Value: "b"},
			{Start: new(int64(100)), Value: "c"},
		}))
	})

	It("Properly sorts repeated lyrics out of order", func() {
		lyrics, err := ToLyrics("xxx", "[00:00.00]  [13:00]Repeated\n[00:10.00][51:00:00.00]Test\n[00:40.00]Not repeated")
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
})
