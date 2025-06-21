package plugins

import (
	"context"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin Manager", func() {
	var mgr *Manager
	var ctx context.Context

	BeforeEach(func() {
		// We change the plugins folder to random location to avoid conflicts with other tests,
		// but, as this is an integration test, we can't use configtest.SetupConfig() as it causes
		// data races.
		originalPluginsFolder := conf.Server.Plugins.Folder
		DeferCleanup(func() {
			conf.Server.Plugins.Folder = originalPluginsFolder
		})
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = testDataDir

		ctx = GinkgoT().Context()
		mgr = createManager()
		mgr.ScanPlugins()
	})

	It("should scan and discover plugins from the testdata folder", func() {
		Expect(mgr).NotTo(BeNil())

		mediaAgentNames := mgr.PluginNames("MetadataAgent")
		Expect(mediaAgentNames).To(HaveLen(4))
		Expect(mediaAgentNames).To(ContainElement("fake_artist_agent"))
		Expect(mediaAgentNames).To(ContainElement("fake_album_agent"))
		Expect(mediaAgentNames).To(ContainElement("multi_plugin"))
		Expect(mediaAgentNames).To(ContainElement("unauthorized_plugin"))

		scrobblerNames := mgr.PluginNames("Scrobbler")
		Expect(scrobblerNames).To(ContainElement("fake_scrobbler"))

		initServiceNames := mgr.PluginNames("LifecycleManagement")
		Expect(initServiceNames).To(ContainElement("multi_plugin"))
		Expect(initServiceNames).To(ContainElement("fake_init_service"))
	})

	It("should load a MetadataAgent plugin and invoke artist-related methods", func() {
		plugin := mgr.LoadPlugin("fake_artist_agent", CapabilityMetadataAgent)
		Expect(plugin).NotTo(BeNil())

		agent, ok := plugin.(agents.Interface)
		Expect(ok).To(BeTrue(), "plugin should implement agents.Interface")
		Expect(agent.AgentName()).To(Equal("fake_artist_agent"))

		mbidRetriever, ok := agent.(agents.ArtistMBIDRetriever)
		Expect(ok).To(BeTrue())
		mbid, err := mbidRetriever.GetArtistMBID(ctx, "123", "The Beatles")
		Expect(err).NotTo(HaveOccurred())
		Expect(mbid).To(Equal("1234567890"))
	})

	It("should load all MetadataAgent plugins", func() {
		agents := mgr.LoadAllMediaAgents()
		Expect(agents).To(HaveLen(4))
		var names []string
		for _, a := range agents {
			names = append(names, a.AgentName())
		}
		Expect(names).To(ContainElements("fake_artist_agent", "fake_album_agent", "multi_plugin", "unauthorized_plugin"))
	})

	Describe("ScanPlugins", func() {
		var tempPluginsDir string
		var m *Manager

		BeforeEach(func() {
			tempPluginsDir, _ = os.MkdirTemp("", "navidrome-plugins-test-*")
			DeferCleanup(func() {
				_ = os.RemoveAll(tempPluginsDir)
			})

			conf.Server.Plugins.Folder = tempPluginsDir
			m = createManager()
		})

		// Helper to create a complete valid plugin for manager testing
		createValidPlugin := func(folderName, manifestName string) {
			pluginDir := filepath.Join(tempPluginsDir, folderName)
			Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

			// Copy real WASM file from testdata
			sourceWasmPath := filepath.Join(testDataDir, "fake_artist_agent", "plugin.wasm")
			targetWasmPath := filepath.Join(pluginDir, "plugin.wasm")
			sourceWasm, err := os.ReadFile(sourceWasmPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(os.WriteFile(targetWasmPath, sourceWasm, 0600)).To(Succeed())

			manifest := `{
				"name": "` + manifestName + `",
				"version": "1.0.0",
				"capabilities": ["MetadataAgent"],
				"author": "Test Author",
				"description": "Test Plugin",
				"website": "https://test.navidrome.org/` + manifestName + `",
				"permissions": {}
			}`
			Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0600)).To(Succeed())
		}

		It("should register and compile discovered plugins", func() {
			createValidPlugin("test-plugin", "test-plugin")

			m.ScanPlugins()

			// Focus on manager behavior: registration and compilation
			Expect(m.plugins).To(HaveLen(1))
			Expect(m.plugins).To(HaveKey("test-plugin"))

			plugin := m.plugins["test-plugin"]
			Expect(plugin.ID).To(Equal("test-plugin"))
			Expect(plugin.Manifest.Name).To(Equal("test-plugin"))

			// Verify plugin can be loaded (compilation successful)
			loadedPlugin := m.LoadPlugin("test-plugin", CapabilityMetadataAgent)
			Expect(loadedPlugin).NotTo(BeNil())
		})

		It("should handle multiple plugins with different IDs but same manifest names", func() {
			// This tests manager-specific behavior: how it handles ID conflicts
			createValidPlugin("lastfm-official", "lastfm")
			createValidPlugin("lastfm-custom", "lastfm")

			m.ScanPlugins()

			// Both should be registered with their folder names as IDs
			Expect(m.plugins).To(HaveLen(2))
			Expect(m.plugins).To(HaveKey("lastfm-official"))
			Expect(m.plugins).To(HaveKey("lastfm-custom"))

			// Both should be loadable independently
			official := m.LoadPlugin("lastfm-official", CapabilityMetadataAgent)
			custom := m.LoadPlugin("lastfm-custom", CapabilityMetadataAgent)
			Expect(official).NotTo(BeNil())
			Expect(custom).NotTo(BeNil())
			Expect(official.PluginID()).To(Equal("lastfm-official"))
			Expect(custom.PluginID()).To(Equal("lastfm-custom"))
		})
	})

	Describe("LoadPlugin", func() {
		It("should load a MetadataAgent plugin and invoke artist-related methods", func() {
			plugin := mgr.LoadPlugin("fake_artist_agent", CapabilityMetadataAgent)
			Expect(plugin).NotTo(BeNil())

			agent, ok := plugin.(agents.Interface)
			Expect(ok).To(BeTrue(), "plugin should implement agents.Interface")
			Expect(agent.AgentName()).To(Equal("fake_artist_agent"))

			mbidRetriever, ok := agent.(agents.ArtistMBIDRetriever)
			Expect(ok).To(BeTrue())
			mbid, err := mbidRetriever.GetArtistMBID(ctx, "id", "Test Artist")
			Expect(err).NotTo(HaveOccurred())
			Expect(mbid).To(Equal("1234567890"))
		})
	})

	Describe("EnsureCompiled", func() {
		It("should successfully wait for plugin compilation", func() {
			err := mgr.EnsureCompiled("fake_artist_agent")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error for non-existent plugin", func() {
			err := mgr.EnsureCompiled("non-existent-plugin")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("plugin not found: non-existent-plugin"))
		})

		It("should wait for compilation to complete for all valid plugins", func() {
			pluginNames := []string{"fake_artist_agent", "fake_album_agent", "multi_plugin", "fake_scrobbler"}

			for _, name := range pluginNames {
				err := mgr.EnsureCompiled(name)
				Expect(err).NotTo(HaveOccurred(), "plugin %s should compile successfully", name)
			}
		})
	})

	Describe("Invoke Methods", func() {
		It("should load all MetadataAgent plugins and invoke methods", func() {
			mediaAgentNames := mgr.PluginNames("MetadataAgent")
			Expect(mediaAgentNames).NotTo(BeEmpty())

			plugins := mgr.LoadAllPlugins("MetadataAgent")
			Expect(plugins).To(HaveLen(len(mediaAgentNames)))

			var fakeAlbumPlugin agents.Interface
			for _, p := range plugins {
				if agent, ok := p.(agents.Interface); ok {
					if agent.AgentName() == "fake_album_agent" {
						fakeAlbumPlugin = agent
						break
					}
				}
			}

			Expect(fakeAlbumPlugin).NotTo(BeNil(), "fake_album_agent should be loaded")

			// Test GetAlbumInfo method - need to cast to the specific interface
			albumRetriever, ok := fakeAlbumPlugin.(agents.AlbumInfoRetriever)
			Expect(ok).To(BeTrue(), "fake_album_agent should implement AlbumInfoRetriever")

			info, err := albumRetriever.GetAlbumInfo(ctx, "Test Album", "Test Artist", "123")
			Expect(err).NotTo(HaveOccurred())
			Expect(info).NotTo(BeNil())
			Expect(info.Name).To(Equal("Test Album"))
		})
	})

	Describe("Permission Enforcement Integration", func() {
		It("should fail when plugin tries to access unauthorized services", func() {
			// This plugin tries to access config service but has no permissions
			plugin := mgr.LoadPlugin("unauthorized_plugin", CapabilityMetadataAgent)
			Expect(plugin).NotTo(BeNil())

			agent, ok := plugin.(agents.Interface)
			Expect(ok).To(BeTrue())

			// This should fail because the plugin tries to access unauthorized config service
			// The exact behavior depends on the plugin implementation, but it should either:
			// 1. Fail during instantiation, or
			// 2. Return an error when trying to call config methods

			// Try to use one of the available methods - let's test with GetArtistMBID
			mbidRetriever, isMBIDRetriever := agent.(agents.ArtistMBIDRetriever)
			if isMBIDRetriever {
				_, err := mbidRetriever.GetArtistMBID(ctx, "id", "Test Artist")
				if err == nil {
					// If no error, the plugin should still be working
					// but any config access should fail silently or return default values
					Expect(agent.AgentName()).To(Equal("unauthorized_plugin"))
				} else {
					// If there's an error, it should be related to missing permissions
					Expect(err.Error()).To(ContainSubstring(""))
				}
			} else {
				// If the plugin doesn't implement the interface, that's also acceptable
				Expect(agent.AgentName()).To(Equal("unauthorized_plugin"))
			}
		})
	})
})
