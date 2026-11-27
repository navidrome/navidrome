package jellyfin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Users", func() {
	var ctx context.Context
	var api *Router
	// The DB holds the full rows; the user carries the id/name-only copy its projection returns.
	authedWithLibraries := func(r *http.Request, libs model.Libraries) *http.Request {
		putLibraries(ctx, api.ds, libs)
		stripped := make(model.Libraries, len(libs))
		for i, lib := range libs {
			stripped[i] = model.Library{ID: lib.ID, Name: lib.Name}
		}
		return r.WithContext(request.WithUser(ctx, model.User{ID: testID("u1"), UserName: "alice", Libraries: stripped}))
	}
	BeforeEach(func() {
		ctx = GinkgoT().Context()
		api = &Router{ds: persistence.New(db.Db())}
	})

	Describe("getUserViews", func() {
		It("returns one view per accessible library", func() {
			libs := model.Libraries{{ID: 1, Name: "Music"}, {ID: 2, Name: "Podcasts"}}
			w := httptest.NewRecorder()
			api.getUserViews(w, authedWithLibraries(httptest.NewRequest(http.MethodGet, "/UserViews", nil), libs))
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).To(HaveLen(2))
			Expect(res.TotalRecordCount).To(Equal(2))

			Expect(res.Items[0].Id).To(Equal(dto.EncodeLibraryID(1)))
			Expect(res.Items[0].Name).To(Equal("Music"))
			Expect(res.Items[0].Type).To(Equal("CollectionFolder"))
			Expect(res.Items[0].CollectionType).To(Equal("music"))
			Expect(res.Items[0].IsFolder).To(BeTrue())

			Expect(res.Items[1].Id).To(Equal(dto.EncodeLibraryID(2)))
			Expect(res.Items[1].Name).To(Equal("Podcasts"))
		})

		// Manet keeps no library, and so syncs no artists/albums/tracks, when these are missing.
		It("describes the library like Jellyfin's CollectionFolder", func() {
			created := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
			libs := model.Libraries{{ID: 1, Name: "Music", Path: "/music", TotalAlbums: 42, CreatedAt: created}}
			w := httptest.NewRecorder()
			api.getUserViews(w, authedWithLibraries(httptest.NewRequest(http.MethodGet, "/UserViews", nil), libs))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())

			item := res.Items[0]
			Expect(*item.ChildCount).To(Equal(42))
			Expect(item.DateCreated).To(Equal("2026-07-01T12:00:00.0000000Z"))
			Expect(item.SortName).To(Equal("Music"))
			Expect(item.Path).To(Equal("/music"))
			Expect(item.LocationType).To(Equal("FileSystem"))
			Expect(item.UserData).ToNot(BeNil())
			Expect(item.UserData.ItemId).To(Equal(dto.EncodeLibraryID(1)))
		})

		It("returns a single view for a user with one library", func() {
			libs := model.Libraries{{ID: 1, Name: "Music"}}
			w := httptest.NewRecorder()
			api.getUserViews(w, authedWithLibraries(httptest.NewRequest(http.MethodGet, "/UserViews", nil), libs))
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).To(HaveLen(1))
			Expect(res.Items[0].Id).To(Equal(dto.EncodeLibraryID(1)))
		})

		It("returns no views for a user with no library access", func() {
			w := httptest.NewRecorder()
			api.getUserViews(w, authedWithLibraries(httptest.NewRequest(http.MethodGet, "/UserViews", nil), nil))
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).To(BeEmpty())
			Expect(res.TotalRecordCount).To(Equal(0))
		})
	})

	It("returns the current user", func() {
		w := httptest.NewRecorder()
		api.getCurrentUser(w, authedWithLibraries(httptest.NewRequest(http.MethodGet, "/Users/Me", nil), nil))
		var u dto.UserDto
		Expect(json.Unmarshal(w.Body.Bytes(), &u)).To(Succeed())
		Expect(u.Name).To(Equal("alice"))
		Expect(u.Policy).ToNot(BeNil())
		Expect(u.Policy.IsAdministrator).To(BeFalse())
		Expect(u.Configuration).ToNot(BeNil())
	})

	Describe("getPublicUsers", func() {
		publicUsers := func() []dto.UserDto {
			w := httptest.NewRecorder()
			api.getPublicUsers(w, httptest.NewRequest(http.MethodGet, "/Users/Public", nil))
			Expect(w.Code).To(Equal(http.StatusOK))
			var users []dto.UserDto
			Expect(json.Unmarshal(w.Body.Bytes(), &users)).To(Succeed())
			return users
		}

		BeforeEach(func() {
			Expect(api.ds.User().Put(ctx, &model.User{ID: testID("u1"), UserName: "alice"})).To(Succeed())
			Expect(api.ds.User().Put(ctx, &model.User{ID: testID("u2"), UserName: "bob"})).To(Succeed())
		})

		It("returns an empty list when the config is unset", func() {
			conf.Update(func(c *conf.Options) { c.Jellyfin.ExposedPublicUsers = "" })
			Expect(publicUsers()).To(BeEmpty())
		})

		It("lists the configured users in order, without leaking policy", func() {
			conf.Update(func(c *conf.Options) { c.Jellyfin.ExposedPublicUsers = "bob, alice" })
			users := publicUsers()
			Expect(users).To(HaveLen(2))
			Expect(users[0].Name).To(Equal("bob"))
			Expect(users[0].Id).To(Equal(dto.EncodeID(testID("u2"))))
			Expect(users[1].Name).To(Equal("alice"))
			// The public list must not expose Policy/Configuration to unauthenticated callers.
			Expect(users[0].Policy).To(BeNil())
			Expect(users[0].Configuration).To(BeNil())
		})

		It("skips a configured username that does not exist", func() {
			conf.Update(func(c *conf.Options) { c.Jellyfin.ExposedPublicUsers = "alice,ghost" })
			users := publicUsers()
			Expect(users).To(HaveLen(1))
			Expect(users[0].Name).To(Equal("alice"))
		})

		It("matches usernames case-insensitively and de-duplicates", func() {
			conf.Update(func(c *conf.Options) { c.Jellyfin.ExposedPublicUsers = "ALICE, alice" })
			users := publicUsers()
			Expect(users).To(HaveLen(1))
			Expect(users[0].Name).To(Equal("alice"))
		})
	})
})

// putLibraries stores libs with their stats and creation dates, which Library().Put does not store.
func putLibraries(ctx context.Context, ds model.DataStore, libs model.Libraries) {
	GinkgoHelper()
	for _, lib := range libs {
		if lib.Path == "" {
			lib.Path = "/" + lib.Name
		}
		if lib.ID != model.DefaultLibraryID {
			Expect(ds.Library().Put(ctx, &lib)).To(Succeed())
		}
		_, err := db.Db().Exec("UPDATE library SET name = ?, path = ?, total_albums = ?, created_at = ? WHERE id = ?",
			lib.Name, lib.Path, lib.TotalAlbums, lib.CreatedAt, lib.ID)
		Expect(err).ToNot(HaveOccurred())
	}
}
