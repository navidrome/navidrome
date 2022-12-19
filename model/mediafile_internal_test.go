package model

import (
	"github.com/navidrome/navidrome/consts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("fixAlbumArtist", func() {
	var album Album
	BeforeEach(func() {
		album = Album{}
	})
	Context("Non-Compilations", func() {
		BeforeEach(func() {
			album.Compilation = false
			album.Artist = "Sparks"
			album.ArtistID = "ar-123"
		})
		It("returns the track artist if no album artist is specified", func() {
			al := fixAlbumArtist(album, nil)
			Expect(al.AlbumArtistID).To(Equal("ar-123"))
			Expect(al.AlbumArtist).To(Equal("Sparks"))
		})
		It("returns the album artist if it is specified", func() {
			album.AlbumArtist = "Sparks Brothers"
			album.AlbumArtistID = "ar-345"
			al := fixAlbumArtist(album, nil)
			Expect(al.AlbumArtistID).To(Equal("ar-345"))
			Expect(al.AlbumArtist).To(Equal("Sparks Brothers"))
		})
	})
	Context("Compilations", func() {
		BeforeEach(func() {
			album.Compilation = true
			album.Name = "Sgt. Pepper Knew My Father"
			album.AlbumArtistID = "ar-000"
			album.AlbumArtist = "The Beatles"
		})

		It("returns VariousArtists if there's more than one album artist", func() {
			al := fixAlbumArtist(album, []string{"ar-123", "ar-345"})
			Expect(al.AlbumArtistID).To(Equal(consts.VariousArtistsID))
			Expect(al.AlbumArtist).To(Equal(consts.VariousArtists))
		})

		It("returns the sole album artist if they are the same", func() {
			al := fixAlbumArtist(album, []string{"ar-000", "ar-000"})
			Expect(al.AlbumArtistID).To(Equal("ar-000"))
			Expect(al.AlbumArtist).To(Equal("The Beatles"))
		})
	})
})
