package app

import (
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/consts"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Auth", func() {
	Describe("mapAuthHeader", func() {
		It("maps the custom header to Authorization header", func() {
			r := httptest.NewRequest("GET", "/index.html", nil)
			r.Header.Set(consts.UIAuthorizationHeader, "test authorization bearer")
			w := httptest.NewRecorder()

			mapAuthHeader()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.Header.Get("Authorization")).To(Equal("test authorization bearer"))
				w.WriteHeader(200)
			})).ServeHTTP(w, r)

			Expect(w.Code).To(Equal(200))
		})
	})
})
