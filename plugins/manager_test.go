package plugins

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager", Ordered, func() {
	var (
		manager     *Manager
		ctx         context.Context
		testdataDir string
	)

	BeforeAll(func() {
		ctx = context.Background()

		// Get testdata directory (where fake-metadata-agent.wasm lives)
		_, currentFile, _, ok := runtime.Caller(0)
		Expect(ok).To(BeTrue())
		testdataDir = filepath.Join(filepath.Dir(currentFile), "testdata")

		// Setup config
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = testdataDir

		// Create manager once for all tests
		manager = &Manager{
			plugins: make(map[string]*pluginInstance),
		}
		err := manager.Start(ctx)
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(func() {
			_ = manager.Stop()
		})
	})

	Describe("LoadPlugin", func() {
		It("auto-loads plugins from folder on Start", func() {
			// Plugin is already loaded by manager.Start() via discoverPlugins
			names := manager.PluginNames(string(CapabilityMetadataAgent))
			Expect(names).To(ContainElement("fake-metadata-agent"))
		})

		It("returns error when plugin file does not exist", func() {
			err := manager.LoadPlugin("nonexistent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("plugin file not found"))
		})

		It("returns error when plugin is already loaded", func() {
			// Plugin was loaded on Start, try to load again
			err := manager.LoadPlugin("fake-metadata-agent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already loaded"))
		})

		It("returns error when plugins folder is not configured", func() {
			originalFolder := conf.Server.Plugins.Folder
			originalDataFolder := conf.Server.DataFolder
			conf.Server.Plugins.Folder = ""
			conf.Server.DataFolder = ""
			defer func() {
				conf.Server.Plugins.Folder = originalFolder
				conf.Server.DataFolder = originalDataFolder
			}()

			err := manager.LoadPlugin("test")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no plugins folder configured"))
		})
	})

	Describe("UnloadPlugin", func() {
		It("removes a loaded plugin", func() {
			// Plugin is already loaded from Start
			err := manager.UnloadPlugin("fake-metadata-agent")
			Expect(err).ToNot(HaveOccurred())

			names := manager.PluginNames(string(CapabilityMetadataAgent))
			Expect(names).ToNot(ContainElement("fake-metadata-agent"))
		})

		It("can reload after unload", func() {
			// Reload the plugin we just unloaded
			err := manager.LoadPlugin("fake-metadata-agent")
			Expect(err).ToNot(HaveOccurred())

			names := manager.PluginNames(string(CapabilityMetadataAgent))
			Expect(names).To(ContainElement("fake-metadata-agent"))
		})

		It("returns error when plugin not found", func() {
			err := manager.UnloadPlugin("nonexistent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})
	})

	Describe("ReloadPlugin", func() {
		It("unloads and reloads a plugin", func() {
			err := manager.ReloadPlugin("fake-metadata-agent")
			Expect(err).ToNot(HaveOccurred())

			names := manager.PluginNames(string(CapabilityMetadataAgent))
			Expect(names).To(ContainElement("fake-metadata-agent"))
		})

		It("returns error when plugin not found", func() {
			err := manager.ReloadPlugin("nonexistent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to unload"))
		})
	})

	It("can call the plugin concurrently", func() {
		// Plugin is already loaded

		const concurrency = 100
		errs := make(chan error, concurrency)
		bios := make(chan string, concurrency)

		g := sync.WaitGroup{}
		g.Add(concurrency)
		for i := range concurrency {
			go func(i int) {
				defer g.Done()
				a, ok := manager.LoadMediaAgent("fake-metadata-agent")
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
