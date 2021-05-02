package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"

	"github.com/navidrome/navidrome/consts"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Auth", func() {
	Describe("Public functions", func() {
		var ds model.DataStore
		var req *http.Request
		var resp *httptest.ResponseRecorder

		BeforeEach(func() {
			ds = &tests.MockDataStore{}
		})

		Describe("CreateAdmin", func() {
			BeforeEach(func() {
				req = httptest.NewRequest("POST", "/createAdmin", strings.NewReader(`{"username":"johndoe", "password":"secret"}`))
				resp = httptest.NewRecorder()
				CreateAdmin(ds)(resp, req)
			})

			It("creates an admin user with the specified password", func() {
				usr := ds.User(context.TODO())
				u, err := usr.FindByUsername("johndoe")
				Expect(err).To(BeNil())
				Expect(u.Password).ToNot(BeEmpty())
				Expect(u.IsAdmin).To(BeTrue())
			})

			It("returns the expected payload", func() {
				Expect(resp.Code).To(Equal(http.StatusOK))
				var parsed map[string]interface{}
				Expect(json.Unmarshal(resp.Body.Bytes(), &parsed)).To(BeNil())
				Expect(parsed["isAdmin"]).To(Equal(true))
				Expect(parsed["username"]).To(Equal("johndoe"))
				Expect(parsed["name"]).To(Equal("Johndoe"))
				Expect(parsed["id"]).ToNot(BeEmpty())
				Expect(parsed["token"]).ToNot(BeEmpty())
			})
		})
		Describe("Login", func() {
			BeforeEach(func() {
				req = httptest.NewRequest("POST", "/login", strings.NewReader(`{"username":"janedoe", "password":"abc123"}`))
				resp = httptest.NewRecorder()
			})

			It("fails if user does not exist", func() {
				Login(ds)(resp, req)
				Expect(resp.Code).To(Equal(http.StatusUnauthorized))
			})

			It("logs in successfully if user exists", func() {
				usr := ds.User(context.TODO())
				_ = usr.Put(&model.User{ID: "111", UserName: "janedoe", NewPassword: "abc123", Name: "Jane", IsAdmin: false})

				Login(ds)(resp, req)
				Expect(resp.Code).To(Equal(http.StatusOK))

				var parsed map[string]interface{}
				Expect(json.Unmarshal(resp.Body.Bytes(), &parsed)).To(BeNil())
				Expect(parsed["isAdmin"]).To(Equal(false))
				Expect(parsed["username"]).To(Equal("janedoe"))
				Expect(parsed["name"]).To(Equal("Jane"))
				Expect(parsed["id"]).ToNot(BeEmpty())
				Expect(parsed["token"]).ToNot(BeEmpty())
			})
		})
	})

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
