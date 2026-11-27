package jellyfin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Browsing", func() {
	var ctx context.Context
	var api *Router
	var ds model.DataStore
	alice := model.User{ID: testID("u1"), UserName: "alice"}
	admin := model.User{ID: testID("admin"), UserName: "admin", IsAdmin: true}

	// ctxUser grants alice exactly libs, in the DB and in the request user.
	ctxUser := func(libs model.Libraries) context.Context {
		ids := make([]int, len(libs))
		for i, lib := range libs {
			ids[i] = lib.ID
		}
		Expect(ds.User().SetUserLibraries(ctx, alice.ID, ids)).To(Succeed())
		u := alice
		u.Libraries = libs
		return request.WithUser(ctx, u)
	}

	// admin has no explicit Libraries; access is granted via the IsAdmin bypass, not membership.
	ctxAdmin := func() context.Context {
		u := admin
		u.Libraries = nil
		return request.WithUser(ctx, u)
	}

	get := func(handler http.HandlerFunc, url string, ctx context.Context) dto.QueryResult {
		GinkgoHelper()
		w := httptest.NewRecorder()
		invoke(handler, w, httptest.NewRequestWithContext(ctx, http.MethodGet, url, nil).WithContext(ctx))
		Expect(w.Code).To(Equal(http.StatusOK))
		var res dto.QueryResult
		Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
		return res
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
		putUser(ctx, ds, alice)
		putUser(ctx, ds, admin)
		persistencetest.SetLibraries(ctx, ds, "/music1", "/music2", "/music3")
		api = &Router{ds: ds}
	})

	Describe("getArtists", func() {
		It("lists artists via /Artists", func() {
			putArtists(ctx, ds, model.Artist{ID: testID("ar1"), Name: "A"})
			res := get(api.getArtists, "/Artists", ctxUser(model.Libraries{{ID: 1}}))
			Expect(res.Items).To(HaveLen(1))
			Expect(res.Items[0].Type).To(Equal("MusicArtist"))
		})

		It("lists only album artists on /Artists/AlbumArtists", func() {
			putArtists(ctx, ds, model.Artist{ID: testID("ar1"), Name: "A"})
			performer := model.Artist{ID: testID("ar2"), Name: "Performer", OrderArtistName: "performer"}
			persistencetest.PutArtists(ctx, ds, 1, performer)
			persistencetest.SetArtistStats(ctx, 1, map[model.Role]model.ArtistStats{model.RoleArtist: {SongCount: 1}}, performer.ID)
			res := get(api.getAlbumArtists, "/Artists/AlbumArtists", ctxUser(model.Libraries{{ID: 1}}))
			Expect(itemNames(res)).To(ConsistOf("A"))
		})

		Context("with one artist in each library", func() {
			BeforeEach(func() {
				putArtistsIn(ctx, ds, 1, model.Artist{ID: testID("ar1"), Name: "One"})
				putArtistsIn(ctx, ds, 2, model.Artist{ID: testID("ar2"), Name: "Two"})
				putArtistsIn(ctx, ds, 3, model.Artist{ID: testID("ar3"), Name: "Three"})
			})

			It("scopes results to the user's accessible libraries", func() {
				res := get(api.getArtists, "/Artists", ctxUser(model.Libraries{{ID: 1}, {ID: 2}}))
				Expect(itemNames(res)).To(ConsistOf("One", "Two"))
			})

			It("scopes to a single library when ParentId is an accessible library id", func() {
				res := get(api.getArtists, "/Artists?ParentId="+dto.EncodeLibraryID(2), ctxUser(model.Libraries{{ID: 1}, {ID: 2}}))
				Expect(itemNames(res)).To(ConsistOf("Two"))
			})

			It("does not let ParentId=<inaccessible library id> narrow the scope", func() {
				res := get(api.getArtists, "/Artists?ParentId="+dto.EncodeLibraryID(3), ctxUser(model.Libraries{{ID: 1}}))
				Expect(itemNames(res)).To(ConsistOf("One"))
			})

			It("does not restrict results for an admin user", func() {
				res := get(api.getArtists, "/Artists", ctxAdmin())
				Expect(itemNames(res)).To(ConsistOf("One", "Two", "Three"))
			})
		})

		It("forwards SearchTerm to the repo's Search method", func() {
			putArtists(ctx, ds, model.Artist{ID: testID("ar1"), Name: "Artist"}, model.Artist{ID: testID("ar2"), Name: "Other"})
			res := get(api.getArtists, "/Artists?SearchTerm=art", ctxUser(model.Libraries{{ID: 1}}))
			Expect(itemNames(res)).To(ConsistOf("Artist"))
		})

		It("bounds a search the client left unbounded, and clamps an oversized one", func() {
			artists := make([]model.Artist, maxSearchLimit+1)
			for i := range artists {
				artists[i] = model.Artist{ID: testID(fmt.Sprintf("ar%05d", i)), Name: fmt.Sprintf("Artist %05d", i)}
			}
			putArtists(ctx, ds, artists...)

			res := get(api.getArtists, "/Artists?SearchTerm=art", ctxUser(model.Libraries{{ID: 1}}))
			Expect(res.Items).To(HaveLen(defaultSearchLimit))
			Expect(res.TotalRecordCount).To(Equal(defaultSearchLimit + 1))

			res = get(api.getArtists, "/Artists?SearchTerm=art&Limit=999999", ctxUser(model.Libraries{{ID: 1}}))
			Expect(res.Items).To(HaveLen(maxSearchLimit))
			Expect(res.TotalRecordCount).To(Equal(maxSearchLimit + 1))
		})

		It("forwards StartIndex/Limit as Offset/Max", func() {
			var artists []model.Artist
			for i := range 20 {
				name := fmt.Sprintf("Artist %02d", i)
				artists = append(artists, model.Artist{ID: testID(name), Name: name})
			}
			putArtists(ctx, ds, artists...)
			all := itemNames(get(api.getArtists, "/Artists", ctxUser(model.Libraries{{ID: 1}})))
			Expect(all).To(HaveLen(20))

			res := get(api.getArtists, "/Artists?StartIndex=5&Limit=10", ctxUser(model.Libraries{{ID: 1}}))
			Expect(itemNames(res)).To(Equal(all[5:15]))
			Expect(res.TotalRecordCount).To(Equal(20))
		})

		DescribeTable("restricts to favorites",
			func(url string, handler func(*Router) http.HandlerFunc) {
				putArtists(ctx, ds,
					model.Artist{ID: testID("ar1"), Name: "Starred"},
					model.Artist{ID: testID("ar2"), Name: "Plain"},
					model.Artist{ID: testID("ar3"), Name: "Starred Missing", Missing: true},
				)
				userCtx := ctxUser(model.Libraries{{ID: 1}})
				Expect(ds.Artist().SetStar(userCtx, true, testID("ar1"), testID("ar3"))).To(Succeed())
				res := get(handler(api), url, userCtx)
				// listArtists always ANDs notMissing, favorites filter or not.
				Expect(itemNames(res)).To(ConsistOf("Starred"))
			},
			Entry("Filters=IsFavorite", "/Artists?Filters=IsFavorite",
				func(a *Router) http.HandlerFunc { return a.getArtists }),
			Entry("isFavorite=true", "/Artists?isFavorite=true",
				func(a *Router) http.HandlerFunc { return a.getArtists }),
			Entry("on /Artists/AlbumArtists", "/Artists/AlbumArtists?Filters=IsFavorite",
				func(a *Router) http.HandlerFunc { return a.getAlbumArtists }),
		)

		It("404s a malformed ParentId instead of listing every library's artists", func() {
			putArtists(ctx, ds, model.Artist{ID: testID("ar1"), Name: "Artist"})
			w := httptest.NewRecorder()
			r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/Artists?ParentId=not-a-valid-id", nil).WithContext(ctxUser(model.Libraries{{ID: 1}}))
			invoke(api.getArtists, w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("getGenres", func() {
		It("lists genres via /Genres", func() {
			res := get(api.getGenres, "/Genres", ctxUser(model.Libraries{{ID: 1}}))
			Expect(res.Items).NotTo(BeNil())
		})

		It("handles /MusicGenres the same way", func() {
			get(api.getGenres, "/MusicGenres", ctxUser(model.Libraries{{ID: 1}}))
		})

		It("treats a negative StartIndex as 0 instead of panicking", func() {
			res := get(api.getGenres, "/Genres?StartIndex=-1&Limit=5", ctxUser(model.Libraries{{ID: 1}}))
			Expect(res.StartIndex).To(Equal(0))
		})
	})

	// putTagPerLibrary stores one tag named after each library it is in.
	putTagPerLibrary := func(name model.TagName, prefix string) {
		for _, libID := range []int{1, 2, 3} {
			Expect(ds.Tag().Add(ctx, libID, model.NewTag(name, fmt.Sprintf("%s%d", prefix, libID)))).To(Succeed())
		}
	}

	Describe("getStudios", func() {
		BeforeEach(func() {
			putTagPerLibrary(model.TagRecordLabel, "Label ")
		})

		It("scopes results to the user's accessible libraries", func() {
			res := get(api.getStudios, "/Studios", ctxUser(model.Libraries{{ID: 1}, {ID: 2}}))
			Expect(itemNames(res)).To(ConsistOf("Label 1", "Label 2"))
		})

		// An empty scope (admin, or a non-admin with no explicit library grants) must be treated
		// as unrestricted, matching accessibleLibraryIDs' documented contract, not as "match nothing".
		It("does not restrict results for an admin user", func() {
			res := get(api.getStudios, "/Studios", ctxAdmin())
			Expect(itemNames(res)).To(ConsistOf("Label 1", "Label 2", "Label 3"))
		})

		It("404s a malformed ParentId instead of listing every library's studios", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/Studios?ParentId=not-a-valid-id", nil).WithContext(ctxUser(model.Libraries{{ID: 1}}))
			invoke(api.getStudios, w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("treats a negative StartIndex as 0 instead of panicking", func() {
			res := get(api.getStudios, "/Studios?StartIndex=-1&Limit=5", ctxUser(model.Libraries{{ID: 1}}))
			Expect(res.StartIndex).To(Equal(0))
		})
	})

	Describe("getQueryFiltersLegacy", func() {
		filters := func(ctx context.Context) dto.QueryFiltersLegacy {
			GinkgoHelper()
			w := httptest.NewRecorder()
			invoke(api.getQueryFiltersLegacy, w, httptest.NewRequestWithContext(ctx, http.MethodGet, "/Items/Filters", nil).WithContext(ctx))
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryFiltersLegacy
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			return res
		}

		BeforeEach(func() {
			putTagPerLibrary(model.TagGenre, "Genre ")
		})

		It("scopes genres to the user's accessible libraries", func() {
			Expect(filters(ctxUser(model.Libraries{{ID: 1}, {ID: 2}})).Genres).To(ConsistOf("Genre 1", "Genre 2"))
		})

		It("does not restrict genres for an admin user", func() {
			Expect(filters(ctxAdmin()).Genres).To(ConsistOf("Genre 1", "Genre 2", "Genre 3"))
		})

		It("404s a malformed ParentId instead of listing every library's filters", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/Items/Filters?ParentId=not-a-valid-id", nil).WithContext(ctxUser(model.Libraries{{ID: 1}}))
			invoke(api.getQueryFiltersLegacy, w, r)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})
})

func itemNames(res dto.QueryResult) []string {
	names := make([]string, len(res.Items))
	for i, it := range res.Items {
		names[i] = it.Name
	}
	return names
}
