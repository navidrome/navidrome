package jellyfin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Users", func() {
	var api *Router
	authedWithLibraries := func(r *http.Request, libs model.Libraries) *http.Request {
		ctx := request.WithUser(context.Background(), model.User{ID: "u1", UserName: "alice", Libraries: libs})
		return r.WithContext(ctx)
	}
	BeforeEach(func() { api = &Router{ds: &tests.MockDataStore{}} })

	Describe("getUserViews", func() {
		It("returns one view per accessible library", func() {
			libs := model.Libraries{{ID: 1, Name: "Music"}, {ID: 2, Name: "Podcasts"}}
			w := httptest.NewRecorder()
			api.getUserViews(w, authedWithLibraries(httptest.NewRequest("GET", "/UserViews", nil), libs))
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).To(HaveLen(2))
			Expect(res.TotalRecordCount).To(Equal(2))

			Expect(res.Items[0].Id).To(Equal(dto.EncodeID("1")))
			Expect(res.Items[0].Name).To(Equal("Music"))
			Expect(res.Items[0].Type).To(Equal("CollectionFolder"))
			Expect(res.Items[0].CollectionType).To(Equal("music"))
			Expect(res.Items[0].IsFolder).To(BeTrue())

			Expect(res.Items[1].Id).To(Equal(dto.EncodeID("2")))
			Expect(res.Items[1].Name).To(Equal("Podcasts"))
		})

		It("returns a single view for a user with one library", func() {
			libs := model.Libraries{{ID: 1, Name: "Music"}}
			w := httptest.NewRecorder()
			api.getUserViews(w, authedWithLibraries(httptest.NewRequest("GET", "/UserViews", nil), libs))
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).To(HaveLen(1))
			Expect(res.Items[0].Id).To(Equal(dto.EncodeID("1")))
		})

		It("returns no views for a user with no library access", func() {
			w := httptest.NewRecorder()
			api.getUserViews(w, authedWithLibraries(httptest.NewRequest("GET", "/UserViews", nil), nil))
			Expect(w.Code).To(Equal(http.StatusOK))
			var res dto.QueryResult
			Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
			Expect(res.Items).To(HaveLen(0))
			Expect(res.TotalRecordCount).To(Equal(0))
		})
	})

	It("returns the current user", func() {
		w := httptest.NewRecorder()
		api.getCurrentUser(w, authedWithLibraries(httptest.NewRequest("GET", "/Users/Me", nil), nil))
		var u dto.UserDto
		Expect(json.Unmarshal(w.Body.Bytes(), &u)).To(Succeed())
		Expect(u.Name).To(Equal("alice"))
		Expect(u.Policy).ToNot(BeNil())
		Expect(u.Policy.IsAdministrator).To(BeFalse())
		Expect(u.Configuration).ToNot(BeNil())
	})

	Describe("getPublicUsers", func() {
		var ur *tests.MockedUserRepo
		publicUsers := func() []dto.UserDto {
			w := httptest.NewRecorder()
			api.getPublicUsers(w, httptest.NewRequest("GET", "/Users/Public", nil))
			Expect(w.Code).To(Equal(http.StatusOK))
			var users []dto.UserDto
			Expect(json.Unmarshal(w.Body.Bytes(), &users)).To(Succeed())
			return users
		}

		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			ur = api.ds.User(context.Background()).(*tests.MockedUserRepo)
			Expect(ur.Put(&model.User{ID: "u1", UserName: "alice"})).To(Succeed())
			Expect(ur.Put(&model.User{ID: "u2", UserName: "bob"})).To(Succeed())
		})

		It("returns an empty list when the config is unset", func() {
			conf.Server.Jellyfin.ExposedPublicUsers = ""
			Expect(publicUsers()).To(BeEmpty())
		})

		It("lists the configured users in order, without leaking policy", func() {
			conf.Server.Jellyfin.ExposedPublicUsers = "bob, alice"
			users := publicUsers()
			Expect(users).To(HaveLen(2))
			Expect(users[0].Name).To(Equal("bob"))
			Expect(users[0].Id).To(Equal(dto.EncodeID("u2")))
			Expect(users[1].Name).To(Equal("alice"))
			// The public list must not expose Policy/Configuration to unauthenticated callers.
			Expect(users[0].Policy).To(BeNil())
			Expect(users[0].Configuration).To(BeNil())
		})

		It("skips a configured username that does not exist", func() {
			conf.Server.Jellyfin.ExposedPublicUsers = "alice,ghost"
			users := publicUsers()
			Expect(users).To(HaveLen(1))
			Expect(users[0].Name).To(Equal("alice"))
		})

		It("matches usernames case-insensitively and de-duplicates", func() {
			conf.Server.Jellyfin.ExposedPublicUsers = "ALICE, alice"
			users := publicUsers()
			Expect(users).To(HaveLen(1))
			Expect(users[0].Name).To(Equal("alice"))
		})
	})
})
