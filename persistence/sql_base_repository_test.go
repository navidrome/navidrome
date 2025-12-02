package persistence

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/hasher"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sqlRepository", func() {
	var r sqlRepository
	BeforeEach(func() {
		r.ctx = request.WithUser(context.Background(), model.User{ID: "user-id"})
		r.tableName = "table"
	})

	Describe("applyOptions", func() {
		var sq squirrel.SelectBuilder
		BeforeEach(func() {
			sq = squirrel.Select("*").From("test")
			r.sortMappings = map[string]string{
				"name": "title",
			}
		})
		It("does not add any clauses when options is empty", func() {
			sq = r.applyOptions(sq, model.QueryOptions{})
			sql, _, _ := sq.ToSql()
			Expect(sql).To(Equal("SELECT * FROM test"))
		})
		It("adds all option clauses", func() {
			sq = r.applyOptions(sq, model.QueryOptions{
				Sort:   "name",
				Order:  "desc",
				Max:    1,
				Offset: 2,
			})
			sql, _, _ := sq.ToSql()
			Expect(sql).To(Equal("SELECT * FROM test ORDER BY title desc LIMIT 1 OFFSET 2"))
		})
	})

	Describe("toSQL", func() {
		It("returns error for invalid SQL", func() {
			sq := squirrel.Select("*").From("test").Where(1)
			_, _, err := r.toSQL(sq)
			Expect(err).To(HaveOccurred())
		})

		It("returns the same query when there are no placeholders", func() {
			sq := squirrel.Select("*").From("test")
			query, params, err := r.toSQL(sq)
			Expect(err).NotTo(HaveOccurred())
			Expect(query).To(Equal("SELECT * FROM test"))
			Expect(params).To(BeEmpty())
		})

		It("replaces one placeholder correctly", func() {
			sq := squirrel.Select("*").From("test").Where(squirrel.Eq{"id": 1})
			query, params, err := r.toSQL(sq)
			Expect(err).NotTo(HaveOccurred())
			Expect(query).To(Equal("SELECT * FROM test WHERE id = {:p0}"))
			Expect(params).To(HaveKeyWithValue("p0", 1))
		})

		It("replaces multiple placeholders correctly", func() {
			sq := squirrel.Select("*").From("test").Where(squirrel.Eq{"id": 1, "name": "test"})
			query, params, err := r.toSQL(sq)
			Expect(err).NotTo(HaveOccurred())
			Expect(query).To(Equal("SELECT * FROM test WHERE id = {:p0} AND name = {:p1}"))
			Expect(params).To(HaveKeyWithValue("p0", 1))
			Expect(params).To(HaveKeyWithValue("p1", "test"))
		})
	})

	Describe("sanitizeSort", func() {
		BeforeEach(func() {
			r.registerModel(&struct {
				Field string `structs:"field"`
			}{}, nil)
			r.sortMappings = map[string]string{
				"sort1": "mappedSort1",
			}
		})

		When("sanitizing sort", func() {
			It("returns empty if the sort key is not found in the model nor in the mappings", func() {
				sort, _ := r.sanitizeSort("unknown", "")
				Expect(sort).To(BeEmpty())
			})

			It("returns the mapped value when sort key exists", func() {
				sort, _ := r.sanitizeSort("sort1", "")
				Expect(sort).To(Equal("mappedSort1"))
			})

			It("is case insensitive", func() {
				sort, _ := r.sanitizeSort("Sort1", "")
				Expect(sort).To(Equal("mappedSort1"))
			})

			It("returns the field if it is a valid field", func() {
				sort, _ := r.sanitizeSort("field", "")
				Expect(sort).To(Equal("field"))
			})

			It("is case insensitive for fields", func() {
				sort, _ := r.sanitizeSort("FIELD", "")
				Expect(sort).To(Equal("field"))
			})
		})
		When("sanitizing order", func() {
			It("returns 'asc' if order is empty", func() {
				_, order := r.sanitizeSort("", "")
				Expect(order).To(Equal(""))
			})

			It("returns 'asc' if order is 'asc'", func() {
				_, order := r.sanitizeSort("", "ASC")
				Expect(order).To(Equal("asc"))
			})

			It("returns 'desc' if order is 'desc'", func() {
				_, order := r.sanitizeSort("", "desc")
				Expect(order).To(Equal("desc"))
			})

			It("returns 'asc' if order is unknown", func() {
				_, order := r.sanitizeSort("", "something")
				Expect(order).To(Equal("asc"))
			})
		})
	})

	Describe("buildSortOrder", func() {
		BeforeEach(func() {
			r.sortMappings = map[string]string{}
		})

		Context("single field", func() {
			It("sorts by specified field", func() {
				sql := r.buildSortOrder("name", "desc")
				Expect(sql).To(Equal("name desc"))
			})
			It("defaults to 'asc'", func() {
				sql := r.buildSortOrder("name", "")
				Expect(sql).To(Equal("name asc"))
			})
			It("inverts pre-defined order", func() {
				sql := r.buildSortOrder("name desc", "desc")
				Expect(sql).To(Equal("name asc"))
			})
			It("forces snake case for field names", func() {
				sql := r.buildSortOrder("AlbumArtist", "asc")
				Expect(sql).To(Equal("album_artist asc"))
			})
		})
		Context("multiple fields", func() {
			It("handles multiple fields", func() {
				sql := r.buildSortOrder("name  desc,age asc,  status desc ", "asc")
				Expect(sql).To(Equal("name desc, age asc, status desc"))
			})
			It("inverts multiple fields", func() {
				sql := r.buildSortOrder("name desc, age, status asc", "desc")
				Expect(sql).To(Equal("name asc, age desc, status desc"))
			})
			It("handles spaces in mapped field", func() {
				r.sortMappings = map[string]string{
					"has_lyrics": "(lyrics != '[]'), updated_at",
				}
				sql := r.buildSortOrder("has_lyrics", "desc")
				Expect(sql).To(Equal("(lyrics != '[]') desc, updated_at desc"))
			})

		})
		Context("function fields", func() {
			It("handles functions with multiple params", func() {
				sql := r.buildSortOrder("substr(id, 7)", "asc")
				Expect(sql).To(Equal("substr(id, 7) asc"))
			})
			It("handles functions with multiple params mixed with multiple fields", func() {
				sql := r.buildSortOrder("name desc, substr(id, 7), status asc", "desc")
				Expect(sql).To(Equal("name asc, substr(id, 7) desc, status desc"))
			})
			It("handles nested functions", func() {
				sql := r.buildSortOrder("name desc, coalesce(nullif(release_date, ''), nullif(original_date, '')), status asc", "desc")
				Expect(sql).To(Equal("name asc, coalesce(nullif(release_date, ''), nullif(original_date, '')) desc, status desc"))
			})
		})
	})

	Describe("resetSeededRandom", func() {
		var id string
		BeforeEach(func() {
			id = r.seedKey()
			hasher.SetSeed(id, "")
		})
		It("does not reset seed if sort is not random", func() {
			var options []model.QueryOptions
			r.resetSeededRandom(options)
			Expect(hasher.CurrentSeed(id)).To(BeEmpty())
		})
		It("resets seed if sort is random", func() {
			options := []model.QueryOptions{{Sort: "random"}}
			r.resetSeededRandom(options)
			Expect(hasher.CurrentSeed(id)).NotTo(BeEmpty())
		})
		It("resets seed if sort is random and seed is provided", func() {
			options := []model.QueryOptions{{Sort: "random", Seed: "seed"}}
			r.resetSeededRandom(options)
			Expect(hasher.CurrentSeed(id)).To(Equal("seed"))
		})
		It("keeps seed when paginating", func() {
			options := []model.QueryOptions{{Sort: "random", Seed: "seed", Offset: 0}}
			r.resetSeededRandom(options)
			Expect(hasher.CurrentSeed(id)).To(Equal("seed"))

			options = []model.QueryOptions{{Sort: "random", Offset: 1}}
			r.resetSeededRandom(options)
			Expect(hasher.CurrentSeed(id)).To(Equal("seed"))
		})
	})

	Describe("applyLibraryFilter", func() {
		var sq squirrel.SelectBuilder

		BeforeEach(func() {
			sq = squirrel.Select("*").From("test_table")
		})

		Context("Admin User", func() {
			BeforeEach(func() {
				r.ctx = request.WithUser(context.Background(), model.User{ID: "admin", IsAdmin: true})
			})

			It("should not apply library filter for admin users", func() {
				result := r.applyLibraryFilter(sq)
				sql, _, _ := result.ToSql()
				Expect(sql).To(Equal("SELECT * FROM test_table"))
			})
		})

		Context("Regular User", func() {
			BeforeEach(func() {
				r.ctx = request.WithUser(context.Background(), model.User{ID: "user123", IsAdmin: false})
			})

			It("should apply library filter for regular users", func() {
				result := r.applyLibraryFilter(sq)
				sql, args, _ := result.ToSql()
				Expect(sql).To(ContainSubstring("IN (SELECT ul.library_id FROM user_library ul WHERE ul.user_id = ?)"))
				Expect(args).To(ContainElement("user123"))
			})

			It("should use custom table name when provided", func() {
				result := r.applyLibraryFilter(sq, "custom_table")
				sql, args, _ := result.ToSql()
				Expect(sql).To(ContainSubstring("custom_table.library_id IN"))
				Expect(args).To(ContainElement("user123"))
			})
		})

		Context("Headless Process (No User Context)", func() {
			BeforeEach(func() {
				r.ctx = context.Background() // No user context
			})

			It("should not apply library filter for headless processes", func() {
				result := r.applyLibraryFilter(sq)
				sql, _, _ := result.ToSql()
				Expect(sql).To(Equal("SELECT * FROM test_table"))
			})

			It("should not apply library filter even with custom table name", func() {
				result := r.applyLibraryFilter(sq, "custom_table")
				sql, _, _ := result.ToSql()
				Expect(sql).To(Equal("SELECT * FROM test_table"))
			})
		})
	})
})
