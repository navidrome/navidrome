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

		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

		// Get references to the mock repositories so we can inspect their Options
		mockAlbumRepo = ds.Album(nil).(*tests.MockAlbumRepo)
		mockArtistRepo = ds.Artist(nil).(*tests.MockArtistRepo)
		mockMediaFileRepo = ds.MediaFile(nil).(*tests.MockMediaFileRepo)
	})

	Context("musicFolderId parameter", func() {
		assertQueryOptions := func(filter squirrel.Sqlizer, expectedQuery string, expectedArgs ...any) {
			GinkgoHelper()
			query, args, err := filter.ToSql()
			Expect(err).ToNot(HaveOccurred())
			Expect(query).To(ContainSubstring(expectedQuery))
			Expect(args).To(ContainElements(expectedArgs...))
		}

		Describe("Search2", func() {
			It("narrows artists with a join-free EXISTS when musicFolderId is a strict subset", func() {
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

				assertQueryOptions(mockAlbumRepo.Options.Filters, "library_id IN (?)", 1)
				assertQueryOptions(mockMediaFileRepo.Options.Filters, "library_id IN (?)", 1)
				// Artists scope through the library_artist junction, so a join-free EXISTS is used.
				assertQueryOptions(mockArtistRepo.Options.Filters,
					"EXISTS (SELECT 1 FROM library_artist", 1)
			})

			It("skips the artist narrowing filter when the request covers all of the user's libraries", func() {
				r := newGetRequest("query=test") // no musicFolderId → resolves to all accessible
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

				// Artist filter is left to the repository when the request spans all the user's libs.
				assertQueryOptions(mockAlbumRepo.Options.Filters, "library_id IN (?,?,?)", 1, 2, 3)
				assertQueryOptions(mockMediaFileRepo.Options.Filters, "library_id IN (?,?,?)", 1, 2, 3)
				Expect(mockArtistRepo.Options.Filters).To(BeNil())
			})

			It("narrows artists when duplicate musicFolderId IDs still form a strict subset", func() {
				// Duplicates inflate the requested count (musicFolderId is not deduplicated):
				// {1,1,2} has 3 entries but is still a strict subset of the 3 libraries, so the
				// narrowing filter must apply (a length-based check would wrongly skip it).
				r := newGetRequest("query=test", "musicFolderId=1", "musicFolderId=1", "musicFolderId=2")
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
				assertQueryOptions(mockArtistRepo.Options.Filters,
					"EXISTS (SELECT 1 FROM library_artist", 1, 2)
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
			It("narrows artists with a join-free EXISTS when musicFolderId is a strict subset", func() {
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

				assertQueryOptions(mockAlbumRepo.Options.Filters, "library_id IN (?)", 1)
				assertQueryOptions(mockMediaFileRepo.Options.Filters, "library_id IN (?)", 1)
				// Artists scope through the library_artist junction, so a join-free EXISTS is used.
				assertQueryOptions(mockArtistRepo.Options.Filters,
					"EXISTS (SELECT 1 FROM library_artist", 1)
			})

			It("skips the artist narrowing filter when the request covers all of the user's libraries", func() {
				r := newGetRequest("query=test") // no musicFolderId → resolves to all accessible
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

				// Artist filter is left to the repository when the request spans all the user's libs.
				assertQueryOptions(mockAlbumRepo.Options.Filters, "library_id IN (?,?,?)", 1, 2, 3)
				assertQueryOptions(mockMediaFileRepo.Options.Filters, "library_id IN (?,?,?)", 1, 2, 3)
				Expect(mockArtistRepo.Options.Filters).To(BeNil())
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
