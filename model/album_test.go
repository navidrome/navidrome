package model_test

import (
	"encoding/json"

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
