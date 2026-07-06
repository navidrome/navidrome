package jellyfin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AuthenticateByName", func() {
	var api *Router
	var ds *tests.MockDataStore
	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		auth.Init(ds)
		ur := ds.User(context.Background()).(*tests.MockedUserRepo)
		Expect(ur.Put(&model.User{ID: "u1", UserName: "alice", NewPassword: "secret"})).To(Succeed())
		api = &Router{ds: ds}
	})

	It("issues a token for valid credentials", func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Users/AuthenticateByName",
			strings.NewReader(`{"Username":"alice","Pw":"secret"}`))
		api.authenticateByName(w, r)

		Expect(w.Code).To(Equal(http.StatusOK))
		var res dto.AuthenticationResult
		Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
		Expect(res.AccessToken).ToNot(BeEmpty())
		Expect(res.User.Name).To(Equal("alice"))
		claims, err := auth.Validate(res.AccessToken)
		Expect(err).ToNot(HaveOccurred())
		Expect(claims.Subject).To(Equal("alice"))

		// Finamp reads Policy/Configuration right after login and null-crashes if they're absent.
		Expect(res.User.Policy).ToNot(BeNil())
		Expect(res.User.Policy.IsAdministrator).To(BeFalse())
		Expect(res.User.Policy.EnableAllFolders).To(BeTrue())
		Expect(res.User.Policy.EnableMediaPlayback).To(BeTrue())
		Expect(res.User.Configuration).ToNot(BeNil())

		// Ours is a partial SessionInfo; a strict client may fail to parse it, and Finamp's
		// login doesn't require it, so it should be omitted entirely rather than sent partial.
		Expect(res.SessionInfo).To(BeNil())
	})

	It("records the login time, like the web UI login does", func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Users/AuthenticateByName",
			strings.NewReader(`{"Username":"alice","Pw":"secret"}`))
		api.authenticateByName(w, r)

		Expect(w.Code).To(Equal(http.StatusOK))
		ur := ds.User(context.Background()).(*tests.MockedUserRepo)
		usr, err := ur.FindByUsername("alice")
		Expect(err).ToNot(HaveOccurred())
		Expect(usr.LastLoginAt).ToNot(BeNil())
	})

	It("reflects an administrator in the User.Policy", func() {
		ur := ds.User(context.Background()).(*tests.MockedUserRepo)
		Expect(ur.Put(&model.User{ID: "admin1", UserName: "root", NewPassword: "secret", IsAdmin: true})).To(Succeed())

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Users/AuthenticateByName",
			strings.NewReader(`{"Username":"root","Pw":"secret"}`))
		api.authenticateByName(w, r)

		Expect(w.Code).To(Equal(http.StatusOK))
		var res dto.AuthenticationResult
		Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
		Expect(res.User.Policy).ToNot(BeNil())
		Expect(res.User.Policy.IsAdministrator).To(BeTrue())
	})

	It("rejects invalid credentials with 401", func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Users/AuthenticateByName",
			strings.NewReader(`{"Username":"alice","Pw":"wrong"}`))
		api.authenticateByName(w, r)
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})

	It("rejects an empty password even for a user with an empty stored password with 401", func() {
		ur := ds.User(context.Background()).(*tests.MockedUserRepo)
		Expect(ur.Put(&model.User{ID: "e", UserName: "empty", NewPassword: ""})).To(Succeed())

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Users/AuthenticateByName",
			strings.NewReader(`{"Username":"empty","Pw":""}`))
		api.authenticateByName(w, r)
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})
})
