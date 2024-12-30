package model

import (
	"github.com/navidrome/navidrome/consts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("fixAlbumArtist", func() {
	var album Album
	BeforeEach(func() {
		album = Album{Participants: Participants{}}
	})
	Context("Non-Compilations", func() {
		BeforeEach(func() {
			album.Compilation = false
			album.Participants.Add(RoleArtist, Artist{ID: "ar-123", Name: "Sparks"})
		})
		It("returns the track artist if no album artist is specified", func() {
			fixAlbumArtist(&album)
			Expect(album.AlbumArtistID).To(Equal("ar-123"))
			Expect(album.AlbumArtist).To(Equal("Sparks"))
		})
		It("returns the album artist if it is specified", func() {
			album.AlbumArtist = "Sparks Brothers"
			album.AlbumArtistID = "ar-345"
			fixAlbumArtist(&album)
			Expect(album.AlbumArtistID).To(Equal("ar-345"))
			Expect(album.AlbumArtist).To(Equal("Sparks Brothers"))
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
			album.Participants.Add(RoleAlbumArtist, Artist{ID: "ar-123", Name: "Sparks"})
			album.Participants.Add(RoleAlbumArtist, Artist{ID: "ar-345", Name: "The Beach"})
			fixAlbumArtist(&album)
			Expect(album.AlbumArtistID).To(Equal(consts.VariousArtistsID))
			Expect(album.AlbumArtist).To(Equal(consts.VariousArtists))
		})

		It("returns the sole album artist if they are the same", func() {
			album.Participants.Add(RoleAlbumArtist, Artist{ID: "ar-000", Name: "The Beatles"})
			fixAlbumArtist(&album)
			Expect(album.AlbumArtistID).To(Equal("ar-000"))
			Expect(album.AlbumArtist).To(Equal("The Beatles"))
		})
	})
})
