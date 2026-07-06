package e2e

import (
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/server/jellyfin/dto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Authentication", func() {
	BeforeEach(func() { setupTestDB() })

	authenticate := func(username, pw string) *httptest.ResponseRecorder {
		body := `{"Username":"` + username + `","Pw":"` + pw + `"}`
		return rawReq("POST", "/Users/AuthenticateByName", body)
	}

	Describe("POST /Users/AuthenticateByName", func() {
		It("authenticates a valid user and returns a usable token", func() {
			w := authenticate("admin", "password")
			var res dto.AuthenticationResult
			parseInto(w, &res)
			Expect(res.AccessToken).ToNot(BeEmpty())
			Expect(res.User).ToNot(BeNil())
			Expect(res.User.Name).To(Equal("admin"))
			Expect(res.User.Id).To(Equal("admin-1"))
			Expect(res.User.Policy.IsAdministrator).To(BeTrue())
			Expect(res.ServerId).ToNot(BeEmpty())

			// The returned token must actually authenticate a protected request.
			r := httptest.NewRequest("GET", "/Users/Me", nil)
			r.Header.Set("X-Emby-Token", res.AccessToken)
			pw := httptest.NewRecorder()
			router.ServeHTTP(pw, r)
			Expect(pw.Code).To(Equal(http.StatusOK))
		})

		It("marks a non-admin user's policy as non-administrator", func() {
			w := authenticate("regular", "password")
			var res dto.AuthenticationResult
			parseInto(w, &res)
			Expect(res.User.Policy.IsAdministrator).To(BeFalse())
		})

		It("rejects a wrong password", func() {
			Expect(authenticate("admin", "wrong").Code).To(Equal(http.StatusUnauthorized))
		})

		It("rejects an empty password", func() {
			Expect(authenticate("admin", "").Code).To(Equal(http.StatusUnauthorized))
		})

		It("rejects an unknown user", func() {
			Expect(authenticate("nobody", "password").Code).To(Equal(http.StatusUnauthorized))
		})

		It("rejects a malformed body", func() {
			Expect(rawReq("POST", "/Users/AuthenticateByName", "not json").Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("GET /Users/Public", func() {
		It("returns an empty list (manual login only)", func() {
			w := rawReq("GET", "/Users/Public", "")
			Expect(w.Code).To(Equal(http.StatusOK))
			var users []dto.UserDto
			parseInto(w, &users)
			Expect(users).To(BeEmpty())
		})
	})

	Describe("current user", func() {
		It("returns the caller from GET /Users/Me", func() {
			var u dto.UserDto
			parseInto(getAs(regularUser, "/Users/Me"), &u)
			Expect(u.Name).To(Equal("regular"))
			Expect(u.Id).To(Equal("regular-1"))
		})

		It("returns the caller from GET /Users/{userId}", func() {
			var u dto.UserDto
			parseInto(get("/Users/admin-1"), &u)
			Expect(u.Name).To(Equal("admin"))
		})
	})

	Describe("auth enforcement", func() {
		It("rejects a protected request with no token", func() {
			Expect(rawReq("GET", "/Users/Me", "").Code).To(Equal(http.StatusUnauthorized))
		})

		It("rejects a protected request with a bogus token", func() {
			r := httptest.NewRequest("GET", "/Users/Me", nil)
			r.Header.Set("X-Emby-Token", "not-a-valid-jwt")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})
	})
})
