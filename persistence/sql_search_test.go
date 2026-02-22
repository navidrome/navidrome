package persistence

import (
	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sqlRepository", func() {
	Describe("formatFullText", func() {
		It("prefixes with a space", func() {
			Expect(formatFullText("legiao urbana")).To(Equal(" legiao urbana"))
		})
	})

	Describe("legacySearchExpr", func() {
		It("returns nil for empty query", func() {
			Expect(legacySearchExpr("media_file", "")).To(BeNil())
		})

		It("generates LIKE filter for single word", func() {
			filter := legacySearchExpr("media_file", "beatles")
			Expect(filter).ToNot(BeNil())
			sql, args, err := filter.where.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("media_file.full_text LIKE"))
			Expect(args).To(ContainElement("% beatles%"))
		})

		It("generates AND of LIKE filters for multiple words", func() {
			filter := legacySearchExpr("media_file", "abbey road")
			Expect(filter).ToNot(BeNil())
			sql, args, err := filter.where.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("AND"))
			Expect(args).To(HaveLen(2))
		})
	})

	Describe("getSearchFilter", func() {
		It("returns FTS5 MATCH filter by default", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "fts"
			conf.Server.Search.FullString = false

			sqlizer := getSearchFilter("media_file", "test")
			Expect(sqlizer).ToNot(BeNil())
			sql, _, err := sqlizer.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("MATCH"))
		})

		It("returns legacy LIKE filter when SearchBackend is legacy", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "legacy"
			conf.Server.Search.FullString = false

			sqlizer := getSearchFilter("media_file", "test")
			Expect(sqlizer).ToNot(BeNil())
			sql, _, err := sqlizer.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("LIKE"))
		})

		It("falls back to legacy LIKE when SearchFullString is enabled", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "fts"
			conf.Server.Search.FullString = true

			sqlizer := getSearchFilter("media_file", "test")
			Expect(sqlizer).ToNot(BeNil())
			sql, _, err := sqlizer.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("LIKE"))
		})

		It("routes CJK queries to LIKE filter", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "fts"
			conf.Server.Search.FullString = false

			sqlizer := getSearchFilter("media_file", "周杰伦")
			Expect(sqlizer).ToNot(BeNil())
			sql, _, err := sqlizer.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("LIKE"))
			Expect(sql).NotTo(ContainSubstring("MATCH"))
		})

		It("returns nil for empty query", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "fts"
			Expect(getSearchFilter("media_file", "")).To(BeNil())
		})
	})

	Describe("applySearchFilter", func() {
		It("adds BM25 ranking for FTS5 queries", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "fts"
			conf.Server.Search.FullString = false

			sq := squirrel.Select("*").From("media_file")
			sq = applySearchFilter(sq, "media_file", "beatles", "media_file.rowid", "title")
			sql, _, err := sq.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("MATCH"))
			Expect(sql).To(ContainSubstring("bm25"))
			Expect(sql).To(ContainSubstring("title"))
		})

		It("falls back to naturalOrder when query produces no filter", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "legacy"

			sq := squirrel.Select("*").From("media_file")
			sq = applySearchFilter(sq, "media_file", "", "media_file.rowid", "title")
			sql, _, err := sq.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("ORDER BY media_file.rowid"))
			Expect(sql).NotTo(ContainSubstring("title"))
		})

		It("uses legacy LIKE when SearchBackend is legacy", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "legacy"
			conf.Server.Search.FullString = false

			sq := squirrel.Select("*").From("media_file")
			sq = applySearchFilter(sq, "media_file", "周杰伦", "media_file.rowid")
			sql, _, err := sq.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("LIKE"))
			Expect(sql).To(ContainSubstring("full_text"))
		})
	})

})
