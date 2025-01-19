package server

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Auth", func() {
	Describe("User login", func() {
		var ds model.DataStore
		var req *http.Request
		var resp *httptest.ResponseRecorder

		BeforeEach(func() {
			ds = &tests.MockDataStore{}
			auth.Init(ds)
		})

		Describe("createAdmin", func() {
			var createdAt time.Time
			BeforeEach(func() {
				req = httptest.NewRequest("POST", "/createAdmin", strings.NewReader(`{"username":"johndoe", "password":"secret"}`))
				resp = httptest.NewRecorder()
				createdAt = time.Now()
				createAdmin(ds)(resp, req)
			})

			It("creates an admin user with the specified password", func() {
				usr := ds.User(context.Background())
				u, err := usr.FindByUsername("johndoe")
				Expect(err).To(BeNil())
				Expect(u.Password).ToNot(BeEmpty())
				Expect(u.IsAdmin).To(BeTrue())
				Expect(*u.LastLoginAt).To(BeTemporally(">=", createdAt, time.Second))
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
			const (
				trustedIpv4   = "192.168.0.42"
				untrustedIpv4 = "8.8.8.8"
				trustedIpv6   = "2001:4860:4860:1234:5678:0000:4242:8888"
				untrustedIpv6 = "5005:0:3003"
			)

			fs := os.DirFS("tests/fixtures")

			BeforeEach(func() {
				usr := ds.User(context.Background())
				_ = usr.Put(&model.User{ID: "111", UserName: "janedoe", NewPassword: "abc123", Name: "Jane", IsAdmin: false})
				req = httptest.NewRequest("GET", "/index.html", nil)
				req.Header.Add("Remote-User", "janedoe")
				resp = httptest.NewRecorder()
				conf.Server.UILoginBackgroundURL = ""
				conf.Server.ReverseProxyWhitelist = "192.168.0.0/16,2001:4860:4860::/48"
			})

			It("sets auth data if IPv4 matches whitelist", func() {
				req = req.WithContext(request.WithReverseProxyIp(req.Context(), trustedIpv4))
				serveIndex(ds, fs, nil)(resp, req)

				config := extractAppConfig(resp.Body.String())
				parsed := config["auth"].(map[string]interface{})

				Expect(parsed["id"]).To(Equal("111"))
			})

			It("sets no auth data if IPv4 does not match whitelist", func() {
				req = req.WithContext(request.WithReverseProxyIp(req.Context(), untrustedIpv4))
				serveIndex(ds, fs, nil)(resp, req)

				config := extractAppConfig(resp.Body.String())
				Expect(config["auth"]).To(BeNil())
			})

			It("sets auth data if IPv6 matches whitelist", func() {
				req = req.WithContext(request.WithReverseProxyIp(req.Context(), trustedIpv6))
				serveIndex(ds, fs, nil)(resp, req)

				config := extractAppConfig(resp.Body.String())
				parsed := config["auth"].(map[string]interface{})

				Expect(parsed["id"]).To(Equal("111"))
			})

			It("sets no auth data if IPv6 does not match whitelist", func() {
				req = req.WithContext(request.WithReverseProxyIp(req.Context(), untrustedIpv6))
				serveIndex(ds, fs, nil)(resp, req)

				config := extractAppConfig(resp.Body.String())
				Expect(config["auth"]).To(BeNil())
			})

			It("creates user and sets auth data if user does not exist", func() {
				newUser := "NEW_USER_" + id.NewRandom()

				req = req.WithContext(request.WithReverseProxyIp(req.Context(), trustedIpv4))
				req.Header.Set("Remote-User", newUser)
				serveIndex(ds, fs, nil)(resp, req)

				config := extractAppConfig(resp.Body.String())
				parsed := config["auth"].(map[string]interface{})

				Expect(parsed["username"]).To(Equal(newUser))
			})

			It("sets auth data if user exists", func() {
				req = req.WithContext(request.WithReverseProxyIp(req.Context(), trustedIpv4))
				serveIndex(ds, fs, nil)(resp, req)

				config := extractAppConfig(resp.Body.String())
				parsed := config["auth"].(map[string]interface{})

				Expect(parsed["id"]).To(Equal("111"))
				Expect(parsed["isAdmin"]).To(BeFalse())
				Expect(parsed["name"]).To(Equal("Jane"))
				Expect(parsed["username"]).To(Equal("janedoe"))
				Expect(parsed["subsonicSalt"]).ToNot(BeEmpty())
				Expect(parsed["subsonicToken"]).ToNot(BeEmpty())
				salt := parsed["subsonicSalt"].(string)
				token := fmt.Sprintf("%x", md5.Sum([]byte("abc123"+salt)))
				Expect(parsed["subsonicToken"]).To(Equal(token))

				// Request Header authentication should not generate a JWT token
				Expect(parsed).ToNot(HaveKey("token"))
			})

			It("does not set auth data when listening on unix socket without whitelist", func() {
				conf.Server.Address = "unix:/tmp/navidrome-test"
				conf.Server.ReverseProxyWhitelist = ""

				// No ReverseProxyIp in request context
				serveIndex(ds, fs, nil)(resp, req)

				config := extractAppConfig(resp.Body.String())
				Expect(config["auth"]).To(BeNil())
			})

			It("does not set auth data when listening on unix socket with incorrect whitelist", func() {
				conf.Server.Address = "unix:/tmp/navidrome-test"

				req = req.WithContext(request.WithReverseProxyIp(req.Context(), "@"))
				serveIndex(ds, fs, nil)(resp, req)

				config := extractAppConfig(resp.Body.String())
				Expect(config["auth"]).To(BeNil())
			})

			It("sets auth data when listening on unix socket with correct whitelist", func() {
				conf.Server.Address = "unix:/tmp/navidrome-test"
				conf.Server.ReverseProxyWhitelist = conf.Server.ReverseProxyWhitelist + ",@"

				req = req.WithContext(request.WithReverseProxyIp(req.Context(), "@"))
				serveIndex(ds, fs, nil)(resp, req)

				config := extractAppConfig(resp.Body.String())
				parsed := config["auth"].(map[string]interface{})

				Expect(parsed["id"]).To(Equal("111"))
			})
		})

		Describe("login", func() {
			BeforeEach(func() {
				req = httptest.NewRequest("POST", "/login", strings.NewReader(`{"username":"janedoe", "password":"abc123"}`))
				resp = httptest.NewRecorder()
			})

			It("fails if user does not exist", func() {
				login(ds)(resp, req)
				Expect(resp.Code).To(Equal(http.StatusUnauthorized))
			})

			It("logs in successfully if user exists", func() {
				usr := ds.User(context.Background())
				_ = usr.Put(&model.User{ID: "111", UserName: "janedoe", NewPassword: "abc123", Name: "Jane", IsAdmin: false})

				login(ds)(resp, req)
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

	Describe("tokenFromHeader", func() {
		It("returns the token when the Authorization header is set correctly", func() {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set(consts.UIAuthorizationHeader, "Bearer testtoken")

			token := tokenFromHeader(req)
			Expect(token).To(Equal("testtoken"))
		})

		It("returns an empty string when the Authorization header is not set", func() {
			req := httptest.NewRequest("GET", "/", nil)

			token := tokenFromHeader(req)
			Expect(token).To(BeEmpty())
		})

		It("returns an empty string when the Authorization header is not a Bearer token", func() {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set(consts.UIAuthorizationHeader, "Basic testtoken")

			token := tokenFromHeader(req)
			Expect(token).To(BeEmpty())
		})

		It("returns an empty string when the Bearer token is too short", func() {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set(consts.UIAuthorizationHeader, "Bearer")

			token := tokenFromHeader(req)
			Expect(token).To(BeEmpty())
		})
	})

	Describe("validateIPAgainstList", func() {
		Context("when provided with empty inputs", func() {
			It("should return false", func() {
				Expect(validateIPAgainstList("", "")).To(BeFalse())
				Expect(validateIPAgainstList("192.168.1.1", "")).To(BeFalse())
				Expect(validateIPAgainstList("", "192.168.0.0/16")).To(BeFalse())
			})
		})

		Context("when provided with invalid IP inputs", func() {
			It("should return false", func() {
				Expect(validateIPAgainstList("invalidIP", "192.168.0.0/16")).To(BeFalse())
			})
		})

		Context("when provided with valid inputs", func() {
			It("should return true when IP is in the list", func() {
				Expect(validateIPAgainstList("192.168.1.1", "192.168.0.0/16,10.0.0.0/8")).To(BeTrue())
				Expect(validateIPAgainstList("10.0.0.1", "192.168.0.0/16,10.0.0.0/8")).To(BeTrue())
			})

			It("should return false when IP is not in the list", func() {
				Expect(validateIPAgainstList("172.16.0.1", "192.168.0.0/16,10.0.0.0/8")).To(BeFalse())
			})
		})

		Context("when provided with invalid CIDR notation in the list", func() {
			It("should ignore invalid CIDR and return the correct result", func() {
				Expect(validateIPAgainstList("192.168.1.1", "192.168.0.0/16,invalidCIDR")).To(BeTrue())
				Expect(validateIPAgainstList("10.0.0.1", "invalidCIDR,10.0.0.0/8")).To(BeTrue())
				Expect(validateIPAgainstList("172.16.0.1", "192.168.0.0/16,invalidCIDR")).To(BeFalse())
			})
		})

		Context("when provided with IP:port format", func() {
			It("should handle IP:port format correctly", func() {
				Expect(validateIPAgainstList("192.168.1.1:8080", "192.168.0.0/16,10.0.0.0/8")).To(BeTrue())
				Expect(validateIPAgainstList("10.0.0.1:1234", "192.168.0.0/16,10.0.0.0/8")).To(BeTrue())
				Expect(validateIPAgainstList("172.16.0.1:9999", "192.168.0.0/16,10.0.0.0/8")).To(BeFalse())
			})
		})
	})
})
