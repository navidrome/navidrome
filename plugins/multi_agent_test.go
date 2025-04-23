package plugins

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("wasmFakeMultiAgent (real plugin)", func() {
	var (
		agent agents.Interface
		ctx   context.Context
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = "plugins/testdata"
		ctx = context.Background()
	})

	It("returns artist MBID", func() {
		mgr := createManager()
		Expect(mgr).NotTo(BeNil())

		Eventually(func() bool {
			_, ok := agents.Map["fake_multi_agent_ArtistMetadataService"]
			return ok
		}, "5s", "100ms").Should(BeTrue(), "ArtistMetadataService agent should be registered")

		constructor, ok := agents.Map["fake_multi_agent_ArtistMetadataService"]
		Expect(ok).To(BeTrue())
		agent = constructor(nil)
		Expect(agent).NotTo(BeNil(), "ArtistMetadataService agent should be constructible")

		mbidRetriever := agent.(agents.ArtistMBIDRetriever)
		mbid, err := mbidRetriever.GetArtistMBID(ctx, "id", "Test Artist")
		Expect(err).NotTo(HaveOccurred())
		Expect(mbid).To(Equal("multi-artist-mbid"))
	})

	It("returns album info", func() {
		mgr := createManager()
		Expect(mgr).NotTo(BeNil())

		Eventually(func() bool {
			_, ok := agents.Map["fake_multi_agent_AlbumMetadataService"]
			return ok
		}, "5s", "100ms").Should(BeTrue(), "AlbumMetadataService agent should be registered")

		constructor, ok := agents.Map["fake_multi_agent_AlbumMetadataService"]
		Expect(ok).To(BeTrue())
		agent = constructor(nil)
		Expect(agent).NotTo(BeNil(), "AlbumMetadataService agent should be constructible")

		infoRetriever := agent.(agents.AlbumInfoRetriever)
		info, err := infoRetriever.GetAlbumInfo(ctx, "Test Album", "Test Artist", "mbid")
		Expect(err).NotTo(HaveOccurred())
		Expect(info).NotTo(BeNil())
		Expect(info.Name).To(Equal("Test Album"))
		Expect(info.MBID).To(Equal("multi-album-mbid"))
		Expect(info.Description).To(Equal("Multi agent album description"))
		Expect(info.URL).To(Equal("https://multi.example.com/album"))
	})
})
