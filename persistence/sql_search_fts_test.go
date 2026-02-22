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
	Entry("converts slashed word to phrase+concat OR", "AC/DC", `("AC DC" OR ACDC*)`),
	Entry("converts hyphenated word to phrase+concat OR", "a-ha", `("a ha" OR aha*)`),
	Entry("converts partial hyphenated word to phrase+concat OR", "a-h", `("a h" OR ah*)`),
	Entry("converts hyphenated name to phrase+concat OR", "Jay-Z", `("Jay Z" OR JayZ*)`),
	Entry("converts contraction to phrase+concat OR", "it's", `("it s" OR its*)`),
	Entry("handles punctuated word mixed with plain words", "best of a-ha", `best* of* ("a ha" OR aha*)`),
	Entry("strips miscellaneous punctuation", "rock & roll, vol. 2", "rock* roll* vol* 2*"),
	Entry("preserves unicode characters with diacritics", "Björk début", "Björk* début*"),
	Entry("collapses dotted abbreviation into phrase", "R.E.M.", `"R E M"`),
	Entry("collapses abbreviation without trailing dot", "R.E.M", `"R E M"`),
	Entry("collapses abbreviation mixed with words", "best of R.E.M.", `best* of* "R E M"`),
	Entry("collapses two-letter abbreviation", "U.K.", `"U K"`),
	Entry("does not collapse single letter surrounded by words", "I am fine", "I* am* fine*"),
	Entry("does not collapse single standalone letter", "A test", "A* test*"),
	Entry("preserves quoted phrase with punctuation verbatim", `"ac/dc"`, `"ac/dc"`),
	Entry("preserves quoted abbreviation verbatim", `"R.E.M."`, `"R.E.M."`),
	Entry("returns empty string for punctuation-only input", "!!!!!!!", ""),
	Entry("returns empty string for mixed punctuation", "!@#$%^&", ""),
	Entry("returns empty string for empty quoted phrase", `""`, ""),
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

var _ = DescribeTable("containsCJK",
	func(input string, expected bool) {
		Expect(containsCJK(input)).To(Equal(expected))
	},
	Entry("returns false for empty string", "", false),
	Entry("returns false for ASCII text", "hello world", false),
	Entry("returns false for Latin with diacritics", "Björk début", false),
	Entry("detects Chinese characters (Han)", "周杰伦", true),
	Entry("detects Japanese Hiragana", "こんにちは", true),
	Entry("detects Japanese Katakana", "カタカナ", true),
	Entry("detects Korean Hangul", "한국어", true),
	Entry("detects CJK mixed with Latin", "best of 周杰伦", true),
	Entry("detects single CJK character", "a曲b", true),
)

var _ = Describe("likeSearchExpr", func() {
	It("returns nil for empty query", func() {
		Expect(likeSearchExpr("media_file", "")).To(BeNil())
	})

	It("returns nil for whitespace-only query", func() {
		Expect(likeSearchExpr("media_file", "   ")).To(BeNil())
	})

	It("generates LIKE filters against core columns for single CJK word", func() {
		expr := likeSearchExpr("media_file", "周杰伦")
		sql, args, err := expr.ToSql()
		Expect(err).ToNot(HaveOccurred())
		// Should have OR between columns for the single word
		Expect(sql).To(ContainSubstring("OR"))
		Expect(sql).To(ContainSubstring("media_file.title LIKE"))
		Expect(sql).To(ContainSubstring("media_file.album LIKE"))
		Expect(sql).To(ContainSubstring("media_file.artist LIKE"))
		Expect(sql).To(ContainSubstring("media_file.album_artist LIKE"))
		Expect(args).To(HaveLen(4))
		for _, arg := range args {
			Expect(arg).To(Equal("%周杰伦%"))
		}
	})

	It("generates AND of OR groups for multi-word query", func() {
		expr := likeSearchExpr("media_file", "周杰伦 greatest")
		sql, args, err := expr.ToSql()
		Expect(err).ToNot(HaveOccurred())
		// Two groups AND'd together, each with 4 columns OR'd
		Expect(sql).To(ContainSubstring("AND"))
		Expect(args).To(HaveLen(8))
	})

	It("uses correct columns for album table", func() {
		expr := likeSearchExpr("album", "周杰伦")
		sql, args, err := expr.ToSql()
		Expect(err).ToNot(HaveOccurred())
		Expect(sql).To(ContainSubstring("album.name LIKE"))
		Expect(sql).To(ContainSubstring("album.album_artist LIKE"))
		Expect(args).To(HaveLen(2))
	})

	It("uses correct columns for artist table", func() {
		expr := likeSearchExpr("artist", "周杰伦")
		sql, args, err := expr.ToSql()
		Expect(err).ToNot(HaveOccurred())
		Expect(sql).To(ContainSubstring("artist.name LIKE"))
		Expect(args).To(HaveLen(1))
	})

	It("returns nil for unknown table", func() {
		Expect(likeSearchExpr("unknown_table", "周杰伦")).To(BeNil())
	})
})

var _ = DescribeTable("qualifyOrderBy",
	func(tableName, orderBy, expected string) {
		Expect(qualifyOrderBy(tableName, orderBy)).To(Equal(expected))
	},
	Entry("returns empty string for empty input", "artist", "", ""),
	Entry("qualifies simple column with table name", "artist", "name", "artist.name"),
	Entry("qualifies column with direction", "artist", "name desc", "artist.name desc"),
	Entry("preserves already-qualified column", "artist", "artist.name", "artist.name"),
	Entry("preserves already-qualified column with direction", "artist", "artist.name desc", "artist.name desc"),
	Entry("returns empty for function call expression", "artist", "sum(json_extract(stats, '$.total.m')) desc", ""),
	Entry("returns empty for expression with comma", "artist", "a, b", ""),
	Entry("qualifies media_file column", "media_file", "title", "media_file.title"),
)

var _ = Describe("ftsSearchExpr", func() {
	It("returns nil for empty query", func() {
		Expect(ftsSearchExpr("media_file", "")).To(BeNil())
	})

	It("returns ftsFilter with correct table names and MATCH expression", func() {
		expr := ftsSearchExpr("media_file", "beatles")
		fts, ok := expr.(*ftsFilter)
		Expect(ok).To(BeTrue())
		Expect(fts.tableName).To(Equal("media_file"))
		Expect(fts.ftsTable).To(Equal("media_file_fts"))
		Expect(fts.matchExpr).To(HavePrefix("{title album artist album_artist"))
		Expect(fts.matchExpr).To(ContainSubstring("beatles*"))
	})

	It("ToSql generates rowid IN subquery with MATCH (fallback path)", func() {
		expr := ftsSearchExpr("media_file", "beatles")
		sql, args, err := expr.ToSql()
		Expect(err).ToNot(HaveOccurred())
		Expect(sql).To(ContainSubstring("media_file.rowid IN"))
		Expect(sql).To(ContainSubstring("media_file_fts"))
		Expect(sql).To(ContainSubstring("MATCH"))
		Expect(args).To(HaveLen(1))
	})

	It("generates correct FTS table name per entity", func() {
		for _, table := range []string{"media_file", "album", "artist"} {
			expr := ftsSearchExpr(table, "test")
			fts, ok := expr.(*ftsFilter)
			Expect(ok).To(BeTrue())
			Expect(fts.tableName).To(Equal(table))
			Expect(fts.ftsTable).To(Equal(table + "_fts"))
		}
	})

	It("builds bm25() rank expression with column weights", func() {
		expr := ftsSearchExpr("media_file", "beatles")
		fts, ok := expr.(*ftsFilter)
		Expect(ok).To(BeTrue())
		Expect(fts.rankExpr).To(HavePrefix("bm25(media_file_fts,"))
		Expect(fts.rankExpr).To(ContainSubstring("10.0"))

		expr = ftsSearchExpr("artist", "beatles")
		fts, ok = expr.(*ftsFilter)
		Expect(ok).To(BeTrue())
		Expect(fts.rankExpr).To(HavePrefix("bm25(artist_fts,"))
	})

	It("falls back to ftsTable.rank for unknown tables", func() {
		expr := ftsSearchExpr("unknown_table", "test")
		fts, ok := expr.(*ftsFilter)
		Expect(ok).To(BeTrue())
		Expect(fts.rankExpr).To(Equal("unknown_table_fts.rank"))
	})

	It("wraps query with column filter for known tables", func() {
		expr := ftsSearchExpr("artist", "Beatles")
		fts, ok := expr.(*ftsFilter)
		Expect(ok).To(BeTrue())
		Expect(fts.matchExpr).To(Equal("{name sort_artist_name search_normalized} : (Beatles*)"))
	})

	It("passes query without column filter for unknown tables", func() {
		expr := ftsSearchExpr("unknown_table", "test")
		fts, ok := expr.(*ftsFilter)
		Expect(ok).To(BeTrue())
		Expect(fts.matchExpr).To(Equal("test*"))
	})

	It("preserves phrase queries inside column filter", func() {
		expr := ftsSearchExpr("media_file", `"the beatles"`)
		fts, ok := expr.(*ftsFilter)
		Expect(ok).To(BeTrue())
		Expect(fts.matchExpr).To(ContainSubstring(`"the beatles"`))
	})

	It("preserves prefix queries inside column filter", func() {
		expr := ftsSearchExpr("media_file", "beat*")
		fts, ok := expr.(*ftsFilter)
		Expect(ok).To(BeTrue())
		Expect(fts.matchExpr).To(ContainSubstring("beat*"))
	})

	It("falls back to LIKE search for punctuation-only query", func() {
		expr := ftsSearchExpr("media_file", "!!!!!!!")
		Expect(expr).ToNot(BeNil())
		_, ok := expr.(*ftsFilter)
		Expect(ok).To(BeFalse(), "punctuation-only should fall back to LIKE, not FTS")
		sql, args, err := expr.ToSql()
		Expect(err).ToNot(HaveOccurred())
		Expect(sql).To(ContainSubstring("LIKE"))
		Expect(args).To(ContainElement("%!!!!!!!%"))
	})

	It("returns nil for empty string even with LIKE fallback", func() {
		Expect(ftsSearchExpr("media_file", "")).To(BeNil())
		Expect(ftsSearchExpr("media_file", "   ")).To(BeNil())
	})

	It("returns nil for empty quoted phrase", func() {
		Expect(ftsSearchExpr("media_file", `""`)).To(BeNil())
	})
})

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
			Expect(results).To(HaveLen(3))
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
			Expect(results).To(HaveLen(2))
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

	Describe("CJK search", func() {
		It("finds media files by CJK title", func() {
			results, err := mr.Search("プラチナ", 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("プラチナ・ジェット"))
			Expect(results[0].ID).To(Equal(songCJK.ID))
		})

		It("finds media files by CJK artist name", func() {
			results, err := mr.Search("シートベルツ", 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Artist).To(Equal("シートベルツ"))
		})

		It("finds albums by CJK artist name", func() {
			results, err := alr.Search("シートベルツ", 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Name).To(Equal("COWBOY BEBOP"))
			Expect(results[0].ID).To(Equal(albumCJK.ID))
		})

		It("finds artists by CJK name", func() {
			results, err := arr.Search("シートベルツ", 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Name).To(Equal("シートベルツ"))
			Expect(results[0].ID).To(Equal(artistCJK.ID))
		})
	})

	Describe("Album version search", func() {
		It("finds albums by version tag via FTS", func() {
			results, err := alr.Search("Deluxe", 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].ID).To(Equal(albumWithVersion.ID))
		})
	})

	Describe("Punctuation-only search", func() {
		It("finds media files with punctuation-only title", func() {
			results, err := mr.Search("!!!!!!!", 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("!!!!!!!"))
			Expect(results[0].ID).To(Equal(songPunctuation.ID))
		})
	})

	Describe("Legacy backend fallback", func() {
		It("returns results using legacy LIKE-based search when configured", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "legacy"

			results, err := mr.Search("Radioactivity", 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("Radioactivity"))
		})
	})
})
