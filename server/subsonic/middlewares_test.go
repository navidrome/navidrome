package subsonic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/deluan/navidrome/engine"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func newGetRequest(queryParams ...string) *http.Request {
	r := httptest.NewRequest("GET", "/ping?"+strings.Join(queryParams, "&"), nil)
	ctx := r.Context()
	return r.WithContext(log.NewContext(ctx))
}

func newPostRequest(queryParam string, formFields ...string) *http.Request {
	r, err := http.NewRequest("POST", "/ping?"+queryParam, strings.NewReader(strings.Join(formFields, "&")))
	if err != nil {
		panic(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	ctx := r.Context()
	return r.WithContext(log.NewContext(ctx))
}

var _ = Describe("Middlewares", func() {
	var next *mockHandler
	var w *httptest.ResponseRecorder

	BeforeEach(func() {
		next = &mockHandler{}
		w = httptest.NewRecorder()
	})

	Describe("ParsePostForm", func() {
		It("converts any filed in a x-www-form-urlencoded POST into query params", func() {
			r := newPostRequest("a=abc", "u=user", "v=1.15", "c=test")
			cp := postFormToQueryParams(next)
			cp.ServeHTTP(w, r)

			Expect(next.req.URL.Query().Get("a")).To(Equal("abc"))
			Expect(next.req.URL.Query().Get("u")).To(Equal("user"))
			Expect(next.req.URL.Query().Get("v")).To(Equal("1.15"))
			Expect(next.req.URL.Query().Get("c")).To(Equal("test"))
		})
		It("adds repeated params", func() {
			r := newPostRequest("a=abc", "id=1", "id=2")
			cp := postFormToQueryParams(next)
			cp.ServeHTTP(w, r)

			Expect(next.req.URL.Query().Get("a")).To(Equal("abc"))
			Expect(next.req.URL.Query()["id"]).To(ConsistOf("1", "2"))
		})
		It("overrides query params with same key", func() {
			r := newPostRequest("a=query", "a=body")
			cp := postFormToQueryParams(next)
			cp.ServeHTTP(w, r)

			Expect(next.req.URL.Query().Get("a")).To(Equal("body"))
		})
	})

	Describe("CheckParams", func() {
		It("passes when all required params are available", func() {
			r := newGetRequest("u=user", "v=1.15", "c=test")
			cp := checkRequiredParameters(next)
			cp.ServeHTTP(w, r)

			Expect(next.req.Context().Value("username")).To(Equal("user"))
			Expect(next.req.Context().Value("version")).To(Equal("1.15"))
			Expect(next.req.Context().Value("client")).To(Equal("test"))
			Expect(next.called).To(BeTrue())
		})

		It("fails when user is missing", func() {
			r := newGetRequest("v=1.15", "c=test")
			cp := checkRequiredParameters(next)
			cp.ServeHTTP(w, r)

			Expect(w.Body.String()).To(ContainSubstring(`code="10"`))
			Expect(next.called).To(BeFalse())
		})

		It("fails when version is missing", func() {
			r := newGetRequest("u=user", "c=test")
			cp := checkRequiredParameters(next)
			cp.ServeHTTP(w, r)

			Expect(w.Body.String()).To(ContainSubstring(`code="10"`))
			Expect(next.called).To(BeFalse())
		})

		It("fails when client is missing", func() {
			r := newGetRequest("u=user", "v=1.15")
			cp := checkRequiredParameters(next)
			cp.ServeHTTP(w, r)

			Expect(w.Body.String()).To(ContainSubstring(`code="10"`))
			Expect(next.called).To(BeFalse())
		})
	})

	Describe("Authenticate", func() {
		var mockedUser *mockUsers
		BeforeEach(func() {
			mockedUser = &mockUsers{}
		})

		It("passes all parameters to users.Authenticate ", func() {
			r := newGetRequest("u=valid", "p=password", "t=token", "s=salt", "jwt=jwt")
			cp := authenticate(mockedUser)(next)
			cp.ServeHTTP(w, r)

			Expect(mockedUser.username).To(Equal("valid"))
			Expect(mockedUser.password).To(Equal("password"))
			Expect(mockedUser.token).To(Equal("token"))
			Expect(mockedUser.salt).To(Equal("salt"))
			Expect(mockedUser.jwt).To(Equal("jwt"))
			Expect(next.called).To(BeTrue())
			user := next.req.Context().Value("user").(*model.User)
			Expect(user.UserName).To(Equal("valid"))
		})

		It("fails authentication with wrong password", func() {
			r := newGetRequest("u=invalid", "", "", "")
			cp := authenticate(mockedUser)(next)
			cp.ServeHTTP(w, r)

			Expect(w.Body.String()).To(ContainSubstring(`code="40"`))
			Expect(next.called).To(BeFalse())
		})
	})
})

type mockHandler struct {
	req    *http.Request
	called bool
}

func (mh *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mh.req = r
	mh.called = true
}

type mockUsers struct {
	engine.Users
	username, password, token, salt, jwt string
}

func (m *mockUsers) Authenticate(ctx context.Context, username, password, token, salt, jwt string) (*model.User, error) {
	m.username = username
	m.password = password
	m.token = token
	m.salt = salt
	m.jwt = jwt
	if username == "valid" {
		return &model.User{UserName: username, Password: password}, nil
	}
	return nil, model.ErrInvalidAuth
}
