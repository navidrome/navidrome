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
	authed := func(r *http.Request) *http.Request {
		ctx := request.WithUser(context.Background(), model.User{ID: "u1", UserName: "alice"})
		return r.WithContext(ctx)
	}
	BeforeEach(func() { api = &Router{ds: &tests.MockDataStore{}} })

	It("returns a single music view", func() {
		w := httptest.NewRecorder()
		api.getUserViews(w, authed(httptest.NewRequest("GET", "/UserViews", nil)))
		Expect(w.Code).To(Equal(http.StatusOK))
		var res dto.QueryResult
		Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
		Expect(res.Items).To(HaveLen(1))
		Expect(res.Items[0].CollectionType).To(Equal("music"))
		Expect(res.Items[0].Type).To(Equal("CollectionFolder"))
	})

	It("returns the current user", func() {
		w := httptest.NewRecorder()
		api.getCurrentUser(w, authed(httptest.NewRequest("GET", "/Users/Me", nil)))
		var u dto.UserDto
		Expect(json.Unmarshal(w.Body.Bytes(), &u)).To(Succeed())
		Expect(u.Name).To(Equal("alice"))
	})
})
