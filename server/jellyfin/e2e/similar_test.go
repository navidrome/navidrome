package e2e

import (
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Similar", func() {
	BeforeEach(func() { setupTestDB() })

	Describe("GET /Artists/{id}/Similar", func() {
		It("returns the provider's similar artists, excluding ones not in the library", func() {
			providerFake.similarArtists = model.Artists{
				{ID: "z", Name: "Led Zeppelin"},
				{ID: "", Name: "Not In Library"}, // no id -> not present -> excluded
			}
			q := queryResult(get("/Artists/" + enc(artistID("The Beatles")) + "/Similar"))
			Expect(names(q.Items)).To(ConsistOf("Led Zeppelin"))
			Expect(q.Items[0].Type).To(Equal("MusicArtist"))
		})

		It("returns an empty result (not 404) when the provider has nothing", func() {
			q := queryResult(get("/Artists/" + enc(artistID("The Beatles")) + "/Similar"))
			Expect(q.Items).To(BeEmpty())
			Expect(q.TotalRecordCount).To(Equal(0))
		})
	})

	Describe("GET /Items/{id}/Similar", func() {
		It("returns similar songs for a track", func() {
			providerFake.similarSongs = model.MediaFiles{{ID: "x1", Title: "Similar Song", LibraryID: 1}}
			q := queryResult(get("/Items/" + enc(songID("So What")) + "/Similar"))
			Expect(names(q.Items)).To(ConsistOf("Similar Song"))
			Expect(q.Items[0].Type).To(Equal("Audio"))
		})

		It("excludes similar songs from libraries the user can't access", func() {
			providerFake.similarSongs = model.MediaFiles{
				{ID: "x1", Title: "In Library", LibraryID: 1},
				{ID: "x2", Title: "Other Library", LibraryID: 2}, // regularUser has no access
			}
			q := queryResult(getAs(regularUser, "/Items/"+enc(songID("So What"))+"/Similar"))
			Expect(names(q.Items)).To(ConsistOf("In Library"))
		})

		It("returns similar albums (derived from similar songs, de-duplicated) for an album", func() {
			providerFake.similarSongs = model.MediaFiles{
				{ID: "x1", AlbumID: albumID("IV")},
				{ID: "x2", AlbumID: albumID("IV")}, // same album -> counted once
				{ID: "x3", AlbumID: albumID("Kind of Blue")},
			}
			q := queryResult(get("/Items/" + enc(albumID("Abbey Road")) + "/Similar"))
			Expect(names(q.Items)).To(Equal([]string{"IV", "Kind of Blue"}))
			Expect(q.Items[0].Type).To(Equal("MusicAlbum"))
		})

		It("excludes similar albums from libraries the user can't access", func() {
			// Seed an album in a second library the regular user has no access to, and point a
			// provider similar-song at it.
			otherLib := model.Library{ID: 2, Name: "Other Library", Path: "fake:///other"}
			Expect(ds.Library(ctx).Put(&otherLib)).To(Succeed())
			otherAlbum := model.Album{ID: "other-album", Name: "Other Album", LibraryID: 2}
			Expect(ds.Album(ctx).Put(&otherAlbum)).To(Succeed())

			providerFake.similarSongs = model.MediaFiles{
				{ID: "x1", AlbumID: albumID("IV")}, // library 1 -> visible
				{ID: "x2", AlbumID: "other-album"}, // library 2 -> filtered for regularUser
			}
			q := queryResult(getAs(regularUser, "/Items/"+enc(albumID("Abbey Road"))+"/Similar"))
			Expect(names(q.Items)).To(ConsistOf("IV"))
		})

		It("returns an empty result (not 404) for an unknown item, so the client stops retrying", func() {
			q := queryResult(get("/Items/" + enc("does-not-exist") + "/Similar"))
			Expect(q.Items).To(BeEmpty())
		})
	})

	// Finamp plays exactly what InstantMix returns, so a track seed must lead its own mix.
	Describe("GET /Items/{id}/InstantMix", func() {
		It("returns the seed track first, followed by similar songs", func() {
			providerFake.similarSongs = model.MediaFiles{{ID: "x1", Title: "Similar Song", LibraryID: 1}}
			q := queryResult(get("/Items/" + enc(songID("So What")) + "/InstantMix?limit=19"))
			Expect(names(q.Items)).To(Equal([]string{"So What", "Similar Song"}))
			Expect(q.Items[0].Type).To(Equal("Audio"))
		})

		It("does not duplicate the seed when the provider returns it", func() {
			providerFake.similarSongs = model.MediaFiles{
				{ID: songID("So What"), Title: "So What", LibraryID: 1},
				{ID: "x1", Title: "Similar Song", LibraryID: 1},
			}
			q := queryResult(get("/Items/" + enc(songID("So What")) + "/InstantMix"))
			Expect(names(q.Items)).To(Equal([]string{"So What", "Similar Song"}))
		})

		It("caps the mix at the requested limit", func() {
			providerFake.similarSongs = model.MediaFiles{
				{ID: "x1", Title: "S1", LibraryID: 1},
				{ID: "x2", Title: "S2", LibraryID: 1},
				{ID: "x3", Title: "S3", LibraryID: 1},
			}
			q := queryResult(get("/Items/" + enc(songID("So What")) + "/InstantMix?limit=2"))
			Expect(names(q.Items)).To(Equal([]string{"So What", "S1"}))
		})

		It("excludes similar songs from libraries the user can't access", func() {
			providerFake.similarSongs = model.MediaFiles{
				{ID: "x1", Title: "In Library", LibraryID: 1},
				{ID: "x2", Title: "Other Library", LibraryID: 2},
			}
			q := queryResult(getAs(regularUser, "/Items/"+enc(songID("So What"))+"/InstantMix"))
			Expect(names(q.Items)).To(Equal([]string{"So What", "In Library"}))
		})

		It("returns a mix of the provider's similar songs for an artist seed", func() {
			providerFake.similarSongs = model.MediaFiles{{ID: "x1", Title: "Artist Mix Song", LibraryID: 1}}
			q := queryResult(get("/Items/" + enc(artistID("Miles Davis")) + "/InstantMix"))
			Expect(names(q.Items)).To(Equal([]string{"Artist Mix Song"}))
		})

		It("returns an empty result (not 404) for an unknown item", func() {
			w := get("/Items/" + enc("does-not-exist") + "/InstantMix")
			Expect(w.Code).To(Equal(200))
			Expect(queryResult(w).Items).To(BeEmpty())
		})

		It("returns only the seed when the provider has nothing", func() {
			q := queryResult(get("/Items/" + enc(songID("Help!")) + "/InstantMix"))
			Expect(names(q.Items)).To(Equal([]string{"Help!"}))
		})
	})
})
