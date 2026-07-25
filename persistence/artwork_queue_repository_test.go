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
		DeferCleanup(clearArtworkTables)
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

	It("EnqueueBump raises priority without resetting a backing-off row's retry_at", func() {
		Expect(repo.Enqueue(item("al", "b1", model.ArtworkPriorityScan))).To(Succeed())
		// Push retry_at into the future so the row is backing off and hidden from dequeue.
		Expect(repo.MarkFailed("al", "b1", model.ImageTypePrimary, time.Now().Add(time.Hour))).To(Succeed())
		Expect(repo.DequeueBatch(10)).To(BeEmpty())

		// A request-triggered bump raises priority but must leave the backoff intact.
		Expect(repo.EnqueueBump(item("al", "b1", model.ArtworkPriorityBump))).To(Succeed())
		Expect(repo.DequeueBatch(10)).To(BeEmpty(), "bump must not reset retry_at")

		// Enqueue (scan/manual), by contrast, resets retry_at and makes it eligible now.
		Expect(repo.Enqueue(item("al", "b1", model.ArtworkPriorityScan))).To(Succeed())
		got, _ := repo.DequeueBatch(10)
		Expect(got).To(HaveLen(1))
		Expect(got[0].Priority).To(Equal(model.ArtworkPriorityBump), "bump's higher priority is preserved")
	})

	It("EnqueueBump inserts a brand-new row eligible immediately", func() {
		Expect(repo.EnqueueBump(item("ar", "n1", model.ArtworkPriorityBump))).To(Succeed())
		got, _ := repo.DequeueBatch(10)
		Expect(got).To(HaveLen(1))
		Expect(got[0].ItemID).To(Equal("n1"))
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

	It("MarkFailedIfUnchanged applies backoff only while retry_at is unchanged", func() {
		Expect(repo.Enqueue(item("al", "m1", model.ArtworkPriorityScan))).To(Succeed())
		// Anchor retry_at in the past (attempts -> 1) so it can never collide with the re-enqueue's now.
		Expect(repo.MarkFailed("al", "m1", model.ImageTypePrimary, time.Now().Add(-time.Hour))).To(Succeed())
		got, err := repo.DequeueBatch(10)
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(HaveLen(1))
		original := got[0].RetryAt

		// A concurrent scan re-enqueues, resetting retry_at to now.
		Expect(repo.Enqueue(item("al", "m1", model.ArtworkPriorityScan))).To(Succeed())

		// Failing with the stale retry_at is a no-op: the re-enqueued row keeps its fresh state.
		future := time.Now().Add(48 * time.Hour)
		Expect(repo.MarkFailedIfUnchanged("al", "m1", model.ImageTypePrimary, original, future)).To(Succeed())
		got, _ = repo.DequeueBatch(10)
		Expect(got).To(HaveLen(1), "the fresh re-enqueue stays immediately eligible")
		Expect(got[0].Attempts).To(Equal(1), "the stale failure must not bump attempts")
		current := got[0].RetryAt

		// Failing with the current retry_at applies the backoff and bumps attempts.
		Expect(repo.MarkFailedIfUnchanged("al", "m1", model.ImageTypePrimary, current, future)).To(Succeed())
		got, _ = repo.DequeueBatch(10)
		Expect(got).To(BeEmpty(), "backed-off row is hidden until the future retry_at")
		all, _ := repo.Count()
		Expect(all).To(Equal(int64(1)))
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
			item("mf", songDayInALife.ID, model.ArtworkPriorityScan),
			item("mf", "no-such-mediafile", model.ArtworkPriorityScan),
		)).To(Succeed())

		purged, err := repo.PurgeDangling()
		Expect(err).ToNot(HaveOccurred())
		Expect(purged).To(Equal(int64(5)))

		got, _ := repo.DequeueBatch(100)
		ids := make([]string, 0, len(got))
		for _, it := range got {
			ids = append(ids, it.ItemID)
		}
		Expect(ids).To(ConsistOf(albumSgtPeppers.ID, artistKraftwerk.ID, plsBest.ID, radioWithHomePage.ID, songDayInALife.ID))
	})

	It("enqueues stale absent states for recheck", func() {
		awRepo := NewArtworkRepository(context.Background(), GetDBXBuilder())
		old := time.Now().Add(-48 * time.Hour)
		Expect(awRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "ar", ItemID: "stale1", ImageType: model.ImageTypePrimary, Hash: "", AttemptedAt: old})).To(Succeed())
		Expect(awRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "ar", ItemID: "fresh1", ImageType: model.ImageTypePrimary, Hash: "", AttemptedAt: time.Now()})).To(Succeed())
		Expect(awRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "ar", ItemID: "found1", ImageType: model.ImageTypePrimary, Hash: "hX", AttemptedAt: old})).To(Succeed())

		n, err := repo.EnqueueStaleAbsent(model.KindArtistArtwork, time.Now().Add(-24*time.Hour))
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(int64(1)))

		items, err := repo.DequeueBatch(10)
		Expect(err).ToNot(HaveOccurred())
		Expect(items).To(HaveLen(1))
		Expect(items[0].ItemID).To(Equal("stale1"))
		Expect(items[0].Priority).To(Equal(model.ArtworkPriorityRecheck))
	})

	It("enqueues entities that have no item_artwork row at all", func() {
		awRepo := NewArtworkRepository(context.Background(), GetDBXBuilder())
		// albumSgtPeppers has a resolved row and albumAbbeyRoad an absent row: both already
		// processed, so neither should be enqueued as "missing".
		Expect(awRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "al", ItemID: albumSgtPeppers.ID, ImageType: model.ImageTypePrimary, Hash: "hX", AttemptedAt: time.Now()})).To(Succeed())
		Expect(awRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "al", ItemID: albumAbbeyRoad.ID, ImageType: model.ImageTypePrimary, Hash: "", AttemptedAt: time.Now()})).To(Succeed())

		n, err := repo.EnqueueMissing(model.KindAlbumArtwork)
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(BeNumerically(">=", 1))

		got, err := repo.DequeueBatch(1000)
		Expect(err).ToNot(HaveOccurred())
		ids := make([]string, 0, len(got))
		for _, it := range got {
			Expect(it.ItemKind).To(Equal("al"))
			Expect(it.Priority).To(Equal(model.ArtworkPriorityRecheck))
			ids = append(ids, it.ItemID)
		}
		Expect(ids).To(ContainElement(albumRadioactivity.ID), "an album with no row must be enqueued")
		Expect(ids).ToNot(ContainElement(albumSgtPeppers.ID), "a resolved album must not be re-enqueued")
		Expect(ids).ToNot(ContainElement(albumAbbeyRoad.ID), "an absent-state album must not be enqueued as missing")
	})

	It("does not disturb an already-queued entity when enqueueing missing rows", func() {
		Expect(repo.Enqueue(item("al", albumRadioactivity.ID, model.ArtworkPriorityBump))).To(Succeed())

		_, err := repo.EnqueueMissing(model.KindAlbumArtwork)
		Expect(err).ToNot(HaveOccurred())

		got, _ := repo.DequeueBatch(1000)
		var count int
		for _, it := range got {
			if it.ItemID == albumRadioactivity.ID {
				count++
				Expect(it.Priority).To(Equal(model.ArtworkPriorityBump), "existing bump priority must survive")
			}
		}
		Expect(count).To(Equal(1), "the already-queued row must not be duplicated")
	})
})
