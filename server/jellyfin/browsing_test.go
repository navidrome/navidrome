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
		return request.WithUser(context.Background(), model.User{ID: "u1", UserName: "alice", Libraries: libs})
	}

	// admin has no explicit Libraries; access is granted via the IsAdmin bypass, not membership.
	ctxAdmin := func() context.Context {
		return request.WithUser(context.Background(), model.User{ID: "admin", IsAdmin: true, Libraries: nil})
	}

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		api = &Router{ds: ds}
	})

	Describe("getArtists", func() {
		It("lists artists via /Artists", func() {
			ds.Artist(context.Background()).(*tests.MockArtistRepo).SetData(model.Artists{{ID: "ar1", Name: "A"}})
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
			ds.Artist(context.Background()).(*tests.MockArtistRepo).SetData(model.Artists{{ID: "ar1", Name: "A"}})
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
			artistRepo.SetData(model.Artists{{ID: "ar1", Name: "Artist"}})
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
			artistRepo.SetData(model.Artists{{ID: "ar1", Name: "Artist"}})
			w := httptest.NewRecorder()
			libs := model.Libraries{{ID: 1}, {ID: 2}}
			r := httptest.NewRequest("GET", "/Artists?ParentId=2", nil).WithContext(ctxUser(libs))
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
			artistRepo.SetData(model.Artists{{ID: "ar1", Name: "Artist"}})
			w := httptest.NewRecorder()
			libs := model.Libraries{{ID: 1}} // no access to library 99
			r := httptest.NewRequest("GET", "/Artists?ParentId=99", nil).WithContext(ctxUser(libs))
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
			artistRepo.SetData(model.Artists{{ID: "ar1", Name: "Artist"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Artists?SearchTerm=art", nil).WithContext(ctxUser(model.Libraries{{ID: 1}}))
			invoke(api.getArtists, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).To(HaveLen(1))
		})

		It("forwards StartIndex/Limit as Offset/Max", func() {
			artistRepo := ds.Artist(context.Background()).(*tests.MockArtistRepo)
			artistRepo.SetData(model.Artists{{ID: "ar1", Name: "Artist"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Artists?StartIndex=5&Limit=10", nil).WithContext(ctxUser(model.Libraries{{ID: 1}}))
			invoke(api.getArtists, w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(artistRepo.Options.Offset).To(Equal(5))
			Expect(artistRepo.Options.Max).To(Equal(10))
		})

		It("does not restrict results for an admin user", func() {
			artistRepo := ds.Artist(context.Background()).(*tests.MockArtistRepo)
			artistRepo.SetData(model.Artists{{ID: "ar1", Name: "Artist"}})
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
})
