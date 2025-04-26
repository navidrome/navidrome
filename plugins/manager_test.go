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
		conf.Server.Plugins.Folder = "./plugins/testdata"

		ctx = GinkgoT().Context()
		mgr = createManager()
		mgr.ScanPlugins()
	})

	It("should scan and discover plugins from the testdata folder", func() {
		Expect(mgr).NotTo(BeNil())

		mediaAgentNames := mgr.PluginNames("MetadataAgent")
		Expect(mediaAgentNames).To(HaveLen(3))
		Expect(mediaAgentNames).To(ContainElement("fake_artist_agent"))
		Expect(mediaAgentNames).To(ContainElement("fake_album_agent"))
		Expect(mediaAgentNames).To(ContainElement("multi_plugin"))

		scrobblerNames := mgr.PluginNames("Scrobbler")
		Expect(scrobblerNames).To(ContainElement("fake_scrobbler"))

		initServiceNames := mgr.PluginNames("LifecycleManagement")
		Expect(initServiceNames).To(ContainElement("multi_plugin"))
		Expect(initServiceNames).To(ContainElement("fake_init_service"))
	})

	It("should load a MetadataAgent plugin and invoke artist-related methods", func() {
		plugin := mgr.LoadPlugin("fake_artist_agent")
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
		Expect(agents).To(HaveLen(3))
		var names []string
		for _, a := range agents {
			names = append(names, a.AgentName())
		}
		Expect(names).To(ContainElements("fake_artist_agent", "fake_album_agent", "multi_plugin"))
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
				"services": ["MetadataAgent"],
				"author": "Test Author",
				"description": "Test Plugin"
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
	})

	Describe("LoadPlugin", func() {
		It("should load a MetadataAgent plugin and invoke artist-related methods", func() {
			plugin := mgr.LoadPlugin("fake_artist_agent")
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
})
