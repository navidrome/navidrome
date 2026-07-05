package dto

import (
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("mappers", func() {
	It("maps a song to an Audio BaseItemDto", func() {
		mf := model.MediaFile{
			ID: "song-1", Title: "Song", Album: "Alb", AlbumID: "alb-1",
			Artist: "Art", AlbumArtist: "AA", TrackNumber: 3, DiscNumber: 1,
			Year: 1999, Duration: 60,
		}
		mf.PlayCount = 2
		mf.Starred = true
		item := SongToBaseItem(mf)
		Expect(item.Type).To(Equal("Audio"))
		Expect(item.IsFolder).To(BeFalse())
		Expect(item.Id).To(Equal("song-1"))
		Expect(item.AlbumId).To(Equal("alb-1"))
		Expect(item.ParentId).To(Equal("alb-1"))
		Expect(item.RunTimeTicks).To(Equal(int64(600_000_000)))
		Expect(*item.IndexNumber).To(Equal(3))
		Expect(item.UserData.IsFavorite).To(BeTrue())
		Expect(item.UserData.PlayCount).To(Equal(2))
		Expect(item.UserData.Played).To(BeTrue())
	})

	It("maps an album to a MusicAlbum folder item", func() {
		al := model.Album{ID: "alb-1", Name: "Alb", AlbumArtist: "AA", AlbumArtistID: "art-1", MaxYear: 1999, SongCount: 10}
		item := AlbumToBaseItem(al)
		Expect(item.Type).To(Equal("MusicAlbum"))
		Expect(item.IsFolder).To(BeTrue())
		Expect(item.ParentId).To(Equal("art-1"))
		Expect(*item.ProductionYear).To(Equal(1999))
		Expect(*item.ChildCount).To(Equal(10))
	})

	It("maps an artist to a MusicArtist folder item", func() {
		ar := model.Artist{ID: "art-1", Name: "AA", AlbumCount: 2, SongCount: 20}
		item := ArtistToBaseItem(ar)
		Expect(item.Type).To(Equal("MusicArtist"))
		Expect(item.IsFolder).To(BeTrue())
		Expect(*item.AlbumCount).To(Equal(2))
	})

	It("maps a genre to a MusicGenre folder item", func() {
		g := model.Genre{ID: "genre-1", Name: "Rock"}
		item := GenreToBaseItem(g)
		Expect(item.Type).To(Equal("MusicGenre"))
		Expect(item.IsFolder).To(BeTrue())
		Expect(item.Id).To(Equal("genre-1"))
		Expect(item.Name).To(Equal("Rock"))
	})
})
