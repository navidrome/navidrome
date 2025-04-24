package plugins

import (
	"context"
	"time"

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

		ctx = GinkgoT().Context()
		mgr = createManager()
		mgr.ScanPlugins()
	})

	It("should scan and discover plugins from the testdata folder", func() {
		Expect(mgr).NotTo(BeNil())

		mediaAgentNames := mgr.PluginNames("MediaMetadataService")
		Expect(mediaAgentNames).To(ContainElement("fake_artist_agent"))
		Expect(mediaAgentNames).To(ContainElement("fake_album_agent"))
		Expect(mediaAgentNames).To(ContainElement("fake_multi_agent"))

		scrobblerNames := mgr.PluginNames("ScrobblerService")
		Expect(scrobblerNames).To(ContainElement("fake_scrobbler"))
	})

	It("should load a MediaMetadataService plugin and invoke artist-related methods", func() {
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

	It("should load all MediaMetadataService plugins and invoke methods", func() {
		mediaAgentNames := mgr.PluginNames("MediaMetadataService")
		Expect(mediaAgentNames).NotTo(BeEmpty())

		plugins := mgr.LoadAllPlugins("MediaMetadataService")
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

	It("should use DevPluginCompilationTimeout config for plugin compilation timeout", func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DevPluginCompilationTimeout = 123 * time.Second
		Expect(pluginCompilationTimeout()).To(Equal(123 * time.Second))

		conf.Server.DevPluginCompilationTimeout = 0
		Expect(pluginCompilationTimeout()).To(Equal(time.Minute))
	})
})
