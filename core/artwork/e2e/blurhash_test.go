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

	// The blurhash is computed inline when the served reader is closed, so by the time the read
	// helpers return, the hash is already persisted — no polling needed.
	storedAlbum := func(id string) model.Album {
		GinkgoHelper()
		updated, err := ds.Album(ctx).Get(id)
		Expect(err).ToNot(HaveOccurred())
		return *updated
	}

	It("persists a real blurhash after album artwork is served", func() {
		setLayout(fstest.MapFS{
			"Artist/Album/01 - Song.mp3": trackFile(1, "Song"),
			"Artist/Album/cover.png":     realPNG("blurhash-album"),
		})
		scan()
		al := firstAlbum()
		Expect(al.BlurHash).To(BeEmpty())

		readArtwork(al.CoverArtID())

		updated := storedAlbum(al.ID)
		Expect(len(updated.BlurHash)).To(BeNumerically(">", 6))
		Expect(updated.BlurHashUpdatedAt).ToNot(BeNil())
		// The snapshot must not be before the artwork version, or the DTO would treat it as
		// stale (it may exceed it: image file mtimes are folded in).
		Expect(updated.BlurHashUpdatedAt.Before(updated.ArtworkUpdatedAt())).To(BeFalse())
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

		updated := storedAlbum(al.ID)
		Expect(updated.BlurHash).ToNot(BeEmpty())
		Expect(updated.BlurHashUpdatedAt).ToNot(BeNil())
		// A future file mtime must be capped at now, or the !Before checks would pin the hash
		// (and the client's cover cache) until wall time caught up.
		Expect(updated.BlurHashUpdatedAt.After(time.Now())).To(BeFalse())
	})

	It("recomputes when the cover is swapped in place", func() {
		setLayout(fstest.MapFS{
			"Artist/Album/01 - Song.mp3": trackFile(1, "Song"),
			"Artist/Album/cover.png":     realPNG("original-cover"),
		})
		scan()
		al := firstAlbum()
		readArtwork(al.CoverArtID())
		firstHash := storedAlbum(al.ID).BlurHash
		Expect(firstHash).ToNot(BeEmpty())

		// Swap the cover bytes and rescan, then serve: the tee hashes the newly-served bytes, so the
		// stored hash moves to describe the new cover.
		setLayout(fstest.MapFS{
			"Artist/Album/01 - Song.mp3": trackFile(1, "Song"),
			"Artist/Album/cover.png":     realPNG("swapped-cover"),
		})
		scan()
		readArtwork(al.CoverArtID())

		updated := storedAlbum(al.ID)
		Expect(updated.BlurHash).ToNot(BeEmpty())
		Expect(updated.BlurHash).ToNot(Equal(firstHash))
	})

	It("clears the stored blurhash when the cover disappears", func() {
		setLayout(fstest.MapFS{
			"Artist/Album/01 - Song.mp3": trackFile(1, "Song"),
			"Artist/Album/cover.png":     realPNG("vanishing-cover"),
		})
		scan()
		al := firstAlbum()
		readArtwork(al.CoverArtID())
		Expect(storedAlbum(al.ID).BlurHash).ToNot(BeEmpty())

		// No rescan: the folder row still lists the cover, but the file is gone. The serve falls back
		// to the placeholder (GetOrPlaceholder, the real Jellyfin/Subsonic path), which clears the
		// stored hash inline.
		setLayout(fstest.MapFS{
			"Artist/Album/01 - Song.mp3": trackFile(1, "Song"),
		})
		Expect(readOrPlaceholder(al.CoverArtID())).To(Equal(placeholderBytes()))

		Expect(storedAlbum(al.ID).BlurHash).To(BeEmpty())
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
		firstHash := storedAlbum(al.ID).BlurHash
		Expect(firstHash).ToNot(BeEmpty())

		// Replace the bytes but keep the SAME mtime and do NOT rescan: only the served bytes change.
		swapped := realPNG("swapped-bytes")
		swapped.ModTime = fixed
		setLayout(fstest.MapFS{
			"Artist/Album/01 - Song.mp3": trackFile(1, "Song"),
			"Artist/Album/cover.png":     swapped,
		})
		readArtwork(al.CoverArtID())

		Expect(storedAlbum(al.ID).BlurHash).ToNot(Equal(firstHash))
	})

	It("advances the album artwork version when only the cover file changes (quick scan)", func() {
		setLayout(fstest.MapFS{
			"Artist/Album/01 - Song.mp3": trackFile(1, "Song"),
			"Artist/Album/cover.png":     realPNG("p1-orig"),
		})
		scan()
		al := firstAlbum()
		readArtwork(al.CoverArtID())
		first := storedAlbum(al.ID)
		Expect(first.BlurHash).ToNot(BeEmpty())
		Expect(first.BlurHashUpdatedAt.Before(first.ArtworkUpdatedAt())).To(BeFalse())

		// Replace only the cover and quick-scan: the album row stays untouched while the folder's
		// images_updated_at advances the artwork version, so hash-keyed clients refetch.
		fakeFS.Add("Artist/Album/cover.png", realPNG("p1-swapped"), time.Now())
		quickScan()

		stale := storedAlbum(al.ID)
		Expect(stale.UpdatedAt).To(Equal(first.UpdatedAt), "premise: image-only change must not touch the album row")
		Expect(stale.BlurHash).To(Equal(first.BlurHash))
		Expect(stale.BlurHashUpdatedAt.Before(stale.ArtworkUpdatedAt())).To(BeTrue(), "stored hash must read as stale")

		// The refetch serves the new bytes; the tee rotates the hash and its version catches up.
		readArtwork(al.CoverArtID())
		fresh := storedAlbum(al.ID)
		Expect(fresh.BlurHash).ToNot(Equal(first.BlurHash))
		Expect(fresh.BlurHashUpdatedAt.Before(fresh.ArtworkUpdatedAt())).To(BeFalse())
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
		stored, err := ds.Playlist(ctx).Get(pl.ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(stored.BlurHash).ToNot(BeEmpty())

		// Remove the sidecar: the serve now falls through to the placeholder, captured by the tee.
		Expect(os.Remove(sidecar)).To(Succeed())
		Expect(readArtwork(pl.CoverArtID())).To(Equal(placeholderBytes()))

		stored, err = ds.Playlist(ctx).Get(pl.ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(stored.BlurHash).To(BeEmpty())
	})

	It("does not persist a blurhash when the served image cannot be decoded", func() {
		setLayout(fstest.MapFS{
			"Artist/Album/01 - Song.mp3": trackFile(1, "Song"),
			"Artist/Album/cover.png":     imageFile("not-a-real-image"),
		})
		scan()
		al := firstAlbum()

		readArtwork(al.CoverArtID())

		Expect(storedAlbum(al.ID).BlurHash).To(BeEmpty())
	})
})
