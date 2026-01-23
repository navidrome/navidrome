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

		It(`returns nil if tries a filter with fullTextExpr("'")`, func() {
			r.filterMappings = map[string]filterFunc{
				"name": fullTextFilter("table"),
			}
			options.Filters = map[string]interface{}{"name": "'"}
			Expect(r.parseRestFilters(context.Background(), options)).To(BeEmpty())
		})

		It("does not add nill filters", func() {
			r.filterMappings = map[string]filterFunc{
				"name": func(string, any) squirrel.Sqlizer {
					return nil
				},
			}
			options.Filters = map[string]interface{}{"name": "joe"}
			Expect(r.parseRestFilters(context.Background(), options)).To(BeEmpty())
		})

		It("returns a '=' condition for 'id' filter", func() {
			options.Filters = map[string]interface{}{"id": "123"}
			Expect(r.parseRestFilters(context.Background(), options)).To(Equal(squirrel.And{squirrel.Eq{"id": "123"}}))
		})

		It("returns a 'in' condition for multiples 'id' filters", func() {
			options.Filters = map[string]interface{}{"id": []string{"123", "456"}}
			Expect(r.parseRestFilters(context.Background(), options)).To(Equal(squirrel.And{squirrel.Eq{"id": []string{"123", "456"}}}))
		})

		It("returns a 'like' condition for other filters", func() {
			options.Filters = map[string]interface{}{"name": "joe"}
			Expect(r.parseRestFilters(context.Background(), options)).To(Equal(squirrel.And{squirrel.Like{"name": "joe%"}}))
		})

		It("uses the custom filter", func() {
			r.filterMappings = map[string]filterFunc{
				"test": func(field string, value interface{}) squirrel.Sqlizer {
					return squirrel.Gt{field: value}
				},
			}
			options.Filters = map[string]interface{}{"test": 100}
			Expect(r.parseRestFilters(context.Background(), options)).To(Equal(squirrel.And{squirrel.Gt{"test": 100}}))
		})
	})

	Describe("fullTextFilter function", func() {
		var filter filterFunc
		var tableName string
		var mbidFields []string

		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
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

				// mbidExpr with no fields returns nil, so cmp.Or falls back to fullTextExpr
				expected := squirrel.And{
					squirrel.Like{"test_table.full_text": "% 550e8400-e29b-41d4-a716-446655440000%"},
				}
				Expect(result).To(Equal(expected))
			})
		})

		Context("when value is not a valid UUID", func() {
			It("returns full text search condition only", func() {
				result := filter("search", "beatles")

				// mbidExpr returns nil for non-UUIDs, so fullTextExpr result is returned directly
				expected := squirrel.And{
					squirrel.Like{"test_table.full_text": "% beatles%"},
				}
				Expect(result).To(Equal(expected))
			})

			It("handles multi-word search terms", func() {
				result := filter("search", "the beatles abbey road")

				// Should return And condition directly
				andCondition, ok := result.(squirrel.And)
				Expect(ok).To(BeTrue())
				Expect(andCondition).To(HaveLen(4))

				// Check that all words are present (order may vary)
				Expect(andCondition).To(ContainElement(squirrel.Like{"test_table.full_text": "% the%"}))
				Expect(andCondition).To(ContainElement(squirrel.Like{"test_table.full_text": "% beatles%"}))
				Expect(andCondition).To(ContainElement(squirrel.Like{"test_table.full_text": "% abbey%"}))
				Expect(andCondition).To(ContainElement(squirrel.Like{"test_table.full_text": "% road%"}))
			})
		})

		Context("when SearchFullString config changes behavior", func() {
			It("uses different separator with SearchFullString=false", func() {
				conf.Server.SearchFullString = false
				result := filter("search", "test query")

				andCondition, ok := result.(squirrel.And)
				Expect(ok).To(BeTrue())
				Expect(andCondition).To(HaveLen(2))

				// Check that all words are present with leading space (order may vary)
				Expect(andCondition).To(ContainElement(squirrel.Like{"test_table.full_text": "% test%"}))
				Expect(andCondition).To(ContainElement(squirrel.Like{"test_table.full_text": "% query%"}))
			})

			It("uses no separator with SearchFullString=true", func() {
				conf.Server.SearchFullString = true
				result := filter("search", "test query")

				andCondition, ok := result.(squirrel.And)
				Expect(ok).To(BeTrue())
				Expect(andCondition).To(HaveLen(2))

				// Check that all words are present without leading space (order may vary)
				Expect(andCondition).To(ContainElement(squirrel.Like{"test_table.full_text": "%test%"}))
				Expect(andCondition).To(ContainElement(squirrel.Like{"test_table.full_text": "%query%"}))
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

				expected := squirrel.And{
					squirrel.Like{"test_table.full_text": "% dont%"}, // str.SanitizeStrings removes quotes
				}
				Expect(result).To(Equal(expected))
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
				expected := squirrel.And{
					squirrel.Like{"test_table.full_text": "% 550e8400-invalid-uuid%"},
				}
				Expect(result).To(Equal(expected))
			})

			It("handles empty mbid fields array", func() {
				emptyMbidFilter := fullTextFilter(tableName, []string{}...)
				result := emptyMbidFilter("search", "test")

				// mbidExpr with empty fields returns nil, so cmp.Or falls back to fullTextExpr
				expected := squirrel.And{
					squirrel.Like{"test_table.full_text": "% test%"},
				}
				Expect(result).To(Equal(expected))
			})

			It("converts value to lowercase before processing", func() {
				result := filter("search", "TEST")

				// The function converts to lowercase internally
				expected := squirrel.And{
					squirrel.Like{"test_table.full_text": "% test%"},
				}
				Expect(result).To(Equal(expected))
			})
		})
	})

})
