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
			providerFake.similarSongs = model.MediaFiles{{ID: "x1", Title: "Similar Song"}}
			q := queryResult(get("/Items/" + enc(songID("So What")) + "/Similar"))
			Expect(names(q.Items)).To(ConsistOf("Similar Song"))
			Expect(q.Items[0].Type).To(Equal("Audio"))
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

		It("returns an empty result (not 404) for an unknown item, so the client stops retrying", func() {
			q := queryResult(get("/Items/" + enc("does-not-exist") + "/Similar"))
			Expect(q.Items).To(BeEmpty())
		})
	})
})
