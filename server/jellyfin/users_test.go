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
})
