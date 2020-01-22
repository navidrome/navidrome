package subsonic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func newTestRequest(queryParams ...string) *http.Request {
	r := httptest.NewRequest("get", "/ping?"+strings.Join(queryParams, "&"), nil)
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

	Describe("CheckParams", func() {
		It("passes when all required params are available", func() {
			r := newTestRequest("u=user", "v=1.15", "c=test")
			cp := checkRequiredParameters(next)
			cp.ServeHTTP(w, r)

			Expect(next.req.Context().Value("username")).To(Equal("user"))
			Expect(next.req.Context().Value("version")).To(Equal("1.15"))
			Expect(next.req.Context().Value("client")).To(Equal("test"))
			Expect(next.called).To(BeTrue())
		})

		It("fails when user is missing", func() {
			r := newTestRequest("v=1.15", "c=test")
			cp := checkRequiredParameters(next)
			cp.ServeHTTP(w, r)

			Expect(w.Body.String()).To(ContainSubstring(`code="10"`))
			Expect(next.called).To(BeFalse())
		})

		It("fails when version is missing", func() {
			r := newTestRequest("u=user", "c=test")
			cp := checkRequiredParameters(next)
			cp.ServeHTTP(w, r)

			Expect(w.Body.String()).To(ContainSubstring(`code="10"`))
			Expect(next.called).To(BeFalse())
		})

		It("fails when client is missing", func() {
			r := newTestRequest("u=user", "v=1.15")
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
			r := newTestRequest("u=valid", "p=password", "t=token", "s=salt")
			cp := authenticate(mockedUser)(next)
			cp.ServeHTTP(w, r)

			Expect(mockedUser.username).To(Equal("valid"))
			Expect(mockedUser.password).To(Equal("password"))
			Expect(mockedUser.token).To(Equal("token"))
			Expect(mockedUser.salt).To(Equal("salt"))
			Expect(next.called).To(BeTrue())
			user := next.req.Context().Value("user").(*model.User)
			Expect(user.UserName).To(Equal("valid"))
		})

		It("fails authentication with wrong password", func() {
			r := newTestRequest("u=invalid", "", "", "")
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
	username, password, token, salt string
}

func (m *mockUsers) Authenticate(ctx context.Context, username, password, token, salt string) (*model.User, error) {
	m.username = username
	m.password = password
	m.token = token
	m.salt = salt
	if username == "valid" {
		return &model.User{UserName: username, Password: password}, nil
	}
	return nil, model.ErrInvalidAuth
}
