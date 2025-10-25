package plugins

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/plugins/api"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Adapter Media Agent", func() {
	var ctx context.Context
	var mgr *managerImpl

	BeforeEach(func() {
		ctx = GinkgoT().Context()

		// Ensure plugins folder is set to testdata
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Folder = testDataDir

		mgr = createManager(nil, metrics.NewNoopInstance())
		mgr.ScanPlugins()

		// Wait for all plugins to compile to avoid race conditions
		err := mgr.EnsureCompiled("multi_plugin")
		Expect(err).NotTo(HaveOccurred(), "multi_plugin should compile successfully")
		err = mgr.EnsureCompiled("fake_album_agent")
		Expect(err).NotTo(HaveOccurred(), "fake_album_agent should compile successfully")
	})

	Describe("AgentName and PluginName", func() {
		It("should return the plugin name", func() {
			agent := mgr.LoadPlugin("multi_plugin", "MetadataAgent")
			Expect(agent).NotTo(BeNil(), "multi_plugin should be loaded")
			Expect(agent.PluginID()).To(Equal("multi_plugin"))
		})
		It("should return the agent name", func() {
			agent, ok := mgr.LoadMediaAgent("multi_plugin")
			Expect(ok).To(BeTrue(), "multi_plugin should be loaded as media agent")
			Expect(agent.AgentName()).To(Equal("multi_plugin"))
		})
	})

	Describe("Album methods", func() {
		var agent *wasmMediaAgent

		BeforeEach(func() {
			a, ok := mgr.LoadMediaAgent("fake_album_agent")
			Expect(ok).To(BeTrue(), "fake_album_agent should be loaded")
			agent = a.(*wasmMediaAgent)
		})

		Context("GetAlbumInfo", func() {
			It("should return album information", func() {
				info, err := agent.GetAlbumInfo(ctx, "Test Album", "Test Artist", "mbid")

				Expect(err).NotTo(HaveOccurred())
				Expect(info).NotTo(BeNil())
				Expect(info.Name).To(Equal("Test Album"))
				Expect(info.MBID).To(Equal("album-mbid-123"))
				Expect(info.Description).To(Equal("This is a test album description"))
				Expect(info.URL).To(Equal("https://example.com/album"))
			})

			It("should return ErrNotFound when plugin returns not found", func() {
				_, err := agent.GetAlbumInfo(ctx, "Test Album", "", "mbid")

				Expect(err).To(Equal(agents.ErrNotFound))
			})

			It("should return ErrNotFound when plugin returns nil response", func() {
				_, err := agent.GetAlbumInfo(ctx, "", "", "")

				Expect(err).To(Equal(agents.ErrNotFound))
			})
		})

		Context("GetAlbumImages", func() {
			It("should return album images", func() {
				images, err := agent.GetAlbumImages(ctx, "Test Album", "Test Artist", "mbid")

				Expect(err).NotTo(HaveOccurred())
				Expect(images).To(Equal([]agents.ExternalImage{
					{URL: "https://example.com/album1.jpg", Size: 300},
					{URL: "https://example.com/album2.jpg", Size: 400},
				}))
			})
		})
	})

	Describe("Artist methods", func() {
		var agent *wasmMediaAgent

		BeforeEach(func() {
			a, ok := mgr.LoadMediaAgent("fake_artist_agent")
			Expect(ok).To(BeTrue(), "fake_artist_agent should be loaded")
			agent = a.(*wasmMediaAgent)
		})

		Context("GetArtistMBID", func() {
			It("should return artist MBID", func() {
				mbid, err := agent.GetArtistMBID(ctx, "artist-id", "Test Artist")

				Expect(err).NotTo(HaveOccurred())
				Expect(mbid).To(Equal("1234567890"))
			})

			It("should return ErrNotFound when plugin returns not found", func() {
				_, err := agent.GetArtistMBID(ctx, "artist-id", "")

				Expect(err).To(Equal(agents.ErrNotFound))
			})
		})

		Context("GetArtistURL", func() {
			It("should return artist URL", func() {
				url, err := agent.GetArtistURL(ctx, "artist-id", "Test Artist", "mbid")

				Expect(err).NotTo(HaveOccurred())
				Expect(url).To(Equal("https://example.com"))
			})
		})

		Context("GetArtistBiography", func() {
			It("should return artist biography", func() {
				bio, err := agent.GetArtistBiography(ctx, "artist-id", "Test Artist", "mbid")

				Expect(err).NotTo(HaveOccurred())
				Expect(bio).To(Equal("This is a test biography"))
			})
		})

		Context("GetSimilarArtists", func() {
			It("should return similar artists", func() {
				artists, err := agent.GetSimilarArtists(ctx, "artist-id", "Test Artist", "mbid", 10)

				Expect(err).NotTo(HaveOccurred())
				Expect(artists).To(Equal([]agents.Artist{
					{Name: "Similar Artist 1", MBID: "mbid1"},
					{Name: "Similar Artist 2", MBID: "mbid2"},
				}))
			})
		})

		Context("GetArtistImages", func() {
			It("should return artist images", func() {
				images, err := agent.GetArtistImages(ctx, "artist-id", "Test Artist", "mbid")

				Expect(err).NotTo(HaveOccurred())
				Expect(images).To(Equal([]agents.ExternalImage{
					{URL: "https://example.com/image1.jpg", Size: 100},
					{URL: "https://example.com/image2.jpg", Size: 200},
				}))
			})
		})

		Context("GetArtistTopSongs", func() {
			It("should return artist top songs", func() {
				songs, err := agent.GetArtistTopSongs(ctx, "artist-id", "Test Artist", "mbid", 10)

				Expect(err).NotTo(HaveOccurred())
				Expect(songs).To(Equal([]agents.Song{
					{Name: "Song 1", MBID: "mbid1"},
					{Name: "Song 2", MBID: "mbid2"},
				}))
			})
		})
	})

	Describe("Helper functions", func() {
		It("convertExternalImages should convert API image objects to agent image objects", func() {
			apiImages := []*api.ExternalImage{
				{Url: "https://example.com/image1.jpg", Size: 100},
				{Url: "https://example.com/image2.jpg", Size: 200},
			}

			agentImages := convertExternalImages(apiImages)
			Expect(agentImages).To(HaveLen(2))

			for i, img := range agentImages {
				Expect(img.URL).To(Equal(apiImages[i].Url))
				Expect(img.Size).To(Equal(int(apiImages[i].Size)))
			}
		})

		It("convertExternalImages should handle empty slice", func() {
			agentImages := convertExternalImages([]*api.ExternalImage{})
			Expect(agentImages).To(BeEmpty())
		})

		It("convertExternalImages should handle nil", func() {
			agentImages := convertExternalImages(nil)
			Expect(agentImages).To(BeEmpty())
		})
	})

	Describe("Error mapping", func() {
		var agent wasmMediaAgent

		It("should map API ErrNotFound to agents.ErrNotFound", func() {
			err := agent.mapError(api.ErrNotFound)
			Expect(err).To(Equal(agents.ErrNotFound))
		})

		It("should map API ErrNotImplemented to agents.ErrNotFound", func() {
			err := agent.mapError(api.ErrNotImplemented)
			Expect(err).To(Equal(agents.ErrNotFound))
		})

		It("should pass through other errors", func() {
			testErr := errors.New("test error")
			err := agent.mapError(testErr)
			Expect(err).To(Equal(testErr))
		})

		It("should handle nil error", func() {
			err := agent.mapError(nil)
			Expect(err).To(BeNil())
		})
	})
})
