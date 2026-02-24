package e2e

import (
	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Browsing Endpoints", func() {
	BeforeEach(func() {
		setupTestDB()
	})

	Describe("getMusicFolders", func() {
		It("returns the configured music library", func() {
			resp := doReq("getMusicFolders")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.MusicFolders).ToNot(BeNil())
			Expect(resp.MusicFolders.Folders).To(HaveLen(1))
			Expect(resp.MusicFolders.Folders[0].Name).To(Equal("Music Library"))
			Expect(resp.MusicFolders.Folders[0].Id).To(Equal(int32(lib.ID)))
		})
	})

	Describe("getIndexes", func() {
		It("returns artist indexes", func() {
			resp := doReq("getIndexes")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Indexes).ToNot(BeNil())
			Expect(resp.Indexes.Index).ToNot(BeEmpty())
		})

		It("includes all artists across indexes", func() {
			resp := doReq("getIndexes")

			var allArtistNames []string
			for _, idx := range resp.Indexes.Index {
				for _, a := range idx.Artists {
					allArtistNames = append(allArtistNames, a.Name)
				}
			}
			Expect(allArtistNames).To(ContainElements("The Beatles", "Led Zeppelin", "Miles Davis", "Various"))
		})
	})

	Describe("getArtists", func() {
		It("returns artist indexes in ID3 format", func() {
			resp := doReq("getArtists")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Artist).ToNot(BeNil())
			Expect(resp.Artist.Index).ToNot(BeEmpty())
		})

		It("includes all artists across ID3 indexes", func() {
			resp := doReq("getArtists")

			var allArtistNames []string
			for _, idx := range resp.Artist.Index {
				for _, a := range idx.Artists {
					allArtistNames = append(allArtistNames, a.Name)
				}
			}
			Expect(allArtistNames).To(ContainElements("The Beatles", "Led Zeppelin", "Miles Davis", "Various"))
		})

		It("reports correct album counts for artists", func() {
			resp := doReq("getArtists")

			var beatlesAlbumCount int32
			for _, idx := range resp.Artist.Index {
				for _, a := range idx.Artists {
					if a.Name == "The Beatles" {
						beatlesAlbumCount = a.AlbumCount
					}
				}
			}
			Expect(beatlesAlbumCount).To(Equal(int32(2)))
		})
	})

	Describe("getMusicDirectory", func() {
		It("returns an artist directory with its albums as children", func() {
			artists, err := ds.Artist(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"name": "The Beatles"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(artists).ToNot(BeEmpty())
			beatlesID := artists[0].ID

			resp := doReq("getMusicDirectory", "id", beatlesID)

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Directory).ToNot(BeNil())
			Expect(resp.Directory.Name).To(Equal("The Beatles"))
			Expect(resp.Directory.Child).To(HaveLen(2)) // Abbey Road, Help!
		})

		It("returns an album directory with its tracks as children", func() {
			albums, err := ds.Album(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"album.name": "Abbey Road"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(albums).ToNot(BeEmpty())
			abbeyRoadID := albums[0].ID

			resp := doReq("getMusicDirectory", "id", abbeyRoadID)

			Expect(resp.Directory).ToNot(BeNil())
			Expect(resp.Directory.Name).To(Equal("Abbey Road"))
			Expect(resp.Directory.Child).To(HaveLen(2)) // Come Together, Something
		})

		It("returns an error for a non-existent ID", func() {
			resp := doReq("getMusicDirectory", "id", "non-existent-id")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})
	})

	Describe("getArtist", func() {
		It("returns artist with albums in ID3 format", func() {
			artists, err := ds.Artist(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"name": "The Beatles"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(artists).ToNot(BeEmpty())
			beatlesID := artists[0].ID

			resp := doReq("getArtist", "id", beatlesID)

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.ArtistWithAlbumsID3).ToNot(BeNil())
			Expect(resp.ArtistWithAlbumsID3.Name).To(Equal("The Beatles"))
			Expect(resp.ArtistWithAlbumsID3.Album).To(HaveLen(2))
		})

		It("returns album names for the artist", func() {
			artists, err := ds.Artist(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"name": "The Beatles"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(artists).ToNot(BeEmpty())
			beatlesID := artists[0].ID

			resp := doReq("getArtist", "id", beatlesID)

			var albumNames []string
			for _, a := range resp.ArtistWithAlbumsID3.Album {
				albumNames = append(albumNames, a.Name)
			}
			Expect(albumNames).To(ContainElements("Abbey Road", "Help!"))
		})

		It("returns an error for a non-existent artist", func() {
			resp := doReq("getArtist", "id", "non-existent-id")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})

		It("returns artist with a single album", func() {
			artists, err := ds.Artist(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"name": "Led Zeppelin"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(artists).ToNot(BeEmpty())
			ledZepID := artists[0].ID

			resp := doReq("getArtist", "id", ledZepID)

			Expect(resp.ArtistWithAlbumsID3).ToNot(BeNil())
			Expect(resp.ArtistWithAlbumsID3.Name).To(Equal("Led Zeppelin"))
			Expect(resp.ArtistWithAlbumsID3.Album).To(HaveLen(1))
			Expect(resp.ArtistWithAlbumsID3.Album[0].Name).To(Equal("IV"))
		})
	})

	Describe("getAlbum", func() {
		It("returns album with its tracks", func() {
			albums, err := ds.Album(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"album.name": "Abbey Road"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(albums).ToNot(BeEmpty())
			abbeyRoadID := albums[0].ID

			resp := doReq("getAlbum", "id", abbeyRoadID)

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.AlbumWithSongsID3).ToNot(BeNil())
			Expect(resp.AlbumWithSongsID3.Name).To(Equal("Abbey Road"))
			Expect(resp.AlbumWithSongsID3.Song).To(HaveLen(2))
		})

		It("includes correct track metadata", func() {
			albums, err := ds.Album(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"album.name": "Abbey Road"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(albums).ToNot(BeEmpty())
			abbeyRoadID := albums[0].ID

			resp := doReq("getAlbum", "id", abbeyRoadID)

			var trackTitles []string
			for _, s := range resp.AlbumWithSongsID3.Song {
				trackTitles = append(trackTitles, s.Title)
			}
			Expect(trackTitles).To(ContainElements("Come Together", "Something"))
		})

		It("returns album with correct artist and year", func() {
			albums, err := ds.Album(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"album.name": "Kind of Blue"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(albums).ToNot(BeEmpty())
			kindOfBlueID := albums[0].ID

			resp := doReq("getAlbum", "id", kindOfBlueID)

			Expect(resp.AlbumWithSongsID3).ToNot(BeNil())
			Expect(resp.AlbumWithSongsID3.Name).To(Equal("Kind of Blue"))
			Expect(resp.AlbumWithSongsID3.Artist).To(Equal("Miles Davis"))
			Expect(resp.AlbumWithSongsID3.Year).To(Equal(int32(1959)))
			Expect(resp.AlbumWithSongsID3.Song).To(HaveLen(1))
		})

		It("returns an error for a non-existent album", func() {
			resp := doReq("getAlbum", "id", "non-existent-id")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})
	})

	Describe("getSong", func() {
		It("returns a song by its ID", func() {
			songs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"title": "Come Together"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(songs).ToNot(BeEmpty())
			songID := songs[0].ID

			resp := doReq("getSong", "id", songID)

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Song).ToNot(BeNil())
			Expect(resp.Song.Title).To(Equal("Come Together"))
			Expect(resp.Song.Album).To(Equal("Abbey Road"))
			Expect(resp.Song.Artist).To(Equal("The Beatles"))
		})

		It("returns an error for a non-existent song", func() {
			resp := doReq("getSong", "id", "non-existent-id")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})

		It("returns correct metadata for a jazz track", func() {
			songs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"title": "So What"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(songs).ToNot(BeEmpty())
			songID := songs[0].ID

			resp := doReq("getSong", "id", songID)

			Expect(resp.Song).ToNot(BeNil())
			Expect(resp.Song.Title).To(Equal("So What"))
			Expect(resp.Song.Album).To(Equal("Kind of Blue"))
			Expect(resp.Song.Artist).To(Equal("Miles Davis"))
		})
	})

	Describe("getGenres", func() {
		It("returns all genres", func() {
			resp := doReq("getGenres")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Genres).ToNot(BeNil())
			Expect(resp.Genres.Genre).To(HaveLen(3))
		})

		It("includes correct genre names", func() {
			resp := doReq("getGenres")

			var genreNames []string
			for _, g := range resp.Genres.Genre {
				genreNames = append(genreNames, g.Name)
			}
			Expect(genreNames).To(ContainElements("Rock", "Jazz", "Pop"))
		})

		It("reports correct song and album counts for Rock", func() {
			resp := doReq("getGenres")

			var rockGenre *responses.Genre
			for i, g := range resp.Genres.Genre {
				if g.Name == "Rock" {
					rockGenre = &resp.Genres.Genre[i]
					break
				}
			}
			Expect(rockGenre).ToNot(BeNil())
			Expect(rockGenre.SongCount).To(Equal(int32(4)))
			Expect(rockGenre.AlbumCount).To(Equal(int32(3)))
		})

		It("reports correct song and album counts for Jazz", func() {
			resp := doReq("getGenres")

			var jazzGenre *responses.Genre
			for i, g := range resp.Genres.Genre {
				if g.Name == "Jazz" {
					jazzGenre = &resp.Genres.Genre[i]
					break
				}
			}
			Expect(jazzGenre).ToNot(BeNil())
			Expect(jazzGenre.SongCount).To(Equal(int32(2)))
			Expect(jazzGenre.AlbumCount).To(Equal(int32(2)))
		})

		It("reports correct song and album counts for Pop", func() {
			resp := doReq("getGenres")

			var popGenre *responses.Genre
			for i, g := range resp.Genres.Genre {
				if g.Name == "Pop" {
					popGenre = &resp.Genres.Genre[i]
					break
				}
			}
			Expect(popGenre).ToNot(BeNil())
			Expect(popGenre.SongCount).To(Equal(int32(1)))
			Expect(popGenre.AlbumCount).To(Equal(int32(1)))
		})
	})

	Describe("getAlbumInfo", func() {
		It("returns album info for a valid album", func() {
			albums, err := ds.Album(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"album.name": "Abbey Road"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(albums).ToNot(BeEmpty())
			abbeyRoadID := albums[0].ID

			resp := doReq("getAlbumInfo", "id", abbeyRoadID)

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.AlbumInfo).ToNot(BeNil())
		})
	})

	Describe("getAlbumInfo2", func() {
		It("returns album info for a valid album", func() {
			albums, err := ds.Album(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"album.name": "Abbey Road"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(albums).ToNot(BeEmpty())
			abbeyRoadID := albums[0].ID

			resp := doReq("getAlbumInfo2", "id", abbeyRoadID)

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.AlbumInfo).ToNot(BeNil())
		})
	})

	Describe("getArtistInfo", func() {
		It("returns artist info for a valid artist", func() {
			artists, err := ds.Artist(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"name": "The Beatles"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(artists).ToNot(BeEmpty())
			beatlesID := artists[0].ID

			resp := doReq("getArtistInfo", "id", beatlesID)

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.ArtistInfo).ToNot(BeNil())
		})
	})

	Describe("getArtistInfo2", func() {
		It("returns artist info2 for a valid artist", func() {
			artists, err := ds.Artist(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"name": "The Beatles"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(artists).ToNot(BeEmpty())
			beatlesID := artists[0].ID

			resp := doReq("getArtistInfo2", "id", beatlesID)

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.ArtistInfo2).ToNot(BeNil())
		})
	})

	Describe("getTopSongs", func() {
		It("returns a response for a known artist name", func() {
			resp := doReq("getTopSongs", "artist", "The Beatles")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.TopSongs).ToNot(BeNil())
			// noopProvider returns empty list, so Songs may be empty
		})

		It("returns an empty list for an unknown artist", func() {
			resp := doReq("getTopSongs", "artist", "Unknown Artist")

			Expect(resp.TopSongs).ToNot(BeNil())
			Expect(resp.TopSongs.Song).To(BeEmpty())
		})
	})

	Describe("getSimilarSongs", func() {
		It("returns a response for a valid song ID", func() {
			songs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"title": "Come Together"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(songs).ToNot(BeEmpty())
			songID := songs[0].ID

			resp := doReq("getSimilarSongs", "id", songID)

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.SimilarSongs).ToNot(BeNil())
			// noopProvider returns empty list
		})
	})

	Describe("getSimilarSongs2", func() {
		It("returns a response for a valid song ID", func() {
			songs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"title": "Come Together"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(songs).ToNot(BeEmpty())
			songID := songs[0].ID

			resp := doReq("getSimilarSongs2", "id", songID)

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.SimilarSongs2).ToNot(BeNil())
			// noopProvider returns empty list
		})
	})
})
