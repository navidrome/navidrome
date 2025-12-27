package plugins

import (
	"context"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
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
			// These tests verify the full flow with actual WASM plugin loading.

			AfterEach(func() {
				// Clean up: unload plugin if loaded, remove copied file
				_ = manager.UnloadPlugin("test-metadata-agent")
				_ = os.Remove(filepath.Join(tmpDir, "test-metadata-agent.wasm"))
			})

			It("loads a plugin on CREATE event", func() {
				copyTestPlugin()
				manager.processPluginEvent("test-metadata-agent", notify.Create)
				Expect(manager.PluginNames(string(CapabilityMetadataAgent))).To(ContainElement("test-metadata-agent"))
			})

			It("reloads a plugin on WRITE event", func() {
				copyTestPlugin()
				err := manager.LoadPlugin("test-metadata-agent")
				Expect(err).ToNot(HaveOccurred())

				manager.processPluginEvent("test-metadata-agent", notify.Write)
				Expect(manager.PluginNames(string(CapabilityMetadataAgent))).To(ContainElement("test-metadata-agent"))
			})

			It("unloads a plugin on REMOVE event", func() {
				copyTestPlugin()
				err := manager.LoadPlugin("test-metadata-agent")
				Expect(err).ToNot(HaveOccurred())

				manager.processPluginEvent("test-metadata-agent", notify.Remove)
				Expect(manager.PluginNames(string(CapabilityMetadataAgent))).ToNot(ContainElement("test-metadata-agent"))
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
				autoReloadManager := &Manager{
					plugins: make(map[string]*plugin),
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
		// These are fast unit tests for the pure routing logic.
		// No WASM compilation, no file I/O - runs in microseconds.

		DescribeTable("returns correct action for event type and loaded state",
			func(eventType notify.Event, isLoaded bool, expected pluginAction) {
				Expect(determinePluginAction(eventType, isLoaded)).To(Equal(expected))
			},
			// CREATE events - always load
			Entry("CREATE when not loaded", notify.Create, false, actionLoad),
			Entry("CREATE when loaded", notify.Create, true, actionLoad),

			// WRITE events - reload if loaded, load if not
			Entry("WRITE when not loaded", notify.Write, false, actionLoad),
			Entry("WRITE when loaded", notify.Write, true, actionReload),

			// REMOVE events - always unload
			Entry("REMOVE when not loaded", notify.Remove, false, actionUnload),
			Entry("REMOVE when loaded", notify.Remove, true, actionUnload),

			// RENAME events - treated same as REMOVE
			Entry("RENAME when not loaded", notify.Rename, false, actionUnload),
			Entry("RENAME when loaded", notify.Rename, true, actionUnload),
		)

		It("returns actionNone for unknown event types", func() {
			// Event type 0 or other unknown values
			Expect(determinePluginAction(0, false)).To(Equal(actionNone))
			Expect(determinePluginAction(0, true)).To(Equal(actionNone))
		})
	})
})
