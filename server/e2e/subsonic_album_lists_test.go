package e2e

import (
	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Album List Endpoints", func() {
	BeforeEach(func() {
		setupTestDB()
	})

	Describe("GetAlbumList", func() {
		It("type=newest returns albums sorted by creation date", func() {
			resp := doReq("getAlbumList", "type", "newest")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.AlbumList).ToNot(BeNil())
			Expect(resp.AlbumList.Album).To(HaveLen(6))
		})

		It("type=alphabeticalByName sorts albums by name", func() {
			resp := doReq("getAlbumList", "type", "alphabeticalByName")

			Expect(resp.AlbumList).ToNot(BeNil())
			albums := resp.AlbumList.Album
			Expect(albums).To(HaveLen(6))
			// Verify alphabetical order: Abbey Road, COWBOY BEBOP, Help!, IV, Kind of Blue, Pop
			Expect(albums[0].Title).To(Equal("Abbey Road"))
			Expect(albums[1].Title).To(Equal("COWBOY BEBOP"))
			Expect(albums[2].Title).To(Equal("Help!"))
			Expect(albums[3].Title).To(Equal("IV"))
			Expect(albums[4].Title).To(Equal("Kind of Blue"))
			Expect(albums[5].Title).To(Equal("Pop"))
		})

		It("type=alphabeticalByArtist sorts albums by artist name", func() {
			resp := doReq("getAlbumList", "type", "alphabeticalByArtist")

			Expect(resp.AlbumList).ToNot(BeNil())
			albums := resp.AlbumList.Album
			Expect(albums).To(HaveLen(6))
			// Articles like "The" are stripped for sorting, so "The Beatles" sorts as "Beatles"
			// Non-compilations first: Beatles (x2), Led Zeppelin, Miles Davis, then compilations: Various, then CJK: シートベルツ
			Expect(albums[0].Artist).To(Equal("The Beatles"))
			Expect(albums[1].Artist).To(Equal("The Beatles"))
			Expect(albums[2].Artist).To(Equal("Led Zeppelin"))
			Expect(albums[3].Artist).To(Equal("Miles Davis"))
			Expect(albums[4].Artist).To(Equal("Various"))
			Expect(albums[5].Artist).To(Equal("シートベルツ"))
		})

		It("type=random returns albums", func() {
			resp := doReq("getAlbumList", "type", "random")

			Expect(resp.AlbumList).ToNot(BeNil())
			Expect(resp.AlbumList.Album).To(HaveLen(6))
		})

		It("type=byGenre filters by genre parameter", func() {
			resp := doReq("getAlbumList", "type", "byGenre", "genre", "Jazz")

			Expect(resp.AlbumList).ToNot(BeNil())
			Expect(resp.AlbumList.Album).To(HaveLen(2))
			for _, a := range resp.AlbumList.Album {
				Expect(a.Genre).To(Equal("Jazz"))
			}
		})

		It("type=byYear filters by fromYear/toYear range", func() {
			resp := doReq("getAlbumList", "type", "byYear", "fromYear", "1965", "toYear", "1970")

			Expect(resp.AlbumList).ToNot(BeNil())
			// Should include Abbey Road (1969) and Help! (1965)
			Expect(resp.AlbumList.Album).To(HaveLen(2))
			years := make([]int32, len(resp.AlbumList.Album))
			for i, a := range resp.AlbumList.Album {
				years[i] = a.Year
			}
			Expect(years).To(ConsistOf(int32(1965), int32(1969)))
		})

		It("respects size parameter", func() {
			resp := doReq("getAlbumList", "type", "newest", "size", "2")

			Expect(resp.AlbumList).ToNot(BeNil())
			Expect(resp.AlbumList.Album).To(HaveLen(2))
		})

		It("supports offset for pagination", func() {
			// First get all albums sorted by name to know the expected order
			resp1 := doReq("getAlbumList", "type", "alphabeticalByName", "size", "5")
			allAlbums := resp1.AlbumList.Album

			// Now get with offset=2, size=2
			resp2 := doReq("getAlbumList", "type", "alphabeticalByName", "size", "2", "offset", "2")

			Expect(resp2.AlbumList).ToNot(BeNil())
			Expect(resp2.AlbumList.Album).To(HaveLen(2))
			Expect(resp2.AlbumList.Album[0].Title).To(Equal(allAlbums[2].Title))
			Expect(resp2.AlbumList.Album[1].Title).To(Equal(allAlbums[3].Title))
		})

		It("returns error when type parameter is missing", func() {
			resp := doReq("getAlbumList")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})

		It("returns error for unknown type", func() {
			resp := doReq("getAlbumList", "type", "invalid_type")

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})

		It("type=frequent returns empty when no albums have been played", func() {
			resp := doReq("getAlbumList", "type", "frequent")

			Expect(resp.AlbumList).ToNot(BeNil())
			Expect(resp.AlbumList.Album).To(BeEmpty())
		})

		It("type=recent returns empty when no albums have been played", func() {
			resp := doReq("getAlbumList", "type", "recent")

			Expect(resp.AlbumList).ToNot(BeNil())
			Expect(resp.AlbumList.Album).To(BeEmpty())
		})
	})

	Describe("GetAlbumList - starred type", Ordered, func() {
		BeforeAll(func() {
			setupTestDB()

			// Star an album so the starred filter returns results
			albums, err := ds.Album(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"album.name": "Abbey Road"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(albums).ToNot(BeEmpty())

			resp := doReq("star", "albumId", albums[0].ID)
			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		It("type=starred returns only starred albums", func() {
			resp := doReq("getAlbumList", "type", "starred")

			Expect(resp.AlbumList).ToNot(BeNil())
			Expect(resp.AlbumList.Album).To(HaveLen(1))
			Expect(resp.AlbumList.Album[0].Title).To(Equal("Abbey Road"))
		})
	})

	Describe("GetAlbumList - highest type", Ordered, func() {
		BeforeAll(func() {
			setupTestDB()

			// Rate an album so the highest filter returns results
			albums, err := ds.Album(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"album.name": "Kind of Blue"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(albums).ToNot(BeEmpty())

			resp := doReq("setRating", "id", albums[0].ID, "rating", "5")
			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		It("type=highest returns only rated albums", func() {
			resp := doReq("getAlbumList", "type", "highest")

			Expect(resp.AlbumList).ToNot(BeNil())
			Expect(resp.AlbumList.Album).To(HaveLen(1))
			Expect(resp.AlbumList.Album[0].Title).To(Equal("Kind of Blue"))
		})
	})

	Describe("GetAlbumList2", func() {
		It("returns albums in AlbumID3 format", func() {
			resp := doReq("getAlbumList2", "type", "alphabeticalByName")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.AlbumList2).ToNot(BeNil())
			albums := resp.AlbumList2.Album
			Expect(albums).To(HaveLen(6))
			// Verify AlbumID3 format fields
			Expect(albums[0].Name).To(Equal("Abbey Road"))
			Expect(albums[0].Id).ToNot(BeEmpty())
			Expect(albums[0].Artist).ToNot(BeEmpty())
		})

		It("type=newest works correctly", func() {
			resp := doReq("getAlbumList2", "type", "newest")

			Expect(resp.AlbumList2).ToNot(BeNil())
			Expect(resp.AlbumList2.Album).To(HaveLen(6))
		})
	})

	Describe("GetStarred", func() {
		It("returns empty lists when nothing is starred", func() {
			resp := doReq("getStarred")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Starred).ToNot(BeNil())
			Expect(resp.Starred.Artist).To(BeEmpty())
			Expect(resp.Starred.Album).To(BeEmpty())
			Expect(resp.Starred.Song).To(BeEmpty())
		})
	})

	Describe("GetStarred2", func() {
		It("returns empty lists when nothing is starred", func() {
			resp := doReq("getStarred2")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Starred2).ToNot(BeNil())
			Expect(resp.Starred2.Artist).To(BeEmpty())
			Expect(resp.Starred2.Album).To(BeEmpty())
			Expect(resp.Starred2.Song).To(BeEmpty())
		})
	})

	Describe("GetNowPlaying", func() {
		It("returns empty list when nobody is playing", func() {
			resp := doReq("getNowPlaying")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.NowPlaying).ToNot(BeNil())
			Expect(resp.NowPlaying.Entry).To(BeEmpty())
		})
	})

	Describe("GetRandomSongs", func() {
		It("returns random songs from library", func() {
			resp := doReq("getRandomSongs")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.RandomSongs).ToNot(BeNil())
			Expect(resp.RandomSongs.Songs).ToNot(BeEmpty())
			Expect(resp.RandomSongs.Songs).To(HaveLen(7))
		})

		It("respects size parameter", func() {
			resp := doReq("getRandomSongs", "size", "2")

			Expect(resp.RandomSongs).ToNot(BeNil())
			Expect(resp.RandomSongs.Songs).To(HaveLen(2))
		})

		It("filters by genre when specified", func() {
			resp := doReq("getRandomSongs", "size", "500", "genre", "Jazz")

			Expect(resp.RandomSongs).ToNot(BeNil())
			Expect(resp.RandomSongs.Songs).To(HaveLen(2))
			for _, s := range resp.RandomSongs.Songs {
				Expect(s.Genre).To(Equal("Jazz"))
			}
		})
	})

	Describe("GetSongsByGenre", func() {
		It("returns songs matching the genre", func() {
			resp := doReq("getSongsByGenre", "genre", "Rock")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.SongsByGenre).ToNot(BeNil())
			// 4 Rock songs: Come Together, Something, Help!, Stairway To Heaven
			Expect(resp.SongsByGenre.Songs).To(HaveLen(4))
			for _, song := range resp.SongsByGenre.Songs {
				Expect(song.Genre).To(Equal("Rock"))
			}
		})

		It("supports count and offset parameters", func() {
			// First get all Rock songs
			resp1 := doReq("getSongsByGenre", "genre", "Rock", "count", "500")
			allSongs := resp1.SongsByGenre.Songs

			// Now get with count=2, offset=1
			resp2 := doReq("getSongsByGenre", "genre", "Rock", "count", "2", "offset", "1")

			Expect(resp2.SongsByGenre).ToNot(BeNil())
			Expect(resp2.SongsByGenre.Songs).To(HaveLen(2))
			Expect(resp2.SongsByGenre.Songs[0].Id).To(Equal(allSongs[1].Id))
		})

		It("returns empty for non-existent genre", func() {
			resp := doReq("getSongsByGenre", "genre", "NonExistentGenre")

			Expect(resp.SongsByGenre).ToNot(BeNil())
			Expect(resp.SongsByGenre.Songs).To(BeEmpty())
		})
	})
})
