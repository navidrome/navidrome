package plugins

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/agents"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MetadataAgent", Ordered, func() {
	var (
		manager     *Manager
		agent       agents.Interface
		ctx         context.Context
		testdataDir string
		tmpDir      string
	)

	BeforeAll(func() {
		ctx = GinkgoT().Context()

		// Get testdata directory
		_, currentFile, _, ok := runtime.Caller(0)
		Expect(ok).To(BeTrue())
		testdataDir = filepath.Join(filepath.Dir(currentFile), "testdata")

		// Create temp dir for plugins
		var err error
		tmpDir, err = os.MkdirTemp("", "metadata-agent-test-*")
		Expect(err).ToNot(HaveOccurred())

		// Copy test plugin to temp dir
		srcPath := filepath.Join(testdataDir, "fake-metadata-agent.wasm")
		destPath := filepath.Join(tmpDir, "fake-metadata-agent.wasm")
		data, err := os.ReadFile(srcPath)
		Expect(err).ToNot(HaveOccurred())
		err = os.WriteFile(destPath, data, 0600)
		Expect(err).ToNot(HaveOccurred())

		// Setup config
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = tmpDir
		conf.Server.CacheFolder = filepath.Join(tmpDir, "cache")

		// Create and start the manager
		manager = &Manager{
			plugins: make(map[string]*pluginInstance),
		}
		err = manager.Start(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Load the agent via manager
		var ok2 bool
		agent, ok2 = manager.LoadMediaAgent("fake-metadata-agent")
		Expect(ok2).To(BeTrue())

		DeferCleanup(func() {
			_ = manager.Stop()
			_ = os.RemoveAll(tmpDir)
		})
	})

	Describe("AgentName", func() {
		It("returns the plugin name", func() {
			Expect(agent.AgentName()).To(Equal("fake-metadata-agent"))
		})
	})

	Describe("GetArtistMBID", func() {
		It("returns the MBID from the plugin", func() {
			retriever := agent.(agents.ArtistMBIDRetriever)
			mbid, err := retriever.GetArtistMBID(ctx, "artist-1", "The Beatles")
			Expect(err).ToNot(HaveOccurred())
			Expect(mbid).To(Equal("test-mbid-The Beatles"))
		})
	})

	Describe("GetArtistURL", func() {
		It("returns the URL from the plugin", func() {
			retriever := agent.(agents.ArtistURLRetriever)
			url, err := retriever.GetArtistURL(ctx, "artist-1", "The Beatles", "some-mbid")
			Expect(err).ToNot(HaveOccurred())
			Expect(url).To(Equal("https://test.example.com/artist/The Beatles"))
		})
	})

	Describe("GetArtistBiography", func() {
		It("returns the biography from the plugin", func() {
			retriever := agent.(agents.ArtistBiographyRetriever)
			bio, err := retriever.GetArtistBiography(ctx, "artist-1", "The Beatles", "some-mbid")
			Expect(err).ToNot(HaveOccurred())
			Expect(bio).To(Equal("Biography for The Beatles"))
		})
	})

	Describe("GetArtistImages", func() {
		It("returns images from the plugin", func() {
			retriever := agent.(agents.ArtistImageRetriever)
			images, err := retriever.GetArtistImages(ctx, "artist-1", "The Beatles", "some-mbid")
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
			artists, err := retriever.GetSimilarArtists(ctx, "artist-1", "The Beatles", "some-mbid", 3)
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
			songs, err := retriever.GetArtistTopSongs(ctx, "artist-1", "The Beatles", "some-mbid", 3)
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
			info, err := retriever.GetAlbumInfo(ctx, "Abbey Road", "The Beatles", "album-mbid")
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
			images, err := retriever.GetAlbumImages(ctx, "Abbey Road", "The Beatles", "album-mbid")
			Expect(err).ToNot(HaveOccurred())
			Expect(images).To(HaveLen(1))
			Expect(images[0].URL).To(Equal("https://test.example.com/albums/Abbey Road/cover.jpg"))
			Expect(images[0].Size).To(Equal(500))
		})
	})
})
