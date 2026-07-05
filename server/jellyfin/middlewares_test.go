package jellyfin

import (
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("authenticate middleware", func() {
	var api *Router
	var ds *tests.MockDataStore
	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		auth.Init(ds)
		ur := ds.User(nil).(*tests.MockedUserRepo)
		Expect(ur.Put(&model.User{ID: "u1", UserName: "alice", NewPassword: "secret"})).To(Succeed())
		api = &Router{ds: ds}
	})

	tokenFor := func(name string) string {
		t, err := auth.CreateToken(&model.User{ID: "u1", UserName: name})
		Expect(err).ToNot(HaveOccurred())
		return t
	}

	It("passes with a valid X-Emby-Token and injects the user", func() {
		var gotUser model.User
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotUser, _ = request.UserFrom(r.Context())
			w.WriteHeader(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/Items", nil)
		r.Header.Set("X-Emby-Token", tokenFor("alice"))
		api.authenticate(next).ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(gotUser.UserName).To(Equal("alice"))
	})

	It("rejects a missing token with 401", func() {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/Items", nil)
		api.authenticate(next).ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})
})
