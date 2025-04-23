package plugins

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("wasmAgent (real plugin)", func() {
	var (
		agent agents.Interface
		ctx   context.Context
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = "plugins/testdata"
		ctx = context.Background()

		mgr := createManager()
		Expect(mgr).NotTo(BeNil())

		// Wait for the agent to be registered, polling with a timeout
		Eventually(func() bool {
			_, ok := agents.Map["agent"]
			return ok
		}, "5s", "100ms").Should(BeTrue(), "plugin agent should be registered")

		constructor, ok := agents.Map["agent"]
		Expect(ok).To(BeTrue()) // Re-check for safety, though Eventually should guarantee it
		agent = constructor(nil)
		Expect(agent).NotTo(BeNil(), "plugin agent should be constructible")
	})

	It("returns the correct agent name", func() {
		Expect(agent.AgentName()).To(Equal("agent"))
	})

	It("returns artist MBID", func() {
		mbidRetriever := agent.(agents.ArtistMBIDRetriever)
		mbid, err := mbidRetriever.GetArtistMBID(ctx, "id", "Test Artist")
		Expect(err).NotTo(HaveOccurred())
		Expect(mbid).To(Equal("1234567890"))
	})

	It("returns artist URL", func() {
		urlRetriever := agent.(agents.ArtistURLRetriever)
		url, err := urlRetriever.GetArtistURL(ctx, "id", "Test Artist", "mbid")
		Expect(err).NotTo(HaveOccurred())
		Expect(url).To(Equal("https://example.com"))
	})

	It("returns artist biography", func() {
		bioRetriever := agent.(agents.ArtistBiographyRetriever)
		bio, err := bioRetriever.GetArtistBiography(ctx, "id", "Test Artist", "mbid")
		Expect(err).NotTo(HaveOccurred())
		Expect(bio).To(Equal("This is a test biography"))
	})

	It("returns similar artists", func() {
		similarRetriever := agent.(agents.ArtistSimilarRetriever)
		artists, err := similarRetriever.GetSimilarArtists(ctx, "id", "Test Artist", "mbid", 2)
		Expect(err).NotTo(HaveOccurred())
		Expect(artists).To(HaveLen(2))
		Expect(artists[0].Name).To(Equal("Similar Artist 1"))
		Expect(artists[0].MBID).To(Equal("mbid1"))
		Expect(artists[1].Name).To(Equal("Similar Artist 2"))
		Expect(artists[1].MBID).To(Equal("mbid2"))
	})

	It("returns artist images", func() {
		imageRetriever := agent.(agents.ArtistImageRetriever)
		images, err := imageRetriever.GetArtistImages(ctx, "id", "Test Artist", "mbid")
		Expect(err).NotTo(HaveOccurred())
		Expect(images).To(HaveLen(2))
		Expect(images[0].URL).To(Equal("https://example.com/image1.jpg"))
		Expect(images[0].Size).To(Equal(100))
		Expect(images[1].URL).To(Equal("https://example.com/image2.jpg"))
		Expect(images[1].Size).To(Equal(200))
	})

	It("returns artist top songs", func() {
		topSongsRetriever := agent.(agents.ArtistTopSongsRetriever)
		songs, err := topSongsRetriever.GetArtistTopSongs(ctx, "id", "Test Artist", "mbid", 2)
		Expect(err).NotTo(HaveOccurred())
		Expect(songs).To(HaveLen(2))
		Expect(songs[0].Name).To(Equal("Song 1"))
		Expect(songs[0].MBID).To(Equal("mbid1"))
		Expect(songs[1].Name).To(Equal("Song 2"))
		Expect(songs[1].MBID).To(Equal("mbid2"))
	})

	Describe("error cases", func() {
		It("returns error for empty name in MBID", func() {
			mbidRetriever := agent.(agents.ArtistMBIDRetriever)
			_, err := mbidRetriever.GetArtistMBID(ctx, "id", "")
			Expect(err).To(HaveOccurred())
		})
		It("returns error for empty name in URL", func() {
			urlRetriever := agent.(agents.ArtistURLRetriever)
			_, err := urlRetriever.GetArtistURL(ctx, "id", "", "mbid")
			Expect(err).To(HaveOccurred())
		})
		It("returns error for empty name in Biography", func() {
			bioRetriever := agent.(agents.ArtistBiographyRetriever)
			_, err := bioRetriever.GetArtistBiography(ctx, "id", "", "mbid")
			Expect(err).To(HaveOccurred())
		})
		It("returns error for empty name in SimilarArtists", func() {
			similarRetriever := agent.(agents.ArtistSimilarRetriever)
			_, err := similarRetriever.GetSimilarArtists(ctx, "id", "", "mbid", 2)
			Expect(err).To(HaveOccurred())
		})
		It("returns error for empty name in Images", func() {
			imageRetriever := agent.(agents.ArtistImageRetriever)
			_, err := imageRetriever.GetArtistImages(ctx, "id", "", "mbid")
			Expect(err).To(HaveOccurred())
		})
		It("returns error for empty artistName in TopSongs", func() {
			topSongsRetriever := agent.(agents.ArtistTopSongsRetriever)
			_, err := topSongsRetriever.GetArtistTopSongs(ctx, "id", "", "mbid", 2)
			Expect(err).To(HaveOccurred())
		})
	})
})
