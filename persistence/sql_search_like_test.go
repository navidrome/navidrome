package persistence

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("newLegacySearch", func() {
	It("returns non-nil for single-character query", func() {
		strategy := newLegacySearch("media_file", "a")
		Expect(strategy).ToNot(BeNil(), "single-char queries must not be rejected; min-length is enforced in doSearch, not here")
		sql, _, err := strategy.ToSql()
		Expect(err).ToNot(HaveOccurred())
		Expect(sql).To(ContainSubstring("LIKE"))
	})
})

var _ = Describe("legacySearchExpr", func() {
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
