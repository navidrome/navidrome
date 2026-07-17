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
			// The snapshot must not be before the artwork version, or the DTO would treat it as
			// stale (it may exceed it: image file mtimes are folded in).
			g.Expect(updated.BlurHashUpdatedAt.Before(updated.ArtworkUpdatedAt())).To(BeFalse())
		}, "10s", "100ms").Should(Succeed())
	})

	It("recomputes when the cover is swapped in place", func() {
		setLayout(fstest.MapFS{
			"Artist/Album/01 - Song.mp3": trackFile(1, "Song"),
			"Artist/Album/cover.png":     realPNG("original-cover"),
		})
		scan()
		al := firstAlbum()
		readArtwork(al.CoverArtID())
		var firstHash string
		Eventually(func(g Gomega) {
			updated, err := ds.Album(ctx).Get(al.ID)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(updated.BlurHash).ToNot(BeEmpty())
			firstHash = updated.BlurHash
		}, "10s", "100ms").Should(Succeed())

		// Swap the cover bytes and rescan: the folder's image version moves, so the reader key moves,
		// the cache misses and the fill re-triggers the compute — no serve-time force hint needed.
		setLayout(fstest.MapFS{
			"Artist/Album/01 - Song.mp3": trackFile(1, "Song"),
			"Artist/Album/cover.png":     realPNG("swapped-cover"),
		})
		scan()
		readArtwork(al.CoverArtID())

		Eventually(func(g Gomega) {
			updated, err := ds.Album(ctx).Get(al.ID)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(updated.BlurHash).ToNot(BeEmpty())
			g.Expect(updated.BlurHash).ToNot(Equal(firstHash))
		}, "10s", "100ms").Should(Succeed())
	})

	It("clears the stored blurhash when the cover disappears", func() {
		setLayout(fstest.MapFS{
			"Artist/Album/01 - Song.mp3": trackFile(1, "Song"),
			"Artist/Album/cover.png":     realPNG("vanishing-cover"),
		})
		scan()
		al := firstAlbum()
		readArtwork(al.CoverArtID())
		Eventually(func(g Gomega) {
			updated, err := ds.Album(ctx).Get(al.ID)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(updated.BlurHash).ToNot(BeEmpty())
		}, "10s", "100ms").Should(Succeed())

		// No rescan: the folder row still lists the cover, but the file is gone — the serve's
		// ErrUnavailable alone must trigger the clear.
		setLayout(fstest.MapFS{
			"Artist/Album/01 - Song.mp3": trackFile(1, "Song"),
		})
		_, err := readArtworkOrErr(al.CoverArtID())
		Expect(err).To(HaveOccurred())

		Eventually(func(g Gomega) {
			updated, err := ds.Album(ctx).Get(al.ID)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(updated.BlurHash).To(BeEmpty())
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
