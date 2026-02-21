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
			expr := legacySearchExpr("media_file", "beatles")
			sql, args, err := expr.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("media_file.full_text LIKE"))
			Expect(args).To(ContainElement("% beatles%"))
		})

		It("generates AND of LIKE filters for multiple words", func() {
			expr := legacySearchExpr("media_file", "abbey road")
			sql, args, err := expr.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("AND"))
			Expect(args).To(HaveLen(2))
		})
	})

	Describe("getSearchExpr", func() {
		It("returns ftsSearchExpr by default", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.SearchBackend = "fts"
			conf.Server.SearchFullString = false

			expr := getSearchExpr()("media_file", "test")
			sql, _, err := expr.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("MATCH"))
		})

		It("returns legacySearchExpr when SearchBackend is legacy", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.SearchBackend = "legacy"
			conf.Server.SearchFullString = false

			expr := getSearchExpr()("media_file", "test")
			sql, _, err := expr.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("LIKE"))
		})

		It("falls back to legacySearchExpr when SearchFullString is enabled", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.SearchBackend = "fts"
			conf.Server.SearchFullString = true

			expr := getSearchExpr()("media_file", "test")
			sql, _, err := expr.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("LIKE"))
		})
	})

})
