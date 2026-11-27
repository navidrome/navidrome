package nativeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

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

var _ = Describe("Library API", func() {
	var ctx context.Context
	var ds model.DataStore
	var router http.Handler
	var adminUser, regularUser model.User
	var library1, library2 model.Library
	var library1Path, library2Path string

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		conf.Update(func(c *conf.Options) { c.EnableSharing = false })
		ds = persistence.New(db.Db())
		auth.Init(ds)
		nativeRouter := newTestRouter(ds, nil, nil, nil)
		router = server.JWTVerifier(nativeRouter)

		// Create test users
		adminUser = model.User{
			ID:          "admin-1",
			UserName:    "admin",
			Name:        "Admin User",
			IsAdmin:     true,
			NewPassword: "adminpass",
		}
		regularUser = model.User{
			ID:          "user-1",
			UserName:    "regular",
			Name:        "Regular User",
			IsAdmin:     false,
			NewPassword: "userpass",
		}

		putUsers(ctx, ds, &adminUser, &regularUser)
		library1Path, library2Path = GinkgoT().TempDir(), GinkgoT().TempDir()
		persistencetest.SetLibraries(ctx, ds, library1Path, library2Path)
		library1 = model.Library{ID: 1, Name: "Test Library 1", Path: library1Path}
		library2 = model.Library{ID: 2, Name: "Test Library 2", Path: library2Path}
		Expect(ds.Library().Put(ctx, &library1)).To(Succeed())
		Expect(ds.Library().Put(ctx, &library2)).To(Succeed())
	})

	Describe("Library CRUD Operations", func() {
		Context("as admin user", func() {
			var adminToken string

			BeforeEach(func() {
				var err error
				adminToken, err = auth.CreateToken(&adminUser)
				Expect(err).ToNot(HaveOccurred())
			})

			Describe("GET /api/library", func() {
				It("returns all libraries", func() {
					req := createAuthenticatedRequest("GET", "/library", nil, adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var libraries []model.Library
					err := json.Unmarshal(w.Body.Bytes(), &libraries)
					Expect(err).ToNot(HaveOccurred())
					Expect(libraries).To(HaveLen(2))
					Expect(libraries[0].Name).To(Equal("Test Library 1"))
					Expect(libraries[1].Name).To(Equal("Test Library 2"))
				})
			})

			Describe("GET /api/library/{id}", func() {
				It("returns a specific library", func() {
					req := createAuthenticatedRequest("GET", "/library/1", nil, adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var library model.Library
					err := json.Unmarshal(w.Body.Bytes(), &library)
					Expect(err).ToNot(HaveOccurred())
					Expect(library.Name).To(Equal("Test Library 1"))
					Expect(library.Path).To(Equal(library1Path))
				})

				It("returns 404 for non-existent library", func() {
					req := createAuthenticatedRequest("GET", "/library/999", nil, adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusNotFound))
				})

				It("returns 400 for invalid library ID", func() {
					req := createAuthenticatedRequest("GET", "/library/invalid", nil, adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusNotFound))
				})
			})

			Describe("POST /api/library", func() {
				It("creates a new library", func() {
					newLibrary := model.Library{
						Name: "New Library",
						Path: GinkgoT().TempDir(),
					}
					body, _ := json.Marshal(newLibrary)
					req := createAuthenticatedRequest("POST", "/library", bytes.NewBuffer(body), adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK), w.Body.String())
					var created struct{ ID string }
					Expect(json.Unmarshal(w.Body.Bytes(), &created)).To(Succeed())
					id, err := strconv.Atoi(created.ID)
					Expect(err).ToNot(HaveOccurred())
					stored, err := ds.Library().Get(ctx, id)
					Expect(err).ToNot(HaveOccurred())
					Expect(stored.Name).To(Equal("New Library"))
					Expect(stored.Path).To(Equal(newLibrary.Path))
				})

				It("validates required fields", func() {
					invalidLibrary := model.Library{
						Name: "", // Missing name
						Path: GinkgoT().TempDir(),
					}
					body, _ := json.Marshal(invalidLibrary)
					req := createAuthenticatedRequest("POST", "/library", bytes.NewBuffer(body), adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(ContainSubstring(`"name":"ra.validation.required"`))
				})

				It("validates path field", func() {
					invalidLibrary := model.Library{
						Name: "Valid Name",
						Path: "", // Missing path
					}
					body, _ := json.Marshal(invalidLibrary)
					req := createAuthenticatedRequest("POST", "/library", bytes.NewBuffer(body), adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(ContainSubstring(`"path":"ra.validation.required"`))
				})
			})

			Describe("PUT /api/library/{id}", func() {
				It("updates an existing library", func() {
					updatedLibrary := model.Library{
						Name: "Updated Library 2",
						Path: GinkgoT().TempDir(),
					}
					body, _ := json.Marshal(updatedLibrary)
					req := createAuthenticatedRequest("PUT", "/library/2", bytes.NewBuffer(body), adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var updated model.Library
					err := json.Unmarshal(w.Body.Bytes(), &updated)
					Expect(err).ToNot(HaveOccurred())
					Expect(updated.ID).To(Equal(2))
					Expect(updated.Name).To(Equal("Updated Library 2"))
					Expect(updated.Path).To(Equal(updatedLibrary.Path))
					stored, err := ds.Library().Get(ctx, 2)
					Expect(err).ToNot(HaveOccurred())
					Expect(stored.Name).To(Equal("Updated Library 2"))
					Expect(stored.Path).To(Equal(updatedLibrary.Path))
				})

				It("validates required fields on update", func() {
					invalidLibrary := model.Library{
						Name: "",
						Path: library2Path,
					}
					body, _ := json.Marshal(invalidLibrary)
					req := createAuthenticatedRequest("PUT", "/library/2", bytes.NewBuffer(body), adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusBadRequest))
					stored, err := ds.Library().Get(ctx, 2)
					Expect(err).ToNot(HaveOccurred())
					Expect(stored.Name).To(Equal("Test Library 2"))
				})
			})

			Describe("DELETE /api/library/{id}", func() {
				It("deletes an existing library", func() {
					req := createAuthenticatedRequest("DELETE", "/library/2", nil, adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))
					_, err := ds.Library().Get(ctx, 2)
					Expect(err).To(MatchError(model.ErrNotFound))
				})

				It("returns 404 for non-existent library", func() {
					req := createAuthenticatedRequest("DELETE", "/library/999", nil, adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusNotFound))
				})
			})
		})

		Context("as regular user", func() {
			var userToken string

			BeforeEach(func() {
				var err error
				userToken, err = auth.CreateToken(&regularUser)
				Expect(err).ToNot(HaveOccurred())
			})

			It("denies access to library management endpoints", func() {
				endpoints := []string{
					"GET /library",
					"POST /library",
					"GET /library/1",
					"PUT /library/1",
					"DELETE /library/1",
				}

				for _, endpoint := range endpoints {
					parts := strings.Split(endpoint, " ")
					method, path := parts[0], parts[1]

					req := createAuthenticatedRequest(method, path, nil, userToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusForbidden))
				}
			})
		})

		Context("without authentication", func() {
			It("denies access to library management endpoints", func() {
				req := createUnauthenticatedRequest("GET", "/library", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})
	})

	Describe("User-Library Association Operations", func() {
		Context("as admin user", func() {
			var adminToken string

			BeforeEach(func() {
				var err error
				adminToken, err = auth.CreateToken(&adminUser)
				Expect(err).ToNot(HaveOccurred())
			})

			Describe("GET /api/user/{id}/library", func() {
				It("returns user's libraries", func() {
					// Set up user libraries
					err := ds.User().SetUserLibraries(ctx, regularUser.ID, []int{1, 2})
					Expect(err).ToNot(HaveOccurred())

					req := createAuthenticatedRequest("GET", fmt.Sprintf("/user/%s/library", regularUser.ID), nil, adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var libraries []model.Library
					err = json.Unmarshal(w.Body.Bytes(), &libraries)
					Expect(err).ToNot(HaveOccurred())
					Expect(libraries).To(HaveLen(2))
				})

				It("returns 404 for non-existent user", func() {
					req := createAuthenticatedRequest("GET", "/user/non-existent/library", nil, adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusNotFound))
				})
			})

			Describe("PUT /api/user/{id}/library", func() {
				It("rejects a body over the size limit with 413", func() {
					body := `{"libraryIds":[1],"pad":"` + strings.Repeat("x", maxSmallJSONBody) + `"}`
					req := createAuthenticatedRequest("PUT", fmt.Sprintf("/user/%s/library", regularUser.ID), bytes.NewBufferString(body), adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusRequestEntityTooLarge))
				})

				It("sets user's libraries", func() {
					Expect(ds.User().SetUserLibraries(ctx, regularUser.ID, []int{1})).To(Succeed())
					request := map[string][]int{
						"libraryIds": {1, 2},
					}
					body, _ := json.Marshal(request)
					req := createAuthenticatedRequest("PUT", fmt.Sprintf("/user/%s/library", regularUser.ID), bytes.NewBuffer(body), adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var libraries []model.Library
					err := json.Unmarshal(w.Body.Bytes(), &libraries)
					Expect(err).ToNot(HaveOccurred())
					Expect(libraries).To(HaveLen(2))
					stored, err := ds.User().GetUserLibraries(ctx, regularUser.ID)
					Expect(err).ToNot(HaveOccurred())
					Expect(stored).To(HaveLen(2))
				})

				It("validates library IDs exist", func() {
					request := map[string][]int{
						"libraryIds": {999}, // Non-existent library
					}
					body, _ := json.Marshal(request)
					req := createAuthenticatedRequest("PUT", fmt.Sprintf("/user/%s/library", regularUser.ID), bytes.NewBuffer(body), adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(ContainSubstring("one or more library IDs are invalid"))
				})

				It("requires at least one library for regular users", func() {
					request := map[string][]int{
						"libraryIds": {}, // Empty libraries
					}
					body, _ := json.Marshal(request)
					req := createAuthenticatedRequest("PUT", fmt.Sprintf("/user/%s/library", regularUser.ID), bytes.NewBuffer(body), adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(ContainSubstring("at least one library must be assigned"))
				})

				It("prevents manual assignment to admin users", func() {
					request := map[string][]int{
						"libraryIds": {1},
					}
					body, _ := json.Marshal(request)
					req := createAuthenticatedRequest("PUT", fmt.Sprintf("/user/%s/library", adminUser.ID), bytes.NewBuffer(body), adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(ContainSubstring("cannot manually assign libraries to admin users"))
				})
			})
		})

		Context("as regular user", func() {
			var userToken string

			BeforeEach(func() {
				var err error
				userToken, err = auth.CreateToken(&regularUser)
				Expect(err).ToNot(HaveOccurred())
			})

			It("denies access to user-library association endpoints", func() {
				req := createAuthenticatedRequest("GET", fmt.Sprintf("/user/%s/library", regularUser.ID), nil, userToken)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusForbidden))
			})
		})
	})
})

// Helper functions

func createAuthenticatedRequest(method, path string, body *bytes.Buffer, token string) *http.Request {
	if body == nil {
		body = &bytes.Buffer{}
	}
	req := httptest.NewRequestWithContext(GinkgoT().Context(), method, path, body)
	req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func createUnauthenticatedRequest(method, path string, body *bytes.Buffer) *http.Request {
	if body == nil {
		body = &bytes.Buffer{}
	}
	req := httptest.NewRequestWithContext(GinkgoT().Context(), method, path, body)
	req.Header.Set("Content-Type", "application/json")
	return req
}
