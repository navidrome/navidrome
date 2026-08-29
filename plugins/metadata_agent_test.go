//go:build !windows

package plugins

import (
	"errors"
	"fmt"
	"time"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/plugins/capabilities"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// The partial-metadata-agent fixture registers through the Go PDK, which exports every method
// and answers -2, so only errNotImplemented reaches agentErr through a real plugin. A plugin
// that omits the export entirely yields errFunctionNotFound, covered here directly.
var _ = Describe("agentErr", func() {
	DescribeTable("classifies a plugin error as a miss or a fault",
		func(err error, wantMiss bool) {
			got := agentErr(err)
			Expect(errors.Is(got, agents.ErrNotFound)).To(Equal(wantMiss))
			Expect(got).To(MatchError(err), "the underlying reason must survive for diagnostics")
		},
		Entry("an unimplemented method is a miss",
			fmt.Errorf("%w: %s", errNotImplemented, FuncGetArtistImages), true),
		Entry("a missing export is a miss",
			fmt.Errorf("%w: %s", errFunctionNotFound, FuncGetArtistImages), true),
		Entry("a call failure is a fault",
			fmt.Errorf("plugin call failed: %w", errors.New("returned status 429")), false),
		Entry("a non-zero exit is a fault",
			errors.New("plugin call exited with code 1"), false),
	)

	DescribeTable("agentErr retry-later",
		func(msg string, wantDelay time.Duration) {
			err := agentErr(errors.New(msg))
			Expect(errors.Is(err, agents.ErrRetryLater)).To(BeTrue())
			retry, _ := errors.AsType[*agents.RetryLaterError](err)
			d := retry.RetryIn
			Expect(d).To(Equal(wantDelay))
		},
		Entry("bare token", "agent(retry_later)", time.Duration(0)),
		Entry("with seconds", "agent(retry_later:120)", 120*time.Second),
		Entry("capped at 1h", "agent(retry_later:999999)", time.Hour),
		// Scaling to nanoseconds before capping wraps past 2^64, landing on ~0.29s.
		Entry("capped before it can overflow", "agent(retry_later:18446744074)", time.Hour),
	)

	It("leaves other plugin errors untouched", func() {
		orig := errors.New("some plugin failure")
		Expect(agentErr(orig)).To(Equal(orig))
	})

	It("does not treat a superstring token as a throttle", func() {
		orig := errors.New("useragent(retry_later)")
		Expect(agentErr(orig)).To(Equal(orig))
	})
})

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

	Describe("GetSimilarSongsByTrack", func() {
		It("returns similar songs from the plugin", func() {
			retriever := agent.(agents.SimilarSongsByTrackRetriever)
			songs, err := retriever.GetSimilarSongsByTrack(GinkgoT().Context(), "track-1", "Yesterday", "The Beatles", "some-mbid", 3)
			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(HaveLen(3))
			Expect(songs[0].Name).To(Equal("Similar to Yesterday #1"))
			Expect(songs[0].Artists).To(HaveLen(1))
			Expect(songs[0].Artists[0].Name).To(Equal("The Beatles"))
		})
	})

	Describe("GetSimilarSongsByAlbum", func() {
		It("returns similar songs from the plugin", func() {
			retriever := agent.(agents.SimilarSongsByAlbumRetriever)
			songs, err := retriever.GetSimilarSongsByAlbum(GinkgoT().Context(), "album-1", "Abbey Road", "The Beatles", "album-mbid", 3)
			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(HaveLen(3))
			Expect(songs[0].Album).To(Equal("Abbey Road"))
		})
	})

	Describe("GetSimilarSongsByArtist", func() {
		It("returns similar songs from the plugin", func() {
			retriever := agent.(agents.SimilarSongsByArtistRetriever)
			songs, err := retriever.GetSimilarSongsByArtist(GinkgoT().Context(), "artist-1", "The Beatles", "some-mbid", 3)
			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(HaveLen(3))
			Expect(songs[0].Name).To(ContainSubstring("The Beatles Style Song"))
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

	DescribeTable("surfaces the plugin failure, not a definitive not-found",
		func(call func() error) {
			err := call()
			Expect(err).To(MatchError(ContainSubstring("simulated plugin error")))
			Expect(err).ToNot(MatchError(agents.ErrNotFound))
		},
		Entry("GetArtistMBID", func() error {
			_, err := errorAgent.(agents.ArtistMBIDRetriever).GetArtistMBID(GinkgoT().Context(), "artist-1", "Test")
			return err
		}),
		Entry("GetArtistURL", func() error {
			_, err := errorAgent.(agents.ArtistURLRetriever).GetArtistURL(GinkgoT().Context(), "artist-1", "Test", "mbid")
			return err
		}),
		Entry("GetArtistBiography", func() error {
			_, err := errorAgent.(agents.ArtistBiographyRetriever).GetArtistBiography(GinkgoT().Context(), "artist-1", "Test", "mbid")
			return err
		}),
		Entry("GetSimilarArtists", func() error {
			_, err := errorAgent.(agents.ArtistSimilarRetriever).GetSimilarArtists(GinkgoT().Context(), "artist-1", "Test", "mbid", 5)
			return err
		}),
		Entry("GetArtistImages", func() error {
			_, err := errorAgent.(agents.ArtistImageRetriever).GetArtistImages(GinkgoT().Context(), "artist-1", "Test", "mbid")
			return err
		}),
		Entry("GetArtistTopSongs", func() error {
			_, err := errorAgent.(agents.ArtistTopSongsRetriever).GetArtistTopSongs(GinkgoT().Context(), "artist-1", "Test", "mbid", 5)
			return err
		}),
		Entry("GetAlbumInfo", func() error {
			_, err := errorAgent.(agents.AlbumInfoRetriever).GetAlbumInfo(GinkgoT().Context(), "Album", "Artist", "mbid")
			return err
		}),
		Entry("GetAlbumImages", func() error {
			_, err := errorAgent.(agents.AlbumImageRetriever).GetAlbumImages(GinkgoT().Context(), "Album", "Artist", "mbid")
			return err
		}),
		Entry("GetSimilarSongsByTrack", func() error {
			_, err := errorAgent.(agents.SimilarSongsByTrackRetriever).GetSimilarSongsByTrack(GinkgoT().Context(), "track-1", "Test", "Artist", "mbid", 5)
			return err
		}),
		Entry("GetSimilarSongsByAlbum", func() error {
			_, err := errorAgent.(agents.SimilarSongsByAlbumRetriever).GetSimilarSongsByAlbum(GinkgoT().Context(), "album-1", "Album", "Artist", "mbid", 5)
			return err
		}),
		Entry("GetSimilarSongsByArtist", func() error {
			_, err := errorAgent.(agents.SimilarSongsByArtistRetriever).GetSimilarSongsByArtist(GinkgoT().Context(), "artist-1", "Artist", "mbid", 5)
			return err
		}),
	)
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

	// An unimplemented optional method is a definitive miss. Reported as a fault it would
	// count against the artwork circuit breaker and keep the item in the retry queue.
	DescribeTable("reports an unimplemented method as a definitive not-found",
		func(call func() error) {
			err := call()
			Expect(err).To(MatchError(errNotImplemented))
			Expect(err).To(MatchError(agents.ErrNotFound))
		},
		Entry("GetArtistMBID", func() error {
			_, err := partialAgent.(agents.ArtistMBIDRetriever).GetArtistMBID(GinkgoT().Context(), "artist-1", "Test Artist")
			return err
		}),
		Entry("GetArtistURL", func() error {
			_, err := partialAgent.(agents.ArtistURLRetriever).GetArtistURL(GinkgoT().Context(), "artist-1", "Test Artist", "mbid")
			return err
		}),
		Entry("GetArtistImages", func() error {
			_, err := partialAgent.(agents.ArtistImageRetriever).GetArtistImages(GinkgoT().Context(), "artist-1", "Test Artist", "mbid")
			return err
		}),
		Entry("GetSimilarArtists", func() error {
			_, err := partialAgent.(agents.ArtistSimilarRetriever).GetSimilarArtists(GinkgoT().Context(), "artist-1", "Test Artist", "mbid", 5)
			return err
		}),
		Entry("GetArtistTopSongs", func() error {
			_, err := partialAgent.(agents.ArtistTopSongsRetriever).GetArtistTopSongs(GinkgoT().Context(), "artist-1", "Test Artist", "mbid", 5)
			return err
		}),
		Entry("GetAlbumInfo", func() error {
			_, err := partialAgent.(agents.AlbumInfoRetriever).GetAlbumInfo(GinkgoT().Context(), "Album", "Artist", "mbid")
			return err
		}),
		Entry("GetAlbumImages", func() error {
			_, err := partialAgent.(agents.AlbumImageRetriever).GetAlbumImages(GinkgoT().Context(), "Album", "Artist", "mbid")
			return err
		}),
		Entry("GetSimilarSongsByTrack", func() error {
			_, err := partialAgent.(agents.SimilarSongsByTrackRetriever).GetSimilarSongsByTrack(GinkgoT().Context(), "track-1", "Test", "Artist", "mbid", 5)
			return err
		}),
		Entry("GetSimilarSongsByAlbum", func() error {
			_, err := partialAgent.(agents.SimilarSongsByAlbumRetriever).GetSimilarSongsByAlbum(GinkgoT().Context(), "album-1", "Album", "Artist", "mbid", 5)
			return err
		}),
		Entry("GetSimilarSongsByArtist", func() error {
			_, err := partialAgent.(agents.SimilarSongsByArtistRetriever).GetSimilarSongsByArtist(GinkgoT().Context(), "artist-1", "Artist", "mbid", 5)
			return err
		}),
	)
})

var _ = Describe("songRefToAgentSong multi-artist", func() {
	It("maps ArtistRef to agents.Artist", func() {
		ref := capabilities.SongRef{Name: "Collab", Artist: "Drake", Artists: []capabilities.ArtistRef{
			{ID: "id-drake", Name: "Drake", MBID: "m-drake"},
			{Name: "Future", MBID: "m-future"},
		}}
		got := songRefToAgentSong(ref)
		Expect(got.Artists).To(Equal([]agents.Artist{
			{ID: "id-drake", Name: "Drake", MBID: "m-drake"},
			{Name: "Future", MBID: "m-future"},
		}))
	})
	It("folds the single Artist/ArtistMBID into a one-element Artists when no Artists provided", func() {
		ref := capabilities.SongRef{Name: "Solo", Artist: "Drake", ArtistMBID: "m-drake"}
		got := songRefToAgentSong(ref)
		Expect(got.Artists).To(Equal([]agents.Artist{{Name: "Drake", MBID: "m-drake"}}))
	})
	It("folds an MBID-only single artist (empty name) so the MBID is not dropped", func() {
		ref := capabilities.SongRef{Name: "Solo", ArtistMBID: "m-drake"}
		got := songRefToAgentSong(ref)
		Expect(got.Artists).To(Equal([]agents.Artist{{MBID: "m-drake"}}))
	})
	It("leaves Artists nil when neither Artists nor the single Artist/ArtistMBID are provided", func() {
		got := songRefToAgentSong(capabilities.SongRef{Name: "Anon"})
		Expect(got.Artists).To(BeNil())
	})
})
