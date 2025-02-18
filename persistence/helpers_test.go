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
				HaveKeyWithValue("updated_at", BeTemporally("~", now)),
				HaveKeyWithValue("created_at", BeTemporally("~", now)),
				Not(HaveKey("Embed")),
			))
		})
	})

	Describe("Exists", func() {
		It("constructs the correct EXISTS query", func() {
			e := Exists("album", squirrel.Eq{"id": 1})
			sql, args, err := e.ToSql()
			Expect(sql).To(Equal("exists (select 1 from album where id = ?)"))
			Expect(args).To(ConsistOf(1))
			Expect(err).To(BeNil())
		})
	})

	Describe("NotExists", func() {
		It("constructs the correct NOT EXISTS query", func() {
			e := NotExists("artist", squirrel.ConcatExpr("id = artist_id"))
			sql, args, err := e.ToSql()
			Expect(sql).To(Equal("not exists (select 1 from artist where id = artist_id)"))
			Expect(args).To(BeEmpty())
			Expect(err).To(BeNil())
		})
	})

	Describe("mapSortOrder", func() {
		It("does not change the sort string if there are no order columns", func() {
			sort := "album_name asc"
			mapped := mapSortOrder("album", sort)
			Expect(mapped).To(Equal(sort))
		})
		It("changes order columns to sort expression", func() {
			sort := "ORDER_ALBUM_NAME asc"
			mapped := mapSortOrder("album", sort)
			Expect(mapped).To(Equal(`(coalesce(nullif(album.sort_album_name,''),album.order_album_name)` +
				` collate nocase) asc`))
		})
		It("changes multiple order columns to sort expressions", func() {
			sort := "compilation, order_title asc, order_album_artist_name desc, year desc"
			mapped := mapSortOrder("album", sort)
			Expect(mapped).To(Equal(`compilation, (coalesce(nullif(album.sort_title,''),album.order_title) collate nocase) asc,` +
				` (coalesce(nullif(album.sort_album_artist_name,''),album.order_album_artist_name) collate nocase) desc, year desc`))
		})
	})
})
