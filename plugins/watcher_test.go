package plugins

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Watcher", func() {
	var (
		manager     *Manager
		ctx         context.Context
		testdataDir string
		tmpDir      string
	)

	BeforeEach(func() {
		ctx = GinkgoT().Context()

		// Use shorter debounce for faster tests
		originalDebounce := debounceDuration
		debounceDuration = 50 * time.Millisecond
		DeferCleanup(func() { debounceDuration = originalDebounce })

		// Get testdata directory
		_, currentFile, _, ok := runtime.Caller(0)
		Expect(ok).To(BeTrue())
		testdataDir = filepath.Join(filepath.Dir(currentFile), "testdata")

		// Create temp dir for plugins
		var err error
		tmpDir, err = os.MkdirTemp("", "plugins-watcher-test-*")
		Expect(err).ToNot(HaveOccurred())

		// Setup config with AutoReload enabled
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = tmpDir
		conf.Server.Plugins.AutoReload = true
		conf.Server.CacheFolder = filepath.Join(tmpDir, "cache")

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

	copyTestPlugin := func(destName string) string {
		srcPath := filepath.Join(testdataDir, "test-plugin.wasm")
		destPath := filepath.Join(tmpDir, destName+".wasm")
		data, err := os.ReadFile(srcPath)
		Expect(err).ToNot(HaveOccurred())
		err = os.WriteFile(destPath, data, 0600)
		Expect(err).ToNot(HaveOccurred())
		return destPath
	}

	Describe("Auto-reload via file watcher", func() {
		It("loads a plugin when a new wasm file is created", func() {
			// Copy plugin file to trigger CREATE event
			copyTestPlugin("watch-create")

			// Wait for debounce + processing
			Eventually(func() []string {
				return manager.PluginNames(string(CapabilityMetadataAgent))
			}, 1*time.Second, 50*time.Millisecond).Should(ContainElement("watch-create"))
		})

		It("reloads a plugin when the wasm file is modified", func() {
			// First, load a plugin
			copyTestPlugin("watch-modify")

			// Wait for it to be loaded
			Eventually(func() []string {
				return manager.PluginNames(string(CapabilityMetadataAgent))
			}, 1*time.Second, 50*time.Millisecond).Should(ContainElement("watch-modify"))

			// Get the original plugin info
			originalInfo := manager.GetPluginInfo()["watch-modify"]
			Expect(originalInfo.Name).ToNot(BeEmpty())

			// Modify the file (re-copy to trigger WRITE event)
			wasmPath := filepath.Join(tmpDir, "watch-modify.wasm")
			data, err := os.ReadFile(wasmPath)
			Expect(err).ToNot(HaveOccurred())

			// Touch the file to trigger write event
			err = os.WriteFile(wasmPath, data, 0600)
			Expect(err).ToNot(HaveOccurred())

			// Wait for reload - the plugin should still be there
			// We can't easily verify it was reloaded without adding tracking,
			// but at least verify it's still loaded
			Consistently(func() []string {
				return manager.PluginNames(string(CapabilityMetadataAgent))
			}, 300*time.Millisecond, 50*time.Millisecond).Should(ContainElement("watch-modify"))
		})

		It("unloads a plugin when the wasm file is removed", func() {
			// First, load a plugin
			wasmPath := copyTestPlugin("watch-remove")

			// Wait for it to be loaded
			Eventually(func() []string {
				return manager.PluginNames(string(CapabilityMetadataAgent))
			}, 1*time.Second, 50*time.Millisecond).Should(ContainElement("watch-remove"))

			// Remove the file
			err := os.Remove(wasmPath)
			Expect(err).ToNot(HaveOccurred())

			// Wait for it to be unloaded
			Eventually(func() []string {
				return manager.PluginNames(string(CapabilityMetadataAgent))
			}, 1*time.Second, 50*time.Millisecond).ShouldNot(ContainElement("watch-remove"))
		})
	})

	Describe("Watcher disabled", func() {
		BeforeEach(func() {
			// Stop existing manager and create one without auto-reload
			_ = manager.Stop()

			conf.Server.Plugins.AutoReload = false
			manager = &Manager{
				plugins: make(map[string]*pluginInstance),
			}
			err := manager.Start(ctx)
			Expect(err).ToNot(HaveOccurred())
		})

		It("does not auto-load plugins when AutoReload is disabled", func() {
			// Copy plugin file
			copyTestPlugin("no-watch")

			// Wait a bit and verify plugin is NOT loaded
			Consistently(func() []string {
				return manager.PluginNames(string(CapabilityMetadataAgent))
			}, 300*time.Millisecond, 50*time.Millisecond).ShouldNot(ContainElement("no-watch"))
		})
	})
})
