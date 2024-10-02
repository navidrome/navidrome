package persistence

import (
	"time"

	"github.com/Masterminds/squirrel"
	. "github.com/onsi/ginkgo/v2"
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
	Describe("toCamelCase", func() {
		It("converts snake_case", func() {
			Expect(toCamelCase("snake_case")).To(Equal("snakeCase"))
		})
		It("converts PascalCase", func() {
			Expect(toCamelCase("PascalCase")).To(Equal("PascalCase"))
		})
		It("converts camelCase", func() {
			Expect(toCamelCase("camelCase")).To(Equal("camelCase"))
		})
		It("converts ALLCAPS", func() {
			Expect(toCamelCase("ALLCAPS")).To(Equal("ALLCAPS"))
		})
	})
	Describe("toSQLArgs", func() {
		type Embed struct{}
		type Model struct {
			Embed     `structs:"-"`
			ID        string     `structs:"id" json:"id"`
			AlbumId   string     `structs:"album_id" json:"albumId"`
			PlayCount int        `structs:"play_count" json:"playCount"`
			UpdatedAt *time.Time `structs:"updated_at"`
			CreatedAt time.Time  `structs:"created_at"`
		}

		It("returns a map with snake_case keys", func() {
			now := time.Now()
			m := &Model{ID: "123", AlbumId: "456", CreatedAt: now, UpdatedAt: &now, PlayCount: 2}
			args, err := toSQLArgs(m)
			Expect(err).To(BeNil())
			Expect(args).To(SatisfyAll(
				HaveKeyWithValue("id", "123"),
				HaveKeyWithValue("album_id", "456"),
				HaveKeyWithValue("play_count", 2),
				HaveKeyWithValue("updated_at", now.Format(time.RFC3339Nano)),
				HaveKeyWithValue("created_at", now.Format(time.RFC3339Nano)),
				Not(HaveKey("Embed")),
			))
		})
	})

	Describe("exists", func() {
		It("constructs the correct EXISTS query", func() {
			e := exists("album", squirrel.Eq{"id": 1})
			sql, args, err := e.ToSql()
			Expect(sql).To(Equal("exists (select 1 from album where id = ?)"))
			Expect(args).To(ConsistOf(1))
			Expect(err).To(BeNil())
		})
	})

	Describe("notExists", func() {
		It("constructs the correct NOT EXISTS query", func() {
			e := notExists("artist", squirrel.ConcatExpr("id = artist_id"))
			sql, args, err := e.ToSql()
			Expect(sql).To(Equal("not exists (select 1 from artist where id = artist_id)"))
			Expect(args).To(BeEmpty())
			Expect(err).To(BeNil())
		})
	})
})
