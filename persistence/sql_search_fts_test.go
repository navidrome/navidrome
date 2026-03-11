package persistence

import (
	"context"

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

var _ = DescribeTable("ftsQueryDegraded",
	func(original, ftsQuery string, expected bool) {
		Expect(ftsQueryDegraded(original, ftsQuery)).To(Equal(expected))
	},
	Entry("not degraded for empty original", "", "1*", false),
	Entry("not degraded for empty ftsQuery", "1+", "", false),
	Entry("not degraded for purely alphanumeric query", "beatles", "beatles*", false),
	Entry("not degraded when long tokens remain", "test^val", "test* val*", false),
	Entry("not degraded for quoted phrase with long tokens", `"the beatles"`, `"the beatles"`, false),
	Entry("degraded for quoted phrase with only short tokens after tokenizer strips special chars", `"1+"`, `"1+"`, true),
	Entry("not degraded for quoted phrase with meaningful content", `"C++ programming"`, `"C++ programming"`, false),
	Entry("degraded when special chars stripped leaving short token", "1+", "1*", true),
	Entry("degraded when special chars stripped leaving two short tokens", "C# 1", "C* 1*", true),
	Entry("not degraded when at least one long token remains", "1+ beatles", "1* beatles*", false),
	Entry("not degraded for OR groups from processPunctuatedWords", "AC/DC", `("AC DC" OR ACDC*)`, false),
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

var _ = Describe("ftsColumnDefs helpers", func() {
	Describe("ftsColumnFilters", func() {
		It("returns column filter for media_file", func() {
			Expect(ftsColumnFilters).To(HaveKeyWithValue("media_file",
				"{title album artist album_artist sort_title sort_album_name sort_artist_name sort_album_artist_name disc_subtitle search_participants search_normalized}",
			))
		})

		It("returns column filter for album", func() {
			Expect(ftsColumnFilters).To(HaveKeyWithValue("album",
				"{name sort_album_name album_artist search_participants discs catalog_num album_version search_normalized}",
			))
		})

		It("returns column filter for artist", func() {
			Expect(ftsColumnFilters).To(HaveKeyWithValue("artist",
				"{name sort_artist_name search_normalized}",
			))
		})

		It("has no entry for unknown table", func() {
			Expect(ftsColumnFilters).ToNot(HaveKey("unknown"))
		})
	})

	Describe("ftsBM25Weights", func() {
		It("returns weight CSV for media_file", func() {
			Expect(ftsBM25Weights).To(HaveKeyWithValue("media_file",
				"10.0, 5.0, 3.0, 3.0, 1.0, 1.0, 1.0, 1.0, 1.0, 2.0, 1.0",
			))
		})

		It("returns weight CSV for album", func() {
			Expect(ftsBM25Weights).To(HaveKeyWithValue("album",
				"10.0, 1.0, 3.0, 2.0, 1.0, 1.0, 1.0, 1.0",
			))
		})

		It("returns weight CSV for artist", func() {
			Expect(ftsBM25Weights).To(HaveKeyWithValue("artist",
				"10.0, 1.0, 1.0",
			))
		})

		It("has no entry for unknown table", func() {
			Expect(ftsBM25Weights).ToNot(HaveKey("unknown"))
		})
	})

	It("has definitions for all known tables", func() {
		for _, table := range []string{"media_file", "album", "artist"} {
			Expect(ftsColumnDefs).To(HaveKey(table))
			Expect(ftsColumnDefs[table]).ToNot(BeEmpty())
		}
	})

	It("has matching column count between filter and weights", func() {
		for table, cols := range ftsColumnDefs {
			// Column filter only includes Weight > 0 columns
			filterCount := 0
			for _, c := range cols {
				if c.Weight > 0 {
					filterCount++
				}
			}
			// For now, all columns have Weight > 0, so filter count == total count
			Expect(filterCount).To(Equal(len(cols)), "table %s: all columns should have positive weights", table)
		}
	})
})

var _ = Describe("newFTSSearch", func() {
	It("returns nil for empty query", func() {
		Expect(newFTSSearch("media_file", "")).To(BeNil())
	})

	It("returns non-nil for single-character query", func() {
		strategy := newFTSSearch("media_file", "a")
		Expect(strategy).ToNot(BeNil(), "single-char queries must not be rejected; min-length is enforced in doSearch, not here")
		sql, _, err := strategy.ToSql()
		Expect(err).ToNot(HaveOccurred())
		Expect(sql).To(ContainSubstring("MATCH"))
	})

	It("returns ftsSearch with correct table names and MATCH expression", func() {
		strategy := newFTSSearch("media_file", "beatles")
		fts, ok := strategy.(*ftsSearch)
		Expect(ok).To(BeTrue())
		Expect(fts.tableName).To(Equal("media_file"))
		Expect(fts.ftsTable).To(Equal("media_file_fts"))
		Expect(fts.matchExpr).To(HavePrefix("{title album artist album_artist"))
		Expect(fts.matchExpr).To(ContainSubstring("beatles*"))
	})

	It("ToSql generates rowid IN subquery with MATCH (fallback path)", func() {
		strategy := newFTSSearch("media_file", "beatles")
		sql, args, err := strategy.ToSql()
		Expect(err).ToNot(HaveOccurred())
		Expect(sql).To(ContainSubstring("media_file.rowid IN"))
		Expect(sql).To(ContainSubstring("media_file_fts"))
		Expect(sql).To(ContainSubstring("MATCH"))
		Expect(args).To(HaveLen(1))
	})

	It("generates correct FTS table name per entity", func() {
		for _, table := range []string{"media_file", "album", "artist"} {
			strategy := newFTSSearch(table, "test")
			fts, ok := strategy.(*ftsSearch)
			Expect(ok).To(BeTrue())
			Expect(fts.tableName).To(Equal(table))
			Expect(fts.ftsTable).To(Equal(table + "_fts"))
		}
	})

	It("builds bm25() rank expression with column weights", func() {
		strategy := newFTSSearch("media_file", "beatles")
		fts, ok := strategy.(*ftsSearch)
		Expect(ok).To(BeTrue())
		Expect(fts.rankExpr).To(HavePrefix("bm25(media_file_fts,"))
		Expect(fts.rankExpr).To(ContainSubstring("10.0"))

		strategy = newFTSSearch("artist", "beatles")
		fts, ok = strategy.(*ftsSearch)
		Expect(ok).To(BeTrue())
		Expect(fts.rankExpr).To(HavePrefix("bm25(artist_fts,"))
	})

	It("falls back to ftsTable.rank for unknown tables", func() {
		strategy := newFTSSearch("unknown_table", "test")
		fts, ok := strategy.(*ftsSearch)
		Expect(ok).To(BeTrue())
		Expect(fts.rankExpr).To(Equal("unknown_table_fts.rank"))
	})

	It("wraps query with column filter for known tables", func() {
		strategy := newFTSSearch("artist", "Beatles")
		fts, ok := strategy.(*ftsSearch)
		Expect(ok).To(BeTrue())
		Expect(fts.matchExpr).To(Equal("{name sort_artist_name search_normalized} : (Beatles*)"))
	})

	It("passes query without column filter for unknown tables", func() {
		strategy := newFTSSearch("unknown_table", "test")
		fts, ok := strategy.(*ftsSearch)
		Expect(ok).To(BeTrue())
		Expect(fts.matchExpr).To(Equal("test*"))
	})

	It("preserves phrase queries inside column filter", func() {
		strategy := newFTSSearch("media_file", `"the beatles"`)
		fts, ok := strategy.(*ftsSearch)
		Expect(ok).To(BeTrue())
		Expect(fts.matchExpr).To(ContainSubstring(`"the beatles"`))
	})

	It("preserves prefix queries inside column filter", func() {
		strategy := newFTSSearch("media_file", "beat*")
		fts, ok := strategy.(*ftsSearch)
		Expect(ok).To(BeTrue())
		Expect(fts.matchExpr).To(ContainSubstring("beat*"))
	})

	It("falls back to LIKE search for punctuation-only query", func() {
		strategy := newFTSSearch("media_file", "!!!!!!!")
		Expect(strategy).ToNot(BeNil())
		_, ok := strategy.(*ftsSearch)
		Expect(ok).To(BeFalse(), "punctuation-only should fall back to LIKE, not FTS")
		sql, args, err := strategy.ToSql()
		Expect(err).ToNot(HaveOccurred())
		Expect(sql).To(ContainSubstring("LIKE"))
		Expect(args).To(ContainElement("%!!!!!!!%"))
	})

	It("falls back to LIKE search for degraded query (special chars stripped leaving short tokens)", func() {
		strategy := newFTSSearch("album", "1+")
		Expect(strategy).ToNot(BeNil())
		_, ok := strategy.(*ftsSearch)
		Expect(ok).To(BeFalse(), "degraded query should fall back to LIKE, not FTS")
		sql, args, err := strategy.ToSql()
		Expect(err).ToNot(HaveOccurred())
		Expect(sql).To(ContainSubstring("LIKE"))
		Expect(args).To(ContainElement("%1+%"))
	})

	It("returns nil for empty string even with LIKE fallback", func() {
		Expect(newFTSSearch("media_file", "")).To(BeNil())
		Expect(newFTSSearch("media_file", "   ")).To(BeNil())
	})

	It("returns nil for empty quoted phrase", func() {
		Expect(newFTSSearch("media_file", `""`)).To(BeNil())
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
			results, err := mr.Search("Radioactivity", model.QueryOptions{Max: 10})
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("Radioactivity"))
			Expect(results[0].ID).To(Equal(songRadioactivity.ID))
		})

		It("finds media files by artist name", func() {
			results, err := mr.Search("Beatles", model.QueryOptions{Max: 10})
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(3))
			for _, r := range results {
				Expect(r.Artist).To(Equal("The Beatles"))
			}
		})
	})

	Describe("Album search", func() {
		It("finds albums by name", func() {
			results, err := alr.Search("Sgt Peppers", model.QueryOptions{Max: 10})
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Name).To(Equal("Sgt Peppers"))
			Expect(results[0].ID).To(Equal(albumSgtPeppers.ID))
		})

		It("finds albums with multi-word search", func() {
			results, err := alr.Search("Abbey Road", model.QueryOptions{Max: 10})
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(2))
		})
	})

	Describe("Artist search", func() {
		It("finds artists by name", func() {
			results, err := arr.Search("Kraftwerk", model.QueryOptions{Max: 10})
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Name).To(Equal("Kraftwerk"))
			Expect(results[0].ID).To(Equal(artistKraftwerk.ID))
		})
	})

	Describe("CJK search", func() {
		It("finds media files by CJK title", func() {
			results, err := mr.Search("プラチナ", model.QueryOptions{Max: 10})
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("プラチナ・ジェット"))
			Expect(results[0].ID).To(Equal(songCJK.ID))
		})

		It("finds media files by CJK artist name", func() {
			results, err := mr.Search("シートベルツ", model.QueryOptions{Max: 10})
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Artist).To(Equal("シートベルツ"))
		})

		It("finds albums by CJK artist name", func() {
			results, err := alr.Search("シートベルツ", model.QueryOptions{Max: 10})
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Name).To(Equal("COWBOY BEBOP"))
			Expect(results[0].ID).To(Equal(albumCJK.ID))
		})

		It("finds artists by CJK name", func() {
			results, err := arr.Search("シートベルツ", model.QueryOptions{Max: 10})
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Name).To(Equal("シートベルツ"))
			Expect(results[0].ID).To(Equal(artistCJK.ID))
		})
	})

	Describe("Album version search", func() {
		It("finds albums by version tag via FTS", func() {
			results, err := alr.Search("Deluxe", model.QueryOptions{Max: 10})
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].ID).To(Equal(albumWithVersion.ID))
		})
	})

	Describe("Punctuation-only search", func() {
		It("finds media files with punctuation-only title", func() {
			results, err := mr.Search("!!!!!!!", model.QueryOptions{Max: 10})
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Title).To(Equal("!!!!!!!"))
			Expect(results[0].ID).To(Equal(songPunctuation.ID))
		})
	})

	Describe("Single-character search (doSearch min-length guard)", func() {
		It("returns empty results for single-char query via Search", func() {
			results, err := mr.Search("a", model.QueryOptions{Max: 10})
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(BeEmpty(), "doSearch should reject single-char queries")
		})
	})

	Describe("Max=0 means no limit (regression: must not produce LIMIT 0)", func() {
		It("returns results with Max=0", func() {
			results, err := mr.Search("Beatles", model.QueryOptions{Max: 0})
			Expect(err).ToNot(HaveOccurred())
			Expect(results).ToNot(BeEmpty(), "Max=0 should mean no limit, not LIMIT 0")
		})
	})
})
