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
			Expect(q.TotalRecordCount).To(Equal(7))
			for _, it := range q.Items {
				Expect(it.Type).To(Equal("Audio"))
				Expect(it.MediaType).To(Equal("Audio"))
				Expect(it.LocationType).To(Equal("FileSystem"))
				Expect(it.ServerId).ToNot(BeEmpty()) // real Jellyfin always sets it
				Expect(it.AlbumId).ToNot(BeEmpty())
			}
		})

		// Real Jellyfin omits MediaSources from a plain list response, returning it only when the
		// client asks via Fields=MediaSources (Finamp's download dialog does).
		It("omits MediaSources unless Fields=MediaSources is requested", func() {
			plain := queryResult(get("/Items?IncludeItemTypes=Audio&Recursive=true"))
			for _, it := range plain.Items {
				Expect(it.MediaSources).To(BeEmpty())
			}
			withSources := queryResult(get("/Items?IncludeItemTypes=Audio&Recursive=true&Fields=MediaSources"))
			for _, it := range withSources.Items {
				Expect(it.MediaSources).To(HaveLen(1))
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

		// "Latest Releases": if PremiereDate isn't recognized, applySort falls through to album-name order.
		It("sorts an artist's tracks by release year for SortBy=PremiereDate (Latest Releases)", func() {
			q := queryResult(get("/Items?IncludeItemTypes=Audio&Recursive=true&AlbumArtistIds=" + enc(artistID("The Beatles")) +
				"&SortBy=PremiereDate%2CAlbum%2CParentIndexNumber%2CIndexNumber%2CSortName&SortOrder=Descending"))
			got := names(q.Items)
			Expect(got).To(HaveLen(3))
			Expect(got[:2]).To(ConsistOf("Come Together", "Something"))
			Expect(got[2]).To(Equal("Help!"))
		})

		It("respects Finamp's explicit ParentIndexNumber/IndexNumber SortBy on an album", func() {
			q := queryResult(get("/Items?IncludeItemTypes=Audio&ParentId=" + enc(albumID("Abbey Road")) + "&SortBy=ParentIndexNumber,IndexNumber,SortName"))
			Expect(names(q.Items)).To(Equal([]string{"Something", "Come Together"}))
		})
	})

	// Finamp's download sync asks a library for the tracks outside any album this way; answering
	// with every track would stream the whole library.
	Describe("Recursive=false", func() {
		lib1 := enc("1")

		It("returns no songs for a library parent", func() {
			q := queryResult(get("/Items?IncludeItemTypes=Audio&ParentId=" + lib1 + "&Recursive=false"))
			Expect(q.Items).To(BeEmpty())
			Expect(q.TotalRecordCount).To(BeZero())
		})

		It("still lists the library's albums", func() {
			q := queryResult(get("/Items?IncludeItemTypes=MusicAlbum&ParentId=" + lib1 + "&Recursive=false"))
			Expect(names(q.Items)).To(ConsistOf("Abbey Road", "Help!", "IV", "Kind of Blue", "Singles"))
		})

		It("still lists an album's tracks", func() {
			q := queryResult(get("/Items?IncludeItemTypes=Audio&ParentId=" + enc(albumID("Abbey Road")) + "&Recursive=false"))
			Expect(names(q.Items)).To(ConsistOf("Come Together", "Something"))
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

		// contributingArtistIds is Jellify's "Featured On" section: albums the artist only appears
		// on, which must exclude their own discography (albums where they are the album artist).
		It("lists Featured On albums (contributingArtistIds) a performer only guests on", func() {
			q := queryResult(get("/Items?IncludeItemTypes=MusicAlbum&Recursive=true&contributingArtistIds=" + enc(artistID("Featured Guest"))))
			Expect(names(q.Items)).To(ConsistOf("Singles"))
		})

		It("excludes an album artist's own discography from Featured On (contributingArtistIds)", func() {
			q := queryResult(get("/Items?IncludeItemTypes=MusicAlbum&Recursive=true&contributingArtistIds=" + enc(artistID("The Beatles"))))
			Expect(names(q.Items)).ToNot(ContainElement("Abbey Road"))
			Expect(names(q.Items)).ToNot(ContainElement("Help!"))
		})
	})

	// Feishin fetches an album's tracks with AlbumIds=<albumId>&IncludeItemTypes=Audio&Recursive=true.
	Describe("album filtering (AlbumIds)", func() {
		It("filters songs by AlbumIds", func() {
			q := queryResult(get("/Items?IncludeItemTypes=Audio&Recursive=true&AlbumIds=" + enc(albumID("Abbey Road"))))
			Expect(names(q.Items)).To(ConsistOf("Come Together", "Something"))
			Expect(q.TotalRecordCount).To(Equal(2))
		})

		It("matches any of multiple comma-separated AlbumIds", func() {
			q := queryResult(get("/Items?IncludeItemTypes=Audio&Recursive=true&AlbumIds=" + enc(albumID("Abbey Road")) + "," + enc(albumID("IV"))))
			Expect(names(q.Items)).To(ConsistOf("Come Together", "Something", "Stairway To Heaven"))
			Expect(q.TotalRecordCount).To(Equal(3))
		})

		It("returns nothing for an unknown album id", func() {
			q := queryResult(get("/Items?IncludeItemTypes=Audio&Recursive=true&AlbumIds=" + enc("no-such-album")))
			Expect(q.Items).To(BeEmpty())
			Expect(q.TotalRecordCount).To(Equal(0))
		})
	})

	// Finamp's genre screen sends ParentId=<libraryId> (scoping) plus GenreIds=<genreId>.
	Describe("genre filtering (GenreIds)", func() {
		lib1 := enc("1")

		It("filters albums by GenreIds", func() {
			q := queryResult(get("/Items?IncludeItemTypes=MusicAlbum&Recursive=true&ParentId=" + lib1 + "&GenreIds=" + enc(genreID("Jazz"))))
			Expect(names(q.Items)).To(ConsistOf("Kind of Blue"))
			Expect(q.TotalRecordCount).To(Equal(1))
		})

		It("filters songs by GenreIds", func() {
			q := queryResult(get("/Items?IncludeItemTypes=Audio&Recursive=true&ParentId=" + lib1 + "&GenreIds=" + enc(genreID("Rock"))))
			Expect(names(q.Items)).To(ConsistOf("Come Together", "Something", "Help!", "Stairway To Heaven"))
			Expect(q.TotalRecordCount).To(Equal(4))
		})

		It("matches any of multiple comma-separated GenreIds", func() {
			q := queryResult(get("/Items?IncludeItemTypes=MusicAlbum&Recursive=true&GenreIds=" + enc(genreID("Jazz")) + "," + enc(genreID("Pop"))))
			Expect(names(q.Items)).To(ConsistOf("Kind of Blue", "Singles"))
		})

		It("matches any of multiple repeated GenreIds params (@jellyfin/sdk spelling)", func() {
			q := queryResult(get("/Items?IncludeItemTypes=MusicAlbum&Recursive=true&GenreIds=" + enc(genreID("Jazz")) + "&GenreIds=" + enc(genreID("Pop"))))
			Expect(names(q.Items)).To(ConsistOf("Kind of Blue", "Singles"))
		})

		It("returns nothing for an unknown genre id", func() {
			q := queryResult(get("/Items?IncludeItemTypes=MusicAlbum&Recursive=true&GenreIds=" + enc("no-such-genre")))
			Expect(q.Items).To(BeEmpty())
			Expect(q.TotalRecordCount).To(Equal(0))
		})

		It("filters album artists by GenreIds on /Artists/AlbumArtists", func() {
			q := queryResult(get("/Artists/AlbumArtists?ParentId=" + lib1 + "&GenreIds=" + enc(genreID("Jazz"))))
			Expect(names(q.Items)).To(ConsistOf("Miles Davis"))
			Expect(q.TotalRecordCount).To(Equal(1))
		})

		It("matches album artists of any of multiple GenreIds", func() {
			q := queryResult(get("/Artists/AlbumArtists?GenreIds=" + enc(genreID("Jazz")) + "," + enc(genreID("Pop"))))
			Expect(names(q.Items)).To(ConsistOf("Miles Davis", "Solo Artist"))
		})

		It("filters album artists by GenreIds via /Items?IncludeItemTypes=MusicArtist", func() {
			q := queryResult(get("/Items?IncludeItemTypes=MusicArtist&Recursive=true&GenreIds=" + enc(genreID("Rock"))))
			Expect(names(q.Items)).To(ConsistOf("The Beatles", "Led Zeppelin"))
		})

		It("returns no artists for an unknown genre id", func() {
			q := queryResult(get("/Artists/AlbumArtists?GenreIds=" + enc("no-such-genre")))
			Expect(q.Items).To(BeEmpty())
		})
	})

	// Jellify (and the official Jellyfin TypeScript SDK) send query params in camelCase
	// (parentId, includeItemTypes, albumArtistIds), where Finamp sends PascalCase. Real Jellyfin
	// binds them case-insensitively; these guard that our dispatcher does too, and that browsing an
	// album with only parentId (no IncludeItemTypes, as Jellify does) returns its tracks.
	Describe("camelCase query params (Jellify / JS SDK)", func() {
		lib1 := enc("1")

		It("filters albums by camelCase albumArtistIds", func() {
			q := queryResult(get("/Items?includeItemTypes=MusicAlbum&recursive=true&parentId=" + lib1 + "&albumArtistIds=" + enc(artistID("The Beatles"))))
			Expect(names(q.Items)).To(ConsistOf("Abbey Road", "Help!"))
		})

		It("filters songs by camelCase artistIds", func() {
			q := queryResult(get("/Items?includeItemTypes=Audio&recursive=true&parentId=" + lib1 + "&artistIds=" + enc(artistID("The Beatles"))))
			Expect(names(q.Items)).To(ConsistOf("Come Together", "Something", "Help!"))
		})

		It("browses an album's tracks with only camelCase parentId (no IncludeItemTypes)", func() {
			q := queryResult(get("/Items?parentId=" + enc(albumID("Abbey Road")) + "&sortBy=ParentIndexNumber&sortBy=IndexNumber&sortBy=SortName"))
			Expect(q.TotalRecordCount).To(Equal(2))
			Expect(names(q.Items)).To(Equal([]string{"Something", "Come Together"}))
		})

		It("browses an artist's albums with only camelCase parentId (no IncludeItemTypes)", func() {
			q := queryResult(get("/Items?parentId=" + enc(artistID("The Beatles"))))
			Expect(names(q.Items)).To(ConsistOf("Abbey Road", "Help!"))
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

		// Finamp restores its saved queue with ids truncated to 16 bytes (see README).
		Describe("Finamp-truncated ids (saved queue restore)", func() {
			It("resolves a truncated id by unique prefix and echoes the requested id", func() {
				full := songID("Come Together")
				truncated := full[:16]
				q := queryResult(get("/Items?ids=" + enc(truncated)))
				Expect(names(q.Items)).To(ConsistOf("Come Together"))
				// Finamp matches restored items by its stored ids, so the requested id must be echoed.
				Expect(q.Items[0].Id).To(Equal(enc(truncated)))
			})

			It("batch-resolves a mixed list of truncated and full ids, keeping order", func() {
				ids := enc(songID("Come Together")[:16]) + "," + enc(songID("So What")) + "," + enc(songID("Help!")[:16])
				q := queryResult(get("/Items?ids=" + ids))
				Expect(names(q.Items)).To(Equal([]string{"Come Together", "So What", "Help!"}))
			})

			It("streams a track by its truncated id", func() {
				full := songID("So What")
				w := get("/Audio/" + enc(full[:16]) + "/stream")
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(streamerSpy.LastMediaFile.ID).To(Equal(full))
			})

			It("still 404s for a truncated id matching nothing", func() {
				Expect(get("/Audio/" + enc("zzzzzzzzzzzzzzzz") + "/stream").Code).To(Equal(http.StatusNotFound))
			})
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
			Expect(q.TotalRecordCount).To(Equal(12)) // 5 albums + 7 songs
		})

		// Chaining the per-type cursors must preserve the merged order.
		It("streams an unbounded multi-type merge, honoring StartIndex", func() {
			all := queryResult(get("/Items?IncludeItemTypes=MusicAlbum,Audio&Recursive=true"))
			Expect(all.Items).To(HaveLen(12))

			skipped := queryResult(get("/Items?IncludeItemTypes=MusicAlbum,Audio&Recursive=true&StartIndex=2"))
			Expect(skipped.Items).To(HaveLen(10))
			Expect(skipped.TotalRecordCount).To(Equal(12))
			Expect(skipped.StartIndex).To(Equal(2))
			Expect(names(skipped.Items)).To(Equal(names(all.Items)[2:]))
		})

		// Paging must ride on the cursor query's LIMIT/OFFSET, not be applied after materializing.
		It("pages songs via StartIndex/Limit while reporting the full total", func() {
			all := queryResult(get("/Items?IncludeItemTypes=Audio&Recursive=true&SortBy=SortName"))
			Expect(all.TotalRecordCount).To(Equal(7))
			Expect(all.Items).To(HaveLen(7))

			p1 := queryResult(get("/Items?IncludeItemTypes=Audio&Recursive=true&SortBy=SortName&Limit=3&StartIndex=0"))
			p2 := queryResult(get("/Items?IncludeItemTypes=Audio&Recursive=true&SortBy=SortName&Limit=3&StartIndex=3"))
			Expect(p1.Items).To(HaveLen(3))
			Expect(p2.Items).To(HaveLen(3))
			Expect(p1.TotalRecordCount).To(Equal(7))
			// The two pages are distinct and match the head of the unpaged, identically-sorted list.
			Expect(names(p1.Items)).ToNot(ContainElement(BeElementOf(names(p2.Items))))
			Expect(append(names(p1.Items), names(p2.Items)...)).To(Equal(names(all.Items)[:6]))
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

		It("includes structured ArtistItems and AlbumArtists on a song (now-playing artist)", func() {
			var item dto.BaseItemDto
			parseInto(get("/Items/"+enc(songID("So What"))), &item)
			Expect(item.ArtistItems).ToNot(BeEmpty())
			Expect(item.ArtistItems[0].Name).To(Equal("Miles Davis"))
			Expect(item.ArtistItems[0].Id).ToNot(BeEmpty())
			Expect(item.AlbumArtists).ToNot(BeEmpty())
			Expect(item.AlbumArtists[0].Name).To(Equal("Miles Davis"))
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
		It("lists album artists only on /Artists/AlbumArtists (excludes performer-only artists)", func() {
			names := names(queryResult(get("/Artists/AlbumArtists")).Items)
			Expect(names).To(ConsistOf("The Beatles", "Led Zeppelin", "Miles Davis", "Solo Artist"))
			Expect(names).ToNot(ContainElement("Featured Guest"))
		})

		It("lists performing artists on /Artists (includes a track's guest artist)", func() {
			names := names(queryResult(get("/Artists")).Items)
			Expect(names).To(ContainElement("Featured Guest"))
			Expect(names).To(ContainElement("Solo Artist"))
		})

		It("returns different lists for album artists and performing artists", func() {
			aa := names(queryResult(get("/Artists/AlbumArtists")).Items)
			ar := names(queryResult(get("/Artists")).Items)
			Expect(aa).ToNot(Equal(ar))
		})

		It("lists genres", func() {
			q := queryResult(get("/Genres"))
			Expect(names(q.Items)).To(ConsistOf("Rock", "Jazz", "Pop"))
		})

		It("pages genres with StartIndex/Limit and still reports the full total", func() {
			q := queryResult(get("/Genres?StartIndex=1&Limit=1"))
			Expect(q.Items).To(HaveLen(1))
			Expect(q.TotalRecordCount).To(Equal(3))
		})
	})
})
