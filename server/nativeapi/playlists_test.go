package nativeapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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
	tracksRepo rest.Repository
}

func (m *mockPlaylistsService) TracksRepository(_ context.Context, _ string, _ bool) rest.Repository {
	return m.tracksRepo
}

var _ = Describe("Playlist Tracks Endpoint", func() {
	var (
		router   http.Handler
		plsSvc   *mockPlaylistsService
		userRepo *tests.MockedUserRepo
		w        *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
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

		nativeRouter := New(ds, nil, plsSvc, nil, tests.NewMockLibraryService(), tests.NewMockUserService(), nil, nil)
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
