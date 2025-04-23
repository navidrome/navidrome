package plugins

import (
	"context"
	"sync"

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
			_, ok := agents.Map["fake_album_agent"]
			return ok
		}, "5s", "100ms").Should(BeTrue(), "plugin agent should be registered")

		constructor, ok := agents.Map["fake_album_agent"]
		Expect(ok).To(BeTrue())
		agent = constructor(nil)
		Expect(agent).NotTo(BeNil(), "plugin agent should be constructible")
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

// Concurrency stress test for CoverArtArchive plugin
var _ = XDescribe("wasmAlbumAgent (coverartarchive concurrency)", func() {
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
			_, ok := agents.Map["coverartarchive"]
			return ok
		}, "5s", "100ms").Should(BeTrue(), "plugin agent should be registered")

		constructor, ok := agents.Map["coverartarchive"]
		Expect(ok).To(BeTrue())
		agent = constructor(nil)
		Expect(agent).NotTo(BeNil(), "plugin agent should be constructible")
	})

	It("handles many concurrent GetAlbumImages calls without crashing", func() {
		imagesRetriever := agent.(interface {
			GetAlbumImages(ctx context.Context, name, artist, mbid string) ([]agents.ExternalImage, error)
		})
		mbid := "08a8025e-8fb9-4e1b-b8a5-c59f66a99006" // Use a real MBID for stress
		wg := sync.WaitGroup{}
		const numThreads = 50
		errs := make(chan error, numThreads)
		for i := 0; i < numThreads; i++ {
			wg.Add(1)
			go func() {
				defer GinkgoRecover()
				defer wg.Done()
				_, err := imagesRetriever.GetAlbumImages(ctx, "", "", mbid)
				if err != nil {
					errs <- err
				}
			}()
		}
		wg.Wait()
		close(errs)
		// All errors should be handled, and there should be no panics or plugin crashes
		for err := range errs {
			println(err.Error())
			Expect(err).ToNot(MatchError("module closed with exit_code(0)"))
		}
	})
})
