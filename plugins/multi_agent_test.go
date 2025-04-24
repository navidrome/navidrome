package plugins

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("wasmMultiAgent (real plugin)", func() {
	var (
		agent agents.Interface
		ctx   context.Context
		mgr   *Manager
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = "./plugins/testdata"

		ctx = context.Background()

		mgr = createManager()
		Expect(mgr).NotTo(BeNil())

		log.Debug("Scanning plugins")
		mgr.ScanPlugins()

		// Check if plugin was discovered - looking for the ArtistMetadataService plugin
		multiAgentName := "fake_multi_agent_ArtistMetadataService"
		found := false
		for name := range mgr.plugins {
			log.Debug("Plugin found", "name", name)
			if name == multiAgentName {
				found = true
			}
		}
		Expect(found).To(BeTrue(), "Plugin should be discovered")

		// Load the plugin directly
		pluginInstance := mgr.LoadPlugin(multiAgentName)
		Expect(pluginInstance).NotTo(BeNil(), "should be able to load the plugin")

		var ok bool
		agent, ok = pluginInstance.(agents.Interface)
		Expect(ok).To(BeTrue(), "plugin should implement agents.Interface")
		Expect(agent).NotTo(BeNil(), "plugin agent should be instantiated")
	})

	It("returns the correct agent name", func() {
		Expect(agent.AgentName()).To(Equal("fake_multi_agent_ArtistMetadataService"))
	})

	It("returns album info", func() {
		// To test album info, we need to load the AlbumMetadataService plugin
		pluginInstance := mgr.LoadPlugin("fake_multi_agent_AlbumMetadataService")
		Expect(pluginInstance).NotTo(BeNil(), "should be able to load the album plugin")

		albumAgent, ok := pluginInstance.(agents.AlbumInfoRetriever)
		Expect(ok).To(BeTrue(), "plugin should implement agents.AlbumInfoRetriever")

		info, err := albumAgent.GetAlbumInfo(ctx, "Test Album", "Test Artist", "mbid")
		Expect(err).NotTo(HaveOccurred())
		Expect(info.Name).To(Equal("Test Album"))
		Expect(info.MBID).To(Equal("multi-album-mbid"))
		Expect(info.URL).To(Equal("https://multi.example.com/album"))
		Expect(info.Description).To(Equal("Multi agent album description"))
	})

	It("returns artist MBID", func() {
		mbidRetriever := agent.(agents.ArtistMBIDRetriever)
		mbid, err := mbidRetriever.GetArtistMBID(ctx, "id", "Test Artist")
		Expect(err).NotTo(HaveOccurred())
		Expect(mbid).To(Equal("multi-artist-mbid"))
	})

	It("returns artist URL", func() {
		urlRetriever := agent.(agents.ArtistURLRetriever)
		url, err := urlRetriever.GetArtistURL(ctx, "id", "Test Artist", "mbid")
		Expect(err).NotTo(HaveOccurred())
		Expect(url).To(Equal("https://multi.example.com/artist"))
	})

	It("returns artist biography", func() {
		biographyRetriever := agent.(agents.ArtistBiographyRetriever)
		biography, err := biographyRetriever.GetArtistBiography(ctx, "id", "Test Artist", "mbid")
		Expect(err).NotTo(HaveOccurred())
		Expect(biography).To(Equal("Multi agent artist bio"))
	})
})
