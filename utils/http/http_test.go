package http

import (
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Http Utils", func() {
	var w *httptest.ResponseRecorder
	var r *http.Request
	var token string

	BeforeEach(func() {
		w = httptest.NewRecorder()
		r = httptest.NewRequest("GET", "/api/events", nil)
		token = "fake-jwt-token" // #nosec G101
	})

	Describe("SetJWTEventCookie", func() {
		It("sets Secure=false and SameSite=LaxMode", func() {
			conf.Server.BaseScheme = "http"
			conf.Server.BaseHost = ""

			SetJWTEventCookie(w, r, token)
			cookie := w.Result().Cookies()[0]

			Expect(cookie.Name).To(Equal(consts.JWTCookie))
			Expect(cookie.Value).To(Equal(token))
			Expect(cookie.Secure).To(BeFalse())
			Expect(cookie.SameSite).To(Equal(http.SameSiteLaxMode))
			Expect(cookie.Path).To(Equal("/api/events"))
			Expect(cookie.HttpOnly).To(BeTrue())
		})

		It("sets Secure=true and SameSite=NoneMode", func() {
			conf.Server.BaseScheme = "https"
			conf.Server.BaseHost = "example.com"

			SetJWTEventCookie(w, r, token)
			cookie := w.Result().Cookies()[0]

			Expect(cookie.Secure).To(BeTrue())
			Expect(cookie.SameSite).To(Equal(http.SameSiteNoneMode))
			Expect(cookie.Path).To(Equal("/api/events"))
			Expect(cookie.HttpOnly).To(BeTrue())
		})
	})

	Describe("setSSECORSHeaders", func() {
		It("sets correct headers", func() {
			conf.Server.BaseScheme = "http"
			conf.Server.BaseHost = ""
			r.Host = "localhost:5173"

			SetSSECORSHeaders(w, r)

			Expect(w.Header().Get("Access-Control-Allow-Origin")).To(Equal("http://localhost:5173"))
			Expect(w.Header().Get("Access-Control-Allow-Credentials")).To(Equal("true"))
		})
	})
})
