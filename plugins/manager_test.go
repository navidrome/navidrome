package plugins

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin Manager", func() {
	var mgr *Manager
	var ctx context.Context

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = "./plugins/testdata"

		ctx = context.Background()
		mgr = createManager()
		mgr.ScanPlugins()
	})

	It("should scan and discover plugins from the testdata folder", func() {
		Expect(mgr).NotTo(BeNil())
		// Check if plugin names are correctly scanned
		artistAgentNames := mgr.PluginNames("ArtistMetadataService")
		Expect(artistAgentNames).To(ContainElement("fake_artist_agent"))

		albumAgentNames := mgr.PluginNames("AlbumMetadataService")
		Expect(albumAgentNames).To(ContainElement("fake_album_agent"))

		scrobblerNames := mgr.PluginNames("ScrobblerService")
		Expect(scrobblerNames).To(ContainElement("fake_scrobbler"))
	})

	It("should be able to load a plugin by name", func() {
		plugin := mgr.LoadPlugin("fake_artist_agent")
		Expect(plugin).NotTo(BeNil())
		agent, ok := plugin.(agents.Interface)
		Expect(ok).To(BeTrue(), "plugin should implement agents.Interface")
		Expect(agent.AgentName()).To(Equal("fake_artist_agent"))

		// Test a specific method
		mbidRetriever, ok := agent.(agents.ArtistMBIDRetriever)
		Expect(ok).To(BeTrue())
		mbid, err := mbidRetriever.GetArtistMBID(ctx, "id", "Test Artist")
		Expect(err).NotTo(HaveOccurred())
		Expect(mbid).To(Equal("1234567890"))
	})

	It("should load plugins of a specific service type", func() {
		// Get the names of album metadata plugins
		albumAgentNames := mgr.PluginNames("AlbumMetadataService")
		// Ensure there's at least one plugin (from our testdata)
		Expect(albumAgentNames).To(ContainElement("fake_album_agent"))

		// Count how many we expect
		expectedPluginCount := len(albumAgentNames)

		// Load all plugins
		plugins := mgr.LoadAllPlugins("AlbumMetadataService")
		Expect(plugins).To(HaveLen(expectedPluginCount))

		// Find our test plugin in the loaded plugins
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

		// Test a specific method to ensure the plugin is working
		albumInfo, err := fakeAlbumPlugin.(agents.AlbumInfoRetriever).GetAlbumInfo(ctx, "Test Album", "Test Artist", "mbid")
		Expect(err).NotTo(HaveOccurred())
		Expect(albumInfo.Name).To(Equal("Test Album"))
	})
})
