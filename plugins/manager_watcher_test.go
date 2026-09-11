package plugins

import (
	"context"
	"net/http"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin Watcher", func() {
	Describe("Integration Tests", Ordered, func() {
		// Uses testdataDir and createTestManager from BeforeSuite
		var (
			manager *Manager
			tmpDir  string
			ctx     context.Context
		)

		BeforeAll(func() {
			ctx = GinkgoT().Context()

			// Create manager for watcher lifecycle tests (no plugin preloaded - tests copy plugin as needed)
			manager, tmpDir = createTestManager(nil)

			// Remove the auto-loaded plugin so tests can control loading
			_ = manager.unloadPlugin("test-metadata-agent")
			_ = os.Remove(filepath.Join(tmpDir, "test-metadata-agent"+PackageExtension))
			// Also remove from DB so tests start with a clean slate
			_ = manager.ds.Plugin(ctx).Delete("test-metadata-agent")
		})

		// Helper to copy test plugin into the temp folder
		copyTestPlugin := func() {
			srcPath := filepath.Join(testdataDir, "test-metadata-agent"+PackageExtension)
			destPath := filepath.Join(tmpDir, "test-metadata-agent"+PackageExtension)
			data, err := os.ReadFile(srcPath)
			Expect(err).ToNot(HaveOccurred())
			err = os.WriteFile(destPath, data, 0600)
			Expect(err).ToNot(HaveOccurred())
		}

		Describe("Plugin event processing (integration)", func() {
			// These tests verify the DB-driven flow with actual WASM plugin loading.

			AfterEach(func() {
				// Clean up: unload plugin if loaded, remove copied file, delete from DB
				_ = manager.unloadPlugin("test-metadata-agent")
				_ = os.Remove(filepath.Join(tmpDir, "test-metadata-agent"+PackageExtension))
				_ = manager.ds.Plugin(ctx).Delete("test-metadata-agent")
			})

			It("adds plugin to DB when file exists", func() {
				copyTestPlugin()
				manager.processPluginEvent("test-metadata-agent")

				// Plugin should be in DB but not loaded (starts disabled)
				Expect(manager.PluginNames(string(CapabilityMetadataAgent))).ToNot(ContainElement("test-metadata-agent"))

				// Verify it was added to DB
				repo := manager.ds.Plugin(ctx)
				plugin, err := repo.Get("test-metadata-agent")
				Expect(err).ToNot(HaveOccurred())
				Expect(plugin.ID).To(Equal("test-metadata-agent"))
				Expect(plugin.Enabled).To(BeFalse())
			})

			It("updates DB and disables plugin when file changes", func() {
				copyTestPlugin()

				// First add and enable the plugin
				manager.processPluginEvent("test-metadata-agent")
				err := manager.EnablePlugin(ctx, "test-metadata-agent")
				Expect(err).ToNot(HaveOccurred())
				Expect(manager.PluginNames(string(CapabilityMetadataAgent))).To(ContainElement("test-metadata-agent"))

				// Modify the stored SHA256 in DB to simulate a file change
				// (In reality, the file would have different content)
				repo := manager.ds.Plugin(ctx)
				plugin, err := repo.Get("test-metadata-agent")
				Expect(err).ToNot(HaveOccurred())
				plugin.SHA256 = "different-hash-to-simulate-change"
				err = repo.Put(plugin)
				Expect(err).ToNot(HaveOccurred())

				// Simulate modification - the plugin should be disabled and unloaded
				manager.processPluginEvent("test-metadata-agent")

				// Should be unloaded
				Expect(manager.PluginNames(string(CapabilityMetadataAgent))).ToNot(ContainElement("test-metadata-agent"))

				// But still in DB (just disabled)
				plugin, err = repo.Get("test-metadata-agent")
				Expect(err).ToNot(HaveOccurred())
				Expect(plugin.Enabled).To(BeFalse())
			})

			It("removes plugin from DB when file is removed", func() {
				copyTestPlugin()

				// First add and enable the plugin
				manager.processPluginEvent("test-metadata-agent")
				err := manager.EnablePlugin(ctx, "test-metadata-agent")
				Expect(err).ToNot(HaveOccurred())

				// Remove the file - plugin should be unloaded and removed from DB
				_ = os.Remove(filepath.Join(tmpDir, "test-metadata-agent"+PackageExtension))
				manager.processPluginEvent("test-metadata-agent")

				// Should be unloaded
				Expect(manager.PluginNames(string(CapabilityMetadataAgent))).ToNot(ContainElement("test-metadata-agent"))

				// And removed from DB
				repo := manager.ds.Plugin(ctx)
				_, err = repo.Get("test-metadata-agent")
				Expect(err).To(HaveOccurred())
			})
		})

		Describe("Watcher lifecycle", func() {
			It("does not start file watcher when AutoReload is disabled", func() {
				Expect(manager.watcherEvents).To(BeNil())
				Expect(manager.watcherDone).To(BeNil())
			})

			It("starts file watcher when AutoReload is enabled", func() {
				_ = manager.Stop()

				conf.Server.Plugins.AutoReload = true

				// Set up a mock DataStore for the auto-reload manager
				mockPluginRepo := tests.CreateMockPluginRepo()
				mockPluginRepo.Permitted = true
				dataStore := &tests.MockDataStore{MockedPlugin: mockPluginRepo}

				autoReloadManager := &Manager{
					plugins:        make(map[string]*plugin),
					ds:             dataStore,
					subsonicRouter: http.NotFoundHandler(),
				}
				err := autoReloadManager.Start(ctx)
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(autoReloadManager.Stop)

				Expect(autoReloadManager.watcherEvents).ToNot(BeNil())
				Expect(autoReloadManager.watcherDone).ToNot(BeNil())
			})
		})
	})

	Describe("determinePluginAction", func() {
		var tmpDir string

		BeforeEach(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "plugin-action-test-*")
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			os.RemoveAll(tmpDir)
		})

		It("returns actionUpdate when file exists", func() {
			filePath := filepath.Join(tmpDir, "test.ndp")
			err := os.WriteFile(filePath, []byte("test"), 0600)
			Expect(err).ToNot(HaveOccurred())

			Expect(determinePluginAction(filePath)).To(Equal(actionUpdate))
		})

		It("returns actionRemove when file does not exist", func() {
			filePath := filepath.Join(tmpDir, "nonexistent.ndp")
			Expect(determinePluginAction(filePath)).To(Equal(actionRemove))
		})
	})
})
