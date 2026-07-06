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
	// alice has access to library 1 only; used by tests that don't care about scoping.
	ctxUser := func() context.Context {
		return request.WithUser(context.Background(), model.User{ID: "u1", UserName: "alice", Libraries: model.Libraries{{ID: 1, Name: "Music"}}})
	}
	ctxUserWithLibraries := func(libs model.Libraries) context.Context {
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

	Describe("getItems", func() {
		It("lists albums when IncludeItemTypes=MusicAlbum", func() {
			ds.Album(context.Background()).(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "a1", Name: "One"}, {ID: "a2", Name: "Two"}})
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
			ds.Album(context.Background()).(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "a1", Name: "One"}})
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{{ID: "s1", Title: "Song", AlbumID: "a1"}})
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
			ds.Album(context.Background()).(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "a1", Name: "One", AlbumArtistID: "ar1"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items?ParentId=ar1&IncludeItemTypes=MusicAlbum", nil).WithContext(ctxUser())
			api.getItems(w, r)
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).To(HaveLen(1))
			albumRepo := ds.Album(context.Background()).(*tests.MockAlbumRepo)
			sql, _, err := albumRepo.Options.Filters.ToSql()
			Expect(err).NotTo(HaveOccurred())
			Expect(sql).To(ContainSubstring("json_tree"))
		})

		It("lists artists when IncludeItemTypes=MusicArtist", func() {
			ds.Artist(context.Background()).(*tests.MockArtistRepo).SetData(model.Artists{{ID: "ar1", Name: "Artist"}})
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

		It("lists playlists when IncludeItemTypes=Playlist", func() {
			ds.Playlist(context.Background()).(*tests.MockPlaylistRepo).SetData(model.Playlists{{ID: "p1", Name: "My Mix", SongCount: 5}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items?IncludeItemTypes=Playlist", nil).WithContext(ctxUser())
			api.getItems(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).To(HaveLen(1))
			Expect(res.Items[0].Type).To(Equal("Playlist"))
			Expect(res.Items[0].Id).To(Equal("p1"))
			Expect(res.TotalRecordCount).To(Equal(1))
		})

		It("merges results from every requested type in IncludeItemTypes", func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{{ID: "s1", Title: "Song"}})
			ds.Album(context.Background()).(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "a1", Name: "One"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items?IncludeItemTypes=Audio,MusicAlbum", nil).WithContext(ctxUser())
			api.getItems(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).To(HaveLen(2))
			types := []string{res.Items[0].Type, res.Items[1].Type}
			Expect(types).To(ConsistOf("Audio", "MusicAlbum"))
			Expect(res.TotalRecordCount).To(Equal(2))
		})

		It("merges a multi-type favorites query (Finamp's favorites screen), skipping playlists (no starred concept)", func() {
			mfRepo := ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo)
			mfRepo.SetData(model.MediaFiles{{ID: "s1", Title: "Song"}})
			albumRepo := ds.Album(context.Background()).(*tests.MockAlbumRepo)
			albumRepo.SetData(model.Albums{{ID: "a1", Name: "One"}})
			ds.Playlist(context.Background()).(*tests.MockPlaylistRepo).SetData(model.Playlists{{ID: "p1", Name: "My Mix"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items?IncludeItemTypes=Audio,MusicAlbum,Playlist&Filters=IsFavorite", nil).WithContext(ctxUser())
			api.getItems(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).To(HaveLen(2))
			types := []string{res.Items[0].Type, res.Items[1].Type}
			Expect(types).To(ConsistOf("Audio", "MusicAlbum"))
			sql, _, err := albumRepo.Options.Filters.ToSql()
			Expect(err).NotTo(HaveOccurred())
			Expect(sql).To(ContainSubstring("starred"))
		})

		It("applies StartIndex/Limit to the merged multi-type result set", func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{{ID: "s1", Title: "Song"}, {ID: "s2", Title: "Song2"}})
			ds.Album(context.Background()).(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "a1", Name: "One"}, {ID: "a2", Name: "Two"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items?IncludeItemTypes=Audio,MusicAlbum&StartIndex=1&Limit=2", nil).WithContext(ctxUser())
			api.getItems(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).To(HaveLen(2))
			Expect(res.TotalRecordCount).To(Equal(4))
			Expect(res.StartIndex).To(Equal(1))
		})

		It("applies a starred filter when Filters=IsFavorite", func() {
			albumRepo := ds.Album(context.Background()).(*tests.MockAlbumRepo)
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
			albumRepo := ds.Album(context.Background()).(*tests.MockAlbumRepo)
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
			albumRepo := ds.Album(context.Background()).(*tests.MockAlbumRepo)
			albumRepo.SetData(model.Albums{{ID: "a1", Name: "One"}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items?IncludeItemTypes=MusicAlbum&StartIndex=5&Limit=10", nil).WithContext(ctxUser())
			api.getItems(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(albumRepo.Options.Offset).To(Equal(5))
			Expect(albumRepo.Options.Max).To(Equal(10))
		})

		Describe("sorting", func() {
			It("maps SortBy=PlayCount to the play_count column", func() {
				albumRepo := ds.Album(context.Background()).(*tests.MockAlbumRepo)
				albumRepo.SetData(model.Albums{{ID: "a1", Name: "One"}})
				w := httptest.NewRecorder()
				r := httptest.NewRequest("GET", "/Items?IncludeItemTypes=MusicAlbum&SortBy=PlayCount", nil).WithContext(ctxUser())
				api.getItems(w, r)
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(albumRepo.Options.Sort).To(Equal("play_count"))
			})

			It("maps SortBy=DatePlayed to the play_date column", func() {
				mfRepo := ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo)
				mfRepo.SetData(model.MediaFiles{{ID: "s1", Title: "Song"}})
				w := httptest.NewRecorder()
				r := httptest.NewRequest("GET", "/Items?IncludeItemTypes=Audio&SortBy=DatePlayed", nil).WithContext(ctxUser())
				api.getItems(w, r)
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(mfRepo.Options.Sort).To(Equal("play_date"))
			})

			It("uses the first recognized key in a comma-separated SortBy list", func() {
				albumRepo := ds.Album(context.Background()).(*tests.MockAlbumRepo)
				albumRepo.SetData(model.Albums{{ID: "a1", Name: "One"}})
				w := httptest.NewRecorder()
				r := httptest.NewRequest("GET", "/Items?IncludeItemTypes=MusicAlbum&SortBy=DateCreated,SortName", nil).WithContext(ctxUser())
				api.getItems(w, r)
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(albumRepo.Options.Sort).To(Equal("recently_added"))
			})

			It("skips unrecognized keys in a comma-separated SortBy list to find one that is", func() {
				mfRepo := ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo)
				mfRepo.SetData(model.MediaFiles{{ID: "s1", Title: "Song"}})
				w := httptest.NewRecorder()
				r := httptest.NewRequest("GET", "/Items?IncludeItemTypes=Audio&SortBy=ParentIndexNumber,IndexNumber,SortName", nil).WithContext(ctxUser())
				api.getItems(w, r)
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(mfRepo.Options.Sort).To(Equal("title"))
			})

			It("leaves Sort at the repo default when no SortBy key is recognized", func() {
				albumRepo := ds.Album(context.Background()).(*tests.MockAlbumRepo)
				albumRepo.SetData(model.Albums{{ID: "a1", Name: "One"}})
				w := httptest.NewRecorder()
				r := httptest.NewRequest("GET", "/Items?IncludeItemTypes=MusicAlbum&SortBy=SeriesSortName", nil).WithContext(ctxUser())
				api.getItems(w, r)
				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(albumRepo.Options.Sort).To(Equal(""))
			})
		})

		Describe("library scoping", func() {
			It("scopes a MusicAlbum listing (no ParentId) to the user's accessible libraries", func() {
				albumRepo := ds.Album(context.Background()).(*tests.MockAlbumRepo)
				albumRepo.SetData(model.Albums{{ID: "a1", Name: "One"}})
				w := httptest.NewRecorder()
				libs := model.Libraries{{ID: 1}, {ID: 2}}
				r := httptest.NewRequest("GET", "/Items?IncludeItemTypes=MusicAlbum", nil).WithContext(ctxUserWithLibraries(libs))
				api.getItems(w, r)
				Expect(w.Code).To(Equal(http.StatusOK))
				sql, args, err := albumRepo.Options.Filters.ToSql()
				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring("library_id"))
				Expect(args).To(ContainElements(1, 2))
			})

			It("scopes a Audio listing (no ParentId) to the user's accessible libraries", func() {
				mfRepo := ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo)
				mfRepo.SetData(model.MediaFiles{{ID: "s1", Title: "Song"}})
				w := httptest.NewRecorder()
				libs := model.Libraries{{ID: 1}, {ID: 2}}
				r := httptest.NewRequest("GET", "/Items?IncludeItemTypes=Audio", nil).WithContext(ctxUserWithLibraries(libs))
				api.getItems(w, r)
				Expect(w.Code).To(Equal(http.StatusOK))
				sql, args, err := mfRepo.Options.Filters.ToSql()
				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring("library_id"))
				Expect(args).To(ContainElements(1, 2))
			})

			It("scopes a MusicArtist listing to the user's accessible libraries", func() {
				artistRepo := ds.Artist(context.Background()).(*tests.MockArtistRepo)
				artistRepo.SetData(model.Artists{{ID: "ar1", Name: "Artist"}})
				w := httptest.NewRecorder()
				libs := model.Libraries{{ID: 1}, {ID: 2}}
				r := httptest.NewRequest("GET", "/Items?IncludeItemTypes=MusicArtist", nil).WithContext(ctxUserWithLibraries(libs))
				api.getItems(w, r)
				Expect(w.Code).To(Equal(http.StatusOK))
				sql, args, err := artistRepo.Options.Filters.ToSql()
				Expect(err).NotTo(HaveOccurred())
				Expect(sql).To(ContainSubstring("library_artist.library_id"))
				Expect(args).To(ContainElements(1, 2))
			})

			It("treats a numeric ParentId matching an accessible library as a library scope, not an artist id", func() {
				albumRepo := ds.Album(context.Background()).(*tests.MockAlbumRepo)
				albumRepo.SetData(model.Albums{{ID: "a1", Name: "One"}})
				w := httptest.NewRecorder()
				libs := model.Libraries{{ID: 1}, {ID: 2}}
				r := httptest.NewRequest("GET", "/Items?ParentId=2&IncludeItemTypes=MusicAlbum", nil).WithContext(ctxUserWithLibraries(libs))
				api.getItems(w, r)
				Expect(w.Code).To(Equal(http.StatusOK))
				sql, args, err := albumRepo.Options.Filters.ToSql()
				Expect(err).NotTo(HaveOccurred())
				Expect(sql).NotTo(ContainSubstring("json_tree")) // not treated as an artist-parent filter
				Expect(sql).To(ContainSubstring("library_id"))
				Expect(args).To(ContainElement(2))
			})

			It("does not let ParentId=<inaccessible library id> scope results to that library", func() {
				albumRepo := ds.Album(context.Background()).(*tests.MockAlbumRepo)
				albumRepo.SetData(model.Albums{{ID: "a1", Name: "One"}})
				w := httptest.NewRecorder()
				libs := model.Libraries{{ID: 1}} // no access to library 99
				r := httptest.NewRequest("GET", "/Items?ParentId=99&IncludeItemTypes=MusicAlbum", nil).WithContext(ctxUserWithLibraries(libs))
				api.getItems(w, r)
				Expect(w.Code).To(Equal(http.StatusOK))
				sql, args, err := albumRepo.Options.Filters.ToSql()
				Expect(err).NotTo(HaveOccurred())
				// Falls back to treating "99" as an (empty-matching) artist-parent id...
				Expect(sql).To(ContainSubstring("json_tree"))
				// ...while still scoping to the user's own accessible libraries.
				Expect(sql).To(ContainSubstring("library_id"))
				Expect(args).To(ContainElement(1))
				Expect(args).NotTo(ContainElement(99))
			})

			It("does not restrict a default MusicAlbum listing for an admin user", func() {
				albumRepo := ds.Album(context.Background()).(*tests.MockAlbumRepo)
				albumRepo.SetData(model.Albums{{ID: "a1", Name: "One", LibraryID: 1}, {ID: "a2", Name: "Two", LibraryID: 2}})
				w := httptest.NewRecorder()
				r := httptest.NewRequest("GET", "/Items?IncludeItemTypes=MusicAlbum", nil).WithContext(ctxAdmin())
				api.getItems(w, r)
				Expect(w.Code).To(Equal(http.StatusOK))
				// accessibleLibraryIDs is empty for an admin (Libraries is nil), so
				// ApplyLibraryFilter([]) is a no-op: no library_id restriction is added.
				if albumRepo.Options.Filters == nil {
					return
				}
				sql, _, err := albumRepo.Options.Filters.ToSql()
				Expect(err).NotTo(HaveOccurred())
				Expect(sql).NotTo(ContainSubstring("library_id"))
			})
		})
	})

	Describe("getItem", func() {
		It("returns an album by id", func() {
			ds.Album(context.Background()).(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "a1", Name: "One", LibraryID: 1}})
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

		It("returns 404 for an album in a library the user can't access", func() {
			ds.Album(context.Background()).(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "a1", Name: "One", LibraryID: 2}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items/a1", nil).WithContext(ctxUser()) // only has access to library 1
			r = withChiURLParam(r, "itemId", "a1")
			api.getItem(w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("returns 404 for a song in a library the user can't access", func() {
			ds.MediaFile(context.Background()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{{ID: "s1", Title: "Song", LibraryID: 2}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items/s1", nil).WithContext(ctxUser()) // only has access to library 1
			r = withChiURLParam(r, "itemId", "s1")
			api.getItem(w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("returns an album to an admin even when it's outside their (empty) Libraries", func() {
			ds.Album(context.Background()).(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "a1", Name: "One", LibraryID: 2}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Items/a1", nil).WithContext(ctxAdmin()) // admin, Libraries: nil
			r = withChiURLParam(r, "itemId", "a1")
			api.getItem(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var item dto.BaseItemDto
			Expect(json.Unmarshal(w.Body.Bytes(), &item)).To(Succeed())
			Expect(item.Id).To(Equal("a1"))
		})
	})

	Describe("getLatest", func() {
		It("returns a bare array of the newest albums", func() {
			ds.Album(context.Background()).(*tests.MockAlbumRepo).SetData(model.Albums{{ID: "a1", Name: "One", LibraryID: 1}})
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/Users/u1/Items/Latest", nil).WithContext(ctxUser())
			api.getLatest(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var items []dto.BaseItemDto
			Expect(json.Unmarshal(w.Body.Bytes(), &items)).To(Succeed())
			Expect(items).To(HaveLen(1))
			Expect(items[0].Id).To(Equal("a1"))
		})

		It("scopes to the user's accessible libraries", func() {
			albumRepo := ds.Album(context.Background()).(*tests.MockAlbumRepo)
			albumRepo.SetData(model.Albums{{ID: "a1", Name: "One", LibraryID: 1}})
			w := httptest.NewRecorder()
			libs := model.Libraries{{ID: 1}, {ID: 2}}
			r := httptest.NewRequest("GET", "/Users/u1/Items/Latest", nil).WithContext(ctxUserWithLibraries(libs))
			api.getLatest(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			sql, args, err := albumRepo.Options.Filters.ToSql()
			Expect(err).NotTo(HaveOccurred())
			Expect(sql).To(ContainSubstring("library_id"))
			Expect(args).To(ContainElements(1, 2))
		})
	})
})
