package subsonic

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/log"
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

			Expect(next.req.Context().Value("user")).To(Equal("user"))
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
		BeforeEach(func() {
			conf.Sonic.User = "admin"
			conf.Sonic.Password = "wordpass"
			conf.Sonic.DevDisableAuthentication = false
		})

		Context("Plaintext password", func() {
			It("authenticates with plaintext password ", func() {
				r := newTestRequest("u=admin", "p=wordpass")
				cp := authenticate(next)
				cp.ServeHTTP(w, r)

				Expect(next.called).To(BeTrue())
			})

			It("fails authentication with wrong password", func() {
				r := newTestRequest("u=admin", "p=INVALID")
				cp := authenticate(next)
				cp.ServeHTTP(w, r)

				Expect(w.Body.String()).To(ContainSubstring(`code="40"`))
				Expect(next.called).To(BeFalse())
			})
		})

		Context("Encoded password", func() {
			It("authenticates with simple encoded password ", func() {
				r := newTestRequest("u=admin", "p=enc:776f726470617373")
				cp := authenticate(next)
				cp.ServeHTTP(w, r)

				Expect(next.called).To(BeTrue())
			})
		})

		Context("Token based authentication", func() {
			It("authenticates with token based authentication", func() {
				r := newTestRequest("u=admin", "t=23b342970e25c7928831c3317edd0b67", "s=retnlmjetrymazgkt")
				cp := authenticate(next)
				cp.ServeHTTP(w, r)

				Expect(next.called).To(BeTrue())
			})

			It("fails if salt is missing", func() {
				r := newTestRequest("u=admin", "t=23b342970e25c7928831c3317edd0b67")
				cp := authenticate(next)
				cp.ServeHTTP(w, r)

				Expect(w.Body.String()).To(ContainSubstring(`code="40"`))
				Expect(next.called).To(BeFalse())
			})
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
