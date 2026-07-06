package dto

import (
	"encoding/json"
	"time"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("mappers", func() {
	It("maps a song to an Audio BaseItemDto", func() {
		mf := model.MediaFile{
			ID: "song-1", Title: "Song", Album: "Alb", AlbumID: "alb-1",
			Artist: "Art", AlbumArtist: "AA", TrackNumber: 3, DiscNumber: 1,
			Year: 1999, Duration: 60, Size: 2_500_000,
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
		Expect(item.MediaSources).To(HaveLen(1))
		Expect(item.MediaSources[0].Size).To(Equal(int64(2_500_000)))
	})

	It("builds a MediaSourceInfo from a media file", func() {
		mf := model.MediaFile{ID: "s1", Size: 5242880, Suffix: "mp3", BitRate: 320, Duration: 100}
		src := MediaSourceFromMediaFile(mf)
		Expect(src.Id).To(Equal("s1"))
		Expect(src.Size).To(Equal(int64(5242880)))
		Expect(src.Container).To(Equal("mp3"))
		Expect(src.Bitrate).To(Equal(320_000))
		Expect(src.RunTimeTicks).To(Equal(int64(1_000_000_000)))
		Expect(src.Protocol).To(Equal("Http"))
		Expect(src.SupportsDirectPlay).To(BeTrue())
	})

	It("serializes all Finamp-required MediaSourceInfo bools and arrays, never as null", func() {
		mf := model.MediaFile{ID: "s1", Size: 5242880, Suffix: "mp3", BitRate: 320, Duration: 100}
		src := MediaSourceFromMediaFile(mf)
		b, err := json.Marshal(src)
		Expect(err).ToNot(HaveOccurred())
		j := string(b)
		Expect(j).To(ContainSubstring(`"SupportsProbing":true`))
		Expect(j).To(ContainSubstring(`"IsInfiniteStream":false`))
		Expect(j).To(ContainSubstring(`"RequiresOpening":false`))
		Expect(j).To(ContainSubstring(`"MediaStreams":[]`))
		Expect(j).To(ContainSubstring(`"MediaAttachments":[]`))
		Expect(j).To(ContainSubstring(`"Formats":[]`))
	})

	It("omits IndexNumber and ParentIndexNumber when track/disc numbers are untagged", func() {
		mf := model.MediaFile{
			ID: "song-2", Title: "Song", Album: "Alb", AlbumID: "alb-1",
			Artist: "Art", AlbumArtist: "AA", TrackNumber: 0, DiscNumber: 0,
			Duration: 60,
		}
		item := SongToBaseItem(mf)
		Expect(item.IndexNumber).To(BeNil())
		Expect(item.ParentIndexNumber).To(BeNil())
	})

	It("maps PlayDate to UserData.LastPlayedDate", func() {
		playDate := time.Date(2023, 5, 17, 12, 30, 0, 0, time.UTC)
		mf := model.MediaFile{
			ID: "song-3", Title: "Song", Album: "Alb", AlbumID: "alb-1",
			Artist: "Art", AlbumArtist: "AA", Duration: 60,
		}
		mf.PlayDate = &playDate
		item := SongToBaseItem(mf)
		Expect(item.UserData.LastPlayedDate).NotTo(BeNil())
		Expect(*item.UserData.LastPlayedDate).To(Equal(playDate.Format(time.RFC3339)))
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

	It("maps a playlist to a Playlist BaseItemDto", func() {
		p := model.Playlist{ID: "pl-1", Name: "Chill", SongCount: 7, Duration: 120}
		item := PlaylistToBaseItem(p)
		Expect(item.Type).To(Equal("Playlist"))
		Expect(item.IsFolder).To(BeTrue())
		Expect(item.Id).To(Equal("pl-1"))
		Expect(item.Name).To(Equal("Chill"))
		Expect(item.MediaType).To(Equal("Audio"))
		Expect(*item.ChildCount).To(Equal(7))
		Expect(item.RunTimeTicks).To(Equal(int64(1_200_000_000)))
	})
})
