package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager", func() {
	var (
		manager     *Manager
		ctx         context.Context
		testdataDir string
		tmpDir      string
	)

	BeforeEach(func() {
		ctx = GinkgoT().Context()

		// Get testdata directory
		_, currentFile, _, ok := runtime.Caller(0)
		Expect(ok).To(BeTrue())
		testdataDir = filepath.Join(filepath.Dir(currentFile), "testdata")

		// Create temp dir for plugins
		var err error
		tmpDir, err = os.MkdirTemp("", "plugins-test-*")
		Expect(err).ToNot(HaveOccurred())

		// Setup config
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = tmpDir
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

	Describe("LoadPlugin", func() {
		It("loads a new plugin by name", func() {
			copyTestPlugin("new-plugin")

			err := manager.LoadPlugin("new-plugin")
			Expect(err).ToNot(HaveOccurred())

			names := manager.PluginNames(string(CapabilityMetadataAgent))
			Expect(names).To(ContainElement("new-plugin"))
		})

		It("returns error when plugin file does not exist", func() {
			err := manager.LoadPlugin("nonexistent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("plugin file not found"))
		})

		It("returns error when plugin is already loaded", func() {
			copyTestPlugin("duplicate")

			err := manager.LoadPlugin("duplicate")
			Expect(err).ToNot(HaveOccurred())

			err = manager.LoadPlugin("duplicate")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already loaded"))
		})

		It("returns error when plugins folder is not configured", func() {
			conf.Server.Plugins.Folder = ""
			conf.Server.DataFolder = ""

			err := manager.LoadPlugin("test")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no plugins folder configured"))
		})
	})

	Describe("UnloadPlugin", func() {
		It("removes a loaded plugin", func() {
			copyTestPlugin("to-unload")
			err := manager.LoadPlugin("to-unload")
			Expect(err).ToNot(HaveOccurred())

			err = manager.UnloadPlugin("to-unload")
			Expect(err).ToNot(HaveOccurred())

			names := manager.PluginNames(string(CapabilityMetadataAgent))
			Expect(names).ToNot(ContainElement("to-unload"))
		})

		It("returns error when plugin not found", func() {
			err := manager.UnloadPlugin("nonexistent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})
	})

	Describe("ReloadPlugin", func() {
		It("unloads and reloads a plugin", func() {
			copyTestPlugin("to-reload")
			err := manager.LoadPlugin("to-reload")
			Expect(err).ToNot(HaveOccurred())

			err = manager.ReloadPlugin("to-reload")
			Expect(err).ToNot(HaveOccurred())

			names := manager.PluginNames(string(CapabilityMetadataAgent))
			Expect(names).To(ContainElement("to-reload"))
		})

		It("returns error when plugin not found", func() {
			err := manager.ReloadPlugin("nonexistent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to unload"))
		})

		It("keeps plugin unloaded if reload fails", func() {
			copyTestPlugin("fail-reload")
			err := manager.LoadPlugin("fail-reload")
			Expect(err).ToNot(HaveOccurred())

			// Remove the wasm file so reload will fail
			wasmPath := filepath.Join(tmpDir, "fail-reload.wasm")
			err = os.Remove(wasmPath)
			Expect(err).ToNot(HaveOccurred())

			err = manager.ReloadPlugin("fail-reload")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to reload"))

			// Plugin should no longer be loaded
			names := manager.PluginNames(string(CapabilityMetadataAgent))
			Expect(names).ToNot(ContainElement("fail-reload"))
		})
	})

	It("can call the plugin concurrently", func() {
		copyTestPlugin("new-plugin")
		err := manager.LoadPlugin("new-plugin")
		Expect(err).ToNot(HaveOccurred())

		const concurrency = 100
		errs := make(chan error, concurrency)
		bios := make(chan string, concurrency)

		g := sync.WaitGroup{}
		g.Add(concurrency)
		for i := range concurrency {
			go func(i int) {
				defer g.Done()
				a, ok := manager.LoadMediaAgent("new-plugin")
				Expect(ok).To(BeTrue())
				agent := a.(agents.ArtistBiographyRetriever)
				bio, err := agent.GetArtistBiography(ctx, fmt.Sprintf("artist-%d", i), fmt.Sprintf("Artist %d", i), "")
				if err != nil {
					errs <- err
					return
				}
				bios <- bio
			}(i)
		}
		g.Wait()

		// Collect results
		for range concurrency {
			select {
			case err := <-errs:
				Expect(err).ToNot(HaveOccurred())
			case bio := <-bios:
				Expect(bio).To(ContainSubstring("Biography for Artist"))
			}
		}
	})

})
