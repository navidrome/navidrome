package jellyfin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
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
		Expect(ur.Put(&model.User{ID: testID("u1"), UserName: "alice", NewPassword: "secret"})).To(Succeed())
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
	})

	Describe("SessionInfo", func() {
		login := func(authHeader string) map[string]any {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/Users/AuthenticateByName",
				strings.NewReader(`{"Username":"alice","Pw":"secret"}`))
			r.Header.Set("Authorization", authHeader)
			api.authenticateByName(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var raw map[string]any
			Expect(json.Unmarshal(w.Body.Bytes(), &raw)).To(Succeed())
			Expect(raw).To(HaveKey("SessionInfo"))
			return raw["SessionInfo"].(map[string]any)
		}
		const jellybox = `MediaBrowser Client="JellyBox", Device="Mac", DeviceId="dev-1", Version="2.1"`

		It("has the fields strict clients require", func() {
			s := login(jellybox)
			Expect(s["Id"]).To(And(BeAssignableToTypeOf(""), Not(BeEmpty())))
			Expect(s["UserId"]).To(Equal(dto.EncodeID(testID("u1"))))
			Expect(s["LastActivityDate"]).To(And(BeAssignableToTypeOf(""), Not(BeEmpty())))
			for _, k := range []string{"SupportsRemoteControl", "SupportsMediaControl", "HasCustomDeviceName"} {
				Expect(s[k]).To(BeAssignableToTypeOf(false), k)
			}
			ps, ok := s["PlayState"].(map[string]any)
			Expect(ok).To(BeTrue())
			for _, k := range []string{"CanSeek", "IsPaused", "IsMuted"} {
				Expect(ps[k]).To(BeAssignableToTypeOf(false), k)
			}
		})

		It("describes the calling client", func() {
			s := login(jellybox)
			Expect(s).To(HaveKeyWithValue("UserName", "alice"))
			Expect(s).To(HaveKeyWithValue("Client", "JellyBox"))
			Expect(s).To(HaveKeyWithValue("DeviceName", "Mac"))
			Expect(s).To(HaveKeyWithValue("DeviceId", "dev-1"))
			Expect(s).To(HaveKeyWithValue("ApplicationVersion", "2.1"))
			Expect(s).To(HaveKeyWithValue("IsActive", true))
		})

		It("keeps the same Id for the same device, and a new one for another device", func() {
			first := login(jellybox)["Id"]
			Expect(login(jellybox)["Id"]).To(Equal(first))
			other := login(`MediaBrowser Client="JellyBox", Device="Mac", DeviceId="dev-2", Version="2.1"`)
			Expect(other["Id"]).ToNot(Equal(first))
		})
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
		Expect(ur.Put(&model.User{ID: testID("admin1"), UserName: "root", NewPassword: "secret", IsAdmin: true})).To(Succeed())

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
		Expect(ur.Put(&model.User{ID: testID("e"), UserName: "empty", NewPassword: ""})).To(Succeed())

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Users/AuthenticateByName",
			strings.NewReader(`{"Username":"empty","Pw":""}`))
		api.authenticateByName(w, r)
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})
})

var _ = Describe("AuthenticateByName body limit", func() {
	It("rejects a request body larger than the limit", func() {
		ds := &tests.MockDataStore{}
		api := &Router{ds: ds}
		w := httptest.NewRecorder()
		body := `{"Username":"alice","Pw":"secret","Padding":"` + strings.Repeat("x", server.MaxLoginBodySize) + `"}`
		r := httptest.NewRequest("POST", "/Users/AuthenticateByName", strings.NewReader(body))
		server.LimitLoginBody(http.HandlerFunc(api.authenticateByName)).ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusBadRequest))
	})
})
