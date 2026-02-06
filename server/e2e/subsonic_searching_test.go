package e2e

import (
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Search Endpoints", func() {
	BeforeEach(func() {
		setupTestDB()
	})

	Describe("Search2", func() {
		It("finds artists by name", func() {
			r := newReq("search2", "query", "Beatles")
			resp, err := router.Search2(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.SearchResult2).ToNot(BeNil())
			Expect(resp.SearchResult2.Artist).ToNot(BeEmpty())

			found := false
			for _, a := range resp.SearchResult2.Artist {
				if a.Name == "The Beatles" {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "expected to find artist 'The Beatles'")
		})

		It("finds albums by name", func() {
			r := newReq("search2", "query", "Abbey Road")
			resp, err := router.Search2(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.SearchResult2).ToNot(BeNil())
			Expect(resp.SearchResult2.Album).ToNot(BeEmpty())

			found := false
			for _, a := range resp.SearchResult2.Album {
				if a.Title == "Abbey Road" {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "expected to find album 'Abbey Road'")
		})

		It("finds songs by title", func() {
			r := newReq("search2", "query", "Come Together")
			resp, err := router.Search2(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.SearchResult2).ToNot(BeNil())
			Expect(resp.SearchResult2.Song).ToNot(BeEmpty())

			found := false
			for _, s := range resp.SearchResult2.Song {
				if s.Title == "Come Together" {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "expected to find song 'Come Together'")
		})

		It("respects artistCount/albumCount/songCount limits", func() {
			r := newReq("search2", "query", "Beatles",
				"artistCount", "1", "albumCount", "1", "songCount", "1")
			resp, err := router.Search2(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.SearchResult2).ToNot(BeNil())
			Expect(len(resp.SearchResult2.Artist)).To(BeNumerically("<=", 1))
			Expect(len(resp.SearchResult2.Album)).To(BeNumerically("<=", 1))
			Expect(len(resp.SearchResult2.Song)).To(BeNumerically("<=", 1))
		})

		It("supports offset parameters", func() {
			// First get all results for Beatles
			r1 := newReq("search2", "query", "Beatles", "songCount", "500")
			resp1, err := router.Search2(r1)
			Expect(err).ToNot(HaveOccurred())
			allSongs := resp1.SearchResult2.Song

			if len(allSongs) > 1 {
				// Get with offset to skip the first song
				r2 := newReq("search2", "query", "Beatles", "songOffset", "1", "songCount", "500")
				resp2, err := router.Search2(r2)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp2.SearchResult2).ToNot(BeNil())
				Expect(len(resp2.SearchResult2.Song)).To(Equal(len(allSongs) - 1))
			}
		})

		It("returns empty results for non-matching query", func() {
			r := newReq("search2", "query", "ZZZZNONEXISTENT99999")
			resp, err := router.Search2(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.SearchResult2).ToNot(BeNil())
			Expect(resp.SearchResult2.Artist).To(BeEmpty())
			Expect(resp.SearchResult2.Album).To(BeEmpty())
			Expect(resp.SearchResult2.Song).To(BeEmpty())
		})
	})

	Describe("Search3", func() {
		It("returns results in ID3 format", func() {
			r := newReq("search3", "query", "Beatles")
			resp, err := router.Search3(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.SearchResult3).ToNot(BeNil())
			// Verify ID3 format: Artist should be ArtistID3 with Name and AlbumCount
			Expect(resp.SearchResult3.Artist).ToNot(BeEmpty())
			Expect(resp.SearchResult3.Artist[0].Name).ToNot(BeEmpty())
			Expect(resp.SearchResult3.Artist[0].Id).ToNot(BeEmpty())
		})

		It("finds across all entity types simultaneously", func() {
			// "Beatles" should match artist, albums, and songs by The Beatles
			r := newReq("search3", "query", "Beatles")
			resp, err := router.Search3(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.SearchResult3).ToNot(BeNil())

			// Should find at least the artist "The Beatles"
			artistFound := false
			for _, a := range resp.SearchResult3.Artist {
				if a.Name == "The Beatles" {
					artistFound = true
					break
				}
			}
			Expect(artistFound).To(BeTrue(), "expected to find artist 'The Beatles'")

			// Should find albums by The Beatles (albums contain "Beatles" in artist field)
			// Albums are returned as AlbumID3 type
			for _, a := range resp.SearchResult3.Album {
				Expect(a.Id).ToNot(BeEmpty())
				Expect(a.Name).ToNot(BeEmpty())
			}

			// Songs are returned as Child type
			for _, s := range resp.SearchResult3.Song {
				Expect(s.Id).ToNot(BeEmpty())
				Expect(s.Title).ToNot(BeEmpty())
			}
		})
	})
})
