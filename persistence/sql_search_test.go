package persistence

import (
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

	Describe("ftsSearchExpr", func() {
		It("returns nil for empty query", func() {
			Expect(ftsSearchExpr("media_file", "")).To(BeNil())
		})

		It("generates rowid IN subquery for single word", func() {
			expr := ftsSearchExpr("media_file", "beatles")
			sql, args, err := expr.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("media_file.rowid IN"))
			Expect(sql).To(ContainSubstring("media_file_fts"))
			Expect(sql).To(ContainSubstring("MATCH"))
			Expect(args).To(ContainElement("beatles*"))
		})

		It("generates correct FTS table name from entity table", func() {
			expr := ftsSearchExpr("album", "test")
			sql, _, err := expr.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("album_fts"))
		})

		It("uses rowid for artist table", func() {
			expr := ftsSearchExpr("artist", "beatles")
			sql, _, err := expr.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("artist.rowid IN"))
			Expect(sql).To(ContainSubstring("artist_fts"))
		})

		It("preserves phrase queries", func() {
			expr := ftsSearchExpr("media_file", `"the beatles"`)
			_, args, err := expr.ToSql()
			Expect(err).ToNot(HaveOccurred())
			// Phrase is preserved as-is, no * appended
			Expect(args).To(ContainElement(`"the beatles"`))
		})

		It("preserves prefix queries", func() {
			expr := ftsSearchExpr("media_file", "beat*")
			_, args, err := expr.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(args).To(ContainElement("beat*"))
		})
	})
})
