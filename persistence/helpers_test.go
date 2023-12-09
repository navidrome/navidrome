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
			Expect(args).To(HaveKeyWithValue("id", "123"))
			Expect(args).To(HaveKeyWithValue("album_id", "456"))
			Expect(args).To(HaveKeyWithValue("play_count", 2))
			Expect(args).To(HaveKeyWithValue("updated_at", now.Format(time.RFC3339Nano)))
			Expect(args).To(HaveKeyWithValue("created_at", now.Format(time.RFC3339Nano)))
			Expect(args).ToNot(HaveKey("Embed"))
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
