package artworke2e_test

import (
	"os"
	"path/filepath"
	"testing/fstest"
	"time"

	"github.com/navidrome/navidrome/model"
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

	It("does not persist a future-dated blurhash timestamp", func() {
		cover := realPNG("future-cover")
		cover.ModTime = time.Now().Add(500 * time.Hour) // clock skew / future-stamped file
		setLayout(fstest.MapFS{
			"Artist/Album/01 - Song.mp3": trackFile(1, "Song"),
			"Artist/Album/cover.png":     cover,
		})
		scan()
		al := firstAlbum()
		readArtwork(al.CoverArtID())

		Eventually(func(g Gomega) {
			updated, err := ds.Album(ctx).Get(al.ID)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(updated.BlurHash).ToNot(BeEmpty())
			g.Expect(updated.BlurHashUpdatedAt).ToNot(BeNil())
			// A future file mtime must be capped at now, or the !Before checks would pin the hash
			// (and the client's cover cache) until wall time caught up.
			g.Expect(updated.BlurHashUpdatedAt.After(time.Now())).To(BeFalse())
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

		// Swap the cover bytes and rescan, then serve: the tee hashes the newly-served bytes, so the
		// stored hash moves to describe the new cover.
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

		// No rescan: the folder row still lists the cover, but the file is gone. The serve falls back
		// to the placeholder (GetOrPlaceholder, the real Jellyfin/Subsonic path), and the worker's
		// gone-recheck confirms the source is really gone and clears the stored hash.
		setLayout(fstest.MapFS{
			"Artist/Album/01 - Song.mp3": trackFile(1, "Song"),
		})
		Expect(readOrPlaceholder(al.CoverArtID())).To(Equal(placeholderBytes()))

		Eventually(func(g Gomega) {
			updated, err := ds.Album(ctx).Get(al.ID)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(updated.BlurHash).To(BeEmpty())
		}, "10s", "100ms").Should(Succeed())
	})

	It("recomputes when cover bytes change under a preserved mtime (cache disabled)", func() {
		cover := realPNG("orig-bytes")
		fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		cover.ModTime = fixed
		setLayout(fstest.MapFS{
			"Artist/Album/01 - Song.mp3": trackFile(1, "Song"),
			"Artist/Album/cover.png":     cover,
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

		// Replace the bytes but keep the SAME mtime and do NOT rescan: only the served bytes change.
		swapped := realPNG("swapped-bytes")
		swapped.ModTime = fixed
		setLayout(fstest.MapFS{
			"Artist/Album/01 - Song.mp3": trackFile(1, "Song"),
			"Artist/Album/cover.png":     swapped,
		})
		readArtwork(al.CoverArtID())

		Eventually(func(g Gomega) {
			updated, err := ds.Album(ctx).Get(al.ID)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(updated.BlurHash).ToNot(Equal(firstHash))
		}, "10s", "100ms").Should(Succeed())
	})

	It("clears a stored playlist hash when it falls back to the placeholder", func() {
		// A playlist with a sidecar cover gets a real hash; removing the sidecar makes the reader chain
		// fall through to fromAlbumPlaceholder(), whose bytes flow through the tee on Get and clear it.
		dir := GinkgoT().TempDir()
		m3uPath := filepath.Join(dir, "MyList.m3u")
		Expect(os.WriteFile(m3uPath, []byte("#EXTM3U\n"), 0600)).To(Succeed())
		sidecar := filepath.Join(dir, "MyList.png")
		Expect(os.WriteFile(sidecar, realPNG("pl-cover").Data, 0600)).To(Succeed())

		pl := putPlaylist(model.Playlist{ID: "pl-blur", Name: "MyList", Path: m3uPath})
		readArtwork(pl.CoverArtID())
		Eventually(func(g Gomega) {
			updated, err := ds.Playlist(ctx).Get(pl.ID)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(updated.BlurHash).ToNot(BeEmpty())
		}, "10s", "100ms").Should(Succeed())

		// Remove the sidecar: the serve now falls through to the placeholder, captured by the tee.
		Expect(os.Remove(sidecar)).To(Succeed())
		Expect(readArtwork(pl.CoverArtID())).To(Equal(placeholderBytes()))

		Eventually(func(g Gomega) {
			updated, err := ds.Playlist(ctx).Get(pl.ID)
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
