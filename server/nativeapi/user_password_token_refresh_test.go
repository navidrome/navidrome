package nativeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type noopPluginUnloader struct{}

func (noopPluginUnloader) UnloadDisabledPlugins(context.Context) {}

// Pins that the token-epoch handoff survives a real request through the real middleware chain.
var _ = Describe("PUT /user/{id}: token refresh on self password change", func() {
	var ds model.DataStore
	var router http.Handler

	BeforeEach(func() {
		// db.Db() is a process-wide singleton that this DeferCleanup closes for the whole binary; keep this the only real-DB spec in this package.
		DeferCleanup(configtest.SetupConfig())
		conf.Server.EnableUserEditing = true
		conf.Server.EnableSharing = false
		conf.Server.SessionTimeout = time.Hour
		conf.Server.DbPath = filepath.Join(GinkgoT().TempDir(), "nativeapi-user-refresh.db") + "?_journal_mode=WAL"
		DeferCleanup(db.Init(GinkgoT().Context()))

		ds = &tests.MockDataStore{RealDS: persistence.New(db.Db())}
		auth.Init(ds)

		userService := core.NewUser(ds, noopPluginUnloader{})
		nativeRouter := New(ds, nil, nil, nil, tests.NewMockLibraryService(), userService, nil, nil, nil, nil)
		router = server.JWTVerifier(nativeRouter)
	})

	It("carries the bumped epoch in the refreshed token, not the epoch the token was minted with", func() {
		usr := model.User{UserName: "selfchanger", Name: "Self Changer", NewPassword: "old-password"}
		Expect(ds.User(GinkgoT().Context()).Put(&usr)).To(Succeed())

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

		reloaded, err := ds.User(GinkgoT().Context()).Get(usr.ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(reloaded.TokenEpoch).To(Equal(1))
		Expect(claims.Epoch).To(Equal(reloaded.TokenEpoch))
	})
})
