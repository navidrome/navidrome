package persistence

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ArtworkQueueRepository", func() {
	var repo model.ArtworkQueueRepository

	item := func(kind, id string, prio int) model.ArtworkQueueItem {
		return model.ArtworkQueueItem{ItemKind: kind, ItemID: id,
			ImageType: model.ImageTypePrimary, Priority: prio}
	}

	BeforeEach(func() {
		clearArtworkTables()
		repo = NewArtworkQueueRepository(context.Background(), GetDBXBuilder())
	})

	It("enqueues and dequeues by priority then FIFO", func() {
		Expect(repo.Enqueue(item("al", "low", model.ArtworkPriorityBackfill))).To(Succeed())
		Expect(repo.Enqueue(item("ar", "high", model.ArtworkPriorityBump))).To(Succeed())

		got, err := repo.DequeueBatch(10)
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(HaveLen(2))
		Expect(got[0].ItemID).To(Equal("high"))
	})

	It("keeps the higher priority on duplicate enqueue", func() {
		Expect(repo.Enqueue(item("al", "a1", model.ArtworkPriorityBump))).To(Succeed())
		Expect(repo.Enqueue(item("al", "a1", model.ArtworkPriorityBackfill))).To(Succeed())
		got, _ := repo.DequeueBatch(10)
		Expect(got).To(HaveLen(1))
		Expect(got[0].Priority).To(Equal(model.ArtworkPriorityBump))
	})

	It("hides failed items until retry_at", func() {
		Expect(repo.Enqueue(item("al", "f1", model.ArtworkPriorityScan))).To(Succeed())
		Expect(repo.MarkFailed("al", "f1", model.ImageTypePrimary, time.Now().Add(time.Hour))).To(Succeed())

		got, err := repo.DequeueBatch(10)
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(BeEmpty())

		Expect(repo.MarkFailed("al", "f1", model.ImageTypePrimary, time.Now().Add(-time.Minute))).To(Succeed())
		got, _ = repo.DequeueBatch(10)
		Expect(got).To(HaveLen(1))
		Expect(got[0].Attempts).To(Equal(2))
	})

	It("deletes on completion and counts", func() {
		Expect(repo.Enqueue(item("al", "c1", 0))).To(Succeed())
		n, _ := repo.Count()
		Expect(n).To(Equal(int64(1)))
		Expect(repo.Delete("al", "c1", model.ImageTypePrimary)).To(Succeed())
		n, _ = repo.Count()
		Expect(n).To(BeZero())
	})

	It("DeleteIfUnchanged deletes only while retry_at is unchanged", func() {
		Expect(repo.Enqueue(item("al", "d1", model.ArtworkPriorityScan))).To(Succeed())
		// Anchor retry_at in the past so it can never collide with the re-enqueue's now.
		Expect(repo.MarkFailed("al", "d1", model.ImageTypePrimary, time.Now().Add(-time.Hour))).To(Succeed())
		got, err := repo.DequeueBatch(10)
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(HaveLen(1))
		original := got[0].RetryAt

		// A concurrent scan re-enqueues, resetting retry_at to now.
		Expect(repo.Enqueue(item("al", "d1", model.ArtworkPriorityScan))).To(Succeed())

		// Deleting with the stale retry_at is a no-op: the re-enqueued row survives.
		Expect(repo.DeleteIfUnchanged("al", "d1", model.ImageTypePrimary, original)).To(Succeed())
		n, _ := repo.Count()
		Expect(n).To(Equal(int64(1)))

		// Deleting with the current retry_at removes it.
		got, _ = repo.DequeueBatch(10)
		Expect(got).To(HaveLen(1))
		Expect(repo.DeleteIfUnchanged("al", "d1", model.ImageTypePrimary, got[0].RetryAt)).To(Succeed())
		n, _ = repo.Count()
		Expect(n).To(BeZero())
	})

	It("purges queue rows whose entity no longer exists, per kind", func() {
		Expect(repo.Enqueue(
			item("al", albumSgtPeppers.ID, model.ArtworkPriorityScan),
			item("al", "no-such-album", model.ArtworkPriorityScan),
			item("ar", artistKraftwerk.ID, model.ArtworkPriorityScan),
			item("ar", "no-such-artist", model.ArtworkPriorityScan),
			item("pl", plsBest.ID, model.ArtworkPriorityScan),
			item("pl", "no-such-playlist", model.ArtworkPriorityScan),
			item("ra", radioWithHomePage.ID, model.ArtworkPriorityScan),
			item("ra", "no-such-radio", model.ArtworkPriorityScan),
		)).To(Succeed())

		purged, err := repo.PurgeDangling()
		Expect(err).ToNot(HaveOccurred())
		Expect(purged).To(Equal(int64(4)))

		got, _ := repo.DequeueBatch(100)
		ids := make([]string, 0, len(got))
		for _, it := range got {
			ids = append(ids, it.ItemID)
		}
		Expect(ids).To(ConsistOf(albumSgtPeppers.ID, artistKraftwerk.ID, plsBest.ID, radioWithHomePage.ID))
	})

	It("enqueues stale absent states for recheck", func() {
		awRepo := NewArtworkRepository(context.Background(), GetDBXBuilder())
		old := time.Now().Add(-48 * time.Hour)
		Expect(awRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "ar", ItemID: "stale1", ImageType: model.ImageTypePrimary, Hash: "", AttemptedAt: old})).To(Succeed())
		Expect(awRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "ar", ItemID: "fresh1", ImageType: model.ImageTypePrimary, Hash: "", AttemptedAt: time.Now()})).To(Succeed())
		Expect(awRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "ar", ItemID: "found1", ImageType: model.ImageTypePrimary, Hash: "hX", AttemptedAt: old})).To(Succeed())

		n, err := repo.EnqueueStaleAbsent("ar", time.Now().Add(-24*time.Hour))
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(int64(1)))

		items, err := repo.DequeueBatch(10)
		Expect(err).ToNot(HaveOccurred())
		Expect(items).To(HaveLen(1))
		Expect(items[0].ItemID).To(Equal("stale1"))
		Expect(items[0].Priority).To(Equal(model.ArtworkPriorityRecheck))
	})
})
