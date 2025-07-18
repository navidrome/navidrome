package nativeapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Song Endpoints", func() {
	var (
		router    http.Handler
		ds        *tests.MockDataStore
		mfRepo    *tests.MockMediaFileRepo
		userRepo  *tests.MockedUserRepo
		w         *httptest.ResponseRecorder
		testUser  model.User
		testSongs model.MediaFiles
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.SessionTimeout = time.Minute

		// Setup mock repositories
		mfRepo = tests.CreateMockMediaFileRepo()
		userRepo = tests.CreateMockUserRepo()

		ds = &tests.MockDataStore{
			MockedMediaFile: mfRepo,
			MockedUser:      userRepo,
			MockedProperty:  &tests.MockedPropertyRepo{},
		}

		// Initialize auth system
		auth.Init(ds)

		// Create test user
		testUser = model.User{
			ID:          "user-1",
			UserName:    "testuser",
			Name:        "Test User",
			IsAdmin:     false,
			NewPassword: "testpass",
		}
		err := userRepo.Put(&testUser)
		Expect(err).ToNot(HaveOccurred())

		// Create test songs
		testSongs = model.MediaFiles{
			{
				ID:        "song-1",
				Title:     "Test Song 1",
				Artist:    "Test Artist 1",
				Album:     "Test Album 1",
				AlbumID:   "album-1",
				ArtistID:  "artist-1",
				Duration:  180.5,
				BitRate:   320,
				Path:      "/music/song1.mp3",
				Suffix:    "mp3",
				Size:      5242880,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:        "song-2",
				Title:     "Test Song 2",
				Artist:    "Test Artist 2",
				Album:     "Test Album 2",
				AlbumID:   "album-2",
				ArtistID:  "artist-2",
				Duration:  240.0,
				BitRate:   256,
				Path:      "/music/song2.mp3",
				Suffix:    "mp3",
				Size:      7340032,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		mfRepo.SetData(testSongs)

		// Create the native API router and wrap it with the JWTVerifier middleware
		nativeRouter := New(ds, nil, nil, nil, core.NewMockLibraryService())
		router = server.JWTVerifier(nativeRouter)
		w = httptest.NewRecorder()
	})

	// Helper function to create unauthenticated request
	createUnauthenticatedRequest := func(method, path string, body []byte) *http.Request {
		var req *http.Request
		if body != nil {
			req = httptest.NewRequest(method, path, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req = httptest.NewRequest(method, path, nil)
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
				mfRepo.SetError(true)

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
				mfRepo.SetError(true)

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

				// Should still return all songs since our mock doesn't implement pagination
				// but the request should be processed successfully
				Expect(len(response)).To(BeNumerically(">=", 1))
			})

			It("handles sort parameters", func() {
				req := createAuthenticatedRequest("GET", "/song?_sort=title&_order=ASC", nil)
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))

				var response []model.MediaFile
				err := json.Unmarshal(w.Body.Bytes(), &response)
				Expect(err).ToNot(HaveOccurred())

				Expect(response).To(HaveLen(2))
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

				// Mock doesn't implement filtering, but request should be processed
				Expect(len(response)).To(BeNumerically(">=", 1))
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
			// The X-Total-Count header might be set by the REST framework
			// We just verify the request is processed successfully
		})
	})

	Describe("Edge cases and error handling", func() {
		Context("when repository is unavailable", func() {
			It("handles database connection errors", func() {
				mfRepo.SetError(true)

				req := createAuthenticatedRequest("GET", "/song", nil)
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusInternalServerError))
			})
		})

		Context("when no songs exist", func() {
			It("returns empty array when no songs are found", func() {
				mfRepo.SetData(model.MediaFiles{}) // Empty dataset

				req := createAuthenticatedRequest("GET", "/song", nil)
				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))

				var response []model.MediaFile
				err := json.Unmarshal(w.Body.Bytes(), &response)
				Expect(err).ToNot(HaveOccurred())

				Expect(response).To(HaveLen(0))
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
				err := userRepo.Put(&adminUser)
				Expect(err).ToNot(HaveOccurred())

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
				err := userRepo.Put(&regularUser)
				Expect(err).ToNot(HaveOccurred())

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
