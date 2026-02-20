package persistence

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("buildFTS5Query",
	func(input, expected string) {
		Expect(buildFTS5Query(input)).To(Equal(expected))
	},
	Entry("returns empty string for empty input", "", ""),
	Entry("returns empty string for whitespace-only input", "   ", ""),
	Entry("appends * to a single word for prefix matching", "beatles", "beatles*"),
	Entry("appends * to each word for prefix matching", "abbey road", "abbey* road*"),
	Entry("preserves quoted phrases without appending *", `"the beatles"`, `"the beatles"`),
	Entry("does not double-append * to existing prefix wildcard", "beat*", "beat*"),
	Entry("strips FTS5 operators and appends * to lowercased words", "AND OR NOT NEAR", "and* or* not* near*"),
	Entry("strips special FTS5 syntax characters and appends *", "test^col:val", "test* col* val*"),
	Entry("handles mixed phrases and words", `"the beatles" abbey`, `"the beatles" abbey*`),
	Entry("handles prefix with multiple words", "beat* abbey", "beat* abbey*"),
	Entry("collapses multiple spaces", "abbey   road", "abbey* road*"),
	Entry("strips leading * from tokens and appends trailing *", "*livia", "livia*"),
	Entry("strips leading * and preserves existing trailing *", "*livia oliv*", "livia* oliv*"),
	Entry("strips standalone *", "*", ""),
	Entry("strips apostrophe from input", "Guns N' Roses", "Guns* N* Roses*"),
	Entry("strips slash and splits into tokens", "AC/DC", "AC* DC*"),
	Entry("strips miscellaneous punctuation", "rock & roll, vol. 2", "rock* roll* vol* 2*"),
	Entry("preserves unicode characters with diacritics", "Björk début", "Björk* début*"),
	Entry("collapses dotted abbreviation into phrase+concat OR", "R.E.M.", `("R E M" OR REM*)`),
	Entry("collapses abbreviation without trailing dot", "R.E.M", `("R E M" OR REM*)`),
	Entry("collapses abbreviation mixed with words", "best of R.E.M.", `best* of* ("R E M" OR REM*)`),
	Entry("collapses two-letter abbreviation", "U.K.", `("U K" OR UK*)`),
	Entry("does not collapse single letter surrounded by words", "I am fine", "I* am* fine*"),
	Entry("does not collapse single standalone letter", "A test", "A* test*"),
	Entry("preserves quoted phrase with punctuation verbatim", `"ac/dc"`, `"ac/dc"`),
	Entry("preserves quoted abbreviation verbatim", `"R.E.M."`, `"R.E.M."`),
)

var _ = DescribeTable("normalizeForFTS",
	func(expected string, values ...string) {
		Expect(normalizeForFTS(values...)).To(Equal(expected))
	},
	Entry("strips dots and concatenates", "REM", "R.E.M."),
	Entry("strips slash", "ACDC", "AC/DC"),
	Entry("strips hyphen", "Aha", "A-ha"),
	Entry("skips unchanged words", "", "The Beatles"),
	Entry("handles mixed input", "REM", "R.E.M.", "Automatic for the People"),
	Entry("deduplicates", "REM", "R.E.M.", "R.E.M."),
	Entry("strips apostrophe from word", "N", "Guns N' Roses"),
	Entry("handles multiple values with punctuation", "REM ACDC", "R.E.M.", "AC/DC"),
)

var _ = Describe("FTS5 Integration Search", func() {
	var (
		mr  model.MediaFileRepository
		alr model.AlbumRepository
		arr model.ArtistRepository
	)

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, adminUser)
		conn := GetDBXBuilder()
		mr = NewMediaFileRepository(ctx, conn)
		alr = NewAlbumRepository(ctx, conn)
		arr = NewArtistRepository(ctx, conn)
	})

	Describe("MediaFile search", func() {
		It("finds media files by title", func() {
			results, err := mr.Search("Radioactivity", 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("Radioactivity"))
			Expect(results[0].ID).To(Equal(songRadioactivity.ID))
		})

		It("finds media files by artist name", func() {
			results, err := mr.Search("Beatles", 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(2))
			for _, r := range results {
				Expect(r.Artist).To(Equal("The Beatles"))
			}
		})
	})

	Describe("Album search", func() {
		It("finds albums by name", func() {
			results, err := alr.Search("Sgt Peppers", 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Name).To(Equal("Sgt Peppers"))
			Expect(results[0].ID).To(Equal(albumSgtPeppers.ID))
		})

		It("finds albums with multi-word search", func() {
			results, err := alr.Search("Abbey Road", 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Name).To(Equal("Abbey Road"))
			Expect(results[0].ID).To(Equal(albumAbbeyRoad.ID))
		})
	})

	Describe("Artist search", func() {
		It("finds artists by name", func() {
			results, err := arr.Search("Kraftwerk", 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Name).To(Equal("Kraftwerk"))
			Expect(results[0].ID).To(Equal(artistKraftwerk.ID))
		})
	})

	Describe("Legacy backend fallback", func() {
		It("returns results using legacy LIKE-based search when configured", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.SearchBackend = "legacy"

			results, err := mr.Search("Radioactivity", 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("Radioactivity"))
		})
	})
})
