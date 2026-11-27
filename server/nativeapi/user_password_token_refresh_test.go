package nativeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/server"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Pins that the token-epoch handoff survives a real request through the real middleware chain.
var _ = Describe("PUT /user/{id}: token refresh on self password change", func() {
	var ctx context.Context
	var ds model.DataStore
	var router http.Handler

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		conf.Update(func(c *conf.Options) {
			c.EnableUserEditing = true
			c.EnableSharing = false
			c.SessionTimeout = time.Hour
		})
		ds = persistence.New(db.Db())
		auth.Init(ctx, ds)
		router = server.JWTVerifier(newTestRouter(ds, nil, nil, nil))
	})

	It("carries the bumped epoch in the refreshed token, not the epoch the token was minted with", func() {
		usr := model.User{UserName: "selfchanger", Name: "Self Changer", NewPassword: "old-password"}
		Expect(ds.User().Put(ctx, &usr)).To(Succeed())

		token, err := auth.CreateToken(&usr)
		Expect(err).ToNot(HaveOccurred())

		body, _ := json.Marshal(map[string]any{
			"userName":        usr.UserName,
			"name":            usr.Name,
			"currentPassword": "old-password",
			"password":        "new-password",
		})
		req := createAuthenticatedRequest(http.MethodPut, "/user/"+usr.ID, bytes.NewBuffer(body), token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK), w.Body.String())

		refreshed := w.Header().Get(consts.UIAuthorizationHeader)
		Expect(refreshed).ToNot(BeEmpty())
		claims, err := auth.Validate(refreshed)
		Expect(err).ToNot(HaveOccurred())

		reloaded, err := ds.User().Get(ctx, usr.ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(reloaded.TokenEpoch).To(Equal(1))
		Expect(claims.Epoch).To(Equal(reloaded.TokenEpoch))
	})
})
