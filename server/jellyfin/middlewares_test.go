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

var _ = Describe("tokenFromRequest", func() {
	It("accepts the lowercase api_key query param", func() {
		r := httptest.NewRequest("GET", "/Items/s1/File?api_key=tok123", nil)
		Expect(tokenFromRequest(r)).To(Equal("tok123"))
	})

	It("accepts a PascalCase ApiKey query param once normalizeQueryKeys has folded it", func() {
		r := httptest.NewRequest("GET", "/Items/s1/File?ApiKey=tok123", nil)
		var got string
		invoke(func(_ http.ResponseWriter, r *http.Request) { got = tokenFromRequest(r) }, httptest.NewRecorder(), r)
		Expect(got).To(Equal("tok123"))
	})
})

var _ = Describe("normalizeQueryKeys", func() {
	// keyFor runs a request through normalizeQueryKeys and reports the value the handler sees for
	// the given (lowercase) key — i.e. what a case-insensitive read would find.
	keyFor := func(rawQuery, key string) string {
		r := httptest.NewRequest("GET", "/Items?"+rawQuery, nil)
		var got string
		invoke(func(_ http.ResponseWriter, r *http.Request) { got = r.URL.Query().Get(key) }, httptest.NewRecorder(), r)
		return got
	}

	It("folds PascalCase (Finamp) and camelCase (Jellify) keys to lowercase", func() {
		Expect(keyFor("ParentId=abc", "parentid")).To(Equal("abc"))
		Expect(keyFor("parentId=abc", "parentid")).To(Equal("abc"))
	})

	It("leaves values untouched", func() {
		Expect(keyFor("IncludeItemTypes=MusicAlbum,Audio", "includeitemtypes")).To(Equal("MusicAlbum,Audio"))
	})

	It("passes already-lowercase keys through unchanged", func() {
		Expect(keyFor("container=mp3", "container")).To(Equal("mp3"))
	})

	It("merges values when two keys fold to the same name instead of dropping one", func() {
		r := httptest.NewRequest("GET", "/Items?Ids=aaa&ids=bbb", nil)
		var got []string
		invoke(func(_ http.ResponseWriter, r *http.Request) { got = r.URL.Query()["ids"] }, httptest.NewRecorder(), r)
		Expect(got).To(ConsistOf("aaa", "bbb"))
	})
})
