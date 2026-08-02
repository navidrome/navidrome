package artwork

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type flakyGetArtworkRepo struct {
	*tests.MockArtworkRepo
}

func (f *flakyGetArtworkRepo) GetMimeByHash() (map[string]string, error) {
	return nil, errors.New("db locked")
}

var _ = Describe("Prune", func() {
	var ds *tests.MockDataStore
	var store *ImageStore
	var awRepo *tests.MockArtworkRepo

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		awRepo = ds.Artwork(context.Background()).(*tests.MockArtworkRepo)
		store = NewImageStore(GinkgoT().TempDir())
	})

	// PutImage refreshes created_at like the SQL repo, so fixtures are aged directly.
	ageArtwork := func(h string, t time.Time) {
		a := awRepo.Data[h]
		a.CreatedAt = t
		awRepo.Data[h] = a
	}

	It("purges dangling item_artwork state for gone entities, summed across kinds", func() {
		Expect(awRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "al", ItemID: "gone-album", ImageType: model.ImageTypePrimary})).To(Succeed())
		Expect(awRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "ar", ItemID: "gone-artist", ImageType: model.ImageTypePrimary})).To(Succeed())
		Expect(awRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "ar", ItemID: "live-artist", ImageType: model.ImageTypePrimary})).To(Succeed())
		awRepo.ExistingIDs = map[string]map[string]bool{
			"al": {},
			"ar": {"live-artist": true},
		}

		Expect(prune(context.Background(), ds, store)).To(Succeed())

		_, err := awRepo.GetItemArtwork(model.KindAlbumArtwork, "gone-album", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound))
		_, err = awRepo.GetItemArtwork(model.KindArtistArtwork, "gone-artist", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound))
		_, err = awRepo.GetItemArtwork(model.KindArtistArtwork, "live-artist", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
	})

	It("purges dangling artwork_queue rows for gone entities", func() {
		queueRepo := tests.CreateMockArtworkQueueRepo()
		Expect(queueRepo.Enqueue(
			model.ArtworkQueueItem{ItemKind: "al", ItemID: "gone-album", ImageType: model.ImageTypePrimary},
			model.ArtworkQueueItem{ItemKind: "al", ItemID: "live-album", ImageType: model.ImageTypePrimary},
		)).To(Succeed())
		queueRepo.ExistingIDs = map[string]map[string]bool{"al": {"live-album": true}}
		ds.MockedArtworkQueue = queueRepo

		Expect(prune(context.Background(), ds, store)).To(Succeed())

		Expect(findQueued(queueRepo, "al", "gone-album")).To(BeNil())
		Expect(findQueued(queueRepo, "al", "live-album")).ToNot(BeNil())
	})

	It("deletes orphan rows and their store files, keeps referenced ones", func() {
		data := []byte("orphan-bytes")
		h, _ := hashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())
		old := time.Now().Add(-2 * time.Hour)
		Expect(os.Chtimes(store.path(h, "image/jpeg"), old, old)).To(Succeed())
		Expect(awRepo.PutImage(&model.Artwork{Hash: h, Mime: "image/jpeg"})).To(Succeed())
		ageArtwork(h, old)

		kept := []byte("kept-bytes")
		hk, _ := hashImage(bytes.NewReader(kept))
		Expect(store.Write(hk, "image/jpeg", bytes.NewReader(kept))).To(Succeed())
		Expect(awRepo.PutImage(&model.Artwork{Hash: hk, Mime: "image/jpeg"})).To(Succeed())

		Expect(prune(context.Background(), ds, store)).To(Succeed())

		_, err := awRepo.GetImage(h)
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
		Expect(awRepo.PutImage(&model.Artwork{Hash: h, Mime: "image/jpeg"})).To(Succeed())
		ageArtwork(h, time.Now().Add(-2*time.Hour))
		Expect(awRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "al", ItemID: "a1",
			ImageType: model.ImageTypePrimary, Hash: h, Source: "folder"})).To(Succeed())

		Expect(prune(context.Background(), ds, store)).To(Succeed())

		_, err := awRepo.GetImage(h)
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
		Expect(awRepo.PutImage(&model.Artwork{Hash: h, Mime: "image/jpeg"})).To(Succeed())

		Expect(prune(context.Background(), ds, store)).To(Succeed())

		_, err := awRepo.GetImage(h)
		Expect(err).ToNot(HaveOccurred())
		rc, err := store.Open(h, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("spares an orphan file freshly touched by an overlapping acquisition", func() {
		data := []byte("racing-bytes")
		h, _ := hashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())
		Expect(awRepo.PutImage(&model.Artwork{Hash: h, Mime: "image/jpeg"})).To(Succeed())
		ageArtwork(h, time.Now().Add(-2*time.Hour))
		// The row is orphaned, but a concurrent acquisition just touched the file's mtime.

		Expect(prune(context.Background(), ds, store)).To(Succeed())

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

		Expect(prune(context.Background(), ds, store)).To(Succeed())

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
		Expect(awRepo.PutImage(&model.Artwork{Hash: h, Mime: "image/jpeg"})).To(Succeed())

		Expect(prune(context.Background(), ds, store)).To(Succeed())

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
		Expect(awRepo.PutImage(&model.Artwork{Hash: hb, Mime: "image/jpeg"})).To(Succeed())
		ageArtwork(hb, old)

		good := []byte("good-bytes")
		hg, _ := hashImage(bytes.NewReader(good))
		Expect(store.Write(hg, "image/jpeg", bytes.NewReader(good))).To(Succeed())
		Expect(os.Chtimes(store.path(hg, "image/jpeg"), old, old)).To(Succeed())
		Expect(awRepo.PutImage(&model.Artwork{Hash: hg, Mime: "image/jpeg"})).To(Succeed())
		ageArtwork(hg, old)

		// A read-only shard directory makes os.Remove fail (EACCES) for hb's file only.
		shardDir := filepath.Dir(store.path(hb, "image/jpeg"))
		Expect(os.Chmod(shardDir, 0500)).To(Succeed())
		DeferCleanup(func() { _ = os.Chmod(shardDir, 0755) })

		Expect(prune(context.Background(), ds, store)).To(Succeed())

		_, err := awRepo.GetImage(hg)
		Expect(err).To(MatchError(model.ErrNotFound))
		_, err = store.Open(hg, "image/jpeg")
		Expect(os.IsNotExist(err)).To(BeTrue())

		// The row purge does not depend on file removal, so only the file survives.
		_, err = awRepo.GetImage(hb)
		Expect(err).To(MatchError(model.ErrNotFound))
		rc, err := store.Open(hb, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("never sweeps files on a transient DB error", func() {
		ds.MockedArtwork = &flakyGetArtworkRepo{MockArtworkRepo: tests.CreateMockArtworkRepo()}

		data := []byte("live-bytes")
		h, _ := hashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())

		Expect(prune(context.Background(), ds, store)).ToNot(Succeed())

		rc, err := store.Open(h, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})
})
