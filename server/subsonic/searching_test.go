package subsonic

import (
	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Search", func() {
	var router *Router
	var ds model.DataStore
	var mockAlbumRepo *tests.MockAlbumRepo
	var mockArtistRepo *tests.MockArtistRepo
	var mockMediaFileRepo *tests.MockMediaFileRepo

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		auth.Init(ds)

		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

		// Get references to the mock repositories so we can inspect their Options
		mockAlbumRepo = ds.Album(nil).(*tests.MockAlbumRepo)
		mockArtistRepo = ds.Artist(nil).(*tests.MockArtistRepo)
		mockMediaFileRepo = ds.MediaFile(nil).(*tests.MockMediaFileRepo)
	})

	Context("musicFolderId parameter", func() {
		assertQueryOptions := func(filter squirrel.Sqlizer, expectedQuery string, expectedArgs ...interface{}) {
			GinkgoHelper()
			query, args, err := filter.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(query).To(ContainSubstring(expectedQuery))
			Expect(args).To(ContainElements(expectedArgs...))
		}

		Describe("Search2", func() {
			It("should accept musicFolderId parameter", func() {
				r := newGetRequest("query=test", "musicFolderId=1")
				ctx := request.WithUser(r.Context(), model.User{
					ID:        "user1",
					UserName:  "testuser",
					Libraries: []model.Library{{ID: 1, Name: "Library 1"}},
				})
				r = r.WithContext(ctx)

				resp, err := router.Search2(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.SearchResult2).ToNot(BeNil())

				// Verify that library filter was applied to all repositories
				assertQueryOptions(mockAlbumRepo.Options.Filters, "library_id IN (?)", 1)
				assertQueryOptions(mockArtistRepo.Options.Filters, "library_id IN (?)", 1)
				assertQueryOptions(mockMediaFileRepo.Options.Filters, "library_id IN (?)", 1)
			})

			It("should return results from all accessible libraries when musicFolderId is not provided", func() {
				r := newGetRequest("query=test")
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

				// Verify that library filter was applied to all repositories with all accessible libraries
				assertQueryOptions(mockAlbumRepo.Options.Filters, "library_id IN (?,?,?)", 1, 2, 3)
				assertQueryOptions(mockArtistRepo.Options.Filters, "library_id IN (?,?,?)", 1, 2, 3)
				assertQueryOptions(mockMediaFileRepo.Options.Filters, "library_id IN (?,?,?)", 1, 2, 3)
			})

			It("should return empty results when user has no accessible libraries", func() {
				r := newGetRequest("query=test")
				ctx := request.WithUser(r.Context(), model.User{
					ID:        "user1",
					UserName:  "testuser",
					Libraries: []model.Library{}, // No libraries
				})
				r = r.WithContext(ctx)

				resp, err := router.Search2(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.SearchResult2).ToNot(BeNil())
				Expect(mockAlbumRepo.Options.Filters).To(BeNil())
				Expect(mockArtistRepo.Options.Filters).To(BeNil())
				Expect(mockMediaFileRepo.Options.Filters).To(BeNil())
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
			It("should accept musicFolderId parameter", func() {
				r := newGetRequest("query=test", "musicFolderId=1")
				ctx := request.WithUser(r.Context(), model.User{
					ID:        "user1",
					UserName:  "testuser",
					Libraries: []model.Library{{ID: 1, Name: "Library 1"}},
				})
				r = r.WithContext(ctx)

				resp, err := router.Search3(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.SearchResult3).ToNot(BeNil())

				// Verify that library filter was applied to all repositories
				assertQueryOptions(mockAlbumRepo.Options.Filters, "library_id IN (?)", 1)
				assertQueryOptions(mockArtistRepo.Options.Filters, "library_id IN (?)", 1)
				assertQueryOptions(mockMediaFileRepo.Options.Filters, "library_id IN (?)", 1)
			})

			It("should return results from all accessible libraries when musicFolderId is not provided", func() {
				r := newGetRequest("query=test")
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

				// Verify that library filter was applied to all repositories with all accessible libraries
				assertQueryOptions(mockAlbumRepo.Options.Filters, "library_id IN (?,?,?)", 1, 2, 3)
				assertQueryOptions(mockArtistRepo.Options.Filters, "library_id IN (?,?,?)", 1, 2, 3)
				assertQueryOptions(mockMediaFileRepo.Options.Filters, "library_id IN (?,?,?)", 1, 2, 3)
			})

			It("should return empty results when user has no accessible libraries", func() {
				r := newGetRequest("query=test")
				ctx := request.WithUser(r.Context(), model.User{
					ID:        "user1",
					UserName:  "testuser",
					Libraries: []model.Library{}, // No libraries
				})
				r = r.WithContext(ctx)

				resp, err := router.Search3(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(resp).ToNot(BeNil())
				Expect(resp.SearchResult3).ToNot(BeNil())
				Expect(mockAlbumRepo.Options.Filters).To(BeNil())
				Expect(mockArtistRepo.Options.Filters).To(BeNil())
				Expect(mockMediaFileRepo.Options.Filters).To(BeNil())
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
})
