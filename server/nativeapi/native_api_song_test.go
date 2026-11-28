package nativeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/server"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type failingMediaFileReads struct{ model.MediaFileRepository }

var errMediaFileReads = errors.New("db unavailable")

func (failingMediaFileReads) Count(context.Context, ...rest.QueryOptions) (int64, error) {
	return 0, errMediaFileReads
}

func (failingMediaFileReads) ReadAll(context.Context, ...rest.QueryOptions) ([]model.MediaFile, error) {
	return nil, errMediaFileReads
}

func (failingMediaFileReads) Read(context.Context, string) (*model.MediaFile, error) {
	return nil, errMediaFileReads
}

var _ = Describe("Song Endpoints", func() {
	var (
		ctx       context.Context
		router    http.Handler
		ds        model.DataStore
		w         *httptest.ResponseRecorder
		testUser  model.User
		testSongs model.MediaFiles
	)

	useFailingMediaFiles := func() {
		router = server.JWTVerifier(newTestRouter(persistencetest.Stub(ds, persistencetest.Repos{
			MediaFile: func(r model.MediaFileRepository) model.MediaFileRepository { return failingMediaFileReads{r} },
		}), nil, nil, nil))
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		conf.Update(func(c *conf.Options) {
			c.EnableSharing = false
			c.SessionTimeout = time.Minute
		})

		ds = persistence.New(db.Db())
		auth.Init(ctx, ds)

		// Create test user
		testUser = model.User{
			ID:          "user-1",
			UserName:    "testuser",
			Name:        "Test User",
			IsAdmin:     false,
			NewPassword: "testpass",
		}
		putUsers(ctx, ds, &testUser)

		// Create test songs
		testSongs = model.MediaFiles{
			{
				ID:         "song-1",
				LibraryID:  model.DefaultLibraryID,
				Title:      "Test Song 1",
				OrderTitle: "test song 1",
				Artist:     "Test Artist 1",
				Album:      "Test Album 1",
				AlbumID:    "album-1",
				ArtistID:   "artist-1",
				Duration:   180.5,
				BitRate:    320,
				Path:       "/music/song1.mp3",
				Suffix:     "mp3",
				Size:       5242880,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
			{
				ID:         "song-2",
				LibraryID:  model.DefaultLibraryID,
				Title:      "Test Song 2",
				OrderTitle: "test song 2",
				Artist:     "Test Artist 2",
				Album:      "Test Album 2",
				AlbumID:    "album-2",
				ArtistID:   "artist-2",
				Duration:   240.0,
				BitRate:    256,
				Path:       "/music/song2.mp3",
				Suffix:     "mp3",
				Size:       7340032,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
		}
		for i := range testSongs {
			Expect(ds.MediaFile().Put(ctx, &testSongs[i])).To(Succeed())
		}

		router = server.JWTVerifier(newTestRouter(ds, nil, nil, nil))
		w = httptest.NewRecorder()
	})

	// Helper function to create unauthenticated request
	createUnauthenticatedRequest := func(method, path string, body []byte) *http.Request {
		var req *http.Request
		if body != nil {
			req = httptest.NewRequestWithContext(ctx, method, path, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req = httptest.NewRequestWithContext(ctx, method, path, nil)
		}
		return req
	}

	// Helper function to create authenticated request with JWT token
	createAuthenticatedRequest := func(method, path string, body []byte) *http.Request {
		req := createUnauthenticatedRequest(method, path, body)

		// Create JWT token for the test user
		token, err := auth.CreateToken(&testUser)
		Expect(err).ToNot(HaveOccurred())

		// Add JWT token to Authorization header
		req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+token)

		return req
	}

	Describe("GET /song", func() {
		Context("when user is authenticated", func() {
			It("returns all songs", func() {
				req := createAuthenticatedRequest("GET", "/song", nil)
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))

				var response []model.MediaFile
				err := json.Unmarshal(w.Body.Bytes(), &response)
				Expect(err).ToNot(HaveOccurred())

				Expect(response).To(HaveLen(2))
				Expect(response[0].ID).To(Equal("song-1"))
				Expect(response[0].Title).To(Equal("Test Song 1"))
				Expect(response[1].ID).To(Equal("song-2"))
				Expect(response[1].Title).To(Equal("Test Song 2"))
			})

			It("handles repository errors gracefully", func() {
				useFailingMediaFiles()

				req := createAuthenticatedRequest("GET", "/song", nil)
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusInternalServerError))
			})
		})

		Context("when user is not authenticated", func() {
			It("returns unauthorized", func() {
				req := createUnauthenticatedRequest("GET", "/song", nil)
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})
	})

	Describe("GET /song/{id}", func() {
		Context("when user is authenticated", func() {
			It("returns the specific song", func() {
				req := createAuthenticatedRequest("GET", "/song/song-1", nil)
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))

				var response model.MediaFile
				err := json.Unmarshal(w.Body.Bytes(), &response)
				Expect(err).ToNot(HaveOccurred())

				Expect(response.ID).To(Equal("song-1"))
				Expect(response.Title).To(Equal("Test Song 1"))
				Expect(response.Artist).To(Equal("Test Artist 1"))
			})

			It("returns 404 for non-existent song", func() {
				req := createAuthenticatedRequest("GET", "/song/non-existent", nil)
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusNotFound))
			})

			It("handles repository errors gracefully", func() {
				useFailingMediaFiles()

				req := createAuthenticatedRequest("GET", "/song/song-1", nil)
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusInternalServerError))
			})
		})

		Context("when user is not authenticated", func() {
			It("returns unauthorized", func() {
				req := createUnauthenticatedRequest("GET", "/song/song-1", nil)
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})
	})

	Describe("Song endpoints are read-only", func() {
		Context("POST /song", func() {
			It("should not be available (songs are not persistable)", func() {
				newSong := model.MediaFile{
					Title:    "New Song",
					Artist:   "New Artist",
					Album:    "New Album",
					Duration: 200.0,
				}

				body, _ := json.Marshal(newSong)
				req := createAuthenticatedRequest("POST", "/song", body)
				router.ServeHTTP(w, req)

				// Should return 405 Method Not Allowed or 404 Not Found
				Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
			})
		})

		Context("PUT /song/{id}", func() {
			It("should not be available (songs are not persistable)", func() {
				updatedSong := model.MediaFile{
					ID:       "song-1",
					Title:    "Updated Song",
					Artist:   "Updated Artist",
					Album:    "Updated Album",
					Duration: 250.0,
				}

				body, _ := json.Marshal(updatedSong)
				req := createAuthenticatedRequest("PUT", "/song/song-1", body)
				router.ServeHTTP(w, req)

				// Should return 405 Method Not Allowed or 404 Not Found
				Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
			})
		})

		Context("DELETE /song/{id}", func() {
			It("should not be available (songs are not persistable)", func() {
				req := createAuthenticatedRequest("DELETE", "/song/song-1", nil)
				router.ServeHTTP(w, req)

				// Should return 405 Method Not Allowed or 404 Not Found
				Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
			})
		})
	})

	Describe("Query parameters and filtering", func() {
		Context("when using query parameters", func() {
			It("handles pagination parameters", func() {
				req := createAuthenticatedRequest("GET", "/song?_start=0&_end=1", nil)
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))

				var response []model.MediaFile
				err := json.Unmarshal(w.Body.Bytes(), &response)
				Expect(err).ToNot(HaveOccurred())

				Expect(response).To(HaveLen(1))
			})

			It("handles sort parameters", func() {
				req := createAuthenticatedRequest("GET", "/song?_sort=title&_order=DESC", nil)
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))

				var response []model.MediaFile
				err := json.Unmarshal(w.Body.Bytes(), &response)
				Expect(err).ToNot(HaveOccurred())

				Expect(response).To(HaveLen(2))
				Expect(response[0].ID).To(Equal("song-2"))
				Expect(response[1].ID).To(Equal("song-1"))
			})

			It("handles filter parameters", func() {
				// Properly encode the URL with query parameters
				baseURL := "/song"
				params := url.Values{}
				params.Add("title", "Test Song 1")
				fullURL := baseURL + "?" + params.Encode()

				req := createAuthenticatedRequest("GET", fullURL, nil)
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))

				var response []model.MediaFile
				err := json.Unmarshal(w.Body.Bytes(), &response)
				Expect(err).ToNot(HaveOccurred())

				Expect(response).To(HaveLen(1))
				Expect(response[0].ID).To(Equal("song-1"))
			})
		})
	})

	Describe("Response headers and content type", func() {
		It("sets correct content type for JSON responses", func() {
			req := createAuthenticatedRequest("GET", "/song", nil)
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(ContainSubstring("application/json"))
		})

		It("includes total count header when available", func() {
			req := createAuthenticatedRequest("GET", "/song", nil)
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("X-Total-Count")).To(Equal("2"))
		})
	})

	Describe("Edge cases and error handling", func() {
		Context("when repository is unavailable", func() {
			It("handles database connection errors", func() {
				useFailingMediaFiles()

				req := createAuthenticatedRequest("GET", "/song", nil)
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusInternalServerError))
			})
		})

		Context("when no songs exist", func() {
			It("returns empty array when no songs are found", func() {
				for _, song := range testSongs {
					Expect(ds.MediaFile().Delete(ctx, song.ID)).To(Succeed())
				}

				req := createAuthenticatedRequest("GET", "/song", nil)
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))

				var response []model.MediaFile
				err := json.Unmarshal(w.Body.Bytes(), &response)
				Expect(err).ToNot(HaveOccurred())

				Expect(response).To(BeEmpty())
			})
		})
	})

	Describe("Authentication middleware integration", func() {
		Context("with different user types", func() {
			It("works with admin users", func() {
				adminUser := model.User{
					ID:          "admin-1",
					UserName:    "admin",
					Name:        "Admin User",
					IsAdmin:     true,
					NewPassword: "adminpass",
				}
				putUsers(ctx, ds, &adminUser)

				// Create JWT token for admin user
				token, err := auth.CreateToken(&adminUser)
				Expect(err).ToNot(HaveOccurred())

				req := createUnauthenticatedRequest("GET", "/song", nil)
				req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+token)

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
			})

			It("works with regular users", func() {
				regularUser := model.User{
					ID:          "user-2",
					UserName:    "regular",
					Name:        "Regular User",
					IsAdmin:     false,
					NewPassword: "userpass",
				}
				putUsers(ctx, ds, &regularUser)

				// Create JWT token for regular user
				token, err := auth.CreateToken(&regularUser)
				Expect(err).ToNot(HaveOccurred())

				req := createUnauthenticatedRequest("GET", "/song", nil)
				req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+token)

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))
			})
		})

		Context("with missing authentication context", func() {
			It("rejects requests without user context", func() {
				req := createUnauthenticatedRequest("GET", "/song", nil)
				// No authentication header added

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})

			It("rejects requests with invalid JWT tokens", func() {
				req := createUnauthenticatedRequest("GET", "/song", nil)
				req.Header.Set(consts.UIAuthorizationHeader, "Bearer invalid.token.here")

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})
	})
})
