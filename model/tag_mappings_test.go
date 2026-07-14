package model

import (
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
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

		Context("with split exceptions", func() {
			BeforeEach(func() {
				conf = TagConf{Split: []string{" and ", ";", "/"}}
				conf.SplitRx = compileSplitRegex("test", conf.Split)
				conf.ExceptionsRx = compileExceptionsRegex([]string{
					"Iron and Wine",
					"Iron and Wine Duo",
					"Ella and Louis",
					"AC/DC",
					"Ólafur Arnalds and Nils Frahm",
				})
			})

			It("does not split a value that is exactly an exception", func() {
				Expect(conf.SplitTagValue([]string{"Iron and Wine"})).To(Equal([]string{"Iron and Wine"}))
			})

			It("protects an exception embedded in a multi-artist value", func() {
				Expect(conf.SplitTagValue([]string{"Iron and Wine and Bob"})).
					To(Equal([]string{"Iron and Wine", "Bob"}))
			})

			It("protects every occurrence, not just the first", func() {
				Expect(conf.SplitTagValue([]string{"Iron and Wine; Bob; Iron and Wine"})).
					To(Equal([]string{"Iron and Wine", "Bob", "Iron and Wine"}))
			})

			It("matches exceptions case-insensitively and keeps the tag's casing", func() {
				Expect(conf.SplitTagValue([]string{"IRON AND WINE and Bob"})).
					To(Equal([]string{"IRON AND WINE", "Bob"}))
			})

			It("prefers the longest exception when entries overlap", func() {
				Expect(conf.SplitTagValue([]string{"Iron and Wine Duo and Bob"})).
					To(Equal([]string{"Iron and Wine Duo", "Bob"}))
			})

			It("protects exceptions containing separator characters", func() {
				Expect(conf.SplitTagValue([]string{"AC/DC/Queen"})).
					To(Equal([]string{"AC/DC", "Queen"}))
			})

			It("does not protect an exception embedded in a longer word", func() {
				// "Ella and Louis" must not match inside "Ella and Louise"
				Expect(conf.SplitTagValue([]string{"Ella and Louise"})).
					To(Equal([]string{"Ella", "Louise"}))
			})

			It("handles names with non-ASCII edges", func() {
				Expect(conf.SplitTagValue([]string{"Ólafur Arnalds and Nils Frahm and Bob"})).
					To(Equal([]string{"Ólafur Arnalds and Nils Frahm", "Bob"}))
			})

			It("splits normally when no exception matches", func() {
				Expect(conf.SplitTagValue([]string{"Foo and Bar"})).To(Equal([]string{"Foo", "Bar"}))
			})
		})
	})

	Describe("compileExceptionsRegex", func() {
		It("returns nil for an empty list", func() {
			Expect(compileExceptionsRegex(nil)).To(BeNil())
			Expect(compileExceptionsRegex([]string{})).To(BeNil())
		})

		It("returns nil when all entries are blank", func() {
			Expect(compileExceptionsRegex([]string{"", "  "})).To(BeNil())
		})

		It("escapes regex metacharacters in names", func() {
			rx := compileExceptionsRegex([]string{"Sigur (Rós)"})
			Expect(rx.FindString("Sigur (Rós)")).To(Equal("Sigur (Rós)"))
			Expect(rx.MatchString("Sigur xRósx")).To(BeFalse())
		})

		It("matches case-insensitively", func() {
			rx := compileExceptionsRegex([]string{"Iron and Wine"})
			Expect(rx.MatchString("IRON AND WINE")).To(BeTrue())
		})
	})

	Describe("artistSplitExceptionsRx", func() {
		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
		})

		It("returns nil when no exceptions are configured", func() {
			conf.Server.Scanner.ArtistSplitExceptions = nil
			Expect(artistSplitExceptionsRx()).To(BeNil())
		})

		It("compiles the configured exceptions", func() {
			conf.Server.Scanner.ArtistSplitExceptions = []string{"Iron and Wine"}
			rx := artistSplitExceptionsRx()
			Expect(rx).ToNot(BeNil())
			Expect(rx.MatchString("iron and wine")).To(BeTrue())
		})

		It("caches the compiled regex until the configuration changes", func() {
			conf.Server.Scanner.ArtistSplitExceptions = []string{"Iron and Wine"}
			first := artistSplitExceptionsRx()
			Expect(artistSplitExceptionsRx()).To(BeIdenticalTo(first))

			conf.Server.Scanner.ArtistSplitExceptions = []string{"AC/DC"}
			second := artistSplitExceptionsRx()
			Expect(second).ToNot(BeIdenticalTo(first))
			Expect(second.MatchString("AC/DC")).To(BeTrue())
		})
	})

	Describe("WithParticipantExceptions", func() {
		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Scanner.ArtistSplitExceptions = []string{"Iron and Wine"}
		})

		It("attaches the exceptions regex to participant tags", func() {
			for _, tag := range []TagName{"artist", "albumartist", "artists", "artistsort", "composer", "lyricist", "composersort"} {
				Expect(TagConf{}.WithParticipantExceptions(tag).ExceptionsRx).ToNot(BeNil(), string(tag))
			}
		})

		It("does not attach the exceptions regex to non-participant tags", func() {
			for _, tag := range []TagName{"genre", "mood", "title", "releasetype"} {
				Expect(TagConf{}.WithParticipantExceptions(tag).ExceptionsRx).To(BeNil(), string(tag))
			}
		})
	})
})
