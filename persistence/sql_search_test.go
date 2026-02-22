package persistence

import (
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
			sql, args, err := filter.Where.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("media_file.full_text LIKE"))
			Expect(args).To(ContainElement("% beatles%"))
		})

		It("generates AND of LIKE filters for multiple words", func() {
			filter := legacySearchExpr("media_file", "abbey road")
			Expect(filter).ToNot(BeNil())
			sql, args, err := filter.Where.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("AND"))
			Expect(args).To(HaveLen(2))
		})
	})

	Describe("getSearchExpr", func() {
		It("returns ftsSearchExpr by default (with BM25 ranking)", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "fts"
			conf.Server.Search.FullString = false

			filter := getSearchExpr()("media_file", "test")
			Expect(filter).ToNot(BeNil())
			Expect(filter.RankOrder).To(ContainSubstring("bm25"))
			sql, _, err := filter.Where.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("MATCH"))
		})

		It("returns legacySearchExpr when SearchBackend is legacy", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "legacy"
			conf.Server.Search.FullString = false

			filter := getSearchExpr()("media_file", "test")
			Expect(filter).ToNot(BeNil())
			Expect(filter.Where).ToNot(BeNil())
			sql, _, err := filter.Where.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("LIKE"))
		})

		It("falls back to legacySearchExpr when SearchFullString is enabled", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "fts"
			conf.Server.Search.FullString = true

			filter := getSearchExpr()("media_file", "test")
			Expect(filter).ToNot(BeNil())
			Expect(filter.Where).ToNot(BeNil())
			sql, _, err := filter.Where.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("LIKE"))
		})

		It("routes CJK queries to likeSearchExpr instead of ftsSearchExpr", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "fts"
			conf.Server.Search.FullString = false

			filter := getSearchExpr()("media_file", "周杰伦")
			Expect(filter).ToNot(BeNil())
			Expect(filter.Where).ToNot(BeNil())
			Expect(filter.RankOrder).To(BeEmpty())
			sql, _, err := filter.Where.ToSql()
			Expect(err).ToNot(HaveOccurred())
			// CJK should use LIKE, not MATCH
			Expect(sql).To(ContainSubstring("LIKE"))
			Expect(sql).NotTo(ContainSubstring("MATCH"))
		})

		It("routes non-CJK queries to ftsSearchExpr (with BM25 ranking)", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "fts"
			conf.Server.Search.FullString = false

			filter := getSearchExpr()("media_file", "beatles")
			Expect(filter).ToNot(BeNil())
			Expect(filter.RankOrder).To(ContainSubstring("bm25"))
			sql, _, err := filter.Where.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("MATCH"))
		})

		It("uses legacy for CJK when SearchBackend is legacy", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "legacy"
			conf.Server.Search.FullString = false

			filter := getSearchExpr()("media_file", "周杰伦")
			Expect(filter).ToNot(BeNil())
			Expect(filter.Where).ToNot(BeNil())
			sql, _, err := filter.Where.ToSql()
			Expect(err).ToNot(HaveOccurred())
			// Legacy should still use full_text column LIKE
			Expect(sql).To(ContainSubstring("LIKE"))
			Expect(sql).To(ContainSubstring("full_text"))
		})
	})

})
