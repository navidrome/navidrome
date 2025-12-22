package plugins

import (
	"context"
	"fmt"
	"sync"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager", Ordered, func() {
	var ctx context.Context

	// Ensure plugin is loaded at the start (might have been unloaded by previous tests)
	BeforeAll(func() {
		ctx = GinkgoT().Context()
		if _, ok := testManager.plugins["fake-metadata-agent"]; !ok {
			err := testManager.LoadPlugin("fake-metadata-agent")
			Expect(err).ToNot(HaveOccurred())
		}
	})

	// Ensure plugin is restored after all tests in this block
	AfterAll(func() {
		if _, ok := testManager.plugins["fake-metadata-agent"]; !ok {
			_ = testManager.LoadPlugin("fake-metadata-agent")
		}
	})

	Describe("LoadPlugin", func() {
		It("auto-loads plugins from folder on Start", func() {
			// Plugin is already loaded by testManager.Start() via discoverPlugins
			names := testManager.PluginNames(string(CapabilityMetadataAgent))
			Expect(names).To(ContainElement("fake-metadata-agent"))
		})

		It("returns error when plugin file does not exist", func() {
			err := testManager.LoadPlugin("nonexistent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("plugin file not found"))
		})

		It("returns error when plugin is already loaded", func() {
			// Plugin was loaded on Start, try to load again
			err := testManager.LoadPlugin("fake-metadata-agent")
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

			err := testManager.LoadPlugin("test")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no plugins folder configured"))
		})
	})

	Describe("UnloadPlugin", func() {
		It("removes a loaded plugin", func() {
			// Plugin is already loaded from Start
			err := testManager.UnloadPlugin("fake-metadata-agent")
			Expect(err).ToNot(HaveOccurred())

			names := testManager.PluginNames(string(CapabilityMetadataAgent))
			Expect(names).ToNot(ContainElement("fake-metadata-agent"))
		})

		It("can reload after unload", func() {
			// Reload the plugin we just unloaded
			err := testManager.LoadPlugin("fake-metadata-agent")
			Expect(err).ToNot(HaveOccurred())

			names := testManager.PluginNames(string(CapabilityMetadataAgent))
			Expect(names).To(ContainElement("fake-metadata-agent"))
		})

		It("returns error when plugin not found", func() {
			err := testManager.UnloadPlugin("nonexistent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})
	})

	Describe("ReloadPlugin", func() {
		It("unloads and reloads a plugin", func() {
			err := testManager.ReloadPlugin("fake-metadata-agent")
			Expect(err).ToNot(HaveOccurred())

			names := testManager.PluginNames(string(CapabilityMetadataAgent))
			Expect(names).To(ContainElement("fake-metadata-agent"))
		})

		It("returns error when plugin not found", func() {
			err := testManager.ReloadPlugin("nonexistent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to unload"))
		})
	})

	Describe("GetPluginInfo", func() {
		It("returns information about all loaded plugins", func() {
			info := testManager.GetPluginInfo()
			Expect(info).To(HaveKey("fake-metadata-agent"))
			Expect(info["fake-metadata-agent"].Name).To(Equal("Test Plugin"))
			Expect(info["fake-metadata-agent"].Version).To(Equal("1.0.0"))
		})
	})

	It("can call the plugin concurrently", func() {
		// Plugin is already loaded

		const concurrency = 30
		errs := make(chan error, concurrency)
		bios := make(chan string, concurrency)

		g := sync.WaitGroup{}
		g.Add(concurrency)
		for i := range concurrency {
			go func(i int) {
				defer g.Done()
				a, ok := testManager.LoadMediaAgent("fake-metadata-agent")
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
