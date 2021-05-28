package persistence

import (
	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo"
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
		})
	})
})
