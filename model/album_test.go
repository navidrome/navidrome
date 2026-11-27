package model_test

import (
	"encoding/json"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	. "github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Album", func() {
	DescribeTable("FullName",
		func(enabled bool, tags Tags, expected string) {
			conf.Update(func(c *conf.Options) { c.Subsonic.AppendAlbumVersion = enabled })
			a := Album{Name: "Album", Tags: tags}
			Expect(a.FullName()).To(Equal(expected))
		},
		Entry("appends version when enabled and tag is present", true, Tags{TagAlbumVersion: []string{"Remastered"}}, "Album (Remastered)"),
		Entry("returns just name when disabled", false, Tags{TagAlbumVersion: []string{"Remastered"}}, "Album"),
		Entry("returns just name when tag is absent", true, Tags{}, "Album"),
		Entry("returns just name when tag is an empty slice", true, Tags{TagAlbumVersion: []string{}}, "Album"),
		Entry("does not double parentheses when version is already parenthesized", true, Tags{TagAlbumVersion: []string{"(Remastered)"}}, "Album (Remastered)"),
		Entry("does not add parentheses when version is wrapped in square brackets", true, Tags{TagAlbumVersion: []string{"[Remastered]"}}, "Album [Remastered]"),
	)

	Describe("RecentlyAddedAt", func() {
		birth := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
		created := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		imported := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
		updated := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)

		DescribeTable("picks the timestamp for the RecentlyAddedBy mode",
			func(mode string, a Album, expected time.Time) {
				conf.Update(func(c *conf.Options) { c.RecentlyAddedBy = mode })
				Expect(a.RecentlyAddedAt()).To(Equal(expected))
			},
			Entry("scan uses CreatedAt", consts.RecentlyAddedByScan, Album{BirthTime: birth, CreatedAt: created, UpdatedAt: updated, ImportedAt: imported}, created),
			Entry("scan falls back to UpdatedAt", consts.RecentlyAddedByScan, Album{BirthTime: birth, UpdatedAt: updated, ImportedAt: imported}, updated),
			Entry("scan falls back to ImportedAt", consts.RecentlyAddedByScan, Album{BirthTime: birth, ImportedAt: imported}, imported),
			Entry("birth uses BirthTime", consts.RecentlyAddedByBirth, Album{BirthTime: birth, CreatedAt: created, UpdatedAt: updated}, birth),
			Entry("birth falls back to CreatedAt", consts.RecentlyAddedByBirth, Album{CreatedAt: created, UpdatedAt: updated}, created),
			Entry("modtime uses UpdatedAt", consts.RecentlyAddedByModTime, Album{BirthTime: birth, CreatedAt: created, UpdatedAt: updated}, updated),
			Entry("modtime falls back to CreatedAt", consts.RecentlyAddedByModTime, Album{BirthTime: birth, CreatedAt: created, ImportedAt: imported}, created),
			Entry("is zero when every timestamp is zero", consts.RecentlyAddedByScan, Album{}, time.Time{}),
		)
	})
})

var _ = Describe("Albums", func() {
	var albums Albums

	Context("JSON Marshalling", func() {
		When("we have a valid Albums object", func() {
			BeforeEach(func() {
				albums = Albums{
					{ID: "1", AlbumArtist: "Artist", AlbumArtistID: "11", SortAlbumArtistName: "SortAlbumArtistName", OrderAlbumArtistName: "OrderAlbumArtistName"},
					{ID: "2", AlbumArtist: "Artist", AlbumArtistID: "11", SortAlbumArtistName: "SortAlbumArtistName", OrderAlbumArtistName: "OrderAlbumArtistName"},
				}
			})
			It("marshals correctly", func() {
				data, err := json.Marshal(albums)
				Expect(err).ToNot(HaveOccurred())

				var albums2 Albums
				err = json.Unmarshal(data, &albums2)
				Expect(err).ToNot(HaveOccurred())
				Expect(albums2).To(Equal(albums))
			})
		})
	})
})
