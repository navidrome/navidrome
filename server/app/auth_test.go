package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/navidrome/navidrome/conf"
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
		Describe("Login from HTTP headers", func() {
			fs := os.DirFS("tests/fixtures")

			BeforeEach(func() {
				req = httptest.NewRequest("GET", "/index.html", nil)
				req.Header.Add("Remote-User", "janedoe")
				resp = httptest.NewRecorder()
				conf.Server.UILoginBackgroundURL = ""
				conf.Server.ReverseProxyWhitelist = "192.168.0.0/16,2001:4860:4860::/48"
			})

			It("sets auth data if IPv4 matches whitelist", func() {
				usr := ds.User(context.TODO())
				_ = usr.Put(&model.User{ID: "111", UserName: "janedoe", NewPassword: "abc123", Name: "Jane", IsAdmin: false})

				req.RemoteAddr = "192.168.0.42:25293"
				serveIndex(ds, fs)(resp, req)

				config := extractAppConfig(resp.Body.String())
				parsed := config["auth"].(map[string]interface{})

				Expect(parsed["id"]).To(Equal("111"))
			})

			It("sets no auth data if IPv4 does not match whitelist", func() {
				usr := ds.User(context.TODO())
				_ = usr.Put(&model.User{ID: "111", UserName: "janedoe", NewPassword: "abc123", Name: "Jane", IsAdmin: false})

				req.RemoteAddr = "8.8.8.8:25293"
				serveIndex(ds, fs)(resp, req)

				config := extractAppConfig(resp.Body.String())
				Expect(config["auth"]).To(BeNil())
			})

			It("sets auth data if IPv6 matches whitelist", func() {
				usr := ds.User(context.TODO())
				_ = usr.Put(&model.User{ID: "111", UserName: "janedoe", NewPassword: "abc123", Name: "Jane", IsAdmin: false})

				req.RemoteAddr = "[2001:4860:4860:1234:5678:0000:4242:8888]:25293"
				serveIndex(ds, fs)(resp, req)

				config := extractAppConfig(resp.Body.String())
				parsed := config["auth"].(map[string]interface{})

				Expect(parsed["id"]).To(Equal("111"))
			})

			It("sets no auth data if IPv6 does not match whitelist", func() {
				usr := ds.User(context.TODO())
				_ = usr.Put(&model.User{ID: "111", UserName: "janedoe", NewPassword: "abc123", Name: "Jane", IsAdmin: false})

				req.RemoteAddr = "[5005:0:3003]:25293"
				serveIndex(ds, fs)(resp, req)

				config := extractAppConfig(resp.Body.String())
				Expect(config["auth"]).To(BeNil())
			})

			It("sets auth data if user exists", func() {
				req.RemoteAddr = "192.168.0.42:25293"

				usr := ds.User(context.TODO())
				_ = usr.Put(&model.User{ID: "111", UserName: "janedoe", NewPassword: "abc123", Name: "Jane", IsAdmin: false})

				serveIndex(ds, fs)(resp, req)

				config := extractAppConfig(resp.Body.String())
				parsed := config["auth"].(map[string]interface{})

				Expect(parsed["id"]).To(Equal("111"))
				Expect(parsed["isAdmin"]).To(BeFalse())
				Expect(parsed["name"]).To(Equal("Jane"))
				Expect(parsed["username"]).To(Equal("janedoe"))
				Expect(parsed["token"]).ToNot(BeEmpty())
				Expect(parsed["subsonicSalt"]).ToNot(BeEmpty())
				Expect(parsed["subsonicToken"]).ToNot(BeEmpty())
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
