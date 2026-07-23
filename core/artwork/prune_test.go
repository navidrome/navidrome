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

func (f *flakyGetArtworkRepo) GetAllMimes() (map[string]string, error) {
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

		Expect(Prune(context.Background(), ds, store)).To(Succeed())

		_, err := awRepo.GetItemArtwork("al", "gone-album", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound))
		_, err = awRepo.GetItemArtwork("ar", "gone-artist", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound))
		_, err = awRepo.GetItemArtwork("ar", "live-artist", model.ImageTypePrimary)
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

		Expect(Prune(context.Background(), ds, store)).To(Succeed())

		Expect(findQueued(queueRepo, "al", "gone-album")).To(BeNil())
		Expect(findQueued(queueRepo, "al", "live-album")).ToNot(BeNil())
	})

	It("deletes orphan rows and their store files, keeps referenced ones", func() {
		data := []byte("orphan-bytes")
		h, _ := HashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())
		old := time.Now().Add(-2 * time.Hour)
		Expect(os.Chtimes(store.path(h, "image/jpeg"), old, old)).To(Succeed())
		Expect(awRepo.PutImage(&model.Artwork{Hash: h, Mime: "image/jpeg"})).To(Succeed())
		ageArtwork(h, old)
		awRepo.OrphanHashes = []string{h}

		kept := []byte("kept-bytes")
		hk, _ := HashImage(bytes.NewReader(kept))
		Expect(store.Write(hk, "image/jpeg", bytes.NewReader(kept))).To(Succeed())
		Expect(awRepo.PutImage(&model.Artwork{Hash: hk, Mime: "image/jpeg"})).To(Succeed())

		Expect(Prune(context.Background(), ds, store)).To(Succeed())

		_, err := awRepo.GetImage(h)
		Expect(err).To(MatchError(model.ErrNotFound))
		_, err = store.Open(h, "image/jpeg")
		Expect(os.IsNotExist(err)).To(BeTrue())
		rc, err := store.Open(hk, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("spares a candidate reacquired between snapshot and delete", func() {
		data := []byte("reacquired-bytes")
		h, _ := HashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())
		Expect(awRepo.PutImage(&model.Artwork{Hash: h, Mime: "image/jpeg"})).To(Succeed())
		ageArtwork(h, time.Now().Add(-2*time.Hour))
		awRepo.OrphanHashes = []string{h}
		// Reacquisition: an item now references the hash the snapshot flagged as orphan.
		Expect(awRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "al", ItemID: "a1",
			ImageType: model.ImageTypePrimary, Hash: h, Source: "folder"})).To(Succeed())

		Expect(Prune(context.Background(), ds, store)).To(Succeed())

		_, err := awRepo.GetImage(h)
		Expect(err).ToNot(HaveOccurred())
		rc, err := store.Open(h, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("spares a candidate whose row was freshly recreated (created_at inside the grace window)", func() {
		data := []byte("fresh-reacquired-bytes")
		h, _ := HashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())
		// Reacquisition refreshed created_at after the snapshot; still unreferenced.
		Expect(awRepo.PutImage(&model.Artwork{Hash: h, Mime: "image/jpeg"})).To(Succeed())
		awRepo.OrphanHashes = []string{h}

		Expect(Prune(context.Background(), ds, store)).To(Succeed())

		_, err := awRepo.GetImage(h)
		Expect(err).ToNot(HaveOccurred())
		rc, err := store.Open(h, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("spares an orphan file freshly touched by an overlapping acquisition", func() {
		data := []byte("racing-bytes")
		h, _ := HashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())
		Expect(awRepo.PutImage(&model.Artwork{Hash: h, Mime: "image/jpeg"})).To(Succeed())
		ageArtwork(h, time.Now().Add(-2*time.Hour))
		awRepo.OrphanHashes = []string{h}
		// The row is legitimately orphaned, but a concurrent acquisition just touched the
		// file's mtime (duplicate Write) and is about to commit a row referencing it.

		Expect(Prune(context.Background(), ds, store)).To(Succeed())

		rc, err := store.Open(h, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("sweeps store files that have no artwork row", func() {
		stray := []byte("no-row-bytes")
		h, _ := HashImage(bytes.NewReader(stray))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(stray))).To(Succeed())
		old := time.Now().Add(-2 * time.Hour)
		Expect(os.Chtimes(store.path(h, "image/jpeg"), old, old)).To(Succeed())

		Expect(Prune(context.Background(), ds, store)).To(Succeed())

		_, err := store.Open(h, "image/jpeg")
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	It("sweeps an obsolete mime variant of a reacquired hash", func() {
		data := []byte("variant-bytes")
		h, _ := HashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/png", bytes.NewReader(data))).To(Succeed())
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())
		old := time.Now().Add(-2 * time.Hour)
		Expect(os.Chtimes(store.path(h, "image/png"), old, old)).To(Succeed())
		Expect(os.Chtimes(store.path(h, "image/jpeg"), old, old)).To(Succeed())
		// The row records the current mime; the .png file is a superseded variant.
		Expect(awRepo.PutImage(&model.Artwork{Hash: h, Mime: "image/jpeg"})).To(Succeed())

		Expect(Prune(context.Background(), ds, store)).To(Succeed())

		_, err := store.Open(h, "image/png")
		Expect(os.IsNotExist(err)).To(BeTrue())
		rc, err := store.Open(h, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("warns and continues past a store.Remove failure instead of aborting the loop", func() {
		tests.SkipOnWindows("uses Unix file permission bits")
		if os.Geteuid() == 0 {
			Skip("read-only dir cannot block root (e.g. tests in a container)")
		}
		old := time.Now().Add(-2 * time.Hour)

		blocked := []byte("blocked-bytes")
		hb, _ := HashImage(bytes.NewReader(blocked))
		Expect(store.Write(hb, "image/jpeg", bytes.NewReader(blocked))).To(Succeed())
		Expect(os.Chtimes(store.path(hb, "image/jpeg"), old, old)).To(Succeed())
		Expect(awRepo.PutImage(&model.Artwork{Hash: hb, Mime: "image/jpeg"})).To(Succeed())
		ageArtwork(hb, old)

		good := []byte("good-bytes")
		hg, _ := HashImage(bytes.NewReader(good))
		Expect(store.Write(hg, "image/jpeg", bytes.NewReader(good))).To(Succeed())
		Expect(os.Chtimes(store.path(hg, "image/jpeg"), old, old)).To(Succeed())
		Expect(awRepo.PutImage(&model.Artwork{Hash: hg, Mime: "image/jpeg"})).To(Succeed())
		ageArtwork(hg, old)

		// A read-only shard directory makes os.Remove fail (EACCES) for hb's file only.
		shardDir := filepath.Dir(store.path(hb, "image/jpeg"))
		Expect(os.Chmod(shardDir, 0500)).To(Succeed())
		DeferCleanup(func() { _ = os.Chmod(shardDir, 0755) })

		// hb (blocked) is processed first: if store.Remove's failure aborted the loop
		// instead of warning and continuing, hg would never be reached.
		awRepo.OrphanHashes = []string{hb, hg}

		// Prune still errors: Sweep independently revisits hb's leftover file and,
		// unlike the loop below, has no warn-and-continue fallback of its own.
		err := Prune(context.Background(), ds, store)
		Expect(err).To(HaveOccurred())

		// hg: reached and fully pruned despite being queued after the failing hb -
		// proof the loop didn't return/break on the first Remove error.
		_, err = awRepo.GetImage(hg)
		Expect(err).To(MatchError(model.ErrNotFound))
		_, err = store.Open(hg, "image/jpeg")
		Expect(os.IsNotExist(err)).To(BeTrue())

		// hb: row still purged (DeleteOrphans doesn't depend on file removal), but the
		// file itself survives since store.Remove failed and only warned.
		_, err = awRepo.GetImage(hb)
		Expect(err).To(MatchError(model.ErrNotFound))
		rc, err := store.Open(hb, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})

	It("never sweeps files on a transient DB error", func() {
		ds.MockedArtwork = &flakyGetArtworkRepo{MockArtworkRepo: tests.CreateMockArtworkRepo()}

		data := []byte("live-bytes")
		h, _ := HashImage(bytes.NewReader(data))
		Expect(store.Write(h, "image/jpeg", bytes.NewReader(data))).To(Succeed())

		Expect(Prune(context.Background(), ds, store)).ToNot(Succeed())

		rc, err := store.Open(h, "image/jpeg")
		Expect(err).ToNot(HaveOccurred())
		rc.Close()
	})
})
