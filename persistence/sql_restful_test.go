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

		It(`returns nil if tries a filter with legacySearchExpr("'")`, func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Search.Backend = "legacy"
			r.filterMappings = map[string]filterFunc{
				"name": fullTextFilter("table"),
			}
			options.Filters = map[string]any{"name": "'"}
			Expect(r.parseRestFilters(context.Background(), options)).To(BeEmpty())
		})

		It("does not add nill filters", func() {
			r.filterMappings = map[string]filterFunc{
				"name": func(string, any) squirrel.Sqlizer {
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
				"test": func(field string, value any) squirrel.Sqlizer {
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
			conf.Server.Search.Backend = "legacy"
			tableName = "test_table"
			mbidFields = []string{"mbid", "artist_mbid"}
			filter = fullTextFilter(tableName, mbidFields...)
		})

		Context("when value is a valid UUID", func() {
			It("returns only the mbid filter (precedence over full text)", func() {
				uuid := "550e8400-e29b-41d4-a716-446655440000"
				result := filter("search", uuid)

				expected := squirrel.Or{
					squirrel.Eq{"test_table.mbid": uuid},
					squirrel.Eq{"test_table.artist_mbid": uuid},
				}
				Expect(result).To(Equal(expected))
			})

			It("falls back to full text when no mbid fields are provided", func() {
				noMbidFilter := fullTextFilter(tableName)
				uuid := "550e8400-e29b-41d4-a716-446655440000"
				result := noMbidFilter("search", uuid)

				// mbidExpr with no fields returns nil, so cmp.Or falls back to search strategy
				sql, args, err := result.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(ContainSubstring("test_table.full_text LIKE"))
				Expect(args).To(ContainElement("% 550e8400-e29b-41d4-a716-446655440000%"))
			})
		})

		Context("when value is not a valid UUID", func() {
			It("returns full text search condition only", func() {
				result := filter("search", "beatles")

				// mbidExpr returns nil for non-UUIDs, so search strategy result is returned directly
				sql, args, err := result.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(ContainSubstring("test_table.full_text LIKE"))
				Expect(args).To(ContainElement("% beatles%"))
			})

			It("handles multi-word search terms", func() {
				result := filter("search", "the beatles abbey road")

				sql, args, err := result.ToSql()
				Expect(err).ToNot(HaveOccurred())
				// All words should be present as LIKE conditions
				Expect(sql).To(ContainSubstring("test_table.full_text LIKE"))
				Expect(args).To(HaveLen(4))
				Expect(args).To(ContainElement("% the%"))
				Expect(args).To(ContainElement("% beatles%"))
				Expect(args).To(ContainElement("% abbey%"))
				Expect(args).To(ContainElement("% road%"))
			})
		})

		Context("when SearchFullString config changes behavior", func() {
			It("uses different separator with SearchFullString=false", func() {
				conf.Server.Search.FullString = false
				result := filter("search", "test query")

				sql, args, err := result.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(ContainSubstring("test_table.full_text LIKE"))
				Expect(args).To(HaveLen(2))
				Expect(args).To(ContainElement("% test%"))
				Expect(args).To(ContainElement("% query%"))
			})

			It("uses no separator with SearchFullString=true", func() {
				conf.Server.Search.FullString = true
				result := filter("search", "test query")

				sql, args, err := result.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(ContainSubstring("test_table.full_text LIKE"))
				Expect(args).To(HaveLen(2))
				Expect(args).To(ContainElement("%test%"))
				Expect(args).To(ContainElement("%query%"))
			})
		})

		Context("single-character queries (regression: must not be rejected)", func() {
			It("returns valid filter for single-char query with legacy backend", func() {
				conf.Server.Search.Backend = "legacy"
				result := filter("search", "a")
				Expect(result).ToNot(BeNil(), "single-char REST filter must not be dropped")
				sql, args, err := result.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(ContainSubstring("LIKE"))
				Expect(args).ToNot(BeEmpty())
			})

			It("returns valid filter for single-char query with FTS backend", func() {
				conf.Server.Search.Backend = "fts"
				conf.Server.Search.FullString = false
				ftsFilter := fullTextFilter(tableName, mbidFields...)
				result := ftsFilter("search", "a")
				Expect(result).ToNot(BeNil(), "single-char REST filter must not be dropped")
				sql, args, err := result.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(ContainSubstring("MATCH"))
				Expect(args).ToNot(BeEmpty())
			})
		})

		Context("edge cases", func() {
			It("returns nil for empty string", func() {
				result := filter("search", "")
				Expect(result).To(BeNil())
			})

			It("returns nil for string with only whitespace", func() {
				result := filter("search", "   ")
				Expect(result).To(BeNil())
			})

			It("handles special characters that are sanitized", func() {
				result := filter("search", "don't")

				sql, args, err := result.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(ContainSubstring("test_table.full_text LIKE"))
				Expect(args).To(ContainElement("% dont%"))
			})

			It("returns nil for single quote (SQL injection protection)", func() {
				result := filter("search", "'")
				Expect(result).To(BeNil())
			})

			It("handles mixed case UUIDs", func() {
				uuid := "550E8400-E29B-41D4-A716-446655440000"
				result := filter("search", uuid)

				// Should return only mbid filter (uppercase UUID should work)
				expected := squirrel.Or{
					squirrel.Eq{"test_table.mbid": strings.ToLower(uuid)},
					squirrel.Eq{"test_table.artist_mbid": strings.ToLower(uuid)},
				}
				Expect(result).To(Equal(expected))
			})

			It("handles invalid UUID format gracefully", func() {
				result := filter("search", "550e8400-invalid-uuid")

				// Should return full text filter since UUID is invalid
				sql, args, err := result.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(ContainSubstring("test_table.full_text LIKE"))
				Expect(args).To(ContainElement("% 550e8400-invalid-uuid%"))
			})

			It("handles empty mbid fields array", func() {
				emptyMbidFilter := fullTextFilter(tableName, []string{}...)
				result := emptyMbidFilter("search", "test")

				// mbidExpr with empty fields returns nil, so search strategy result is returned directly
				sql, args, err := result.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(ContainSubstring("test_table.full_text LIKE"))
				Expect(args).To(ContainElement("% test%"))
			})

			It("converts value to lowercase before processing", func() {
				result := filter("search", "TEST")

				sql, args, err := result.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(ContainSubstring("test_table.full_text LIKE"))
				Expect(args).To(ContainElement("% test%"))
			})
		})
	})

})
