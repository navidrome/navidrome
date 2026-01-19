package plugins

import (
	"crypto/sha256"
	"encoding/hex"
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

var _ = Describe("Manager Loader", func() {
	Describe("loadPlugin with complex config values", func() {
		var manager *Manager
		var tmpDir string

		BeforeEach(func() {
			// Create temp directory
			var err error
			tmpDir, err = os.MkdirTemp("", "plugins-loader-test-*")
			Expect(err).ToNot(HaveOccurred())

			// Copy test plugin to temp dir
			srcPath := filepath.Join(testdataDir, "test-metadata-agent"+PackageExtension)
			destPath := filepath.Join(tmpDir, "test-metadata-agent"+PackageExtension)
			data, err := os.ReadFile(srcPath)
			Expect(err).ToNot(HaveOccurred())
			err = os.WriteFile(destPath, data, 0600)
			Expect(err).ToNot(HaveOccurred())

			// Compute SHA256 for the plugin
			hash := sha256.Sum256(data)
			hashHex := hex.EncodeToString(hash[:])

			// Setup config
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Plugins.Enabled = true
			conf.Server.Plugins.Folder = tmpDir
			conf.Server.Plugins.AutoReload = false
			conf.Server.CacheFolder = filepath.Join(tmpDir, "cache")

			// Setup mock DataStore with plugin having complex config (arrays and objects)
			mockPluginRepo := tests.CreateMockPluginRepo()
			mockPluginRepo.Permitted = true
			mockPluginRepo.SetData(model.Plugins{{
				ID:       "test-metadata-agent",
				Path:     destPath,
				SHA256:   hashHex,
				Enabled:  true,
				AllUsers: true,
				// Config with arrays and objects - this should be properly serialized
				Config: `{"api_key":"secret123","users":[{"username":"admin","token":"tok1"},{"username":"user2","token":"tok2"}],"settings":{"enabled":true,"count":5}}`,
			}})
			dataStore := &tests.MockDataStore{MockedPlugin: mockPluginRepo}

			// Create and start manager
			manager = &Manager{
				plugins:        make(map[string]*plugin),
				ds:             dataStore,
				subsonicRouter: http.NotFoundHandler(),
			}
			err = manager.Start(GinkgoT().Context())
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func() {
				_ = manager.Stop()
				_ = os.RemoveAll(tmpDir)
			})
		})

		It("should load plugin with array values in config", func() {
			manager.mu.RLock()
			p, ok := manager.plugins["test-metadata-agent"]
			manager.mu.RUnlock()
			Expect(ok).To(BeTrue())
			Expect(p.manifest.Name).To(Equal("Test Plugin"))
			// If we got here without error, the complex config (arrays, objects)
			// was properly parsed and serialized for extism
		})
	})
})
