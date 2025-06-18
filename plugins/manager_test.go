package plugins

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
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

	It("should use DevPluginCompilationTimeout config for plugin compilation timeout", func() {
		conf.Server.DevPluginCompilationTimeout = 123 * time.Second
		Expect(pluginCompilationTimeout()).To(Equal(123 * time.Second))

		conf.Server.DevPluginCompilationTimeout = 0
		Expect(pluginCompilationTimeout()).To(Equal(time.Minute))
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

		It("processes symlinks correctly", func() {
			// Create a real plugin directory
			pluginDir := filepath.Join(tempPluginsDir, "real-plugin")
			err := os.MkdirAll(pluginDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			// Create plugin.wasm (empty file for testing)
			wasmPath := filepath.Join(pluginDir, "plugin.wasm")
			err = os.WriteFile(wasmPath, []byte{}, 0644) //nolint:gosec
			Expect(err).ToNot(HaveOccurred())

			// Create manifest.json
			manifestPath := filepath.Join(pluginDir, "manifest.json")
			manifestContent := `{
				"name": "real-plugin",
				"version": "1.0.0",
				"capabilities": ["MetadataAgent"],
				"author": "Test Author",
				"description": "Test Plugin",
				"permissions": {}
			}`
			err = os.WriteFile(manifestPath, []byte(manifestContent), 0644) //nolint:gosec
			Expect(err).ToNot(HaveOccurred())

			// Create a symlink to the real plugin
			symlinkPath := filepath.Join(tempPluginsDir, "symlinked-plugin")
			err = os.Symlink(pluginDir, symlinkPath)
			Expect(err).ToNot(HaveOccurred())

			log.Debug("Created symlink", "source", symlinkPath, "target", pluginDir)

			// Verify symlink exists and is a symlink
			symlinkInfo, err := os.Lstat(symlinkPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(symlinkInfo.Mode()&os.ModeSymlink).ToNot(BeZero(), "should be a symlink")

			// Scan plugins
			m.ScanPlugins()

			// Print the plugins map for debugging
			var pluginNames []string
			for name := range m.plugins {
				pluginNames = append(pluginNames, name)
			}
			log.Debug("Plugins after scan", "plugins", pluginNames)

			// We should have one plugin loaded (not duplicated due to symlink)
			Expect(m.plugins).To(HaveLen(1), "should only find one plugin, not duplicates")

			// Verify the plugin was loaded with correct name
			pluginNames = m.PluginNames("MetadataAgent")
			Expect(pluginNames).To(HaveLen(1), "should only have one MetadataAgent plugin")
			Expect(pluginNames).To(ContainElement("real-plugin"), "should have loaded the real-plugin")
		})

		It("should allow multiple plugins with same manifest.name to coexist in different folders", func() {
			// This test validates the scenario where multiple plugins with the same manifest.name
			// can coexist by using folder names as unique identifiers

			// Helper function to create a plugin with given folder name and manifest name
			createPlugin := func(folderName, manifestName string) {
				pluginDir := filepath.Join(tempPluginsDir, folderName)
				err := os.MkdirAll(pluginDir, 0755)
				Expect(err).ToNot(HaveOccurred())

				// Copy a real WASM file from testdata (use fake_artist_agent as source)
				sourceWasmPath := filepath.Join(testDataDir, "fake_artist_agent", "plugin.wasm")
				targetWasmPath := filepath.Join(pluginDir, "plugin.wasm")

				sourceWasm, err := os.ReadFile(sourceWasmPath)
				Expect(err).ToNot(HaveOccurred())
				err = os.WriteFile(targetWasmPath, sourceWasm, 0644) //nolint:gosec
				Expect(err).ToNot(HaveOccurred())

				// Create manifest.json with same name but different folder
				manifestPath := filepath.Join(pluginDir, "manifest.json")
				manifestContent := `{
					"name": "` + manifestName + `",
					"version": "1.0.0",
					"capabilities": ["MetadataAgent"],
					"author": "Test Author",
					"description": "Test Plugin in ` + folderName + `",
					"permissions": {}
				}`
				err = os.WriteFile(manifestPath, []byte(manifestContent), 0644) //nolint:gosec
				Expect(err).ToNot(HaveOccurred())
			}

			// Create three plugins with same manifest.name but different folders
			createPlugin("lastfm-official", "lastfm")
			createPlugin("lastfm-custom", "lastfm")
			createPlugin("lastfm-dev", "lastfm")

			// Scan plugins
			m.ScanPlugins()

			// Verify all three plugins are discovered and can coexist
			pluginNames := m.PluginNames("MetadataAgent")
			Expect(pluginNames).To(HaveLen(3), "should find all three lastfm plugins")
			Expect(pluginNames).To(ConsistOf("lastfm-official", "lastfm-custom", "lastfm-dev"))

			// Verify each plugin can be loaded independently by folder name
			officialPlugin := m.LoadPlugin("lastfm-official", CapabilityMetadataAgent)
			Expect(officialPlugin).NotTo(BeNil(), "should load lastfm-official plugin")
			Expect(officialPlugin.PluginID()).To(Equal("lastfm-official"))

			customPlugin := m.LoadPlugin("lastfm-custom", CapabilityMetadataAgent)
			Expect(customPlugin).NotTo(BeNil(), "should load lastfm-custom plugin")
			Expect(customPlugin.PluginID()).To(Equal("lastfm-custom"))

			devPlugin := m.LoadPlugin("lastfm-dev", CapabilityMetadataAgent)
			Expect(devPlugin).NotTo(BeNil(), "should load lastfm-dev plugin")
			Expect(devPlugin.PluginID()).To(Equal("lastfm-dev"))

			// Verify the plugins map contains all three with folder names as keys
			Expect(m.plugins).To(SatisfyAll(
				HaveLen(3),
				HaveKey("lastfm-official"),
				HaveKey("lastfm-custom"),
				HaveKey("lastfm-dev"),
			))

			// Verify all manifest names are the same (demonstrating coexistence despite same name)
			Expect(m.plugins["lastfm-official"].Manifest.Name).To(Equal("lastfm"))
			Expect(m.plugins["lastfm-custom"].Manifest.Name).To(Equal("lastfm"))
			Expect(m.plugins["lastfm-dev"].Manifest.Name).To(Equal("lastfm"))
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

			albumInfo, err := fakeAlbumPlugin.(agents.AlbumInfoRetriever).GetAlbumInfo(ctx, "Test Album", "Test Artist", "mbid")
			Expect(err).NotTo(HaveOccurred())
			Expect(albumInfo.Name).To(Equal("Test Album"))
		})
	})

	Describe("Permission Enforcement Integration", func() {
		It("should fail when plugin tries to access unauthorized services", func() {
			// Load the unauthorized_plugin which has no permissions but tries to call HTTP service
			plugin := mgr.LoadPlugin("unauthorized_plugin", CapabilityMetadataAgent)
			Expect(plugin).NotTo(BeNil(), "unauthorized_plugin should be loaded")

			agent, ok := plugin.(agents.Interface)
			Expect(ok).To(BeTrue(), "plugin should implement agents.Interface")
			Expect(agent.AgentName()).To(Equal("unauthorized_plugin"))

			// Try to call GetAlbumInfo which attempts to make an HTTP call without permission
			// This should fail because the plugin has no "http" permission in its manifest
			albumRetriever, ok := agent.(agents.AlbumInfoRetriever)
			Expect(ok).To(BeTrue(), "plugin should implement AlbumInfoRetriever")

			_, err := albumRetriever.GetAlbumInfo(ctx, "Test Album", "Test Artist", "test-mbid")
			Expect(err).To(HaveOccurred(), "should fail when trying to access unauthorized HTTP service")

			// The error should indicate that the HTTP function is not available
			// This happens because the HTTP host functions were not loaded into the WASM runtime
			// since the plugin doesn't have "http" permission
			Expect(err.Error()).To(ContainSubstring("is not exported"), "error should indicate missing HTTP functions")
		})
	})
})
