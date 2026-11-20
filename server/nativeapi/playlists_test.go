package nativeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"time"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
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

	DescribeTable("uploadPlaylistImage guard",
		func(enableArtworkUpload, isAdmin bool, expectedStatus int) {
			conf.Update(func(c *conf.Options) { c.EnableArtworkUpload = enableArtworkUpload })
			handler := uploadPlaylistImage(&mockPlaylistsService{})

			req := httptest.NewRequest(http.MethodPost, "/playlist/pls-1/image", nil)
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
			conf.Update(func(c *conf.Options) { c.EnableArtworkUpload = enableArtworkUpload })
			handler := deletePlaylistImage(&mockPlaylistsService{})

			req := httptest.NewRequest(http.MethodDelete, "/playlist/pls-1/image", nil)
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
		conf.Update(func(c *conf.Options) {
			c.EnableSharing = false
			c.SessionTimeout = time.Minute
		})

		plsSvc = &mockPlaylistsService{repo: &mockPlaylistRestRepo{}}
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
		err := userRepo.Put(GinkgoT().Context(), &testUser)
		Expect(err).ToNot(HaveOccurred())

		nativeRouter := New(ds, nil, plsSvc, nil, tests.NewMockLibraryService(), tests.NewMockUserService(), nil, nil, nil, nil, nil)
		router = server.JWTVerifier(nativeRouter)
		w = httptest.NewRecorder()
	})

	createAuthenticatedRequestWithBody := func(method, path string, body io.Reader) *http.Request {
		req := httptest.NewRequest(method, path, body)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		testUser := model.User{ID: "user-1", UserName: "testuser"}
		token, err := auth.CreateToken(&testUser)
		Expect(err).ToNot(HaveOccurred())
		req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+token)
		return req
	}

	createAuthenticatedRequest := func(method, path string) *http.Request {
		return createAuthenticatedRequestWithBody(method, path, nil)
	}

	jsonBody := func(v any) io.Reader {
		b, err := json.Marshal(v)
		Expect(err).ToNot(HaveOccurred())
		return bytes.NewReader(b)
	}

	Describe("POST /playlist/{playlistId}/tracks", func() {
		DescribeTable("adds ids up to DevMaxPlaylistTracksPerRequest",
			func(limit, sent int) {
				conf.Update(func(c *conf.Options) { c.DevMaxPlaylistTracksPerRequest = limit })
				ids := make([]string, sent)
				for i := range ids {
					ids[i] = fmt.Sprintf("%022d", i)
				}
				req := createAuthenticatedRequestWithBody("POST", "/playlist/pls-1/tracks", jsonBody(map[string]any{"ids": ids}))
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
				Expect(plsSvc.addedIds).To(HaveLen(sent))
			},
			Entry("exactly the limit", 5, 5),
			Entry("any number when the limit is 0", 0, 6),
			Entry("a raised limit, in a body larger than 4 MiB", 200_000, 200_000),
		)

		DescribeTable("rejects oversized requests with 413 without adding anything",
			func(payload func() any) {
				req := createAuthenticatedRequestWithBody("POST", "/playlist/pls-1/tracks", jsonBody(payload()))
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusRequestEntityTooLarge))
				Expect(plsSvc.addCalls).To(BeZero())
			},
			Entry("more than DevMaxPlaylistTracksPerRequest ids", func() any {
				conf.Update(func(c *conf.Options) { c.DevMaxPlaylistTracksPerRequest = 5 })
				return map[string]any{"ids": make([]string, 6)}
			}),
			Entry("too many album ids", func() any {
				return map[string]any{"albumIds": make([]string, maxContainerIDs+1)}
			}),
			Entry("too many album, artist and disc ids combined", func() any {
				return map[string]any{
					"albumIds":  make([]string, maxContainerIDs/2),
					"artistIds": make([]string, maxContainerIDs/2),
					"discs":     []model.DiscID{{AlbumID: "al-1", DiscNumber: 1}},
				}
			}),
			Entry("a body over the size limit", func() any {
				return map[string]any{"ids": []string{strings.Repeat("x", int(maxPlaylistJSONBody()))}}
			}),
		)
	})

	Describe("POST /playlist", func() {
		It("treats JSON with a charset as JSON, not as an M3U upload", func() {
			plsSvc.importM3UFn = func(context.Context, io.Reader) (*model.Playlist, error) { return &model.Playlist{}, nil }
			req := createAuthenticatedRequestWithBody("POST", "/playlist", jsonBody(map[string]any{"name": "Mix"}))
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(plsSvc.repo.(*mockPlaylistRestRepo).saved).To(HaveExactElements(HaveField("Name", "Mix")))
		})

		It("accepts a JSON body larger than 4 MiB when the track limit is raised", func() {
			conf.Update(func(c *conf.Options) { c.DevMaxPlaylistTracksPerRequest = 100_000 })
			body := jsonBody(map[string]any{"name": strings.Repeat("x", 4<<20)})
			req := createAuthenticatedRequestWithBody("POST", "/playlist", body)
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(plsSvc.repo.(*mockPlaylistRestRepo).saved).To(HaveLen(1))
		})

		It("rejects a JSON body over the size limit with 413", func() {
			body := jsonBody(map[string]any{"name": strings.Repeat("x", int(maxPlaylistJSONBody()))})
			req := createAuthenticatedRequestWithBody("POST", "/playlist", body)
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusRequestEntityTooLarge))
			Expect(plsSvc.repo.(*mockPlaylistRestRepo).saved).To(BeEmpty())
		})
	})

	Describe("PUT /playlist/{id}", func() {
		It("rejects a JSON body over the size limit with 413", func() {
			body := jsonBody(map[string]any{"comment": strings.Repeat("x", int(maxPlaylistJSONBody()))})
			req := createAuthenticatedRequestWithBody("PUT", "/playlist/pls-1", body)
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusRequestEntityTooLarge))
			Expect(plsSvc.repo.(*mockPlaylistRestRepo).updated).To(BeZero())
		})
	})

	Describe("POST /playlist/{id}/image", func() {
		It("still reads uploads larger than the playlist JSON limit", func() {
			var buf bytes.Buffer
			mw := multipart.NewWriter(&buf)
			part, err := mw.CreateFormFile("image", "cover.png")
			Expect(err).ToNot(HaveOccurred())
			_, err = part.Write(bytes.Repeat([]byte{0}, int(maxPlaylistJSONBody())+1))
			Expect(err).ToNot(HaveOccurred())
			Expect(mw.Close()).To(Succeed())

			req := createAuthenticatedRequestWithBody("POST", "/playlist/pls-1/image", &buf)
			req.Header.Set("Content-Type", mw.FormDataContentType())
			conf.Update(func(c *conf.Options) { c.EnableArtworkUpload = true })
			router.ServeHTTP(w, req)

			// Reaches image decoding: the zero bytes are not an image.
			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("invalid image file"))
		})
	})

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
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/playlist/pls-1", nil))
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

var _ = Describe("createPlaylistFromM3U", func() {
	// Drains the body like the real importer, so the size limit is hit while reading.
	drainingImport := func(_ context.Context, reader io.Reader) (*model.Playlist, error) {
		if _, err := io.Copy(io.Discard, reader); err != nil {
			return nil, err
		}
		pl := &model.Playlist{Name: "Uploaded"}
		pl.AddMediaFiles(model.MediaFiles{{ID: "1", Path: "/music/song.mp3", Title: "Song"}})
		return pl, nil
	}
	upload := func(svc *mockPlaylistsService, body io.Reader) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/playlist", body)
		req.Header.Set("Content-Type", "audio/x-mpegurl")
		w := httptest.NewRecorder()
		createPlaylistFromM3U(svc).ServeHTTP(w, req)
		return w
	}

	It("creates the playlist and returns it as M3U", func() {
		w := upload(&mockPlaylistsService{importM3UFn: drainingImport}, strings.NewReader("/music/song.mp3\n"))

		Expect(w.Code).To(Equal(http.StatusCreated))
		Expect(w.Body.String()).To(ContainSubstring(filepath.FromSlash("/music/song.mp3")))
	})

	It("reads an upload larger than 32 MiB when the track limit is raised", func() {
		conf.Update(func(c *conf.Options) { c.DevMaxPlaylistTracksPerRequest = 100_000 })
		body := strings.NewReader(strings.Repeat("x", 32<<20+1))
		w := upload(&mockPlaylistsService{importM3UFn: drainingImport}, body)

		Expect(w.Code).To(Equal(http.StatusCreated))
	})

	It("rejects a body larger than the upload limit", func() {
		body := strings.NewReader(strings.Repeat("x", int(maxM3UUploadSize())+1))
		w := upload(&mockPlaylistsService{importM3UFn: drainingImport}, body)

		Expect(w.Code).To(Equal(http.StatusRequestEntityTooLarge))
	})
})

var _ = Describe("reorderItem", func() {
	It("rejects a body over the size limit with 413", func() {
		body := `{"insert_before":"` + strings.Repeat("1", maxSmallJSONBody) + `"}`
		req := httptest.NewRequest(http.MethodPut, "/playlist/pls-1/tracks/1?:id=1", strings.NewReader(body))
		w := httptest.NewRecorder()
		reorderItem(&mockPlaylistsService{})(w, req)
		Expect(w.Code).To(Equal(http.StatusRequestEntityTooLarge))
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
		Entry("too many entries -> 413", fmt.Errorf("wrapped: %w", playlists.ErrTooManyEntries), http.StatusRequestEntityTooLarge),
		Entry("body too large -> 413", &http.MaxBytesError{Limit: 1}, http.StatusRequestEntityTooLarge),
		Entry("unrecognized -> default", model.ErrValidation, http.StatusBadRequest),
	)
})

type mockPlaylistTrackRepo struct {
	model.PlaylistTrackRepository
	tracks model.PlaylistTracks
}

func (m *mockPlaylistTrackRepo) Count(context.Context, ...rest.QueryOptions) (int64, error) {
	return int64(len(m.tracks)), nil
}

func (m *mockPlaylistTrackRepo) ReadAll(context.Context, ...rest.QueryOptions) ([]model.PlaylistTrack, error) {
	return m.tracks, nil
}

func (m *mockPlaylistTrackRepo) Read(_ context.Context, id string) (*model.PlaylistTrack, error) {
	for _, t := range m.tracks {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, rest.ErrNotFound
}

type mockPlaylistsService struct {
	playlists.Playlists
	repo          rest.Repository[model.Playlist]
	tracksRepo    rest.Repository[model.PlaylistTrack]
	playlist      *model.Playlist
	addCalls      int
	addedIds      []string
	removeImageFn func(ctx context.Context, id string) error
	setImageFn    func(ctx context.Context, id string, reader io.Reader, ext string) error
	importM3UFn   func(ctx context.Context, reader io.Reader) (*model.Playlist, error)
}

func (m *mockPlaylistsService) ImportM3U(ctx context.Context, reader io.Reader) (*model.Playlist, error) {
	return m.importM3UFn(ctx, reader)
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

func (m *mockPlaylistsService) Repository() rest.Repository[model.Playlist] {
	return m.repo
}

func (m *mockPlaylistsService) GetWithTracks(_ context.Context, _ string) (*model.Playlist, error) {
	if m.playlist == nil {
		return nil, model.ErrNotFound
	}
	return m.playlist, nil
}

func (m *mockPlaylistsService) TracksRepository(_ context.Context, _ string, _ bool) rest.Repository[model.PlaylistTrack] {
	return m.tracksRepo
}

func (m *mockPlaylistsService) AddTracks(_ context.Context, _ string, ids []string) (int, error) {
	m.addCalls++
	m.addedIds = append(m.addedIds, ids...)
	return len(ids), nil
}

func (m *mockPlaylistsService) AddAlbums(_ context.Context, _ string, ids []string) (int, error) {
	m.addCalls++
	return len(ids), nil
}

func (m *mockPlaylistsService) AddArtists(_ context.Context, _ string, ids []string) (int, error) {
	m.addCalls++
	return len(ids), nil
}

func (m *mockPlaylistsService) AddDiscs(_ context.Context, _ string, discs []model.DiscID) (int, error) {
	m.addCalls++
	return len(discs), nil
}

// mockPlaylistRestRepo records what rest.Post and rest.Put hand to the playlist repository.
type mockPlaylistRestRepo struct {
	saved   []model.Playlist
	updated int
}

func (m *mockPlaylistRestRepo) Count(context.Context, ...rest.QueryOptions) (int64, error) {
	return 0, nil
}

func (m *mockPlaylistRestRepo) Read(_ context.Context, id string) (*model.Playlist, error) {
	return &model.Playlist{ID: id}, nil
}

func (m *mockPlaylistRestRepo) ReadAll(context.Context, ...rest.QueryOptions) ([]model.Playlist, error) {
	return nil, nil
}

func (m *mockPlaylistRestRepo) Save(_ context.Context, entity *model.Playlist) (string, error) {
	m.saved = append(m.saved, *entity)
	return "new-id", nil
}

func (m *mockPlaylistRestRepo) Update(context.Context, string, model.Playlist, ...string) error {
	m.updated++
	return nil
}

func (m *mockPlaylistRestRepo) Delete(context.Context, ...string) error {
	return nil
}
