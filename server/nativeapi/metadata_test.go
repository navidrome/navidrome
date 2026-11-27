package nativeapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/server"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fakeProvider struct {
	external.Provider
	mu     sync.Mutex
	called []string
}

func (f *fakeProvider) RefreshInfo(_ context.Context, kind model.Kind, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.called = append(f.called, kind.Prefix()+"/"+id)
	return nil
}

func (f *fakeProvider) calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.called)
}

type failingAlbumGet struct{ model.AlbumRepository }

func (failingAlbumGet) Get(context.Context, string) (*model.Album, error) {
	return nil, errors.New("db unavailable")
}

var _ = Describe("Metadata API", func() {
	var ctx context.Context
	var ds model.DataStore
	var provider *fakeProvider
	var router http.Handler
	var adminToken, userToken string

	newRouter := func(ds model.DataStore) http.Handler {
		return server.JWTVerifier(newTestRouter(ds, nil, nil, provider))
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		conf.Update(func(c *conf.Options) { c.EnableSharing = false })
		ds = persistence.New(db.Db())
		auth.Init(ctx, ds)
		provider = &fakeProvider{}
		router = newRouter(ds)

		adminUser := model.User{ID: "admin-1", UserName: "admin", IsAdmin: true, NewPassword: "adminpass"}
		regularUser := model.User{ID: "user-1", UserName: "regular", IsAdmin: false, NewPassword: "userpass"}
		putUsers(ctx, ds, &adminUser, &regularUser)

		adminCtx := request.WithUser(ctx, adminUser)
		Expect(ds.Album().Put(ctx, &model.Album{ID: "al-1", Name: "Kid A", LibraryID: model.DefaultLibraryID})).To(Succeed())
		persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, model.Artist{ID: "ar-1", Name: "Radiohead"})
		Expect(ds.Playlist().Put(adminCtx, &model.Playlist{ID: "pl-1", Name: "My Playlist", OwnerID: adminUser.ID})).To(Succeed())

		var err error
		adminToken, err = auth.CreateToken(&adminUser)
		Expect(err).ToNot(HaveOccurred())
		userToken, err = auth.CreateToken(&regularUser)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("POST /api/metadata/{kind}/{id}/refresh", func() {
		It("clears state and enqueues a Bump for admins", func() {
			Expect(ds.Artwork().PutItemArtwork(ctx, &model.ItemArtwork{
				ItemKind: "al", ItemID: "al-1", ImageType: model.ImageTypePrimary, Hash: "oldhash", Source: "external",
				AttemptedAt: time.Now(), UpdatedAt: time.Now(),
			})).To(Succeed())
			_, err := ds.Artwork().GetItemArtwork(ctx, model.KindAlbumArtwork, "al-1", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())

			req := createAuthenticatedRequest("POST", "/metadata/al/al-1/refresh", nil, adminToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNoContent))

			_, err = ds.Artwork().GetItemArtwork(ctx, model.KindAlbumArtwork, "al-1", model.ImageTypePrimary)
			Expect(err).To(MatchError(model.ErrNotFound))

			queued, err := ds.ArtworkQueue().Get(ctx, model.KindAlbumArtwork, "al-1", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(queued.Priority).To(Equal(model.ArtworkPriorityBump))
		})

		It("returns 400 for an invalid kind", func() {
			req := createAuthenticatedRequest("POST", "/metadata/xx/id-1/refresh", nil, adminToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("denies access to regular users", func() {
			req := createAuthenticatedRequest("POST", "/metadata/al/al-1/refresh", nil, userToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusForbidden))
		})

		It("denies access without authentication", func() {
			req := createUnauthenticatedRequest("POST", "/metadata/al/al-1/refresh", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})

		It("triggers an external info refresh for albums", func() {
			req := createAuthenticatedRequest("POST", "/metadata/al/al-1/refresh", nil, adminToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNoContent))
			Eventually(provider.calls).Should(ContainElement("al/al-1"))
		})

		It("triggers an external info refresh for artists", func() {
			req := createAuthenticatedRequest("POST", "/metadata/ar/ar-1/refresh", nil, adminToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNoContent))
			Eventually(provider.calls).Should(ContainElement("ar/ar-1"))
		})

		It("skips the external info refresh for kinds without external info", func() {
			req := createAuthenticatedRequest("POST", "/metadata/pl/pl-1/refresh", nil, adminToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNoContent))
			Consistently(provider.calls).ShouldNot(ContainElement("pl/pl-1"))
		})

		It("returns 404 for an unknown id", func() {
			req := createAuthenticatedRequest("POST", "/metadata/al/nope/refresh", nil, adminToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("returns 500 when the lookup fails for a reason other than not-found", func() {
			router = newRouter(persistencetest.Stub(ds, persistencetest.Repos{
				Album: func(r model.AlbumRepository) model.AlbumRepository { return failingAlbumGet{r} },
			}))

			req := createAuthenticatedRequest("POST", "/metadata/al/al-1/refresh", nil, adminToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))
		})
	})
})
