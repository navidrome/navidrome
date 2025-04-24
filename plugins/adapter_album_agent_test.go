package plugins

import (
	"context"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("wasmAlbumAgent (real plugin)", func() {
	var (
		agent agents.Interface
		ctx   context.Context
		mgr   *Manager
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		absPath, err := filepath.Abs("plugins/testdata")
		Expect(err).To(BeNil())
		conf.Server.Plugins.Folder = absPath
		conf.Server.Plugins.Enabled = true

		ctx = context.Background()

		mgr = createManager()
		Expect(mgr).NotTo(BeNil())
		mgr.ScanPlugins()

		// Load the plugin directly
		pluginInstance := mgr.LoadPlugin("fake_album_agent")
		Expect(pluginInstance).NotTo(BeNil(), "should be able to load the plugin")

		var ok bool
		agent, ok = pluginInstance.(agents.Interface)
		Expect(ok).To(BeTrue(), "plugin should implement agents.Interface")
		Expect(agent).NotTo(BeNil(), "plugin agent should be instantiated")
	})

	It("returns the correct agent name", func() {
		Expect(agent.AgentName()).To(Equal("fake_album_agent"))
	})

	It("returns album info", func() {
		infoRetriever := agent.(agents.AlbumInfoRetriever)
		info, err := infoRetriever.GetAlbumInfo(ctx, "Test Album", "Test Artist", "mbid")
		Expect(err).NotTo(HaveOccurred())
		Expect(info).NotTo(BeNil())
		Expect(info.Name).To(Equal("Test Album"))
		Expect(info.MBID).To(Equal("album-mbid-123"))
		Expect(info.Description).To(Equal("This is a test album description"))
		Expect(info.URL).To(Equal("https://example.com/album"))
	})

	It("returns album images", func() {
		imagesRetriever := agent.(interface {
			GetAlbumImages(ctx context.Context, name, artist, mbid string) ([]agents.ExternalImage, error)
		})
		images, err := imagesRetriever.GetAlbumImages(ctx, "Test Album", "Test Artist", "mbid")
		Expect(err).NotTo(HaveOccurred())
		Expect(images).To(HaveLen(2))
		Expect(images[0].URL).To(Equal("https://example.com/album1.jpg"))
		Expect(images[0].Size).To(Equal(300))
		Expect(images[1].URL).To(Equal("https://example.com/album2.jpg"))
		Expect(images[1].Size).To(Equal(400))
	})

	Describe("error cases", func() {
		It("returns error for empty name in AlbumInfo", func() {
			infoRetriever := agent.(agents.AlbumInfoRetriever)
			_, err := infoRetriever.GetAlbumInfo(ctx, "", "Test Artist", "mbid")
			Expect(err).To(HaveOccurred())
			_, err = infoRetriever.GetAlbumInfo(ctx, "Test Album", "", "mbid")
			Expect(err).To(HaveOccurred())
		})
		It("returns error for empty name in AlbumImages", func() {
			imagesRetriever := agent.(interface {
				GetAlbumImages(ctx context.Context, name, artist, mbid string) ([]agents.ExternalImage, error)
			})
			_, err := imagesRetriever.GetAlbumImages(ctx, "", "Test Artist", "mbid")
			Expect(err).To(HaveOccurred())
			_, err = imagesRetriever.GetAlbumImages(ctx, "Test Album", "", "mbid")
			Expect(err).To(HaveOccurred())
		})
	})
})
