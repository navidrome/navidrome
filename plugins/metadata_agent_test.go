package plugins

import (
	"context"
	"path/filepath"
	"runtime"

	extism "github.com/extism/go-sdk"
	"github.com/navidrome/navidrome/core/agents"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MetadataAgent", func() {
	var (
		agent *MetadataAgent
		ctx   context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Load the test plugin
		_, currentFile, _, ok := runtime.Caller(0)
		Expect(ok).To(BeTrue())
		testdataDir := filepath.Join(filepath.Dir(currentFile), "testdata")
		wasmPath := filepath.Join(testdataDir, "test-plugin.wasm")

		manifest := extism.Manifest{
			Wasm: []extism.Wasm{
				extism.WasmFile{Path: wasmPath},
			},
			AllowedHosts: []string{"test.example.com"},
		}

		plugin, err := extism.NewPlugin(ctx, manifest, extism.PluginConfig{
			EnableWasi: true,
		}, nil)
		Expect(err).ToNot(HaveOccurred())

		agent = NewMetadataAgent("test-plugin", plugin)
	})

	AfterEach(func() {
		if agent != nil {
			_ = agent.Close()
		}
	})

	Describe("AgentName", func() {
		It("returns the plugin name", func() {
			Expect(agent.AgentName()).To(Equal("test-plugin"))
		})
	})

	Describe("GetArtistMBID", func() {
		It("returns the MBID from the plugin", func() {
			mbid, err := agent.GetArtistMBID(ctx, "artist-1", "The Beatles")
			Expect(err).ToNot(HaveOccurred())
			Expect(mbid).To(Equal("test-mbid-The Beatles"))
		})
	})

	Describe("GetArtistURL", func() {
		It("returns the URL from the plugin", func() {
			url, err := agent.GetArtistURL(ctx, "artist-1", "The Beatles", "some-mbid")
			Expect(err).ToNot(HaveOccurred())
			Expect(url).To(Equal("https://test.example.com/artist/The Beatles"))
		})
	})

	Describe("GetArtistBiography", func() {
		It("returns the biography from the plugin", func() {
			bio, err := agent.GetArtistBiography(ctx, "artist-1", "The Beatles", "some-mbid")
			Expect(err).ToNot(HaveOccurred())
			Expect(bio).To(Equal("Biography for The Beatles"))
		})
	})

	Describe("GetArtistImages", func() {
		It("returns images from the plugin", func() {
			images, err := agent.GetArtistImages(ctx, "artist-1", "The Beatles", "some-mbid")
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
			artists, err := agent.GetSimilarArtists(ctx, "artist-1", "The Beatles", "some-mbid", 3)
			Expect(err).ToNot(HaveOccurred())
			Expect(artists).To(HaveLen(3))
			Expect(artists[0].Name).To(Equal("The Beatles Similar A"))
			Expect(artists[1].Name).To(Equal("The Beatles Similar B"))
			Expect(artists[2].Name).To(Equal("The Beatles Similar C"))
		})
	})

	Describe("GetArtistTopSongs", func() {
		It("returns top songs from the plugin", func() {
			songs, err := agent.GetArtistTopSongs(ctx, "artist-1", "The Beatles", "some-mbid", 3)
			Expect(err).ToNot(HaveOccurred())
			Expect(songs).To(HaveLen(3))
			Expect(songs[0].Name).To(Equal("The Beatles Song 1"))
			Expect(songs[1].Name).To(Equal("The Beatles Song 2"))
			Expect(songs[2].Name).To(Equal("The Beatles Song 3"))
		})
	})

	Describe("GetAlbumInfo", func() {
		It("returns album info from the plugin", func() {
			info, err := agent.GetAlbumInfo(ctx, "Abbey Road", "The Beatles", "album-mbid")
			Expect(err).ToNot(HaveOccurred())
			Expect(info.Name).To(Equal("Abbey Road"))
			Expect(info.MBID).To(Equal("test-album-mbid-Abbey Road"))
			Expect(info.Description).To(Equal("Description for Abbey Road by The Beatles"))
			Expect(info.URL).To(Equal("https://test.example.com/album/Abbey Road"))
		})
	})

	Describe("GetAlbumImages", func() {
		It("returns album images from the plugin", func() {
			images, err := agent.GetAlbumImages(ctx, "Abbey Road", "The Beatles", "album-mbid")
			Expect(err).ToNot(HaveOccurred())
			Expect(images).To(HaveLen(1))
			Expect(images[0].URL).To(Equal("https://test.example.com/albums/Abbey Road/cover.jpg"))
			Expect(images[0].Size).To(Equal(500))
		})
	})

	Describe("interface assertions", func() {
		It("implements all required interfaces", func() {
			var _ agents.Interface = agent
			var _ agents.ArtistMBIDRetriever = agent
			var _ agents.ArtistURLRetriever = agent
			var _ agents.ArtistBiographyRetriever = agent
			var _ agents.ArtistSimilarRetriever = agent
			var _ agents.ArtistImageRetriever = agent
			var _ agents.ArtistTopSongsRetriever = agent
			var _ agents.AlbumInfoRetriever = agent
			var _ agents.AlbumImageRetriever = agent
		})
	})
})
