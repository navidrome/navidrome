package nativeapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Playlist Image Endpoints", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
	})

	DescribeTable("uploadPlaylistImage guard",
		func(enableArtworkUpload, isAdmin bool, expectedStatus int) {
			conf.Server.EnableArtworkUpload = enableArtworkUpload
			handler := uploadPlaylistImage(&mockPlaylistsService{})

			req := httptest.NewRequest("POST", "/playlist/pls-1/image", nil)
			ctx := request.WithUser(GinkgoT().Context(), model.User{ID: "user-1", IsAdmin: isAdmin})
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(expectedStatus))
		},
		Entry("enabled, regular user passes guard", true, false, http.StatusBadRequest),
		Entry("enabled, admin passes guard", true, true, http.StatusBadRequest),
		Entry("disabled, admin passes guard", false, true, http.StatusBadRequest),
		Entry("disabled, regular user is forbidden", false, false, http.StatusForbidden),
	)

	DescribeTable("deletePlaylistImage guard",
		func(enableArtworkUpload, isAdmin bool, expectedStatus int) {
			conf.Server.EnableArtworkUpload = enableArtworkUpload
			handler := deletePlaylistImage(&mockPlaylistsService{})

			req := httptest.NewRequest("DELETE", "/playlist/pls-1/image", nil)
			ctx := request.WithUser(GinkgoT().Context(), model.User{ID: "user-1", IsAdmin: isAdmin})
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(expectedStatus))
		},
		Entry("enabled, regular user passes guard", true, false, http.StatusNotFound),
		Entry("enabled, admin passes guard", true, true, http.StatusNotFound),
		Entry("disabled, admin passes guard", false, true, http.StatusNotFound),
		Entry("disabled, regular user is forbidden", false, false, http.StatusForbidden),
	)
})

var _ = Describe("Playlist Tracks Endpoint", func() {
	var (
		router   http.Handler
		plsSvc   *mockPlaylistsService
		userRepo *tests.MockedUserRepo
		w        *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.EnableSharing = false
		conf.Server.SessionTimeout = time.Minute

		plsSvc = &mockPlaylistsService{}
		userRepo = tests.CreateMockUserRepo()

		ds := &tests.MockDataStore{
			MockedUser:     userRepo,
			MockedProperty: &tests.MockedPropertyRepo{},
		}

		auth.Init(ds)

		testUser := model.User{
			ID:          "user-1",
			UserName:    "testuser",
			Name:        "Test User",
			IsAdmin:     false,
			NewPassword: "testpass",
		}
		err := userRepo.Put(&testUser)
		Expect(err).ToNot(HaveOccurred())

		nativeRouter := New(ds, nil, plsSvc, nil, tests.NewMockLibraryService(), tests.NewMockUserService(), nil, nil, nil, nil, nil)
		router = server.JWTVerifier(nativeRouter)
		w = httptest.NewRecorder()
	})

	createAuthenticatedRequest := func(method, path string) *http.Request {
		req := httptest.NewRequest(method, path, nil)
		testUser := model.User{ID: "user-1", UserName: "testuser"}
		token, err := auth.CreateToken(&testUser)
		Expect(err).ToNot(HaveOccurred())
		req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+token)
		return req
	}

	Describe("GET /playlist/{playlistId}/tracks", func() {
		It("returns 404 when playlist does not exist", func() {
			req := createAuthenticatedRequest("GET", "/playlist/non-existent/tracks")
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("returns tracks when playlist exists", func() {
			plsSvc.tracksRepo = &mockPlaylistTrackRepo{
				tracks: model.PlaylistTracks{
					{ID: "1", MediaFileID: "mf-1", PlaylistID: "pls-1"},
					{ID: "2", MediaFileID: "mf-2", PlaylistID: "pls-1"},
				},
			}

			req := createAuthenticatedRequest("GET", "/playlist/pls-1/tracks")
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))

			var response []model.PlaylistTrack
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).ToNot(HaveOccurred())
			Expect(response).To(HaveLen(2))
			Expect(response[0].ID).To(Equal("1"))
			Expect(response[1].ID).To(Equal("2"))
		})
	})

	Describe("GET /playlist/{playlistId}/tracks/{id}", func() {
		It("returns 404 when playlist does not exist", func() {
			req := createAuthenticatedRequest("GET", "/playlist/non-existent/tracks/1")
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("returns the track when playlist exists", func() {
			plsSvc.tracksRepo = &mockPlaylistTrackRepo{
				tracks: model.PlaylistTracks{
					{ID: "1", MediaFileID: "mf-1", PlaylistID: "pls-1"},
				},
			}

			req := createAuthenticatedRequest("GET", "/playlist/pls-1/tracks/1")
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))

			var response model.PlaylistTrack
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.ID).To(Equal("1"))
			Expect(response.MediaFileID).To(Equal("mf-1"))
		})

		It("returns 404 when track does not exist in playlist", func() {
			plsSvc.tracksRepo = &mockPlaylistTrackRepo{
				tracks: model.PlaylistTracks{},
			}

			req := createAuthenticatedRequest("GET", "/playlist/pls-1/tracks/999")
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})
})

var _ = Describe("handleExportPlaylist", func() {
	export := func(name string) *httptest.ResponseRecorder {
		r := chi.NewRouter()
		r.Get("/playlist/{playlistId}", handleExportPlaylist(&mockPlaylistsService{
			playlist: &model.Playlist{ID: "pls-1", Name: name},
		}))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/playlist/pls-1", nil))
		return w
	}

	It("names the download after the playlist", func() {
		w := export("Road Trip")

		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Content-Disposition")).To(Equal(`attachment; filename="Road Trip.m3u"`))
	})

	It("does not let the playlist name inject a second filename parameter", func() {
		w := export(`party"; filename="evil.html`)

		Expect(w.Header().Get("Content-Disposition")).To(Equal(`attachment; filename="party_; filename=_evil.html.m3u"`))
	})

	It("keeps non-ASCII names in filename*", func() {
		w := export("Кино")

		Expect(w.Header().Get("Content-Disposition")).To(Equal(`attachment; filename="download.m3u"; filename*=utf-8''%D0%9A%D0%B8%D0%BD%D0%BE.m3u`))
	})
})

var _ = Describe("writePlaylistError", func() {
	DescribeTable("maps a service error to an HTTP status",
		func(err error, expected int) {
			w := httptest.NewRecorder()
			writePlaylistError(w, err, http.StatusBadRequest)
			Expect(w.Code).To(Equal(expected))
		},
		Entry("not found -> 404", model.ErrNotFound, http.StatusNotFound),
		Entry("not authorized -> 403", model.ErrNotAuthorized, http.StatusForbidden),
		Entry("rest permission denied -> 403", rest.ErrPermissionDenied, http.StatusForbidden),
		Entry("not editable -> 409", model.ErrPlaylistNotEditable, http.StatusConflict),
		Entry("unrecognized -> default", model.ErrValidation, http.StatusBadRequest),
	)
})

type mockPlaylistTrackRepo struct {
	model.PlaylistTrackRepository
	tracks model.PlaylistTracks
}

func (m *mockPlaylistTrackRepo) Count(...rest.QueryOptions) (int64, error) {
	return int64(len(m.tracks)), nil
}

func (m *mockPlaylistTrackRepo) ReadAll(...rest.QueryOptions) (any, error) {
	return m.tracks, nil
}

func (m *mockPlaylistTrackRepo) EntityName() string {
	return "playlist_track"
}

func (m *mockPlaylistTrackRepo) NewInstance() any {
	return &model.PlaylistTrack{}
}

func (m *mockPlaylistTrackRepo) Read(id string) (any, error) {
	for _, t := range m.tracks {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, rest.ErrNotFound
}

type mockPlaylistsService struct {
	playlists.Playlists
	tracksRepo    rest.Repository
	playlist      *model.Playlist
	removeImageFn func(ctx context.Context, id string) error
	setImageFn    func(ctx context.Context, id string, reader io.Reader, ext string) error
}

func (m *mockPlaylistsService) RemoveImage(ctx context.Context, id string) error {
	if m.removeImageFn != nil {
		return m.removeImageFn(ctx, id)
	}
	return model.ErrNotFound
}

func (m *mockPlaylistsService) SetImage(ctx context.Context, id string, reader io.Reader, ext string) error {
	if m.setImageFn != nil {
		return m.setImageFn(ctx, id, reader, ext)
	}
	return model.ErrNotFound
}

func (m *mockPlaylistsService) GetWithTracks(_ context.Context, _ string) (*model.Playlist, error) {
	if m.playlist == nil {
		return nil, model.ErrNotFound
	}
	return m.playlist, nil
}

func (m *mockPlaylistsService) TracksRepository(_ context.Context, _ string, _ bool) rest.Repository {
	return m.tracksRepo
}
