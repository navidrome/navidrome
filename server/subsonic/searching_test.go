package subsonic

import (
	"context"
	"net/http"
	"strconv"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/slice"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Search", func() {
	var router *Router
	var ds model.DataStore
	var ctx context.Context

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
		auth.Init(ctx, ds)

		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	})

	Context("musicFolderId parameter", func() {
		// Every library N holds artist ar-N, album al-N and song mf-N, all matching "test".
		// user1 can read libraries 1-3; library 4 is someone else's.
		BeforeEach(func() {
			putUsers(ctx, ds, model.User{ID: "user1", UserName: "testuser"})
			libIDs := persistencetest.SetLibraries(ctx, ds, "/lib1", "/lib2", "/lib3", "/lib4")
			Expect(ds.User().SetUserLibraries(ctx, "user1", libIDs[:3])).To(Succeed())
			for _, id := range libIDs {
				n := strconv.Itoa(id)
				putAlbumArtists(ctx, ds, id, model.Artist{ID: "ar-" + n, Name: "Test Artist " + n})
				persistencetest.PutAlbums(ctx, ds, model.Album{ID: "al-" + n, Name: "Test Album " + n, LibraryID: id})
				persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "mf-" + n, Title: "Test Song " + n, LibraryID: id})
			}
		})

		idsFor := func(prefix string, libs []string) []string {
			ids := make([]string, len(libs))
			for i, n := range libs {
				ids[i] = prefix + n
			}
			return ids
		}
		expectResults2 := func(res *responses.SearchResult2, libs ...string) {
			GinkgoHelper()
			Expect(slice.Map(res.Artist, func(a responses.Artist) string { return a.Id })).To(ConsistOf(idsFor("ar-", libs)))
			Expect(slice.Map(res.Album, func(c responses.Child) string { return c.Id })).To(ConsistOf(idsFor("al-", libs)))
			Expect(slice.Map(res.Song, func(c responses.Child) string { return c.Id })).To(ConsistOf(idsFor("mf-", libs)))
		}
		expectResults3 := func(res *responses.SearchResult3, libs ...string) {
			GinkgoHelper()
			Expect(slice.Map(res.Artist, func(a responses.ArtistID3) string { return a.Id })).To(ConsistOf(idsFor("ar-", libs)))
			Expect(slice.Map(res.Album, func(a responses.AlbumID3) string { return a.Id })).To(ConsistOf(idsFor("al-", libs)))
			Expect(slice.Map(res.Song, func(c responses.Child) string { return c.Id })).To(ConsistOf(idsFor("mf-", libs)))
		}

		Describe("Search2", func() {
			It("scopes all entity types to the requested libraries", func() {
				// The subsonic layer passes the same library_id filter to all three repos; the
				// artist repository translates it to the join-free library_artist predicate itself.
				r := newGetRequest("query=test", "musicFolderId=1")
				ctx := request.WithUser(r.Context(), model.User{
					ID:       "user1",
					UserName: "testuser",
					Libraries: []model.Library{
						{ID: 1, Name: "Library 1"},
						{ID: 2, Name: "Library 2"},
					},
				})
				r = r.WithContext(ctx)

				resp, err := router.Search2(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.SearchResult2).ToNot(BeNil())

				expectResults2(resp.SearchResult2, "1")
			})

			It("applies no library filter when musicFolderId is not provided", func() {
				r := newGetRequest("query=test") // no musicFolderId → all accessible libraries
				ctx := request.WithUser(r.Context(), model.User{
					ID:       "user1",
					UserName: "testuser",
					Libraries: []model.Library{
						{ID: 1, Name: "Library 1"},
						{ID: 2, Name: "Library 2"},
						{ID: 3, Name: "Library 3"},
					},
				})
				r = r.WithContext(ctx)

				resp, err := router.Search2(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.SearchResult2).ToNot(BeNil())

				expectResults2(resp.SearchResult2, "1", "2", "3")
			})

			It("should return empty results when user has no accessible libraries", func() {
				r := newGetRequest("query=test")
				ctx := request.WithUser(r.Context(), model.User{
					ID:        "user1",
					UserName:  "testuser",
					Libraries: []model.Library{}, // No libraries
				})
				Expect(ds.User().SetUserLibraries(ctx, "user1", nil)).To(Succeed())
				r = r.WithContext(ctx)

				resp, err := router.Search2(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.SearchResult2).ToNot(BeNil())
				expectResults2(resp.SearchResult2)
			})

			It("should return error for inaccessible musicFolderId", func() {
				r := newGetRequest("query=test", "musicFolderId=999")
				ctx := request.WithUser(r.Context(), model.User{
					ID:        "user1",
					UserName:  "testuser",
					Libraries: []model.Library{{ID: 1, Name: "Library 1"}},
				})
				r = r.WithContext(ctx)

				resp, err := router.Search2(r)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Library 999 not found or not accessible"))
				Expect(resp).To(BeNil())
			})
		})

		Describe("Search3", func() {
			It("scopes all entity types to the requested libraries", func() {
				r := newGetRequest("query=test", "musicFolderId=1")
				ctx := request.WithUser(r.Context(), model.User{
					ID:       "user1",
					UserName: "testuser",
					Libraries: []model.Library{
						{ID: 1, Name: "Library 1"},
						{ID: 2, Name: "Library 2"},
					},
				})
				r = r.WithContext(ctx)

				resp, err := router.Search3(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.SearchResult3).ToNot(BeNil())

				expectResults3(resp.SearchResult3, "1")
			})

			It("applies no library filter when musicFolderId is not provided", func() {
				r := newGetRequest("query=test") // no musicFolderId → all accessible libraries
				ctx := request.WithUser(r.Context(), model.User{
					ID:       "user1",
					UserName: "testuser",
					Libraries: []model.Library{
						{ID: 1, Name: "Library 1"},
						{ID: 2, Name: "Library 2"},
						{ID: 3, Name: "Library 3"},
					},
				})
				r = r.WithContext(ctx)

				resp, err := router.Search3(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.SearchResult3).ToNot(BeNil())

				expectResults3(resp.SearchResult3, "1", "2", "3")
			})

			It("should return empty results when user has no accessible libraries", func() {
				r := newGetRequest("query=test")
				ctx := request.WithUser(r.Context(), model.User{
					ID:        "user1",
					UserName:  "testuser",
					Libraries: []model.Library{}, // No libraries
				})
				Expect(ds.User().SetUserLibraries(ctx, "user1", nil)).To(Succeed())
				r = r.WithContext(ctx)

				resp, err := router.Search3(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.SearchResult3).ToNot(BeNil())
				expectResults3(resp.SearchResult3)
			})

			It("should return error for inaccessible musicFolderId", func() {
				// Test that the endpoint returns an error when user tries to access a library they don't have access to
				r := newGetRequest("query=test", "musicFolderId=999")
				ctx := request.WithUser(r.Context(), model.User{
					ID:        "user1",
					UserName:  "testuser",
					Libraries: []model.Library{{ID: 1, Name: "Library 1"}},
				})
				r = r.WithContext(ctx)

				resp, err := router.Search3(r)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Library 999 not found or not accessible"))
				Expect(resp).To(BeNil())
			})
		})
	})

	Describe("artwork emission", func() {
		newSearchRequest := func() *http.Request {
			r := newGetRequest("query=test")
			ctx := request.WithUser(r.Context(), model.User{ID: "user1", Libraries: []model.Library{{ID: 1}}})
			return r.WithContext(ctx)
		}

		BeforeEach(func() {
			putUsers(ctx, ds, model.User{ID: "user1", UserName: "testuser"})
		})

		It("omits coverArt and imageUrl for a known-absent artist", func() {
			putAlbumArtists(ctx, ds, model.DefaultLibraryID, model.Artist{ID: "ar-1", Name: "Absent Test Artist"})
			putItemArtwork(ctx, ds, model.KindArtistArtwork, "ar-1", "")
			resp, err := router.Search2(newSearchRequest())
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.SearchResult2.Artist).To(HaveLen(1))
			Expect(resp.SearchResult2.Artist[0].CoverArt).To(BeEmpty())
			Expect(resp.SearchResult2.Artist[0].ArtistImageUrl).To(BeEmpty())
		})

		It("emits coverArt and imageUrl for an unresolved artist", func() {
			putAlbumArtists(ctx, ds, model.DefaultLibraryID, model.Artist{ID: "ar-1", Name: "Unresolved Test Artist"})
			resp, err := router.Search2(newSearchRequest())
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.SearchResult2.Artist).To(HaveLen(1))
			Expect(resp.SearchResult2.Artist[0].CoverArt).To(Equal("ar-ar-1"))
			Expect(resp.SearchResult2.Artist[0].ArtistImageUrl).ToNot(BeEmpty())
		})
	})
})
