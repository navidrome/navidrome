package persistence

import (
	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sqlRepository", func() {
	r := sqlRepository{}
	Describe("applyOptions", func() {
		var sq squirrel.SelectBuilder
		BeforeEach(func() {
			sq = squirrel.Select("*").From("test")
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
			Expect(sql).To(Equal("SELECT * FROM test ORDER BY name desc LIMIT 1 OFFSET 2"))
		})
	})

	Describe("toSQL", func() {
		var r sqlRepository

		BeforeEach(func() {
			r = sqlRepository{}
		})

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

	Describe("sortMapping", func() {
		BeforeEach(func() {
			r.sortMappings = map[string]string{
				"sort1":      "mappedSort1",
				"sortTwo":    "mappedSort2",
				"sort_three": "mappedSort3",
			}
		})

		It("returns the mapped value when sort key exists", func() {
			Expect(r.sortMapping("sort1")).To(Equal("mappedSort1"))
		})

		Context("when sort key does not exist", func() {
			It("returns the original sort key, snake cased", func() {
				Expect(r.sortMapping("NotFoundSort")).To(Equal("not_found_sort"))
			})
		})

		Context("when sort key is camel cased", func() {
			It("returns the mapped value when camel case sort key exists", func() {
				Expect(r.sortMapping("sortTwo")).To(Equal("mappedSort2"))
			})
			It("returns the mapped value when passing a snake case key", func() {
				Expect(r.sortMapping("sort_two")).To(Equal("mappedSort2"))
			})
		})

		Context("when sort key is snake cased", func() {
			It("returns the mapped value when snake case sort key exists", func() {
				Expect(r.sortMapping("sort_three")).To(Equal("mappedSort3"))
			})
			It("returns the mapped value when passing a camel case key", func() {
				Expect(r.sortMapping("sortThree")).To(Equal("mappedSort3"))
			})
		})
	})

	Describe("buildSortOrder", func() {
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
})
