package persistence

import (
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
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
		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
		})

		It("does not change the sort string if there are no order columns", func() {
			Expect(mapSortOrder("album", "album_name asc")).To(Equal("album_name asc"))
		})

		DescribeTable("maps order columns to a collated expression",
			func(preferSortTags, naturalSorting bool, expected string) {
				conf.Server.PreferSortTags = preferSortTags
				conf.Server.EnableNaturalSorting = naturalSorting
				Expect(mapSortOrder("album", "ORDER_ALBUM_NAME asc")).To(Equal(expected))
			},
			Entry("qualified column", false, false,
				"(album.order_album_name collate nocase) asc"),
			Entry("natural collation", false, true,
				"(album.order_album_name collate NATSORT) asc"),
			Entry("sort tags preferred", true, false,
				`(coalesce(nullif(album.sort_album_name,''),album.order_album_name) collate nocase) asc`),
			Entry("sort tags preferred, natural collation", true, true,
				`(coalesce(nullif(album.sort_album_name,''),album.order_album_name) collate NATSORT) asc`),
		)

		It("changes multiple order columns to sort expressions", func() {
			conf.Server.PreferSortTags = true
			sort := "compilation, order_title asc, order_album_artist_name desc, year desc"
			Expect(mapSortOrder("album", sort)).To(Equal(
				`compilation, (coalesce(nullif(album.sort_title,''),album.order_title) collate nocase) asc,` +
					` (coalesce(nullif(album.sort_album_artist_name,''),album.order_album_artist_name) collate nocase) desc, year desc`))
		})
	})

	Describe("naturalSort", func() {
		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
		})

		It("leaves the column alone by default, keeping its declared collation", func() {
			Expect(naturalSort("media_file.title")).To(Equal("media_file.title"))
		})

		It("applies the natural collation when enabled", func() {
			conf.Server.EnableNaturalSorting = true
			Expect(naturalSort("media_file.title")).To(Equal("(media_file.title collate NATSORT)"))
		})
	})
})
