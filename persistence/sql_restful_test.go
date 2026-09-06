package persistence

import (
	"context"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sqlRestful", func() {
	Describe("parseRestFilters", func() {
		var r sqlRepository
		var options rest.QueryOptions

		BeforeEach(func() {
			r = sqlRepository{}
		})

		It("returns nil if filters is empty", func() {
			options.Filters = nil
			Expect(r.parseRestFilters(context.Background(), options)).To(BeNil())
		})

		It(`returns nil for a single-quote FTS filter`, func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "fts"
			r.filterMappings = map[string]filterFunc{
				"name": fullTextFilter("table"),
			}
			options.Filters = map[string]any{"name": "'"}
			Expect(r.parseRestFilters(context.Background(), options)).To(BeEmpty())
		})

		It("does not add nill filters", func() {
			r.filterMappings = map[string]filterFunc{
				"name": func(context.Context, string, any) squirrel.Sqlizer {
					return nil
				},
			}
			options.Filters = map[string]any{"name": "joe"}
			Expect(r.parseRestFilters(context.Background(), options)).To(BeEmpty())
		})

		It("returns a '=' condition for 'id' filter", func() {
			options.Filters = map[string]any{"id": "123"}
			Expect(r.parseRestFilters(context.Background(), options)).To(Equal(squirrel.And{squirrel.Eq{"id": "123"}}))
		})

		It("returns a 'in' condition for multiples 'id' filters", func() {
			options.Filters = map[string]any{"id": []string{"123", "456"}}
			Expect(r.parseRestFilters(context.Background(), options)).To(Equal(squirrel.And{squirrel.Eq{"id": []string{"123", "456"}}}))
		})

		It("returns a 'like' condition for other filters", func() {
			options.Filters = map[string]any{"name": "joe"}
			Expect(r.parseRestFilters(context.Background(), options)).To(Equal(squirrel.And{squirrel.Like{"name": "joe%"}}))
		})

		It("uses the custom filter", func() {
			r.filterMappings = map[string]filterFunc{
				"test": func(_ context.Context, field string, value any) squirrel.Sqlizer {
					return squirrel.Gt{field: value}
				},
			}
			options.Filters = map[string]any{"test": 100}
			Expect(r.parseRestFilters(context.Background(), options)).To(Equal(squirrel.And{squirrel.Gt{"test": 100}}))
		})
	})

	Describe("fullTextFilter function", func() {
		var filter filterFunc
		var tableName string
		var mbidFields []string

		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "fts"
			conf.Server.Search.FullString = false
			tableName = "test_table"
			mbidFields = []string{"mbid", "artist_mbid"}
			filter = fullTextFilter(tableName, mbidFields...)
		})

		Context("when value is a valid UUID", func() {
			It("returns only the mbid filter (precedence over full text)", func() {
				uuid := "550e8400-e29b-41d4-a716-446655440000"
				result := filter(context.Background(), "search", uuid)

				expected := squirrel.Or{
					squirrel.Eq{"test_table.mbid": uuid},
					squirrel.Eq{"test_table.artist_mbid": uuid},
				}
				Expect(result).To(Equal(expected))
			})

			It("falls back to full text when no mbid fields are provided", func() {
				noMbidFilter := fullTextFilter(tableName)
				uuid := "550e8400-e29b-41d4-a716-446655440000"
				result := noMbidFilter(context.Background(), "search", uuid)

				sql, args, err := result.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(ContainSubstring("MATCH"))
				Expect(args).ToNot(BeEmpty())
			})
		})

		Context("when value is not a valid UUID", func() {
			It("returns FTS search condition only", func() {
				result := filter(context.Background(), "search", "beatles")

				sql, args, err := result.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(ContainSubstring("MATCH"))
				Expect(args).ToNot(BeEmpty())
			})

			It("handles multi-word search terms", func() {
				result := filter(context.Background(), "search", "the beatles abbey road")

				sql, args, err := result.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(ContainSubstring("MATCH"))
				Expect(args).ToNot(BeEmpty())
			})
		})

		Context("FTS query handling", func() {
			It("uses FTS for multi-word queries", func() {
				result := filter(context.Background(), "search", "test query")

				sql, args, err := result.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(ContainSubstring("MATCH"))
				Expect(args).ToNot(BeEmpty())
			})
		})

		Context("single-character queries (regression: must not be rejected)", func() {
			It("returns valid filter for single-char query with FTS backend", func() {
				result := filter(context.Background(), "search", "a")
				Expect(result).ToNot(BeNil(), "single-char REST filter must not be dropped")
				sql, args, err := result.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(ContainSubstring("MATCH"))
				Expect(args).ToNot(BeEmpty())
			})
		})

		Context("edge cases", func() {
			It("returns nil for empty string", func() {
				result := filter(context.Background(), "search", "")
				Expect(result).To(BeNil())
			})

			It("returns nil for string with only whitespace", func() {
				result := filter(context.Background(), "search", "   ")
				Expect(result).To(BeNil())
			})

			It("handles special characters that are sanitized", func() {
				result := filter(context.Background(), "search", "don't")

				sql, args, err := result.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(ContainSubstring("MATCH"))
				Expect(args).ToNot(BeEmpty())
			})

			It("returns nil for single quote (SQL injection protection)", func() {
				result := filter(context.Background(), "search", "'")
				Expect(result).To(BeNil())
			})

			It("handles mixed case UUIDs", func() {
				uuid := "550E8400-E29B-41D4-A716-446655440000"
				result := filter(context.Background(), "search", uuid)

				// Should return only mbid filter (uppercase UUID should work)
				expected := squirrel.Or{
					squirrel.Eq{"test_table.mbid": strings.ToLower(uuid)},
					squirrel.Eq{"test_table.artist_mbid": strings.ToLower(uuid)},
				}
				Expect(result).To(Equal(expected))
			})

			It("handles invalid UUID format gracefully", func() {
				result := filter(context.Background(), "search", "550e8400-invalid-uuid")

				sql, args, err := result.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(ContainSubstring("MATCH"))
				Expect(args).ToNot(BeEmpty())
			})

			It("handles empty mbid fields array", func() {
				emptyMbidFilter := fullTextFilter(tableName, []string{}...)
				result := emptyMbidFilter(context.Background(), "search", "test")

				sql, args, err := result.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(ContainSubstring("MATCH"))
				Expect(args).ToNot(BeEmpty())
			})

			It("converts value to lowercase before processing", func() {
				result := filter(context.Background(), "search", "TEST")

				sql, args, err := result.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(ContainSubstring("MATCH"))
				Expect(args).ToNot(BeEmpty())
			})
		})
	})

})
