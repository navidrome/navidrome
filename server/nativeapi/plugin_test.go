package nativeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin API", func() {
	var ds *tests.MockDataStore
	var mockManager *tests.MockPluginManager
	var router http.Handler
	var adminUser, regularUser model.User
	var testPlugin1, testPlugin2 model.Plugin

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		ds = &tests.MockDataStore{}
		mockManager = &tests.MockPluginManager{}
		auth.Init(ds)
		nativeRouter := New(ds, nil, nil, nil, tests.NewMockLibraryService(), tests.NewMockUserService(), nil, mockManager)
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

		// Create test plugins
		testPlugin1 = model.Plugin{
			ID:       "test-plugin-1",
			Path:     "/plugins/test1.wasm",
			Manifest: `{"name":"Test Plugin 1","version":"1.0.0"}`,
			SHA256:   "abc123",
			Enabled:  false,
		}
		testPlugin2 = model.Plugin{
			ID:       "test-plugin-2",
			Path:     "/plugins/test2.wasm",
			Manifest: `{"name":"Test Plugin 2","version":"2.0.0"}`,
			Config:   `{"setting":"value"}`,
			SHA256:   "def456",
			Enabled:  true,
		}

		// Store users in mock datastore
		Expect(ds.User(GinkgoT().Context()).Put(&adminUser)).To(Succeed())
		Expect(ds.User(GinkgoT().Context()).Put(&regularUser)).To(Succeed())
	})

	Context("when plugins are disabled", func() {
		BeforeEach(func() {
			conf.Server.Plugins.Enabled = false
		})

		It("returns 404 for all plugin endpoints", func() {
			adminToken, err := auth.CreateToken(&adminUser)
			Expect(err).ToNot(HaveOccurred())

			req := httptest.NewRequest("GET", "/plugin", nil)
			req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+adminToken)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})

	Context("when plugins are enabled", func() {
		Describe("as admin user", func() {
			var adminToken string

			BeforeEach(func() {
				var err error
				adminToken, err = auth.CreateToken(&adminUser)
				Expect(err).ToNot(HaveOccurred())

				// Store test plugins as admin
				ctx := GinkgoT().Context()
				adminCtx := request.WithUser(ctx, adminUser)
				Expect(ds.Plugin(adminCtx).Put(&testPlugin1)).To(Succeed())
				Expect(ds.Plugin(adminCtx).Put(&testPlugin2)).To(Succeed())
			})

			Describe("GET /api/plugin", func() {
				It("returns all plugins", func() {
					req := httptest.NewRequest("GET", "/plugin", nil)
					req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var plugins []model.Plugin
					err := json.Unmarshal(w.Body.Bytes(), &plugins)
					Expect(err).ToNot(HaveOccurred())
					Expect(plugins).To(HaveLen(2))
				})
			})

			Describe("GET /api/plugin/{id}", func() {
				It("returns a specific plugin", func() {
					req := httptest.NewRequest("GET", "/plugin/test-plugin-1", nil)
					req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var plugin model.Plugin
					err := json.Unmarshal(w.Body.Bytes(), &plugin)
					Expect(err).ToNot(HaveOccurred())
					Expect(plugin.ID).To(Equal("test-plugin-1"))
					Expect(plugin.Path).To(Equal("/plugins/test1.wasm"))
				})

				It("returns 404 for non-existent plugin", func() {
					req := httptest.NewRequest("GET", "/plugin/non-existent", nil)
					req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusNotFound))
				})
			})

			Describe("PUT /api/plugin/{id}", func() {
				It("updates plugin enabled state", func() {
					// Configure mock to update the repo when EnablePlugin is called
					mockManager.EnablePluginFn = func(ctx context.Context, id string) error {
						adminCtx := request.WithUser(ctx, adminUser)
						p, _ := ds.Plugin(adminCtx).Get(id)
						p.Enabled = true
						return ds.Plugin(adminCtx).Put(p)
					}

					body := bytes.NewBufferString(`{"enabled":true}`)
					req := httptest.NewRequest("PUT", "/plugin/test-plugin-1", body)
					req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+adminToken)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var plugin model.Plugin
					err := json.Unmarshal(w.Body.Bytes(), &plugin)
					Expect(err).ToNot(HaveOccurred())
					Expect(plugin.Enabled).To(BeTrue())
					Expect(mockManager.EnablePluginCalls).To(ContainElement("test-plugin-1"))
				})

				It("updates plugin config with valid JSON", func() {
					// Configure mock to update the repo when UpdatePluginConfig is called
					mockManager.UpdatePluginConfigFn = func(ctx context.Context, id, configJSON string) error {
						adminCtx := request.WithUser(ctx, adminUser)
						p, _ := ds.Plugin(adminCtx).Get(id)
						p.Config = configJSON
						return ds.Plugin(adminCtx).Put(p)
					}

					body := bytes.NewBufferString(`{"config":"{\"key\":\"value\"}"}`)
					req := httptest.NewRequest("PUT", "/plugin/test-plugin-1", body)
					req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+adminToken)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var plugin model.Plugin
					err := json.Unmarshal(w.Body.Bytes(), &plugin)
					Expect(err).ToNot(HaveOccurred())
					Expect(plugin.Config).To(Equal(`{"key":"value"}`))
					Expect(mockManager.UpdatePluginConfigCalls).To(HaveLen(1))
					Expect(mockManager.UpdatePluginConfigCalls[0].ConfigJSON).To(Equal(`{"key":"value"}`))
				})

				It("rejects invalid JSON in config field", func() {
					body := bytes.NewBufferString(`{"config":"not valid json"}`)
					req := httptest.NewRequest("PUT", "/plugin/test-plugin-1", body)
					req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+adminToken)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(ContainSubstring("Invalid JSON"))
				})

				It("allows empty config", func() {
					// Configure mock to update the repo when UpdatePluginConfig is called
					mockManager.UpdatePluginConfigFn = func(ctx context.Context, id, configJSON string) error {
						adminCtx := request.WithUser(ctx, adminUser)
						p, _ := ds.Plugin(adminCtx).Get(id)
						p.Config = configJSON
						return ds.Plugin(adminCtx).Put(p)
					}

					body := bytes.NewBufferString(`{"config":""}`)
					req := httptest.NewRequest("PUT", "/plugin/test-plugin-1", body)
					req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+adminToken)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var plugin model.Plugin
					err := json.Unmarshal(w.Body.Bytes(), &plugin)
					Expect(err).ToNot(HaveOccurred())
					Expect(plugin.Config).To(Equal(""))
				})

				It("updates users field", func() {
					// Configure mock to update the repo when UpdatePluginUsers is called
					mockManager.UpdatePluginUsersFn = func(ctx context.Context, id, usersJSON string, allUsers bool) error {
						adminCtx := request.WithUser(ctx, adminUser)
						p, _ := ds.Plugin(adminCtx).Get(id)
						p.Users = usersJSON
						p.AllUsers = allUsers
						return ds.Plugin(adminCtx).Put(p)
					}

					body := bytes.NewBufferString(`{"users":"[\"user1\",\"user2\"]"}`)
					req := httptest.NewRequest("PUT", "/plugin/test-plugin-1", body)
					req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+adminToken)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var plugin model.Plugin
					err := json.Unmarshal(w.Body.Bytes(), &plugin)
					Expect(err).ToNot(HaveOccurred())
					Expect(plugin.Users).To(Equal(`["user1","user2"]`))
					Expect(mockManager.UpdatePluginUsersCalls).To(HaveLen(1))
					Expect(mockManager.UpdatePluginUsersCalls[0].UsersJSON).To(Equal(`["user1","user2"]`))
				})

				It("updates allUsers field", func() {
					// Configure mock to update the repo when UpdatePluginUsers is called
					mockManager.UpdatePluginUsersFn = func(ctx context.Context, id, usersJSON string, allUsers bool) error {
						adminCtx := request.WithUser(ctx, adminUser)
						p, _ := ds.Plugin(adminCtx).Get(id)
						p.Users = usersJSON
						p.AllUsers = allUsers
						return ds.Plugin(adminCtx).Put(p)
					}

					body := bytes.NewBufferString(`{"allUsers":true}`)
					req := httptest.NewRequest("PUT", "/plugin/test-plugin-1", body)
					req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+adminToken)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var plugin model.Plugin
					err := json.Unmarshal(w.Body.Bytes(), &plugin)
					Expect(err).ToNot(HaveOccurred())
					Expect(plugin.AllUsers).To(BeTrue())
					Expect(mockManager.UpdatePluginUsersCalls).To(HaveLen(1))
					Expect(mockManager.UpdatePluginUsersCalls[0].AllUsers).To(BeTrue())
				})

				It("updates both users and allUsers fields together", func() {
					// Configure mock to update the repo when UpdatePluginUsers is called
					mockManager.UpdatePluginUsersFn = func(ctx context.Context, id, usersJSON string, allUsers bool) error {
						adminCtx := request.WithUser(ctx, adminUser)
						p, _ := ds.Plugin(adminCtx).Get(id)
						p.Users = usersJSON
						p.AllUsers = allUsers
						return ds.Plugin(adminCtx).Put(p)
					}

					body := bytes.NewBufferString(`{"users":"[\"user1\"]","allUsers":false}`)
					req := httptest.NewRequest("PUT", "/plugin/test-plugin-1", body)
					req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+adminToken)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var plugin model.Plugin
					err := json.Unmarshal(w.Body.Bytes(), &plugin)
					Expect(err).ToNot(HaveOccurred())
					Expect(plugin.Users).To(Equal(`["user1"]`))
					Expect(plugin.AllUsers).To(BeFalse())
					Expect(mockManager.UpdatePluginUsersCalls).To(HaveLen(1))
				})

				It("rejects invalid JSON in users field", func() {
					body := bytes.NewBufferString(`{"users":"not valid json"}`)
					req := httptest.NewRequest("PUT", "/plugin/test-plugin-1", body)
					req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+adminToken)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusBadRequest))
					Expect(w.Body.String()).To(ContainSubstring("Invalid JSON"))
				})

				It("allows empty users", func() {
					// Configure mock to update the repo when UpdatePluginUsers is called
					mockManager.UpdatePluginUsersFn = func(ctx context.Context, id, usersJSON string, allUsers bool) error {
						adminCtx := request.WithUser(ctx, adminUser)
						p, _ := ds.Plugin(adminCtx).Get(id)
						p.Users = usersJSON
						p.AllUsers = allUsers
						return ds.Plugin(adminCtx).Put(p)
					}

					body := bytes.NewBufferString(`{"users":""}`)
					req := httptest.NewRequest("PUT", "/plugin/test-plugin-1", body)
					req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+adminToken)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))

					var plugin model.Plugin
					err := json.Unmarshal(w.Body.Bytes(), &plugin)
					Expect(err).ToNot(HaveOccurred())
					Expect(plugin.Users).To(Equal(""))
				})

				It("returns 404 for non-existent plugin", func() {
					body := bytes.NewBufferString(`{"enabled":true}`)
					req := httptest.NewRequest("PUT", "/plugin/non-existent", body)
					req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+adminToken)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusNotFound))
				})

				It("returns 400 for invalid request body", func() {
					body := bytes.NewBufferString(`not json`)
					req := httptest.NewRequest("PUT", "/plugin/test-plugin-1", body)
					req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+adminToken)
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusBadRequest))
				})
			})

			Describe("POST /api/plugin/rescan", func() {
				It("triggers plugin rescan", func() {
					req := httptest.NewRequest("POST", "/plugin/rescan", nil)
					req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusOK))
					Expect(mockManager.RescanPluginsCalls).To(Equal(1))
				})

				It("returns error when rescan fails", func() {
					mockManager.RescanError = errors.New("folder not configured")

					req := httptest.NewRequest("POST", "/plugin/rescan", nil)
					req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+adminToken)
					w := httptest.NewRecorder()

					router.ServeHTTP(w, req)

					Expect(w.Code).To(Equal(http.StatusInternalServerError))
					Expect(w.Body.String()).To(ContainSubstring("folder not configured"))
				})
			})
		})

		Describe("as regular user", func() {
			var userToken string

			BeforeEach(func() {
				var err error
				userToken, err = auth.CreateToken(&regularUser)
				Expect(err).ToNot(HaveOccurred())
			})

			It("denies access to GET /api/plugin", func() {
				req := httptest.NewRequest("GET", "/plugin", nil)
				req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+userToken)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusForbidden))
			})

			It("denies access to GET /api/plugin/{id}", func() {
				req := httptest.NewRequest("GET", "/plugin/test-plugin-1", nil)
				req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+userToken)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusForbidden))
			})

			It("denies access to PUT /api/plugin/{id}", func() {
				body := bytes.NewBufferString(`{"enabled":true}`)
				req := httptest.NewRequest("PUT", "/plugin/test-plugin-1", body)
				req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+userToken)
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusForbidden))
			})

			It("denies access to POST /api/plugin/rescan", func() {
				req := httptest.NewRequest("POST", "/plugin/rescan", nil)
				req.Header.Set(consts.UIAuthorizationHeader, "Bearer "+userToken)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusForbidden))
			})
		})

		Describe("without authentication", func() {
			It("denies access to plugin endpoints", func() {
				req := httptest.NewRequest("GET", "/plugin", nil)
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				Expect(w.Code).To(Equal(http.StatusUnauthorized))
			})
		})
	})
})
