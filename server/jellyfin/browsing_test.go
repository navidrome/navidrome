package jellyfin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Browsing", func() {
	var api *Router
	var ds *tests.MockDataStore
	ctxUser := func(libs model.Libraries) context.Context {
		return request.WithUser(context.Background(), model.User{ID: testID("u1"), UserName: "alice", Libraries: libs})
	}

	// admin has no explicit Libraries; access is granted via the IsAdmin bypass, not membership.
	ctxAdmin := func() context.Context {
		return request.WithUser(context.Background(), model.User{ID: testID("admin"), IsAdmin: true, Libraries: nil})
	}

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		api = &Router{ds: ds}
	})

	Describe("getArtists", func() {
		It("lists artists via /Artists", func() {
			ds.Artist(context.Background()).(*tests.MockArtistRepo).SetData(model.Artists{{ID: testID("ar1"), Name: "A"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Artists", nil).WithContext(ctxUser(model.Libraries{{ID: 1}}))
			invoke(api.getArtists, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).To(HaveLen(1))
			Expect(res.Items[0].Type).To(Equal("MusicArtist"))
		})

		It("handles /Artists/AlbumArtists the same way", func() {
			ds.Artist(context.Background()).(*tests.MockArtistRepo).SetData(model.Artists{{ID: testID("ar1"), Name: "A"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Artists/AlbumArtists", nil).WithContext(ctxUser(model.Libraries{{ID: 1}}))
			invoke(api.getArtists, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).To(HaveLen(1))
		})

		It("scopes results to the user's accessible libraries", func() {
			artistRepo := ds.Artist(context.Background()).(*tests.MockArtistRepo)
			artistRepo.SetData(model.Artists{{ID: testID("ar1"), Name: "Artist"}})
			w := httptest.NewRecorder()
			libs := model.Libraries{{ID: 1}, {ID: 2}}
			r := httptest.NewRequest("GET", "/Artists", nil).WithContext(ctxUser(libs))
			invoke(api.getArtists, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			sql, args, err := artistRepo.Options.Filters.ToSql()
			Expect(err).NotTo(HaveOccurred())
			Expect(sql).To(ContainSubstring("library_artist.library_id"))
			Expect(args).To(ContainElements(1, 2))
		})

		It("scopes to a single library when ParentId is an accessible library id", func() {
			artistRepo := ds.Artist(context.Background()).(*tests.MockArtistRepo)
			artistRepo.SetData(model.Artists{{ID: testID("ar1"), Name: "Artist"}})
			w := httptest.NewRecorder()
			libs := model.Libraries{{ID: 1}, {ID: 2}}
			r := httptest.NewRequest("GET", "/Artists?ParentId="+dto.EncodeLibraryID(2), nil).WithContext(ctxUser(libs))
			invoke(api.getArtists, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			sql, args, err := artistRepo.Options.Filters.ToSql()
			Expect(err).NotTo(HaveOccurred())
			Expect(sql).To(ContainSubstring("library_artist.library_id"))
			Expect(args).To(ContainElement(2))
			Expect(args).NotTo(ContainElement(1))
		})

		It("does not let ParentId=<inaccessible library id> narrow the scope", func() {
			artistRepo := ds.Artist(context.Background()).(*tests.MockArtistRepo)
			artistRepo.SetData(model.Artists{{ID: testID("ar1"), Name: "Artist"}})
			w := httptest.NewRecorder()
			libs := model.Libraries{{ID: 1}} // no access to library 99
			r := httptest.NewRequest("GET", "/Artists?ParentId="+dto.EncodeLibraryID(99), nil).WithContext(ctxUser(libs))
			invoke(api.getArtists, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			sql, args, err := artistRepo.Options.Filters.ToSql()
			Expect(err).NotTo(HaveOccurred())
			Expect(sql).To(ContainSubstring("library_artist.library_id"))
			Expect(args).To(ContainElement(1))
			Expect(args).NotTo(ContainElement(99))
		})

		It("forwards SearchTerm to the repo's Search method", func() {
			artistRepo := ds.Artist(context.Background()).(*tests.MockArtistRepo)
			artistRepo.SetData(model.Artists{{ID: testID("ar1"), Name: "Artist"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Artists?SearchTerm=art", nil).WithContext(ctxUser(model.Libraries{{ID: 1}}))
			invoke(api.getArtists, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).To(HaveLen(1))
		})

		It("bounds a search the client left unbounded, and clamps an oversized one", func() {
			artistRepo := ds.Artist(context.Background()).(*tests.MockArtistRepo)
			artistRepo.SetData(model.Artists{{ID: testID("ar1"), Name: "Artist"}})

			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Artists?SearchTerm=art", nil).WithContext(ctxUser(model.Libraries{{ID: 1}}))
			invoke(api.getArtists, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(artistRepo.Options.Max).To(Equal(defaultSearchLimit + 1))

			w = httptest.NewRecorder()
			r = httptest.NewRequest("GET", "/Artists?SearchTerm=art&Limit=999999", nil).WithContext(ctxUser(model.Libraries{{ID: 1}}))
			invoke(api.getArtists, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(artistRepo.Options.Max).To(Equal(maxSearchLimit + 1))
		})

		It("forwards StartIndex/Limit as Offset/Max", func() {
			artistRepo := ds.Artist(context.Background()).(*tests.MockArtistRepo)
			artistRepo.SetData(model.Artists{{ID: testID("ar1"), Name: "Artist"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Artists?StartIndex=5&Limit=10", nil).WithContext(ctxUser(model.Libraries{{ID: 1}}))
			invoke(api.getArtists, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(artistRepo.Options.Offset).To(Equal(5))
			Expect(artistRepo.Options.Max).To(Equal(10))
		})

		It("does not restrict results for an admin user", func() {
			artistRepo := ds.Artist(context.Background()).(*tests.MockArtistRepo)
			artistRepo.SetData(model.Artists{{ID: testID("ar1"), Name: "Artist"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Artists", nil).WithContext(ctxAdmin())
			invoke(api.getArtists, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			// accessibleLibraryIDs is empty for an admin (Libraries is nil), so
			// ApplyArtistLibraryFilter([]) is a no-op: no library_id restriction is added.
			if artistRepo.Options.Filters == nil {
				return
			}
			sql, _, err := artistRepo.Options.Filters.ToSql()
			Expect(err).NotTo(HaveOccurred())
			Expect(sql).NotTo(ContainSubstring("library_artist.library_id"))
		})

		DescribeTable("restricts to favorites",
			func(url string, handler func(*Router) http.HandlerFunc) {
				artistRepo := ds.Artist(context.Background()).(*tests.MockArtistRepo)
				artistRepo.SetData(model.Artists{{ID: testID("ar1"), Name: "Artist"}})
				w := httptest.NewRecorder()
				r := httptest.NewRequest("GET", url, nil).WithContext(ctxUser(model.Libraries{{ID: 1}}))
				invoke(handler(api), w, r)
				Expect(w.Code).To(Equal(http.StatusOK))
				sql, args, err := artistRepo.Options.Filters.ToSql()
				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring("starred"))
				// listArtists always ANDs notMissing, favorites filter or not.
				Expect(sql).To(ContainSubstring("missing"))
				Expect(args).To(ContainElement(true))
			},
			Entry("Filters=IsFavorite", "/Artists?Filters=IsFavorite",
				func(a *Router) http.HandlerFunc { return a.getArtists }),
			Entry("isFavorite=true", "/Artists?isFavorite=true",
				func(a *Router) http.HandlerFunc { return a.getArtists }),
			Entry("on /Artists/AlbumArtists", "/Artists/AlbumArtists?Filters=IsFavorite",
				func(a *Router) http.HandlerFunc { return a.getAlbumArtists }),
		)

		It("404s a malformed ParentId instead of listing every library's artists", func() {
			artistRepo := ds.Artist(context.Background()).(*tests.MockArtistRepo)
			artistRepo.SetData(model.Artists{{ID: testID("ar1"), Name: "Artist"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Artists?ParentId=not-a-valid-id", nil).WithContext(ctxUser(model.Libraries{{ID: 1}}))
			invoke(api.getArtists, w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("getGenres", func() {
		It("lists genres via /Genres", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Genres", nil).WithContext(ctxUser(model.Libraries{{ID: 1}}))
			invoke(api.getGenres, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).NotTo(BeNil())
		})

		It("handles /MusicGenres the same way", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/MusicGenres", nil).WithContext(ctxUser(model.Libraries{{ID: 1}}))
			invoke(api.getGenres, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("getStudios", func() {
		It("scopes results to the user's accessible libraries", func() {
			tagRepo := ds.Tag(context.Background()).(*tests.MockTagRepo)
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Studios", nil).WithContext(ctxUser(model.Libraries{{ID: 1}, {ID: 2}}))
			invoke(api.getStudios, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			sql, args, err := tagRepo.Options.Filters.ToSql()
			Expect(err).NotTo(HaveOccurred())
			Expect(sql).To(ContainSubstring("library_tag.library_id"))
			Expect(args).To(ContainElements(1, 2))
		})

		// An empty scope (admin, or a non-admin with no explicit library grants) must be treated
		// as unrestricted, matching accessibleLibraryIDs' documented contract, not as "match nothing".
		It("does not restrict results for an admin user", func() {
			tagRepo := ds.Tag(context.Background()).(*tests.MockTagRepo)
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Studios", nil).WithContext(ctxAdmin())
			invoke(api.getStudios, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(tagRepo.Options.Filters).To(BeNil())
		})

		It("404s a malformed ParentId instead of listing every library's studios", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Studios?ParentId=not-a-valid-id", nil).WithContext(ctxUser(model.Libraries{{ID: 1}}))
			invoke(api.getStudios, w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("getQueryFiltersLegacy", func() {
		It("scopes genres to the user's accessible libraries", func() {
			genreRepo := ds.Genre(context.Background()).(*tests.MockedGenreRepo)
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items/Filters", nil).WithContext(ctxUser(model.Libraries{{ID: 1}, {ID: 2}}))
			invoke(api.getQueryFiltersLegacy, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			sql, args, err := genreRepo.Options.Filters.ToSql()
			Expect(err).NotTo(HaveOccurred())
			Expect(sql).To(ContainSubstring("library_tag.library_id"))
			Expect(args).To(ContainElements(1, 2))
		})

		It("does not restrict genres for an admin user", func() {
			genreRepo := ds.Genre(context.Background()).(*tests.MockedGenreRepo)
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items/Filters", nil).WithContext(ctxAdmin())
			invoke(api.getQueryFiltersLegacy, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(genreRepo.Options.Filters).To(BeNil())
		})

		It("404s a malformed ParentId instead of listing every library's filters", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items/Filters?ParentId=not-a-valid-id", nil).WithContext(ctxUser(model.Libraries{{ID: 1}}))
			invoke(api.getQueryFiltersLegacy, w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})
})
