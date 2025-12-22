package plugins

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rjeczalik/notify"
)

var _ = Describe("Watcher Integration", Ordered, func() {
	var (
		manager     *Manager
		ctx         context.Context
		testdataDir string
		tmpDir      string
	)

	BeforeAll(func() {
		if testing.Short() {
			Skip("Skipping integration test in short mode")
		}

		ctx = GinkgoT().Context()

		// Get testdata directory
		_, currentFile, _, ok := runtime.Caller(0)
		Expect(ok).To(BeTrue())
		testdataDir = filepath.Join(filepath.Dir(currentFile), "testdata")

		// Create temp dir for plugins
		var err error
		tmpDir, err = os.MkdirTemp("", "plugins-watcher-integration-*")
		Expect(err).ToNot(HaveOccurred())

		// Setup config (AutoReload disabled - tests inject events directly)
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = tmpDir
		conf.Server.Plugins.AutoReload = false

		// Create a fresh manager for each test
		manager = &Manager{
			plugins: make(map[string]*pluginInstance),
		}
		err = manager.Start(ctx)
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(func() {
			_ = manager.Stop()
			_ = os.RemoveAll(tmpDir)
		})
	})

	// Helper to copy test plugin into the temp folder
	copyTestPlugin := func() {
		srcPath := filepath.Join(testdataDir, "fake-metadata-agent.wasm")
		destPath := filepath.Join(tmpDir, "fake-metadata-agent.wasm")
		data, err := os.ReadFile(srcPath)
		Expect(err).ToNot(HaveOccurred())
		err = os.WriteFile(destPath, data, 0600)
		Expect(err).ToNot(HaveOccurred())
	}

	Describe("Plugin event processing (integration)", func() {
		// These tests verify the full flow with actual WASM plugin loading.

		AfterEach(func() {
			// Clean up: unload plugin if loaded, remove copied file
			_ = manager.UnloadPlugin("fake-metadata-agent")
			_ = os.Remove(filepath.Join(tmpDir, "fake-metadata-agent.wasm"))
		})

		It("loads a plugin on CREATE event", func() {
			copyTestPlugin()
			manager.processPluginEvent("fake-metadata-agent", notify.Create)
			Expect(manager.PluginNames(string(CapabilityMetadataAgent))).To(ContainElement("fake-metadata-agent"))
		})

		It("reloads a plugin on WRITE event", func() {
			copyTestPlugin()
			err := manager.LoadPlugin("fake-metadata-agent")
			Expect(err).ToNot(HaveOccurred())

			manager.processPluginEvent("fake-metadata-agent", notify.Write)
			Expect(manager.PluginNames(string(CapabilityMetadataAgent))).To(ContainElement("fake-metadata-agent"))
		})

		It("unloads a plugin on REMOVE event", func() {
			copyTestPlugin()
			err := manager.LoadPlugin("fake-metadata-agent")
			Expect(err).ToNot(HaveOccurred())

			manager.processPluginEvent("fake-metadata-agent", notify.Remove)
			Expect(manager.PluginNames(string(CapabilityMetadataAgent))).ToNot(ContainElement("fake-metadata-agent"))
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
			manager = &Manager{
				plugins: make(map[string]*pluginInstance),
			}
			err := manager.Start(ctx)
			Expect(err).ToNot(HaveOccurred())

			Expect(manager.watcherEvents).ToNot(BeNil())
			Expect(manager.watcherDone).ToNot(BeNil())
		})
	})
})
