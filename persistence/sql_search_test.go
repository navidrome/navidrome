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

	Describe("getSearchStrategy", func() {
		It("returns FTS strategy by default", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "fts"
			conf.Server.Search.FullString = false

			strategy := getSearchStrategy("media_file", "test")
			Expect(strategy).ToNot(BeNil())
			sql, _, err := strategy.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("MATCH"))
		})

		It("returns legacy LIKE strategy when SearchBackend is legacy", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "legacy"
			conf.Server.Search.FullString = false

			strategy := getSearchStrategy("media_file", "test")
			Expect(strategy).ToNot(BeNil())
			sql, _, err := strategy.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("LIKE"))
		})

		It("falls back to legacy LIKE strategy when SearchFullString is enabled", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "fts"
			conf.Server.Search.FullString = true

			strategy := getSearchStrategy("media_file", "test")
			Expect(strategy).ToNot(BeNil())
			sql, _, err := strategy.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("LIKE"))
		})

		It("routes CJK queries to LIKE strategy instead of FTS", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "fts"
			conf.Server.Search.FullString = false

			strategy := getSearchStrategy("media_file", "周杰伦")
			Expect(strategy).ToNot(BeNil())
			sql, _, err := strategy.ToSql()
			Expect(err).ToNot(HaveOccurred())
			// CJK should use LIKE, not MATCH
			Expect(sql).To(ContainSubstring("LIKE"))
			Expect(sql).NotTo(ContainSubstring("MATCH"))
		})

		It("routes non-CJK queries to FTS strategy", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "fts"
			conf.Server.Search.FullString = false

			strategy := getSearchStrategy("media_file", "beatles")
			Expect(strategy).ToNot(BeNil())
			sql, _, err := strategy.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(sql).To(ContainSubstring("MATCH"))
		})

		It("returns non-nil for single-character query (no min-length in strategy)", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "fts"
			conf.Server.Search.FullString = false

			strategy := getSearchStrategy("media_file", "a")
			Expect(strategy).ToNot(BeNil(), "single-char queries must be accepted by strategies (min-length is enforced in doSearch)")
		})

		It("returns non-nil for single-character query with legacy backend", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "legacy"
			conf.Server.Search.FullString = false

			strategy := getSearchStrategy("media_file", "a")
			Expect(strategy).ToNot(BeNil(), "single-char queries must be accepted by legacy strategy (min-length is enforced in doSearch)")
		})

		It("uses legacy for CJK when SearchBackend is legacy", func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "legacy"
			conf.Server.Search.FullString = false

			strategy := getSearchStrategy("media_file", "周杰伦")
			Expect(strategy).ToNot(BeNil())
			sql, _, err := strategy.ToSql()
			Expect(err).ToNot(HaveOccurred())
			// Legacy should still use full_text column LIKE
			Expect(sql).To(ContainSubstring("LIKE"))
			Expect(sql).To(ContainSubstring("full_text"))
		})
	})
})
