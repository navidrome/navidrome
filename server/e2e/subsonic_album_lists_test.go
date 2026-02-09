package e2e

import (
	"net/http/httptest"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Album List Endpoints", func() {
	BeforeEach(func() {
		setupTestDB()
	})

	Describe("GetAlbumList", func() {
		It("type=newest returns albums sorted by creation date", func() {
			w, r := newRawReq("getAlbumList", "type", "newest")
			resp, err := router.GetAlbumList(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.AlbumList).ToNot(BeNil())
			Expect(resp.AlbumList.Album).To(HaveLen(5))
		})

		It("type=alphabeticalByName sorts albums by name", func() {
			w, r := newRawReq("getAlbumList", "type", "alphabeticalByName")
			resp, err := router.GetAlbumList(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.AlbumList).ToNot(BeNil())
			albums := resp.AlbumList.Album
			Expect(albums).To(HaveLen(5))
			// Verify alphabetical order: Abbey Road, Help!, IV, Kind of Blue, Pop
			Expect(albums[0].Title).To(Equal("Abbey Road"))
			Expect(albums[1].Title).To(Equal("Help!"))
			Expect(albums[2].Title).To(Equal("IV"))
			Expect(albums[3].Title).To(Equal("Kind of Blue"))
			Expect(albums[4].Title).To(Equal("Pop"))
		})

		It("type=alphabeticalByArtist sorts albums by artist name", func() {
			w, r := newRawReq("getAlbumList", "type", "alphabeticalByArtist")
			resp, err := router.GetAlbumList(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.AlbumList).ToNot(BeNil())
			albums := resp.AlbumList.Album
			Expect(albums).To(HaveLen(5))
			// Articles like "The" are stripped for sorting, so "The Beatles" sorts as "Beatles"
			// Non-compilations first: Beatles (x2), Led Zeppelin, Miles Davis, then compilations: Various
			Expect(albums[0].Artist).To(Equal("The Beatles"))
			Expect(albums[1].Artist).To(Equal("The Beatles"))
			Expect(albums[2].Artist).To(Equal("Led Zeppelin"))
			Expect(albums[3].Artist).To(Equal("Miles Davis"))
			Expect(albums[4].Artist).To(Equal("Various"))
		})

		It("type=random returns albums", func() {
			w, r := newRawReq("getAlbumList", "type", "random")
			resp, err := router.GetAlbumList(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.AlbumList).ToNot(BeNil())
			Expect(resp.AlbumList.Album).To(HaveLen(5))
		})

		It("type=byGenre filters by genre parameter", func() {
			w, r := newRawReq("getAlbumList", "type", "byGenre", "genre", "Jazz")
			resp, err := router.GetAlbumList(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.AlbumList).ToNot(BeNil())
			Expect(resp.AlbumList.Album).To(HaveLen(1))
			Expect(resp.AlbumList.Album[0].Title).To(Equal("Kind of Blue"))
		})

		It("type=byYear filters by fromYear/toYear range", func() {
			w, r := newRawReq("getAlbumList", "type", "byYear", "fromYear", "1965", "toYear", "1970")
			resp, err := router.GetAlbumList(w, r)

			Expect(err).ToNot(HaveOccurred())
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
			w, r := newRawReq("getAlbumList", "type", "newest", "size", "2")
			resp, err := router.GetAlbumList(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.AlbumList).ToNot(BeNil())
			Expect(resp.AlbumList.Album).To(HaveLen(2))
		})

		It("supports offset for pagination", func() {
			// First get all albums sorted by name to know the expected order
			w1, r1 := newRawReq("getAlbumList", "type", "alphabeticalByName", "size", "5")
			resp1, err := router.GetAlbumList(w1, r1)
			Expect(err).ToNot(HaveOccurred())
			allAlbums := resp1.AlbumList.Album

			// Now get with offset=2, size=2
			w2, r2 := newRawReq("getAlbumList", "type", "alphabeticalByName", "size", "2", "offset", "2")
			resp2, err := router.GetAlbumList(w2, r2)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp2.AlbumList).ToNot(BeNil())
			Expect(resp2.AlbumList.Album).To(HaveLen(2))
			Expect(resp2.AlbumList.Album[0].Title).To(Equal(allAlbums[2].Title))
			Expect(resp2.AlbumList.Album[1].Title).To(Equal(allAlbums[3].Title))
		})

		It("returns error when type parameter is missing", func() {
			w := httptest.NewRecorder()
			r := newReq("getAlbumList")
			_, err := router.GetAlbumList(w, r)

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(req.ErrMissingParam))
		})

		It("returns error for unknown type", func() {
			w, r := newRawReq("getAlbumList", "type", "invalid_type")
			_, err := router.GetAlbumList(w, r)

			Expect(err).To(HaveOccurred())
		})

		It("type=frequent returns empty when no albums have been played", func() {
			w, r := newRawReq("getAlbumList", "type", "frequent")
			resp, err := router.GetAlbumList(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.AlbumList).ToNot(BeNil())
			Expect(resp.AlbumList.Album).To(BeEmpty())
		})

		It("type=recent returns empty when no albums have been played", func() {
			w, r := newRawReq("getAlbumList", "type", "recent")
			resp, err := router.GetAlbumList(w, r)

			Expect(err).ToNot(HaveOccurred())
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

			r := newReq("star", "albumId", albums[0].ID)
			_, err = router.Star(r)
			Expect(err).ToNot(HaveOccurred())
		})

		It("type=starred returns only starred albums", func() {
			w, r := newRawReq("getAlbumList", "type", "starred")
			resp, err := router.GetAlbumList(w, r)

			Expect(err).ToNot(HaveOccurred())
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

			r := newReq("setRating", "id", albums[0].ID, "rating", "5")
			_, err = router.SetRating(r)
			Expect(err).ToNot(HaveOccurred())
		})

		It("type=highest returns only rated albums", func() {
			w, r := newRawReq("getAlbumList", "type", "highest")
			resp, err := router.GetAlbumList(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.AlbumList).ToNot(BeNil())
			Expect(resp.AlbumList.Album).To(HaveLen(1))
			Expect(resp.AlbumList.Album[0].Title).To(Equal("Kind of Blue"))
		})
	})

	Describe("GetAlbumList2", func() {
		It("returns albums in AlbumID3 format", func() {
			w, r := newRawReq("getAlbumList2", "type", "alphabeticalByName")
			resp, err := router.GetAlbumList2(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.AlbumList2).ToNot(BeNil())
			albums := resp.AlbumList2.Album
			Expect(albums).To(HaveLen(5))
			// Verify AlbumID3 format fields
			Expect(albums[0].Name).To(Equal("Abbey Road"))
			Expect(albums[0].Id).ToNot(BeEmpty())
			Expect(albums[0].Artist).ToNot(BeEmpty())
		})

		It("type=newest works correctly", func() {
			w, r := newRawReq("getAlbumList2", "type", "newest")
			resp, err := router.GetAlbumList2(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.AlbumList2).ToNot(BeNil())
			Expect(resp.AlbumList2.Album).To(HaveLen(5))
		})
	})

	Describe("GetStarred", func() {
		It("returns empty lists when nothing is starred", func() {
			r := newReq("getStarred")
			resp, err := router.GetStarred(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Starred).ToNot(BeNil())
			Expect(resp.Starred.Artist).To(BeEmpty())
			Expect(resp.Starred.Album).To(BeEmpty())
			Expect(resp.Starred.Song).To(BeEmpty())
		})
	})

	Describe("GetStarred2", func() {
		It("returns empty lists when nothing is starred", func() {
			r := newReq("getStarred2")
			resp, err := router.GetStarred2(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Starred2).ToNot(BeNil())
			Expect(resp.Starred2.Artist).To(BeEmpty())
			Expect(resp.Starred2.Album).To(BeEmpty())
			Expect(resp.Starred2.Song).To(BeEmpty())
		})
	})

	Describe("GetNowPlaying", func() {
		It("returns empty list when nobody is playing", func() {
			r := newReq("getNowPlaying")
			resp, err := router.GetNowPlaying(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.NowPlaying).ToNot(BeNil())
			Expect(resp.NowPlaying.Entry).To(BeEmpty())
		})
	})

	Describe("GetRandomSongs", func() {
		It("returns random songs from library", func() {
			r := newReq("getRandomSongs")
			resp, err := router.GetRandomSongs(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.RandomSongs).ToNot(BeNil())
			Expect(resp.RandomSongs.Songs).ToNot(BeEmpty())
			Expect(len(resp.RandomSongs.Songs)).To(BeNumerically("<=", 6))
		})

		It("respects size parameter", func() {
			r := newReq("getRandomSongs", "size", "2")
			resp, err := router.GetRandomSongs(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.RandomSongs).ToNot(BeNil())
			Expect(resp.RandomSongs.Songs).To(HaveLen(2))
		})

		It("filters by genre when specified", func() {
			r := newReq("getRandomSongs", "size", "500", "genre", "Jazz")
			resp, err := router.GetRandomSongs(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.RandomSongs).ToNot(BeNil())
			Expect(resp.RandomSongs.Songs).To(HaveLen(1))
			Expect(resp.RandomSongs.Songs[0].Genre).To(Equal("Jazz"))
		})
	})

	Describe("GetSongsByGenre", func() {
		It("returns songs matching the genre", func() {
			r := newReq("getSongsByGenre", "genre", "Rock")
			resp, err := router.GetSongsByGenre(r)

			Expect(err).ToNot(HaveOccurred())
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
			r1 := newReq("getSongsByGenre", "genre", "Rock", "count", "500")
			resp1, err := router.GetSongsByGenre(r1)
			Expect(err).ToNot(HaveOccurred())
			allSongs := resp1.SongsByGenre.Songs

			// Now get with count=2, offset=1
			r2 := newReq("getSongsByGenre", "genre", "Rock", "count", "2", "offset", "1")
			resp2, err := router.GetSongsByGenre(r2)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp2.SongsByGenre).ToNot(BeNil())
			Expect(resp2.SongsByGenre.Songs).To(HaveLen(2))
			Expect(resp2.SongsByGenre.Songs[0].Id).To(Equal(allSongs[1].Id))
		})

		It("returns empty for non-existent genre", func() {
			r := newReq("getSongsByGenre", "genre", "NonExistentGenre")
			resp, err := router.GetSongsByGenre(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.SongsByGenre).ToNot(BeNil())
			Expect(resp.SongsByGenre.Songs).To(BeEmpty())
		})
	})
})
