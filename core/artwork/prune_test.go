package artwork

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type flakyGetArtworkRepo struct {
	model.ArtworkRepository
}

func (f *flakyGetArtworkRepo) GetMimeByHash(context.Context) (map[string]string, error) {
	return nil, errors.New("db locked")
}

var _ = Describe("Prune", func() {
	var ctx context.Context
	var ds model.DataStore
	var store *ImageStore
	var awRepo model.ArtworkRepository

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = newTestDS(ctx)
		awRepo = ds.Artwork()
		store = NewImageStore(GinkgoT().TempDir())
	})

	// PutImage refreshes created_at, so fixtures are aged directly.
	ageArtwork := func(h string, t time.Time) {
		GinkgoHelper()
		_, err := db.Db().Exec("UPDATE artwork SET created_at = ? WHERE hash = ?", t, h)
		Expect(err).ToNot(HaveOccurred())
	}

	It("purges dangling item_artwork state for gone entities, summed across kinds", func() {
		putArtists(ctx, ds, model.Artist{ID: "live-artist", Name: "Live"})
		Expect(awRepo.PutItemArtwork(ctx, &model.ItemArtwork{ItemKind: "al", ItemID: "gone-album", ImageType: model.ImageTypePrimary})).To(Succeed())
		Expect(awRepo.PutItemArtwork(ctx, &model.ItemArtwork{ItemKind: "ar", ItemID: "gone-artist", ImageType: model.ImageTypePrimary})).To(Succeed())
		Expect(awRepo.PutItemArtwork(ctx, &model.ItemArtwork{ItemKind: "ar", ItemID: "live-artist", ImageType: model.ImageTypePrimary})).To(Succeed())

		Expect(prune(ctx, ds, store)).To(Succeed())

		_, err := awRepo.GetItemArtwork(ctx, model.KindAlbumArtwork, "gone-album", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound))
		_, err = awRepo.GetItemArtwork(ctx, model.KindArtistArtwork, "gone-artist", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound))
		_, err = awRepo.GetItemArtwork(ctx, model.KindArtistArtwork, "live-artist", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
	})

	It("purges dangling artwork_queue rows for gone entities", func() {
		putAlbums(ctx, ds, model.Album{ID: "live-album", Name: "Live"})
		Expect(ds.ArtworkQueue().Enqueue(ctx,
			model.ArtworkQueueItem{ItemKind: "al", ItemID: "gone-album", ImageType: model.ImageTypePrimary},
			model.ArtworkQueueItem{ItemKind: "al", ItemID: "live-album", ImageType: model.ImageTypePrimary},
		)).To(Succeed())

		Expect(prune(ctx, ds, store)).To(Succeed())

		Expect(queued(ctx, ds, "al", "gone-album")).To(BeNil())
		Expect(queued(ctx, ds, "al", "live-album")).ToNot(BeNil())
	})

	It("deletes orphan rows and their store files, keeps referenced ones", func() {
		data := []byte("orphan-bytes")
		h, _ := hashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())
		old := time.Now().Add(-2 * time.Hour)
		Expect(os.Chtimes(store.path(h, "image/jpeg"), old, old)).To(Succeed())
		Expect(awRepo.PutImage(ctx, &model.Artwork{Hash: h, Mime: "image/jpeg"})).To(Succeed())
		ageArtwork(h, old)

		kept := []byte("kept-bytes")
		hk, _ := hashImage(bytes.NewReader(kept))
		Expect(store.Write(hk, "image/jpeg", bytes.NewReader(kept))).To(Succeed())
		Expect(awRepo.PutImage(ctx, &model.Artwork{Hash: hk, Mime: "image/jpeg"})).To(Succeed())

		Expect(prune(ctx, ds, store)).To(Succeed())

		_, err := awRepo.GetImage(ctx, h)
		Expect(err).To(MatchError(model.ErrNotFound))
		_, err = store.Open(h, "image/jpeg")
		Expect(os.IsNotExist(err)).To(BeTrue())
		rc, err := store.Open(hk, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("spares an aged row that item_artwork state still references", func() {
		data := []byte("reacquired-bytes")
		h, _ := hashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())
		Expect(awRepo.PutImage(ctx, &model.Artwork{Hash: h, Mime: "image/jpeg"})).To(Succeed())
		ageArtwork(h, time.Now().Add(-2*time.Hour))
		putAlbums(ctx, ds, model.Album{ID: "a1", Name: "Album"})
		Expect(awRepo.PutItemArtwork(ctx, &model.ItemArtwork{ItemKind: "al", ItemID: "a1",
			ImageType: model.ImageTypePrimary, Hash: h, Source: "folder"})).To(Succeed())

		Expect(prune(ctx, ds, store)).To(Succeed())

		_, err := awRepo.GetImage(ctx, h)
		Expect(err).ToNot(HaveOccurred())
		rc, err := store.Open(h, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("spares an unreferenced row recreated inside the grace window", func() {
		data := []byte("fresh-reacquired-bytes")
		h, _ := hashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())
		// Reacquisition refreshed created_at, so the row is unreferenced but too young to drop.
		Expect(awRepo.PutImage(ctx, &model.Artwork{Hash: h, Mime: "image/jpeg"})).To(Succeed())

		Expect(prune(ctx, ds, store)).To(Succeed())

		_, err := awRepo.GetImage(ctx, h)
		Expect(err).ToNot(HaveOccurred())
		rc, err := store.Open(h, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("spares an orphan file freshly touched by an overlapping acquisition", func() {
		data := []byte("racing-bytes")
		h, _ := hashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())
		Expect(awRepo.PutImage(ctx, &model.Artwork{Hash: h, Mime: "image/jpeg"})).To(Succeed())
		ageArtwork(h, time.Now().Add(-2*time.Hour))
		// The row is orphaned, but a concurrent acquisition just touched the file's mtime.

		Expect(prune(ctx, ds, store)).To(Succeed())

		rc, err := store.Open(h, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("sweeps store files that have no artwork row", func() {
		stray := []byte("no-row-bytes")
		h, _ := hashImage(bytes.NewReader(stray))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(stray))).To(Succeed())
		old := time.Now().Add(-2 * time.Hour)
		Expect(os.Chtimes(store.path(h, "image/jpeg"), old, old)).To(Succeed())

		Expect(prune(ctx, ds, store)).To(Succeed())

		_, err := store.Open(h, "image/jpeg")
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	It("sweeps an obsolete mime variant of a reacquired hash", func() {
		data := []byte("variant-bytes")
		h, _ := hashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/png", bytes.NewReader(data))).To(Succeed())
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())
		old := time.Now().Add(-2 * time.Hour)
		Expect(os.Chtimes(store.path(h, "image/png"), old, old)).To(Succeed())
		Expect(os.Chtimes(store.path(h, "image/jpeg"), old, old)).To(Succeed())
		// The row records the current mime; the .png file is a superseded variant.
		Expect(awRepo.PutImage(ctx, &model.Artwork{Hash: h, Mime: "image/jpeg"})).To(Succeed())

		Expect(prune(ctx, ds, store)).To(Succeed())

		_, err := store.Open(h, "image/png")
		Expect(os.IsNotExist(err)).To(BeTrue())
		rc, err := store.Open(h, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("reclaims the other orphan files when one of them cannot be removed", func() {
		tests.SkipOnWindows("uses Unix file permission bits")
		if os.Geteuid() == 0 {
			Skip("read-only dir cannot block root (e.g. tests in a container)")
		}
		old := time.Now().Add(-2 * time.Hour)

		blocked := []byte("blocked-bytes")
		hb, _ := hashImage(bytes.NewReader(blocked))
		Expect(store.Write(hb, "image/jpeg", bytes.NewReader(blocked))).To(Succeed())
		Expect(os.Chtimes(store.path(hb, "image/jpeg"), old, old)).To(Succeed())
		Expect(awRepo.PutImage(ctx, &model.Artwork{Hash: hb, Mime: "image/jpeg"})).To(Succeed())
		ageArtwork(hb, old)

		good := []byte("good-bytes")
		hg, _ := hashImage(bytes.NewReader(good))
		Expect(store.Write(hg, "image/jpeg", bytes.NewReader(good))).To(Succeed())
		Expect(os.Chtimes(store.path(hg, "image/jpeg"), old, old)).To(Succeed())
		Expect(awRepo.PutImage(ctx, &model.Artwork{Hash: hg, Mime: "image/jpeg"})).To(Succeed())
		ageArtwork(hg, old)

		// A read-only shard directory makes os.Remove fail (EACCES) for hb's file only.
		shardDir := filepath.Dir(store.path(hb, "image/jpeg"))
		Expect(shardDir).ToNot(Equal(filepath.Dir(store.path(hg, "image/jpeg"))), "fixtures must land in different shards")
		Expect(os.Chmod(shardDir, 0500)).To(Succeed())
		DeferCleanup(func() { _ = os.Chmod(shardDir, 0755) })

		Expect(prune(ctx, ds, store)).To(Succeed())

		_, err := awRepo.GetImage(ctx, hg)
		Expect(err).To(MatchError(model.ErrNotFound))
		_, err = store.Open(hg, "image/jpeg")
		Expect(os.IsNotExist(err)).To(BeTrue())

		// The row purge does not depend on file removal, so only the file survives.
		_, err = awRepo.GetImage(ctx, hb)
		Expect(err).To(MatchError(model.ErrNotFound))
		rc, err := store.Open(hb, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("never sweeps files on a transient DB error", func() {
		ds = persistencetest.Overlay(ds, func(ds model.DataStore) model.DataStore {
			return &artworkOverlay{DataStore: ds, artwork: &flakyGetArtworkRepo{ArtworkRepository: ds.Artwork()}}
		})

		data := []byte("live-bytes")
		h, _ := hashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())

		Expect(prune(ctx, ds, store)).ToNot(Succeed())

		rc, err := store.Open(h, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})
})

type artworkOverlay struct {
	model.DataStore
	artwork model.ArtworkRepository
}

func (o *artworkOverlay) Artwork() model.ArtworkRepository { return o.artwork }
