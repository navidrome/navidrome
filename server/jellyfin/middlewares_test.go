package jellyfin

import (
	"context"
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
		ur := ds.User(context.Background()).(*tests.MockedUserRepo)
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

	It("rejects a garbage token with 401 and does not call next", func() {
		nextCalled := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/Items", nil)
		r.Header.Set("X-Emby-Token", "not-a-jwt")
		api.authenticate(next).ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
		Expect(nextCalled).To(BeFalse())
	})

	It("rejects a valid token whose subject user does not exist with 401", func() {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
		t, err := auth.CreateToken(&model.User{ID: "x", UserName: "ghost"})
		Expect(err).ToNot(HaveOccurred())

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/Items", nil)
		r.Header.Set("X-Emby-Token", t)
		api.authenticate(next).ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})
})
