//go:build !windows

package plugins

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakeNativeAuth is a mock native auth middleware that authenticates by looking up
// the "X-Test-User" header and setting the user in the context.
func fakeNativeAuth(ds model.DataStore) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username := r.Header.Get("X-Test-User")
			if username == "" {
				http.Error(w, "Not authenticated", http.StatusUnauthorized)
				return
			}
			user, err := ds.User(r.Context()).FindByUsername(username)
			if err != nil {
				http.Error(w, "Not authenticated", http.StatusUnauthorized)
				return
			}
			ctx := request.WithUser(r.Context(), *user)
			ctx = request.WithUsername(ctx, user.UserName)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// fakeSubsonicAuth is a mock subsonic auth that validates by looking up
// the "u" query parameter.
func fakeSubsonicAuth(ds model.DataStore, r *http.Request) (*model.User, error) {
	username := r.URL.Query().Get("u")
	if username == "" {
		return nil, model.ErrInvalidAuth
	}
	user, err := ds.User(r.Context()).FindByUsername(username)
	if err != nil {
		return nil, model.ErrInvalidAuth
	}
	return user, nil
}

var _ = Describe("HTTP Endpoint Handler", Ordered, func() {
	var (
		manager   *Manager
		tmpDir    string
		userRepo  *tests.MockedUserRepo
		dataStore *tests.MockDataStore
		router    http.Handler
	)

	BeforeAll(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "http-endpoint-test-*")
		Expect(err).ToNot(HaveOccurred())

		// Copy both test plugins
		for _, pluginName := range []string{"test-http-endpoint", "test-http-endpoint-public"} {
			srcPath := filepath.Join(testdataDir, pluginName+PackageExtension)
			destPath := filepath.Join(tmpDir, pluginName+PackageExtension)
			data, err := os.ReadFile(srcPath)
			Expect(err).ToNot(HaveOccurred())
			err = os.WriteFile(destPath, data, 0600)
			Expect(err).ToNot(HaveOccurred())
		}

		// Setup config
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = tmpDir
		conf.Server.Plugins.AutoReload = false
		conf.Server.CacheFolder = filepath.Join(tmpDir, "cache")

		// Setup mock data store
		userRepo = tests.CreateMockUserRepo()
		dataStore = &tests.MockDataStore{MockedUser: userRepo}

		// Add test users
		_ = userRepo.Put(&model.User{
			ID:       "user1",
			UserName: "testuser",
			Name:     "Test User",
			IsAdmin:  false,
		})
		_ = userRepo.Put(&model.User{
			ID:       "admin1",
			UserName: "adminuser",
			Name:     "Admin User",
			IsAdmin:  true,
		})

		// Build enabled plugins list
		var enabledPlugins model.Plugins
		for _, pluginName := range []string{"test-http-endpoint", "test-http-endpoint-public"} {
			pluginPath := filepath.Join(tmpDir, pluginName+PackageExtension)
			data, err := os.ReadFile(pluginPath)
			Expect(err).ToNot(HaveOccurred())
			hash := sha256.Sum256(data)
			hashHex := hex.EncodeToString(hash[:])

			enabledPlugins = append(enabledPlugins, model.Plugin{
				ID:       pluginName,
				Path:     pluginPath,
				SHA256:   hashHex,
				Enabled:  true,
				AllUsers: true,
			})
		}

		// Setup mock plugin repo
		mockPluginRepo := dataStore.Plugin(GinkgoT().Context()).(*tests.MockPluginRepo)
		mockPluginRepo.Permitted = true
		mockPluginRepo.SetData(enabledPlugins)

		// Create and start manager
		manager = &Manager{
			plugins:        make(map[string]*plugin),
			ds:             dataStore,
			metrics:        noopMetricsRecorder{},
			subsonicRouter: http.NotFoundHandler(),
		}
		err = manager.Start(GinkgoT().Context())
		Expect(err).ToNot(HaveOccurred())

		// Create the endpoint router with fake auth functions
		router = NewEndpointRouter(manager, dataStore, fakeSubsonicAuth, fakeNativeAuth)

		DeferCleanup(func() {
			_ = manager.Stop()
			_ = os.RemoveAll(tmpDir)
		})
	})

	Describe("Plugin Loading", func() {
		It("loads the authenticated endpoint plugin", func() {
			manager.mu.RLock()
			p := manager.plugins["test-http-endpoint"]
			manager.mu.RUnlock()

			Expect(p).ToNot(BeNil())
			Expect(p.manifest.Name).To(Equal("Test HTTP Endpoint Plugin"))
			Expect(p.manifest.Permissions.Endpoints).ToNot(BeNil())
			Expect(string(p.manifest.Permissions.Endpoints.Auth)).To(Equal("subsonic"))
			Expect(hasCapability(p.capabilities, CapabilityHTTPEndpoint)).To(BeTrue())
		})

		It("loads the public endpoint plugin", func() {
			manager.mu.RLock()
			p := manager.plugins["test-http-endpoint-public"]
			manager.mu.RUnlock()

			Expect(p).ToNot(BeNil())
			Expect(p.manifest.Name).To(Equal("Test HTTP Endpoint Public Plugin"))
			Expect(p.manifest.Permissions.Endpoints).ToNot(BeNil())
			Expect(string(p.manifest.Permissions.Endpoints.Auth)).To(Equal("none"))
			Expect(hasCapability(p.capabilities, CapabilityHTTPEndpoint)).To(BeTrue())
		})
	})

	Describe("Subsonic Auth Endpoints", func() {
		It("returns hello response with valid auth", func() {
			req := httptest.NewRequest("GET", "/test-http-endpoint/hello?u=testuser", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(Equal("Hello from plugin!"))
			Expect(w.Header().Get("Content-Type")).To(Equal("text/plain"))
		})

		It("returns echo response with request details", func() {
			req := httptest.NewRequest("POST", "/test-http-endpoint/echo?u=testuser&foo=bar", strings.NewReader("test body"))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("application/json"))

			var resp map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp["method"]).To(Equal("POST"))
			Expect(resp["path"]).To(Equal("/echo"))
			Expect(resp["body"]).To(Equal("test body"))
			Expect(resp["hasUser"]).To(BeTrue())
			Expect(resp["username"]).To(Equal("testuser"))
		})

		It("returns plugin-defined error status", func() {
			req := httptest.NewRequest("GET", "/test-http-endpoint/error?u=testuser", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusInternalServerError))
			Expect(w.Body.String()).To(Equal("Something went wrong"))
		})

		It("returns plugin 404 for unknown paths", func() {
			req := httptest.NewRequest("GET", "/test-http-endpoint/unknown?u=testuser", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNotFound))
			Expect(w.Body.String()).To(Equal("Not found: /unknown"))
		})

		It("returns 401 without auth credentials", func() {
			req := httptest.NewRequest("GET", "/test-http-endpoint/hello", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})

		It("returns 401 with invalid auth credentials", func() {
			req := httptest.NewRequest("GET", "/test-http-endpoint/hello?u=nonexistent", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})
	})

	Describe("Public Endpoints (auth: none)", func() {
		It("returns webhook response without auth", func() {
			req := httptest.NewRequest("POST", "/test-http-endpoint-public/webhook", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(Equal("webhook received"))
		})

		It("does not pass user info to public endpoints", func() {
			req := httptest.NewRequest("GET", "/test-http-endpoint-public/check-no-user", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(Equal("hasUser=false"))
		})
	})

	Describe("Unknown Plugin", func() {
		It("returns 404 for nonexistent plugin", func() {
			req := httptest.NewRequest("GET", "/nonexistent-plugin/hello", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("User Authorization", func() {
		var restrictedRouter http.Handler

		BeforeAll(func() {
			// Create a manager with a plugin restricted to specific users
			restrictedTmpDir, err := os.MkdirTemp("", "http-endpoint-restricted-test-*")
			Expect(err).ToNot(HaveOccurred())

			srcPath := filepath.Join(testdataDir, "test-http-endpoint"+PackageExtension)
			destPath := filepath.Join(restrictedTmpDir, "test-http-endpoint"+PackageExtension)
			data, err := os.ReadFile(srcPath)
			Expect(err).ToNot(HaveOccurred())
			err = os.WriteFile(destPath, data, 0600)
			Expect(err).ToNot(HaveOccurred())

			hash := sha256.Sum256(data)
			hashHex := hex.EncodeToString(hash[:])

			DeferCleanup(configtest.SetupConfig())
			conf.Server.Plugins.Enabled = true
			conf.Server.Plugins.Folder = restrictedTmpDir
			conf.Server.Plugins.AutoReload = false
			conf.Server.CacheFolder = filepath.Join(restrictedTmpDir, "cache")

			restrictedPluginRepo := tests.CreateMockPluginRepo()
			restrictedPluginRepo.Permitted = true
			restrictedPluginRepo.SetData(model.Plugins{{
				ID:       "test-http-endpoint",
				Path:     destPath,
				SHA256:   hashHex,
				Enabled:  true,
				AllUsers: false,
				Users:    `["admin1"]`, // Only admin1 is allowed
			}})
			restrictedDS := &tests.MockDataStore{
				MockedPlugin: restrictedPluginRepo,
				MockedUser:   userRepo,
			}

			restrictedManager := &Manager{
				plugins:        make(map[string]*plugin),
				ds:             restrictedDS,
				metrics:        noopMetricsRecorder{},
				subsonicRouter: http.NotFoundHandler(),
			}
			err = restrictedManager.Start(GinkgoT().Context())
			Expect(err).ToNot(HaveOccurred())

			restrictedRouter = NewEndpointRouter(restrictedManager, restrictedDS, fakeSubsonicAuth, fakeNativeAuth)

			DeferCleanup(func() {
				_ = restrictedManager.Stop()
				_ = os.RemoveAll(restrictedTmpDir)
			})
		})

		It("allows authorized users", func() {
			req := httptest.NewRequest("GET", "/test-http-endpoint/hello?u=adminuser", nil)
			w := httptest.NewRecorder()
			restrictedRouter.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.String()).To(Equal("Hello from plugin!"))
		})

		It("denies unauthorized users", func() {
			req := httptest.NewRequest("GET", "/test-http-endpoint/hello?u=testuser", nil)
			w := httptest.NewRecorder()
			restrictedRouter.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusForbidden))
		})
	})

	Describe("Request without trailing path", func() {
		It("handles requests to plugin root", func() {
			req := httptest.NewRequest("GET", "/test-http-endpoint-public/webhook", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("Request body handling", func() {
		It("passes request body to the plugin", func() {
			body := `{"event":"push","ref":"refs/heads/main"}`
			req := httptest.NewRequest("POST", "/test-http-endpoint/echo?u=testuser", strings.NewReader(body))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))

			respBody, err := io.ReadAll(w.Body)
			Expect(err).ToNot(HaveOccurred())

			var resp map[string]any
			err = json.Unmarshal(respBody, &resp)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp["body"]).To(Equal(body))
		})
	})
})
