package jellyfin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// withChiURLParam simulates chi's routing having captured a path parameter, since these
// tests call handlers directly instead of going through the full router.
func withChiURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

var _ = Describe("Items", func() {
	var api *Router
	var ds *tests.MockDataStore
	ctxUser := func() context.Context {
		return request.WithUser(context.Background(), model.User{ID: "u1", UserName: "alice"})
	}
	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		api = &Router{ds: ds}
	})

	Describe("getItems", func() {
		It("lists albums when IncludeItemTypes=MusicAlbum", func() {
			ds.Album(nil).(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "a1", Name: "One"}, {ID: "a2", Name: "Two"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items?IncludeItemTypes=MusicAlbum&Recursive=true", nil).WithContext(ctxUser())
			api.getItems(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).To(HaveLen(2))
			Expect(res.Items[0].Type).To(Equal("MusicAlbum"))
			Expect(res.TotalRecordCount).To(Equal(2))
		})

		It("lists an album's songs when ParentId is an album and type is Audio", func() {
			ds.Album(nil).(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "a1", Name: "One"}})
			ds.MediaFile(nil).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{{ID: "s1", Title: "Song", AlbumID: "a1"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items?ParentId=a1&IncludeItemTypes=Audio", nil).WithContext(ctxUser())
			api.getItems(w, r)
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).To(HaveLen(1))
			Expect(res.Items[0].Type).To(Equal("Audio"))
			Expect(res.Items[0].Id).To(Equal("s1"))
		})

		It("lists an artist's albums when ParentId is an artist and type is MusicAlbum", func() {
			ds.Album(nil).(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "a1", Name: "One", AlbumArtistID: "ar1"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items?ParentId=ar1&IncludeItemTypes=MusicAlbum", nil).WithContext(ctxUser())
			api.getItems(w, r)
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).To(HaveLen(1))
			albumRepo := ds.Album(nil).(*tests.MockAlbumRepo)
			sql, _, err := albumRepo.Options.Filters.ToSql()
			Expect(err).NotTo(HaveOccurred())
			Expect(sql).To(ContainSubstring("json_tree"))
		})

		It("lists artists when IncludeItemTypes=MusicArtist", func() {
			ds.Artist(nil).(*tests.MockArtistRepo).SetData(model.Artists{{ID: "ar1", Name: "Artist"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items?IncludeItemTypes=MusicArtist", nil).WithContext(ctxUser())
			api.getItems(w, r)
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).To(HaveLen(1))
			Expect(res.Items[0].Type).To(Equal("MusicArtist"))
		})

		It("lists genres when IncludeItemTypes=MusicGenre", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items?IncludeItemTypes=MusicGenre", nil).WithContext(ctxUser())
			api.getItems(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).NotTo(BeNil())
		})

		It("applies a starred filter when Filters=IsFavorite", func() {
			albumRepo := ds.Album(nil).(*tests.MockAlbumRepo)
			albumRepo.SetData(model.Albums{{ID: "a1", Name: "One"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items?IncludeItemTypes=MusicAlbum&Filters=IsFavorite", nil).WithContext(ctxUser())
			api.getItems(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			sql, _, err := albumRepo.Options.Filters.ToSql()
			Expect(err).NotTo(HaveOccurred())
			Expect(sql).To(ContainSubstring("starred"))
		})

		It("forwards SearchTerm to the repo's Search method", func() {
			albumRepo := ds.Album(nil).(*tests.MockAlbumRepo)
			albumRepo.SetData(model.Albums{{ID: "a1", Name: "One"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items?IncludeItemTypes=MusicAlbum&SearchTerm=one", nil).WithContext(ctxUser())
			api.getItems(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).To(HaveLen(1))
		})

		It("forwards StartIndex/Limit as Offset/Max", func() {
			albumRepo := ds.Album(nil).(*tests.MockAlbumRepo)
			albumRepo.SetData(model.Albums{{ID: "a1", Name: "One"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items?IncludeItemTypes=MusicAlbum&StartIndex=5&Limit=10", nil).WithContext(ctxUser())
			api.getItems(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(albumRepo.Options.Offset).To(Equal(5))
			Expect(albumRepo.Options.Max).To(Equal(10))
		})
	})

	Describe("getItem", func() {
		It("returns an album by id", func() {
			ds.Album(nil).(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "a1", Name: "One"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items/a1", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "a1")
			api.getItem(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var item dto.BaseItemDto
			Expect(json.Unmarshal(w.Body.Bytes(), &item)).To(Succeed())
			Expect(item.Id).To(Equal("a1"))
			Expect(item.Type).To(Equal("MusicAlbum"))
		})

		It("returns 404 when the id doesn't match any entity", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items/missing", nil).WithContext(ctxUser())
			r = withChiURLParam(r, "itemId", "missing")
			api.getItem(w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("getLatest", func() {
		It("returns a bare array of the newest albums", func() {
			ds.Album(nil).(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "a1", Name: "One"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Users/u1/Items/Latest", nil).WithContext(ctxUser())
			api.getLatest(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var items []dto.BaseItemDto
			Expect(json.Unmarshal(w.Body.Bytes(), &items)).To(Succeed())
			Expect(items).To(HaveLen(1))
			Expect(items[0].Id).To(Equal("a1"))
		})
	})
})
