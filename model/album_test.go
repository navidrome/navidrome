package model_test

import (
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Album", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
	})
	DescribeTable("FullName",
		func(enabled bool, tags Tags, expected string) {
			conf.Server.Subsonic.AppendAlbumVersion = enabled
			a := Album{Name: "Album", Tags: tags}
			Expect(a.FullName()).To(Equal(expected))
		},
		Entry("appends version when enabled and tag is present", true, Tags{TagAlbumVersion: []string{"Remastered"}}, "Album (Remastered)"),
		Entry("returns just name when disabled", false, Tags{TagAlbumVersion: []string{"Remastered"}}, "Album"),
		Entry("returns just name when tag is absent", true, Tags{}, "Album"),
		Entry("returns just name when tag is an empty slice", true, Tags{TagAlbumVersion: []string{}}, "Album"),
	)

	Describe("UploadedImagePath", func() {
		BeforeEach(func() {
			conf.Server.DataFolder = conf.NewDir("/data")
		})
		It("returns empty when no image was uploaded", func() {
			Expect(Album{ID: "al-1"}.UploadedImagePath()).To(BeEmpty())
		})
		It("returns the path under the data folder when set", func() {
			a := Album{ID: "al-1", UploadedImage: "al-1_cover.jpg"}
			Expect(a.UploadedImagePath()).To(Equal(filepath.Join("/data", "artwork", "album", "al-1_cover.jpg")))
		})
	})

	Describe("CoverArtID", func() {
		It("changes when a cover is uploaded, even if updated_at is in the future", func() {
			future := time.Now().Add(365 * 24 * time.Hour)
			stamp := time.Now()
			a := Album{ID: "al-1", UpdatedAt: future}
			before := a.CoverArtID().String()
			a.CoverArtUpdatedAt = &stamp
			Expect(a.CoverArtID().String()).ToNot(Equal(before))
		})
		It("does not collide when updated_at decreases by the cover-stamp delta", func() {
			// With an additive combination, updated_at dropping by N while the cover
			// stamp advances by N re-emits the same id — the mix must not.
			u1 := time.Unix(1_000_000_000, 0)
			c1 := time.Unix(2_000_000_000, 0)
			u2 := u1.Add(-100 * time.Second)
			c2 := c1.Add(100 * time.Second)
			id1 := Album{ID: "al-1", UpdatedAt: u1, CoverArtUpdatedAt: &c1}.CoverArtID().String()
			id2 := Album{ID: "al-1", UpdatedAt: u2, CoverArtUpdatedAt: &c2}.CoverArtID().String()
			Expect(id1).ToNot(Equal(id2))
		})
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
				Expect(err).To(BeNil())

				var albums2 Albums
				err = json.Unmarshal(data, &albums2)
				Expect(err).To(BeNil())
				Expect(albums2).To(Equal(albums))
			})
		})
	})
})
