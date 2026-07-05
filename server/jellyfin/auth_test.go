package jellyfin

import (
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
		ur := ds.User(nil).(*tests.MockedUserRepo)
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
	})

	It("rejects invalid credentials with 401", func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Users/AuthenticateByName",
			strings.NewReader(`{"Username":"alice","Pw":"wrong"}`))
		api.authenticateByName(w, r)
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})
})
