//go:build !windows

package plugins

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SubsonicAPI Host Function", Ordered, func() {
	var (
		manager   *Manager
		tmpDir    string
		router    *fakeSubsonicRouter
		userRepo  *tests.MockedUserRepo
		dataStore *tests.MockDataStore
	)

	BeforeAll(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "subsonicapi-test-*")
		Expect(err).ToNot(HaveOccurred())

		// Copy test plugin to temp dir
		srcPath := filepath.Join(testdataDir, "test-subsonicapi-plugin"+PackageExtension)
		destPath := filepath.Join(tmpDir, "test-subsonicapi-plugin"+PackageExtension)
		data, err := os.ReadFile(srcPath)
		Expect(err).ToNot(HaveOccurred())
		err = os.WriteFile(destPath, data, 0600)
		Expect(err).ToNot(HaveOccurred())

		// Setup config
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = tmpDir
		conf.Server.Plugins.AutoReload = false
		conf.Server.CacheFolder = filepath.Join(tmpDir, "cache")

		// Setup mock router and data store
		router = &fakeSubsonicRouter{}
		userRepo = tests.CreateMockUserRepo()
		dataStore = &tests.MockDataStore{MockedUser: userRepo}

		// Add test users
		_ = userRepo.Put(&model.User{
			ID:       "user1",
			UserName: "testuser",
			IsAdmin:  false,
		})
		_ = userRepo.Put(&model.User{
			ID:       "admin1",
			UserName: "adminuser",
			IsAdmin:  true,
		})

		// Create and configure manager
		manager = &Manager{
			plugins: make(map[string]*plugin),
			ds:      dataStore,
		}
		manager.SetSubsonicRouter(router)

		// Pre-enable the plugin in the mock repo so it loads on startup
		// Compute SHA256 of the plugin file to match what syncPlugins will compute
		pluginPath := filepath.Join(tmpDir, "test-subsonicapi-plugin"+PackageExtension)
		wasmData, err := os.ReadFile(pluginPath)
		Expect(err).ToNot(HaveOccurred())
		hash := sha256.Sum256(wasmData)
		hashHex := hex.EncodeToString(hash[:])

		mockPluginRepo := dataStore.Plugin(GinkgoT().Context()).(*tests.MockPluginRepo)
		mockPluginRepo.Permitted = true
		enabledPlugin := model.Plugin{
			ID:       "test-subsonicapi-plugin",
			Path:     pluginPath,
			SHA256:   hashHex,
			Enabled:  true,
			AllUsers: true, // Allow all users for test plugin
		}
		mockPluginRepo.SetData(model.Plugins{enabledPlugin})

		// Start the manager
		err = manager.Start(GinkgoT().Context())
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(func() {
			_ = manager.Stop()
			_ = os.RemoveAll(tmpDir)
		})
	})

	Describe("Plugin Loading", func() {
		It("loads the plugin with SubsonicAPI permission", func() {
			manager.mu.RLock()
			plugin := manager.plugins["test-subsonicapi-plugin"]
			manager.mu.RUnlock()

			Expect(plugin).ToNot(BeNil())
		})

		It("has the correct manifest", func() {
			manager.mu.RLock()
			plugin := manager.plugins["test-subsonicapi-plugin"]
			manager.mu.RUnlock()

			Expect(plugin).ToNot(BeNil())
			Expect(plugin.manifest.Name).To(Equal("Test SubsonicAPI Plugin"))
			Expect(plugin.manifest.Permissions.Subsonicapi).ToNot(BeNil())
		})
	})

	Describe("SubsonicAPI Call", func() {
		var plugin *plugin

		BeforeEach(func() {
			manager.mu.RLock()
			plugin = manager.plugins["test-subsonicapi-plugin"]
			manager.mu.RUnlock()
			Expect(plugin).ToNot(BeNil())
		})

		It("successfully calls the ping endpoint", func() {
			instance, err := plugin.instance(GinkgoT().Context())
			Expect(err).ToNot(HaveOccurred())
			defer instance.Close(GinkgoT().Context())

			exit, output, err := instance.Call("call_subsonic_api", []byte("/ping?u=testuser"))
			Expect(err).ToNot(HaveOccurred())
			Expect(exit).To(Equal(uint32(0)))

			// Verify the response contains the expected structure
			var response map[string]any
			err = json.Unmarshal(output, &response)
			Expect(err).ToNot(HaveOccurred())

			subsonicResponse, ok := response["subsonic-response"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(subsonicResponse["status"]).To(Equal("ok"))
		})

		It("adds required parameters (c, f, v) to the request", func() {
			instance, err := plugin.instance(GinkgoT().Context())
			Expect(err).ToNot(HaveOccurred())
			defer instance.Close(GinkgoT().Context())

			_, _, err = instance.Call("call_subsonic_api", []byte("/getAlbumList?u=testuser&type=newest"))
			Expect(err).ToNot(HaveOccurred())

			// Verify the parameters were added
			Expect(router.lastRequest).ToNot(BeNil())
			query := router.lastRequest.URL.Query()
			Expect(query.Get("c")).To(Equal("test-subsonicapi-plugin"))
			Expect(query.Get("f")).To(Equal("json"))
			Expect(query.Get("v")).To(Equal("1.16.1"))
			Expect(query.Get("type")).To(Equal("newest"))
		})

		It("returns error when username is missing", func() {
			instance, err := plugin.instance(GinkgoT().Context())
			Expect(err).ToNot(HaveOccurred())
			defer instance.Close(GinkgoT().Context())

			exit, _, err := instance.Call("call_subsonic_api", []byte("/ping"))
			Expect(err).To(HaveOccurred())
			Expect(exit).To(Equal(uint32(1)))
			Expect(err.Error()).To(ContainSubstring("missing required parameter"))
		})
	})
})

var _ = Describe("SubsonicAPIService", func() {
	var (
		router    *fakeSubsonicRouter
		userRepo  *tests.MockedUserRepo
		dataStore *tests.MockDataStore
	)

	BeforeEach(func() {
		router = &fakeSubsonicRouter{}
		userRepo = tests.CreateMockUserRepo()
		dataStore = &tests.MockDataStore{MockedUser: userRepo}

		_ = userRepo.Put(&model.User{
			ID:       "user1",
			UserName: "testuser",
			IsAdmin:  false,
		})
		_ = userRepo.Put(&model.User{
			ID:       "admin1",
			UserName: "adminuser",
			IsAdmin:  true,
		})
		_ = userRepo.Put(&model.User{
			ID:       "user2",
			UserName: "alloweduser",
			IsAdmin:  false,
		})
	})

	Describe("Permission Enforcement", func() {
		Context("with specific user IDs allowed", func() {
			It("blocks users not in the allowed list", func() {
				// allowedUserIDs contains "user2", but testuser is "user1"
				service := newSubsonicAPIService("test-plugin", router, dataStore, []string{"user2"}, false)

				ctx := GinkgoT().Context()
				_, err := service.Call(ctx, "/ping?u=testuser")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not authorized"))
			})

			It("allows users in the allowed list", func() {
				// allowedUserIDs contains "user2" which is "alloweduser"
				service := newSubsonicAPIService("test-plugin", router, dataStore, []string{"user2"}, false)

				ctx := GinkgoT().Context()
				response, err := service.Call(ctx, "/ping?u=alloweduser")
				Expect(err).ToNot(HaveOccurred())
				Expect(response).To(ContainSubstring("ok"))
			})

			It("blocks admin users when not in allowed list", func() {
				// allowedUserIDs only contains "user1" (testuser), not "admin1"
				service := newSubsonicAPIService("test-plugin", router, dataStore, []string{"user1"}, false)

				ctx := GinkgoT().Context()
				_, err := service.Call(ctx, "/ping?u=adminuser")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not authorized"))
			})

			It("allows admin users when in allowed list", func() {
				// allowedUserIDs contains "admin1"
				service := newSubsonicAPIService("test-plugin", router, dataStore, []string{"admin1"}, false)

				ctx := GinkgoT().Context()
				response, err := service.Call(ctx, "/ping?u=adminuser")
				Expect(err).ToNot(HaveOccurred())
				Expect(response).To(ContainSubstring("ok"))
			})
		})

		Context("with allUsers=true", func() {
			It("allows all users regardless of allowed list", func() {
				service := newSubsonicAPIService("test-plugin", router, dataStore, nil, true)

				ctx := GinkgoT().Context()
				response, err := service.Call(ctx, "/ping?u=testuser")
				Expect(err).ToNot(HaveOccurred())
				Expect(response).To(ContainSubstring("ok"))
			})

			It("allows admin users when allUsers is true", func() {
				service := newSubsonicAPIService("test-plugin", router, dataStore, nil, true)

				ctx := GinkgoT().Context()
				response, err := service.Call(ctx, "/ping?u=adminuser")
				Expect(err).ToNot(HaveOccurred())
				Expect(response).To(ContainSubstring("ok"))
			})
		})

		Context("with no users configured", func() {
			It("returns error when no users are configured", func() {
				service := newSubsonicAPIService("test-plugin", router, dataStore, nil, false)

				ctx := GinkgoT().Context()
				_, err := service.Call(ctx, "/ping?u=testuser")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no users configured"))
			})

			It("returns error for empty user list", func() {
				service := newSubsonicAPIService("test-plugin", router, dataStore, []string{}, false)

				ctx := GinkgoT().Context()
				_, err := service.Call(ctx, "/ping?u=testuser")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no users configured"))
			})
		})
	})

	Describe("URL Handling", func() {
		It("returns error for missing username parameter", func() {
			service := newSubsonicAPIService("test-plugin", router, dataStore, nil, true)

			ctx := GinkgoT().Context()
			_, err := service.Call(ctx, "/ping")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing required parameter"))
		})

		It("returns error for invalid URL", func() {
			service := newSubsonicAPIService("test-plugin", router, dataStore, nil, true)

			ctx := GinkgoT().Context()
			_, err := service.Call(ctx, "://invalid")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid URL"))
		})

		It("extracts endpoint from path correctly", func() {
			service := newSubsonicAPIService("test-plugin", router, dataStore, []string{"user1"}, false)

			ctx := GinkgoT().Context()
			_, err := service.Call(ctx, "/rest/ping.view?u=testuser")
			Expect(err).ToNot(HaveOccurred())

			// The endpoint should be extracted as "ping.view"
			Expect(router.lastRequest.URL.Path).To(Equal("/ping.view"))
		})
	})

	Describe("Router Availability", func() {
		It("returns error when router is nil", func() {
			service := newSubsonicAPIService("test-plugin", nil, dataStore, nil, true)

			ctx := GinkgoT().Context()
			_, err := service.Call(ctx, "/ping?u=testuser")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("router not available"))
		})
	})
})

// fakeSubsonicRouter is a mock Subsonic router that returns predictable responses.
type fakeSubsonicRouter struct {
	lastRequest *http.Request
}

func (r *fakeSubsonicRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.lastRequest = req

	// Return a successful ping response
	response := map[string]any{
		"subsonic-response": map[string]any{
			"status":  "ok",
			"version": "1.16.1",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
