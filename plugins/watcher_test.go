package plugins

import (
	"context"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rjeczalik/notify"
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
			_ = manager.UnloadPlugin("test-metadata-agent")
			_ = os.Remove(filepath.Join(tmpDir, "test-metadata-agent.wasm"))
		})

		// Helper to copy test plugin into the temp folder
		copyTestPlugin := func() {
			srcPath := filepath.Join(testdataDir, "test-metadata-agent.wasm")
			destPath := filepath.Join(tmpDir, "test-metadata-agent.wasm")
			data, err := os.ReadFile(srcPath)
			Expect(err).ToNot(HaveOccurred())
			err = os.WriteFile(destPath, data, 0600)
			Expect(err).ToNot(HaveOccurred())
		}

		Describe("Plugin event processing (integration)", func() {
			// These tests verify the DB-driven flow with actual WASM plugin loading.

			AfterEach(func() {
				// Clean up: unload plugin if loaded, remove copied file
				_ = manager.UnloadPlugin("test-metadata-agent")
				_ = os.Remove(filepath.Join(tmpDir, "test-metadata-agent.wasm"))
			})

			It("adds plugin to DB on CREATE event", func() {
				copyTestPlugin()
				manager.processPluginEvent("test-metadata-agent", notify.Create)

				// Plugin should be in DB but not loaded (starts disabled)
				Expect(manager.PluginNames(string(CapabilityMetadataAgent))).ToNot(ContainElement("test-metadata-agent"))

				// Verify it was added to DB
				repo := manager.ds.Plugin(ctx)
				plugin, err := repo.Get("test-metadata-agent")
				Expect(err).ToNot(HaveOccurred())
				Expect(plugin.ID).To(Equal("test-metadata-agent"))
				Expect(plugin.Enabled).To(BeFalse())
			})

			It("updates DB and disables plugin on WRITE event when file changes", func() {
				copyTestPlugin()

				// First add and enable the plugin
				manager.processPluginEvent("test-metadata-agent", notify.Create)
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
				manager.processPluginEvent("test-metadata-agent", notify.Write)

				// Should be unloaded
				Expect(manager.PluginNames(string(CapabilityMetadataAgent))).ToNot(ContainElement("test-metadata-agent"))

				// But still in DB (just disabled)
				plugin, err = repo.Get("test-metadata-agent")
				Expect(err).ToNot(HaveOccurred())
				Expect(plugin.Enabled).To(BeFalse())
			})

			It("removes plugin from DB on REMOVE event", func() {
				copyTestPlugin()

				// First add and enable the plugin
				manager.processPluginEvent("test-metadata-agent", notify.Create)
				err := manager.EnablePlugin(ctx, "test-metadata-agent")
				Expect(err).ToNot(HaveOccurred())

				// Simulate removal - plugin should be unloaded and removed from DB
				_ = os.Remove(filepath.Join(tmpDir, "test-metadata-agent.wasm"))
				manager.processPluginEvent("test-metadata-agent", notify.Remove)

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
					plugins: make(map[string]*plugin),
				}
				autoReloadManager.SetDataStore(dataStore)
				err := autoReloadManager.Start(ctx)
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(autoReloadManager.Stop)

				Expect(autoReloadManager.watcherEvents).ToNot(BeNil())
				Expect(autoReloadManager.watcherDone).ToNot(BeNil())
			})
		})
	})

	Describe("determinePluginAction", func() {
		// These are fast unit tests for the pure routing logic.
		// No WASM compilation, no file I/O - runs in microseconds.

		DescribeTable("returns correct action for event type",
			func(eventType notify.Event, expected pluginAction) {
				Expect(determinePluginAction(eventType)).To(Equal(expected))
			},
			// CREATE events - add to DB
			Entry("CREATE", notify.Create, actionAdd),

			// WRITE events - update in DB
			Entry("WRITE", notify.Write, actionUpdate),

			// REMOVE events - remove from DB
			Entry("REMOVE", notify.Remove, actionRemove),

			// RENAME events - treated same as REMOVE
			Entry("RENAME", notify.Rename, actionRemove),
		)

		It("returns actionNone for unknown event types", func() {
			// Event type 0 or other unknown values
			Expect(determinePluginAction(0)).To(Equal(actionNone))
		})
	})
})
