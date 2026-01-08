package plugins

import (
	"github.com/navidrome/navidrome/core/agents"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MetadataAgent", Ordered, func() {
	var agent agents.Interface

	BeforeAll(func() {
		// Load the agent via shared manager
		var ok bool
		agent, ok = testManager.LoadMediaAgent("test-metadata-agent")
		Expect(ok).To(BeTrue())
	})

	Describe("AgentName", func() {
		It("returns the plugin name", func() {
			Expect(agent.AgentName()).To(Equal("test-metadata-agent"))
		})
	})

	Describe("GetArtistMBID", func() {
		It("returns the MBID from the plugin", func() {
			retriever := agent.(agents.ArtistMBIDRetriever)
			mbid, err := retriever.GetArtistMBID(GinkgoT().Context(), "artist-1", "The Beatles")
			Expect(err).ToNot(HaveOccurred())
			Expect(mbid).To(Equal("test-mbid-The Beatles"))
		})
	})

	Describe("GetArtistURL", func() {
		It("returns the URL from the plugin", func() {
			retriever := agent.(agents.ArtistURLRetriever)
			url, err := retriever.GetArtistURL(GinkgoT().Context(), "artist-1", "The Beatles", "some-mbid")
			Expect(err).ToNot(HaveOccurred())
			Expect(url).To(Equal("https://test.example.com/artist/The Beatles"))
		})
	})

	Describe("GetArtistBiography", func() {
		It("returns the biography from the plugin", func() {
			retriever := agent.(agents.ArtistBiographyRetriever)
			bio, err := retriever.GetArtistBiography(GinkgoT().Context(), "artist-1", "The Beatles", "some-mbid")
			Expect(err).ToNot(HaveOccurred())
			Expect(bio).To(Equal("Biography for The Beatles"))
		})
	})

	Describe("GetArtistImages", func() {
		It("returns images from the plugin", func() {
			retriever := agent.(agents.ArtistImageRetriever)
			images, err := retriever.GetArtistImages(GinkgoT().Context(), "artist-1", "The Beatles", "some-mbid")
			Expect(err).ToNot(HaveOccurred())
			Expect(images).To(HaveLen(2))
			Expect(images[0].URL).To(Equal("https://test.example.com/images/The Beatles/large.jpg"))
			Expect(images[0].Size).To(Equal(500))
			Expect(images[1].URL).To(Equal("https://test.example.com/images/The Beatles/small.jpg"))
			Expect(images[1].Size).To(Equal(100))
		})
	})

	Describe("GetSimilarArtists", func() {
		It("returns similar artists from the plugin", func() {
			retriever := agent.(agents.ArtistSimilarRetriever)
			artists, err := retriever.GetSimilarArtists(GinkgoT().Context(), "artist-1", "The Beatles", "some-mbid", 3)
			Expect(err).ToNot(HaveOccurred())
			Expect(artists).To(HaveLen(3))
			Expect(artists[0].Name).To(Equal("The Beatles Similar A"))
			Expect(artists[1].Name).To(Equal("The Beatles Similar B"))
			Expect(artists[2].Name).To(Equal("The Beatles Similar C"))
		})
	})

	Describe("GetArtistTopSongs", func() {
		It("returns top songs from the plugin", func() {
			retriever := agent.(agents.ArtistTopSongsRetriever)
			songs, err := retriever.GetArtistTopSongs(GinkgoT().Context(), "artist-1", "The Beatles", "some-mbid", 3)
			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(HaveLen(3))
			Expect(songs[0].Name).To(Equal("The Beatles Song 1"))
			Expect(songs[1].Name).To(Equal("The Beatles Song 2"))
			Expect(songs[2].Name).To(Equal("The Beatles Song 3"))
		})
	})

	Describe("GetAlbumInfo", func() {
		It("returns album info from the plugin", func() {
			retriever := agent.(agents.AlbumInfoRetriever)
			info, err := retriever.GetAlbumInfo(GinkgoT().Context(), "Abbey Road", "The Beatles", "album-mbid")
			Expect(err).ToNot(HaveOccurred())
			Expect(info.Name).To(Equal("Abbey Road"))
			Expect(info.MBID).To(Equal("test-album-mbid-Abbey Road"))
			Expect(info.Description).To(Equal("Description for Abbey Road by The Beatles"))
			Expect(info.URL).To(Equal("https://test.example.com/album/Abbey Road"))
		})
	})

	Describe("GetAlbumImages", func() {
		It("returns album images from the plugin", func() {
			retriever := agent.(agents.AlbumImageRetriever)
			images, err := retriever.GetAlbumImages(GinkgoT().Context(), "Abbey Road", "The Beatles", "album-mbid")
			Expect(err).ToNot(HaveOccurred())
			Expect(images).To(HaveLen(1))
			Expect(images[0].URL).To(Equal("https://test.example.com/albums/Abbey Road/cover.jpg"))
			Expect(images[0].Size).To(Equal(500))
		})
	})
})

var _ = Describe("MetadataAgent error handling", Ordered, func() {
	// Tests error paths when plugin is configured to return errors
	var (
		errorManager *Manager
		errorAgent   agents.Interface
	)

	BeforeAll(func() {
		// Create manager with error injection config
		errorManager, _ = createTestManager(map[string]map[string]string{
			"test-metadata-agent": {
				"error": "simulated plugin error",
			},
		})

		// Load the agent
		var ok bool
		errorAgent, ok = errorManager.LoadMediaAgent("test-metadata-agent")
		Expect(ok).To(BeTrue())
	})

	It("returns error from GetArtistMBID", func() {
		retriever := errorAgent.(agents.ArtistMBIDRetriever)
		_, err := retriever.GetArtistMBID(GinkgoT().Context(), "artist-1", "Test")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("simulated plugin error"))
	})

	It("returns error from GetArtistURL", func() {
		retriever := errorAgent.(agents.ArtistURLRetriever)
		_, err := retriever.GetArtistURL(GinkgoT().Context(), "artist-1", "Test", "mbid")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("simulated plugin error"))
	})

	It("returns error from GetArtistBiography", func() {
		retriever := errorAgent.(agents.ArtistBiographyRetriever)
		_, err := retriever.GetArtistBiography(GinkgoT().Context(), "artist-1", "Test", "mbid")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("simulated plugin error"))
	})

	It("returns error from GetArtistImages", func() {
		retriever := errorAgent.(agents.ArtistImageRetriever)
		_, err := retriever.GetArtistImages(GinkgoT().Context(), "artist-1", "Test", "mbid")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("simulated plugin error"))
	})

	It("returns error from GetSimilarArtists", func() {
		retriever := errorAgent.(agents.ArtistSimilarRetriever)
		_, err := retriever.GetSimilarArtists(GinkgoT().Context(), "artist-1", "Test", "mbid", 5)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("simulated plugin error"))
	})

	It("returns error from GetArtistTopSongs", func() {
		retriever := errorAgent.(agents.ArtistTopSongsRetriever)
		_, err := retriever.GetArtistTopSongs(GinkgoT().Context(), "artist-1", "Test", "mbid", 5)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("simulated plugin error"))
	})

	It("returns error from GetAlbumInfo", func() {
		retriever := errorAgent.(agents.AlbumInfoRetriever)
		_, err := retriever.GetAlbumInfo(GinkgoT().Context(), "Album", "Artist", "mbid")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("simulated plugin error"))
	})

	It("returns error from GetAlbumImages", func() {
		retriever := errorAgent.(agents.AlbumImageRetriever)
		_, err := retriever.GetAlbumImages(GinkgoT().Context(), "Album", "Artist", "mbid")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("simulated plugin error"))
	})
})

var _ = Describe("MetadataAgent partial implementation", Ordered, func() {
	// Tests the "not implemented" code path when a plugin only implements some methods
	var (
		partialManager *Manager
		partialAgent   agents.Interface
	)

	BeforeAll(func() {
		// Create manager with the partial metadata agent plugin
		partialManager, _ = createTestManagerWithPlugins(nil, "partial-metadata-agent"+PackageExtension)

		// Load the agent
		var ok bool
		partialAgent, ok = partialManager.LoadMediaAgent("partial-metadata-agent")
		Expect(ok).To(BeTrue())
	})

	It("returns data from implemented method (GetArtistBiography)", func() {
		retriever := partialAgent.(agents.ArtistBiographyRetriever)
		bio, err := retriever.GetArtistBiography(GinkgoT().Context(), "artist-1", "Test Artist", "mbid")
		Expect(err).ToNot(HaveOccurred())
		Expect(bio).To(Equal("Partial agent biography for Test Artist"))
	})

	It("returns ErrNotFound for unimplemented method (GetArtistMBID)", func() {
		retriever := partialAgent.(agents.ArtistMBIDRetriever)
		_, err := retriever.GetArtistMBID(GinkgoT().Context(), "artist-1", "Test Artist")
		Expect(err).To(MatchError(errNotImplemented))
	})

	It("returns ErrNotFound for unimplemented method (GetArtistURL)", func() {
		retriever := partialAgent.(agents.ArtistURLRetriever)
		_, err := retriever.GetArtistURL(GinkgoT().Context(), "artist-1", "Test Artist", "mbid")
		Expect(err).To(MatchError(errNotImplemented))
	})

	It("returns ErrNotFound for unimplemented method (GetArtistImages)", func() {
		retriever := partialAgent.(agents.ArtistImageRetriever)
		_, err := retriever.GetArtistImages(GinkgoT().Context(), "artist-1", "Test Artist", "mbid")
		Expect(err).To(MatchError(errNotImplemented))
	})

	It("returns ErrNotFound for unimplemented method (GetSimilarArtists)", func() {
		retriever := partialAgent.(agents.ArtistSimilarRetriever)
		_, err := retriever.GetSimilarArtists(GinkgoT().Context(), "artist-1", "Test Artist", "mbid", 5)
		Expect(err).To(MatchError(errNotImplemented))

	})

	It("returns ErrNotFound for unimplemented method (GetArtistTopSongs)", func() {
		retriever := partialAgent.(agents.ArtistTopSongsRetriever)
		_, err := retriever.GetArtistTopSongs(GinkgoT().Context(), "artist-1", "Test Artist", "mbid", 5)
		Expect(err).To(MatchError(errNotImplemented))

	})

	It("returns ErrNotFound for unimplemented method (GetAlbumInfo)", func() {
		retriever := partialAgent.(agents.AlbumInfoRetriever)
		_, err := retriever.GetAlbumInfo(GinkgoT().Context(), "Album", "Artist", "mbid")
		Expect(err).To(MatchError(errNotImplemented))

	})

	It("returns ErrNotFound for unimplemented method (GetAlbumImages)", func() {
		retriever := partialAgent.(agents.AlbumImageRetriever)
		_, err := retriever.GetAlbumImages(GinkgoT().Context(), "Album", "Artist", "mbid")
		Expect(err).To(MatchError(errNotImplemented))

	})
})
