package model_test

import (
	. "github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ToLyrics", func() {
	num := int64(1551)

	It("should parse tags with spaces", func() {
		lyrics, err := ToLyrics("xxx", "[offset: 1551 ]\n[ti: A title ]\n[ar: An artist ]\n[00:00.00]Hi there")
		Expect(err).ToNot(HaveOccurred())
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
		var a, b, c, d = int64(0), int64(10040), int64(40000), int64(1000 * 60 * 60)
		lyrics, err := ToLyrics("xxx", "[00:00.00]Hi there\n\n\n[00:10.040] \n[00:40]Test\n[01:00:00]late")
		Expect(err).ToNot(HaveOccurred())
		Expect(lyrics.Synced).To(BeTrue())
		Expect(lyrics.Line).To(Equal([]Line{
			{Start: &a, Value: "Hi there"},
			{Start: &b, Value: ""},
			{Start: &c, Value: "Test"},
			{Start: &d, Value: "late"},
		}))
	})
})
