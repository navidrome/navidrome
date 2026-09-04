package nativeapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GET /artwork/explain", func() {
	var router http.Handler
	var ds *tests.MockDataStore
	var artRepo *tests.MockArtworkRepo
	var queueRepo *tests.MockArtworkQueueRepo
	var adminToken, userToken string

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.EnableSharing = false
		conf.Server.ArtistArtPriority = "external"
		artRepo = tests.CreateMockArtworkRepo()
		queueRepo = tests.CreateMockArtworkQueueRepo()
		ds = &tests.MockDataStore{MockedArtwork: artRepo, MockedArtworkQueue: queueRepo}
		Expect(ds.Artist(context.Background()).Put(&model.Artist{ID: "ar-1", Name: "Radiohead"})).To(Succeed())
		auth.Init(ds)

		nativeRouter := New(ds, nil, nil, nil, tests.NewMockLibraryService(), tests.NewMockUserService(), nil, nil, nil, nil, nil)
		router = server.JWTVerifier(nativeRouter)

		adminUser := model.User{ID: "admin-1", UserName: "admin", IsAdmin: true, NewPassword: "adminpass"}
		regularUser := model.User{ID: "user-1", UserName: "regular", IsAdmin: false, NewPassword: "userpass"}
		Expect(ds.User(context.Background()).Put(&adminUser)).To(Succeed())
		Expect(ds.User(context.Background()).Put(&regularUser)).To(Succeed())

		var err error
		adminToken, err = auth.CreateToken(&adminUser)
		Expect(err).ToNot(HaveOccurred())
		userToken, err = auth.CreateToken(&regularUser)
		Expect(err).ToNot(HaveOccurred())
	})

	It("returns 403 for a non-admin", func() {
		req := createAuthenticatedRequest("GET", "/artwork/explain?kind=ar&id=ar-1", nil, userToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusForbidden))
	})

	It("returns 400 for an unknown kind", func() {
		req := createAuthenticatedRequest("GET", "/artwork/explain?kind=zz&id=ar-1", nil, adminToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusBadRequest))
	})

	It("returns 404 for an unknown id", func() {
		req := createAuthenticatedRequest("GET", "/artwork/explain?kind=ar&id=missing", nil, adminToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})

	It("returns the report for an admin", func() {
		// Storage shape from core/artwork/trace.go's storedStep: single-letter keys, "d" optional.
		trace := `[{"c":"external:deezer","o":"hit","d":"https://cdn/x.jpg"}]`
		gaveUpTrace := `[{"c":"external:deezer","o":"error","d":"connection reset"}]`
		attemptedAt := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
		Expect(artRepo.PutItemArtwork(&model.ItemArtwork{
			ItemKind: model.KindArtistArtwork.Prefix(), ItemID: "ar-1", ImageType: model.ImageTypePrimary,
			Hash: "abc", Source: "external:deezer", SourcePath: "/music/Radiohead/folder.jpg", Trace: trace,
			LastFailure: gaveUpTrace, AttemptedAt: attemptedAt,
		})).To(Succeed())

		req := createAuthenticatedRequest("GET", "/artwork/explain?kind=ar&id=ar-1", nil, adminToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))

		var got map[string]any
		Expect(json.Unmarshal(w.Body.Bytes(), &got)).To(Succeed())
		Expect(got["name"]).To(Equal("Radiohead"))
		Expect(got["result"]).To(Equal("resolved from external:deezer"))
		Expect(got["config"]).To(HaveKeyWithValue("setting", "ArtistArtPriority"))

		Expect(got["stored"]).To(Equal(map[string]any{
			"source":      "external:deezer",
			"sourcePath":  "/music/Radiohead/folder.jpg",
			"attemptedAt": attemptedAt.Format(time.RFC3339),
		}))
		Expect(got["steps"]).To(Equal([]any{
			map[string]any{"candidate": "external:deezer", "outcome": "hit", "detail": "https://cdn/x.jpg"},
		}))
		Expect(got["gaveUpAfter"]).To(Equal([]any{
			map[string]any{"candidate": "external:deezer", "outcome": "error", "detail": "connection reset"},
		}))
	})

	It("returns the queue state and the last-attempt trace", func() {
		Expect(queueRepo.Enqueue(model.ArtworkQueueItem{
			ItemKind: model.KindArtistArtwork.Prefix(), ItemID: "ar-1", ImageType: model.ImageTypePrimary,
			Priority: model.ArtworkPriorityScan,
		})).To(Succeed())
		seen, err := queueRepo.Get(model.KindArtistArtwork, "ar-1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())

		retryAt := time.Date(2026, 9, 2, 8, 0, 0, 0, time.UTC)
		failTrace := `[{"c":"external:deezer","o":"error","d":"timeout"}]`
		Expect(queueRepo.MarkFailedIfUnchanged(model.KindArtistArtwork.Prefix(), "ar-1", model.ImageTypePrimary,
			seen.RetryAt, retryAt, failTrace)).To(Succeed())

		req := createAuthenticatedRequest("GET", "/artwork/explain?kind=ar&id=ar-1", nil, adminToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))

		var got map[string]any
		Expect(json.Unmarshal(w.Body.Bytes(), &got)).To(Succeed())
		Expect(got["queued"]).To(Equal(map[string]any{
			"priorityName": "scan",
			"attempts":     float64(1),
			"retryAt":      retryAt.Format(time.RFC3339),
		}))
		Expect(got["lastAttemptFailed"]).To(Equal([]any{
			map[string]any{"candidate": "external:deezer", "outcome": "error", "detail": "timeout"},
		}))
	})

	It("omits empty sections", func() {
		req := createAuthenticatedRequest("GET", "/artwork/explain?kind=ar&id=ar-1", nil, adminToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))

		var got map[string]any
		Expect(json.Unmarshal(w.Body.Bytes(), &got)).To(Succeed())
		Expect(got).ToNot(HaveKey("stored"))
		Expect(got).ToNot(HaveKey("queued"))
		Expect(got).ToNot(HaveKey("lastAttemptFailed"))
		Expect(got).ToNot(HaveKey("gaveUpAfter"))
		Expect(got).ToNot(HaveKey("chainOrigin"))
	})
})
