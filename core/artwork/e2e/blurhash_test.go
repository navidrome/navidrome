package artworke2e_test

import (
	"testing/fstest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BlurHash", func() {
	BeforeEach(func() {
		setupHarness()
	})

	It("persists a real blurhash after album artwork is served", func() {
		setLayout(fstest.MapFS{
			"Artist/Album/01 - Song.mp3": trackFile(1, "Song"),
			"Artist/Album/cover.png":     realPNG("blurhash-album"),
		})
		scan()
		al := firstAlbum()
		Expect(al.BlurHash).To(BeEmpty())

		// Serving the artwork enqueues the async blurhash computation.
		readArtwork(al.CoverArtID())

		Eventually(func(g Gomega) {
			updated, err := ds.Album(ctx).Get(al.ID)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(len(updated.BlurHash)).To(BeNumerically(">", 6))
			g.Expect(updated.BlurHashUpdatedAt).ToNot(BeNil())
			// The snapshot must match the artwork version, or the DTO layer would treat it as stale.
			g.Expect(updated.BlurHashUpdatedAt.Equal(updated.ArtworkUpdatedAt())).To(BeTrue())
		}, "10s", "100ms").Should(Succeed())
	})

	It("does not persist a blurhash when the served image cannot be decoded", func() {
		setLayout(fstest.MapFS{
			"Artist/Album/01 - Song.mp3": trackFile(1, "Song"),
			"Artist/Album/cover.png":     imageFile("not-a-real-image"),
		})
		scan()
		al := firstAlbum()

		readArtwork(al.CoverArtID())

		Consistently(func(g Gomega) {
			updated, err := ds.Album(ctx).Get(al.ID)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(updated.BlurHash).To(BeEmpty())
		}, "600ms", "100ms").Should(Succeed())
	})
})
