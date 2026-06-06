package model

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TagConf", func() {
	Describe("SplitTagValue", func() {
		var conf TagConf

		BeforeEach(func() {
			conf = TagConf{Split: []string{";", "/", ","}}
			conf.SplitRx = compileSplitRegex("test", conf.Split)
		})

		It("splits a single value on configured separators", func() {
			Expect(conf.SplitTagValue([]string{"Rock/Pop;Punk"})).To(Equal([]string{"Rock", "Pop", "Punk"}))
		})

		It("trims whitespace around split values", func() {
			Expect(conf.SplitTagValue([]string{"Love, Emotional, Ballad"})).To(Equal([]string{"Love", "Emotional", "Ballad"}))
		})

		// Regression test for https://github.com/navidrome/navidrome/issues/5065
		//
		// When multiple ID3v2 frames map to the same logical tag (e.g. TMOO + TXXX:MOOD),
		// TagLib's PropertyMap merges them into a slice with several entries. Previously
		// SplitTagValue had a `len(values) != 1` guard that skipped splitting in this case.
		It("splits each value individually when given multiple inputs", func() {
			input := []string{"Love, Emotional, Ballad", "Love; Emotional; Ballad"}
			Expect(conf.SplitTagValue(input)).To(Equal([]string{
				"Love", "Emotional", "Ballad",
				"Love", "Emotional", "Ballad",
			}))
		})

		It("matches separators case-insensitively when the split pattern allows", func() {
			c := TagConf{Split: []string{" AND "}}
			c.SplitRx = compileSplitRegex("test", c.Split)
			Expect(c.SplitTagValue([]string{"foo and bar AND baz"})).To(Equal([]string{"foo", "bar", "baz"}))
		})

		It("returns values unchanged when no separators are configured", func() {
			c := TagConf{}
			Expect(c.SplitTagValue([]string{"Foo, Bar"})).To(Equal([]string{"Foo, Bar"}))
			Expect(c.SplitTagValue([]string{"a", "b"})).To(Equal([]string{"a", "b"}))
		})

		It("returns an empty slice for empty input", func() {
			Expect(conf.SplitTagValue([]string{})).To(BeEmpty())
		})

		It("handles a value with no separator as a single-element result", func() {
			Expect(conf.SplitTagValue([]string{"JustOneMood"})).To(Equal([]string{"JustOneMood"}))
		})

		It("produces empty strings when separators are adjacent (dedup happens downstream)", func() {
			// SplitTagValue itself does not filter empties; that is the job of
			// filterDuplicatedOrEmptyValues in the metadata pipeline.
			Expect(conf.SplitTagValue([]string{"Rock//Pop"})).To(Equal([]string{"Rock", "", "Pop"}))
		})
	})
})
