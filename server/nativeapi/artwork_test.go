package nativeapi

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Artwork API", func() {
	var ds *tests.MockDataStore
	var artRepo *tests.MockArtworkRepo
	var queueRepo *tests.MockArtworkQueueRepo
	var router http.Handler
	var adminToken, userToken string

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.EnableSharing = false
		artRepo = tests.CreateMockArtworkRepo()
		queueRepo = tests.CreateMockArtworkQueueRepo()
		ds = &tests.MockDataStore{MockedArtwork: artRepo, MockedArtworkQueue: queueRepo}
		auth.Init(ds)
		nativeRouter := New(ds, nil, nil, nil, tests.NewMockLibraryService(), tests.NewMockUserService(), nil, nil, nil)
		router = server.JWTVerifier(nativeRouter)

		adminUser := model.User{ID: "admin-1", UserName: "admin", IsAdmin: true, NewPassword: "adminpass"}
		regularUser := model.User{ID: "user-1", UserName: "regular", IsAdmin: false, NewPassword: "userpass"}
		Expect(ds.User(context.TODO()).Put(&adminUser)).To(Succeed())
		Expect(ds.User(context.TODO()).Put(&regularUser)).To(Succeed())

		var err error
		adminToken, err = auth.CreateToken(&adminUser)
		Expect(err).ToNot(HaveOccurred())
		userToken, err = auth.CreateToken(&regularUser)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("POST /api/artwork/{kind}/{id}/refresh", func() {
		It("clears state and enqueues a Bump for admins", func() {
			Expect(artRepo.PutItemArtwork(&model.ItemArtwork{
				ItemKind: "al", ItemID: "al-1", Hash: "oldhash", Source: "external",
			})).To(Succeed())

			req := createAuthenticatedRequest("POST", "/artwork/al/al-1/refresh", nil, adminToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNoContent))

			_, err := artRepo.GetItemArtwork("al", "al-1", model.ImageTypePrimary)
			Expect(err).To(MatchError(model.ErrNotFound))

			queued, err := queueRepo.DequeueBatch(1000)
			Expect(err).ToNot(HaveOccurred())
			Expect(queued).To(ContainElement(SatisfyAll(
				HaveField("ItemKind", "al"),
				HaveField("ItemID", "al-1"),
				HaveField("Priority", model.ArtworkPriorityBump),
			)))
		})

		It("returns 400 for an invalid kind", func() {
			req := createAuthenticatedRequest("POST", "/artwork/xx/id-1/refresh", nil, adminToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("denies access to regular users", func() {
			req := createAuthenticatedRequest("POST", "/artwork/al/al-1/refresh", nil, userToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusForbidden))
		})

		It("denies access without authentication", func() {
			req := createUnauthenticatedRequest("POST", "/artwork/al/al-1/refresh", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})
	})
})
