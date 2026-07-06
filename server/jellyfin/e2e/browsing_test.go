package e2e

import (
	"net/http"
	"time"

	"github.com/navidrome/navidrome/server/jellyfin/dto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func names(items []dto.BaseItemDto) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Name
	}
	return out
}

var _ = Describe("Browsing", func() {
	BeforeEach(func() { setupTestDB() })

	Describe("GET /UserViews", func() {
		It("returns the user's libraries as CollectionFolders", func() {
			q := queryResult(get("/UserViews"))
			Expect(q.TotalRecordCount).To(Equal(1))
			Expect(q.Items[0].Name).To(Equal("Music Library"))
			Expect(q.Items[0].Type).To(Equal("CollectionFolder"))
			Expect(q.Items[0].CollectionType).To(Equal("music"))
		})
	})

	Describe("GET /Items by type", func() {
		It("lists all albums", func() {
			q := queryResult(get("/Items?IncludeItemTypes=MusicAlbum&Recursive=true"))
			Expect(q.TotalRecordCount).To(Equal(5))
			Expect(names(q.Items)).To(ConsistOf("Abbey Road", "Help!", "IV", "Kind of Blue", "Singles"))
		})

		It("lists all songs with Audio type and an AlbumId", func() {
			q := queryResult(get("/Items?IncludeItemTypes=Audio&Recursive=true"))
			Expect(q.TotalRecordCount).To(Equal(6))
			for _, it := range q.Items {
				Expect(it.Type).To(Equal("Audio"))
				Expect(it.MediaType).To(Equal("Audio"))
				Expect(it.AlbumId).ToNot(BeEmpty())
			}
		})

		It("lists all album artists", func() {
			q := queryResult(get("/Items?IncludeItemTypes=MusicArtist&Recursive=true"))
			Expect(q.TotalRecordCount).To(Equal(4))
			Expect(names(q.Items)).To(ConsistOf("The Beatles", "Led Zeppelin", "Miles Davis", "Solo Artist"))
		})

		It("lists all genres", func() {
			q := queryResult(get("/Items?IncludeItemTypes=MusicGenre&Recursive=true"))
			Expect(q.TotalRecordCount).To(Equal(3))
			Expect(names(q.Items)).To(ConsistOf("Rock", "Jazz", "Pop"))
		})

		It("returns no playlists when none exist", func() {
			q := queryResult(get("/Items?IncludeItemTypes=Playlist&Recursive=true"))
			Expect(q.TotalRecordCount).To(Equal(0))
			Expect(q.Items).To(BeEmpty())
		})

		It("defaults to albums when IncludeItemTypes is unrecognized", func() {
			q := queryResult(get("/Items?IncludeItemTypes=Nonsense&Recursive=true"))
			Expect(q.TotalRecordCount).To(Equal(5))
		})
	})

	Describe("ParentId browsing", func() {
		It("browses an artist's albums", func() {
			q := queryResult(get("/Items?IncludeItemTypes=MusicAlbum&ParentId=" + enc(artistID("The Beatles"))))
			Expect(names(q.Items)).To(ConsistOf("Abbey Road", "Help!"))
		})

		It("browses an album's tracks in track order by default", func() {
			q := queryResult(get("/Items?IncludeItemTypes=Audio&ParentId=" + enc(albumID("Abbey Road"))))
			Expect(q.TotalRecordCount).To(Equal(2))
			// Track order (Something=1, Come Together=2) differs from alphabetical title order,
			// proving the sort is by track number, not name.
			Expect(names(q.Items)).To(Equal([]string{"Something", "Come Together"}))
			Expect(*q.Items[0].IndexNumber).To(Equal(1))
			Expect(*q.Items[1].IndexNumber).To(Equal(2))
		})

		It("respects Finamp's explicit ParentIndexNumber/IndexNumber SortBy on an album", func() {
			q := queryResult(get("/Items?IncludeItemTypes=Audio&ParentId=" + enc(albumID("Abbey Road")) + "&SortBy=ParentIndexNumber,IndexNumber,SortName"))
			Expect(names(q.Items)).To(Equal([]string{"Something", "Come Together"}))
		})
	})

	// Finamp's artist screen sends ParentId=<libraryId> (scoping) plus AlbumArtistIds/ArtistIds
	// for the actual artist filter, not ParentId=<artistId>.
	Describe("artist filtering (AlbumArtistIds / ArtistIds)", func() {
		lib1 := enc("1")

		It("filters albums by AlbumArtistIds", func() {
			q := queryResult(get("/Items?IncludeItemTypes=MusicAlbum&Recursive=true&ParentId=" + lib1 + "&AlbumArtistIds=" + enc(artistID("The Beatles"))))
			Expect(names(q.Items)).To(ConsistOf("Abbey Road", "Help!"))
		})

		It("filters songs by ArtistIds", func() {
			q := queryResult(get("/Items?IncludeItemTypes=Audio&Recursive=true&ParentId=" + lib1 + "&ArtistIds=" + enc(artistID("The Beatles"))))
			Expect(names(q.Items)).To(ConsistOf("Come Together", "Something", "Help!"))
		})

		It("filters albums by a single-album artist", func() {
			q := queryResult(get("/Items?IncludeItemTypes=MusicAlbum&Recursive=true&AlbumArtistIds=" + enc(artistID("Led Zeppelin"))))
			Expect(names(q.Items)).To(ConsistOf("IV"))
		})

		It("filters songs by a single-track artist", func() {
			q := queryResult(get("/Items?IncludeItemTypes=Audio&Recursive=true&ArtistIds=" + enc(artistID("Miles Davis"))))
			Expect(names(q.Items)).To(ConsistOf("So What"))
		})
	})

	Describe("search, batch and pagination", func() {
		It("searches albums by term", func() {
			q := queryResult(get("/Items?IncludeItemTypes=MusicAlbum&Recursive=true&SearchTerm=Abbey"))
			Expect(names(q.Items)).To(ContainElement("Abbey Road"))
		})

		It("batch-fetches specific items by Ids", func() {
			ids := enc(albumID("Abbey Road")) + "," + enc(albumID("IV"))
			q := queryResult(get("/Items?ids=" + ids))
			Expect(q.TotalRecordCount).To(Equal(2))
			Expect(names(q.Items)).To(ConsistOf("Abbey Road", "IV"))
		})

		It("applies Limit while reporting the full TotalRecordCount", func() {
			q := queryResult(get("/Items?IncludeItemTypes=MusicAlbum&Recursive=true&Limit=2"))
			Expect(q.Items).To(HaveLen(2))
			Expect(q.TotalRecordCount).To(Equal(5))
		})

		It("pages distinct items via StartIndex", func() {
			p1 := queryResult(get("/Items?IncludeItemTypes=MusicAlbum&Recursive=true&SortBy=SortName&Limit=2&StartIndex=0"))
			p2 := queryResult(get("/Items?IncludeItemTypes=MusicAlbum&Recursive=true&SortBy=SortName&Limit=2&StartIndex=2"))
			Expect(p1.Items).To(HaveLen(2))
			Expect(p2.Items).To(HaveLen(2))
			Expect(names(p1.Items)).ToNot(ContainElement(BeElementOf(names(p2.Items))))
		})

		It("merges multiple types into one paginated result", func() {
			q := queryResult(get("/Items?IncludeItemTypes=MusicAlbum,Audio&Recursive=true"))
			Expect(q.TotalRecordCount).To(Equal(11)) // 5 albums + 6 songs
		})
	})

	Describe("GET /Items/{id}", func() {
		It("resolves an album", func() {
			var item dto.BaseItemDto
			parseInto(get("/Items/"+enc(albumID("Kind of Blue"))), &item)
			Expect(item.Name).To(Equal("Kind of Blue"))
			Expect(item.Type).To(Equal("MusicAlbum"))
		})

		It("resolves a song", func() {
			var item dto.BaseItemDto
			parseInto(get("/Items/"+enc(songID("So What"))), &item)
			Expect(item.Type).To(Equal("Audio"))
		})

		It("includes a parseable DateCreated (Date Added) on a song", func() {
			var item dto.BaseItemDto
			parseInto(get("/Items/"+enc(songID("So What"))), &item)
			Expect(item.DateCreated).ToNot(BeEmpty())
			_, err := time.Parse(time.RFC3339, item.DateCreated)
			Expect(err).ToNot(HaveOccurred())
		})

		It("resolves an artist", func() {
			var item dto.BaseItemDto
			parseInto(get("/Items/"+enc(artistID("Miles Davis"))), &item)
			Expect(item.Type).To(Equal("MusicArtist"))
		})

		It("returns 404 for an unknown id", func() {
			Expect(get("/Items/" + enc("does-not-exist")).Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("GET /Users/{userId}/Items/Latest", func() {
		It("returns recent albums as a bare array, respecting Limit", func() {
			var items []dto.BaseItemDto
			parseInto(get("/Users/admin-1/Items/Latest?Limit=3"), &items)
			Expect(items).To(HaveLen(3))
			for _, it := range items {
				Expect(it.Type).To(Equal("MusicAlbum"))
			}
		})
	})

	Describe("GET /Artists and /Genres", func() {
		It("lists artists", func() {
			q := queryResult(get("/Artists"))
			Expect(q.TotalRecordCount).To(Equal(4))
		})

		It("lists genres", func() {
			q := queryResult(get("/Genres"))
			Expect(names(q.Items)).To(ConsistOf("Rock", "Jazz", "Pop"))
		})
	})
})
