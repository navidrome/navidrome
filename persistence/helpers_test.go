package persistence

import (
	"time"

	"github.com/Masterminds/squirrel"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Helpers", func() {
	Describe("toSnakeCase", func() {
		It("converts camelCase", func() {
			Expect(toSnakeCase("camelCase")).To(Equal("camel_case"))
		})
		It("converts PascalCase", func() {
			Expect(toSnakeCase("PascalCase")).To(Equal("pascal_case"))
		})
		It("converts ALLCAPS", func() {
			Expect(toSnakeCase("ALLCAPS")).To(Equal("allcaps"))
		})
		It("does not converts snake_case", func() {
			Expect(toSnakeCase("snake_case")).To(Equal("snake_case"))
		})
	})
	Describe("toSqlArgs", func() {
		type Model struct {
			ID        string `json:"id"`
			AlbumId   string `json:"albumId"`
			PlayCount int    `json:"playCount"`
			CreatedAt *time.Time
		}

		It("returns a map with snake_case keys", func() {
			now := time.Now()
			m := &Model{ID: "123", AlbumId: "456", CreatedAt: &now, PlayCount: 2}
			args, err := toSqlArgs(m)
			Expect(err).To(BeNil())
			Expect(args).To(HaveKeyWithValue("id", "123"))
			Expect(args).To(HaveKeyWithValue("album_id", "456"))
			Expect(args).To(HaveKey("created_at"))
			Expect(args).To(HaveLen(3))
		})

		It("remove null fields", func() {
			m := &Model{ID: "123", AlbumId: "456"}
			args, err := toSqlArgs(m)
			Expect(err).To(BeNil())
			Expect(args).To(HaveKey("id"))
			Expect(args).To(HaveKey("album_id"))
			Expect(args).To(HaveLen(2))
		})
	})

	Describe("Exists", func() {
		It("constructs the correct EXISTS query", func() {
			e := exists("album", squirrel.Eq{"id": 1})
			sql, args, err := e.ToSql()
			Expect(sql).To(Equal("exists (select 1 from album where id = ?)"))
			Expect(args).To(Equal([]interface{}{1}))
			Expect(err).To(BeNil())
		})
	})
})
