package persistence

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pocketbase/dbx"
)

var _ = Describe("ArtworkQueueRepository", func() {
	var repo model.ArtworkQueueRepository

	item := func(kind, id string, prio int) model.ArtworkQueueItem {
		return model.ArtworkQueueItem{ItemKind: kind, ItemID: id,
			ImageType: model.ImageTypePrimary, Priority: prio}
	}

	// Writes the post-failure state directly: the production path, MarkFailedIfUnchanged, needs
	// a retry_at only a dequeue can hand it.
	backOff := func(kind, id string, retryAt time.Time) {
		GinkgoHelper()
		r := repo.(*artworkQueueRepository)
		_, err := r.executeSQL(squirrel.Update(r.tableName).
			Set("attempts", squirrel.Expr("attempts + 1")).
			Set("retry_at", retryAt).
			Where(squirrel.Eq{"item_kind": kind, "item_id": id, "image_type": model.ImageTypePrimary}))
		Expect(err).ToNot(HaveOccurred())
	}

	remove := func(kind, id string) {
		GinkgoHelper()
		r := repo.(*artworkQueueRepository)
		Expect(r.delete(squirrel.Eq{"item_kind": kind, "item_id": id, "image_type": model.ImageTypePrimary})).To(Succeed())
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

	It("Get returns a queued row, including one still backing off", func() {
		Expect(repo.Enqueue(item("ar", "g1", model.ArtworkPriorityScan))).To(Succeed())
		backOff("ar", "g1", time.Now().Add(time.Hour))

		got, err := repo.Get(model.KindArtistArtwork, "g1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Priority).To(Equal(model.ArtworkPriorityScan))
		Expect(got.Attempts).To(Equal(1))
		Expect(got.RetryAt).To(BeTemporally(">", time.Now()))
	})

	It("Get reports ErrNotFound when the item is not queued", func() {
		_, err := repo.Get(model.KindArtistArtwork, "nope", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("keeps the higher priority on duplicate enqueue", func() {
		Expect(repo.Enqueue(item("al", "a1", model.ArtworkPriorityBump))).To(Succeed())
		Expect(repo.Enqueue(item("al", "a1", model.ArtworkPriorityBackfill))).To(Succeed())
		got, _ := repo.DequeueBatch(10)
		Expect(got).To(HaveLen(1))
		Expect(got[0].Priority).To(Equal(model.ArtworkPriorityBump))
	})

	It("EnqueuePreservingBackoff raises priority without resetting a backing-off row's retry_at", func() {
		Expect(repo.Enqueue(item("al", "b1", model.ArtworkPriorityScan))).To(Succeed())
		backOff("al", "b1", time.Now().Add(time.Hour))
		Expect(repo.DequeueBatch(10)).To(BeEmpty())

		Expect(repo.EnqueuePreservingBackoff(item("al", "b1", model.ArtworkPriorityBump))).To(Succeed())
		Expect(repo.DequeueBatch(10)).To(BeEmpty(), "bump must not reset retry_at")

		// Enqueue (scan/manual), by contrast, resets retry_at and makes it eligible now.
		Expect(repo.Enqueue(item("al", "b1", model.ArtworkPriorityScan))).To(Succeed())
		got, _ := repo.DequeueBatch(10)
		Expect(got).To(HaveLen(1))
		Expect(got[0].Priority).To(Equal(model.ArtworkPriorityBump), "bump's higher priority is preserved")
	})

	It("EnqueuePreservingBackoff inserts a brand-new row eligible immediately", func() {
		Expect(repo.EnqueuePreservingBackoff(item("ar", "n1", model.ArtworkPriorityBump))).To(Succeed())
		got, _ := repo.DequeueBatch(10)
		Expect(got).To(HaveLen(1))
		Expect(got[0].ItemID).To(Equal("n1"))
	})

	It("hides failed items until retry_at", func() {
		Expect(repo.Enqueue(item("al", "f1", model.ArtworkPriorityScan))).To(Succeed())
		backOff("al", "f1", time.Now().Add(time.Hour))

		got, err := repo.DequeueBatch(10)
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(BeEmpty())

		backOff("al", "f1", time.Now().Add(-time.Minute))
		got, _ = repo.DequeueBatch(10)
		Expect(got).To(HaveLen(1))
		Expect(got[0].Attempts).To(Equal(2))
	})

	It("MarkFailedIfUnchanged applies backoff only while retry_at is unchanged", func() {
		Expect(repo.Enqueue(item("al", "m1", model.ArtworkPriorityScan))).To(Succeed())
		// Anchor retry_at in the past so it can never collide with the re-enqueue's now.
		backOff("al", "m1", time.Now().Add(-time.Hour))
		got, err := repo.DequeueBatch(10)
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(HaveLen(1))
		original := got[0].RetryAt

		// A concurrent scan re-enqueues, resetting retry_at to now.
		Expect(repo.Enqueue(item("al", "m1", model.ArtworkPriorityScan))).To(Succeed())

		future := time.Now().Add(48 * time.Hour)
		Expect(repo.MarkFailedIfUnchanged("al", "m1", model.ImageTypePrimary, original, future, "[]")).To(Succeed())
		got, _ = repo.DequeueBatch(10)
		Expect(got).To(HaveLen(1), "the fresh re-enqueue stays immediately eligible")
		Expect(got[0].Attempts).To(BeZero(), "re-enqueue clears attempts, and the stale failure must not bump them")
		current := got[0].RetryAt

		Expect(repo.MarkFailedIfUnchanged("al", "m1", model.ImageTypePrimary, current, future, `[{"c":"read","o":"error"}]`)).To(Succeed())
		got, _ = repo.DequeueBatch(10)
		Expect(got).To(BeEmpty(), "backed-off row is hidden until the future retry_at")
		all, _ := repo.Count()
		Expect(all).To(Equal(int64(1)))
	})

	It("Enqueue clears a prior lifecycle's failure trace; EnqueuePreservingBackoff keeps it", func() {
		// Fail an attempt so the queue row carries a failure trace.
		Expect(repo.Enqueue(item("al", "t1", model.ArtworkPriorityScan))).To(Succeed())
		backOff("al", "t1", time.Now().Add(-time.Hour))
		got, _ := repo.DequeueBatch(10)
		Expect(got).To(HaveLen(1))
		future := time.Now().Add(48 * time.Hour)
		Expect(repo.MarkFailedIfUnchanged("al", "t1", model.ImageTypePrimary, got[0].RetryAt, future, `[{"c":"read","o":"error"}]`)).To(Succeed())

		// A continuation of the same lifecycle must retain the trace.
		Expect(repo.EnqueuePreservingBackoff(item("al", "t1", model.ArtworkPriorityBump))).To(Succeed())
		kept, err := repo.Get(model.KindAlbumArtwork, "t1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(kept.Trace).To(Equal(`[{"c":"read","o":"error"}]`))

		// A fresh Enqueue resets attempts to 0, so the stale failure trace must be cleared with it.
		Expect(repo.Enqueue(item("al", "t1", model.ArtworkPriorityScan))).To(Succeed())
		fresh, err := repo.Get(model.KindAlbumArtwork, "t1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(fresh.Attempts).To(BeZero())
		Expect(fresh.Trace).To(Equal("[]"), "a fresh lifecycle has no last-attempt trace")
	})

	It("Enqueue restarts the retry budget an existing row had spent", func() {
		Expect(repo.Enqueue(item("al", "e1", model.ArtworkPriorityScan))).To(Succeed())
		backOff("al", "e1", time.Now().Add(-time.Hour))
		stale := time.Now().Add(-48 * time.Hour)
		_, err := GetDBXBuilder().NewQuery("UPDATE artwork_queue SET enqueued_at = {:t} WHERE item_id = 'e1'").
			Bind(dbx.Params{"t": stale}).Execute()
		Expect(err).ToNot(HaveOccurred())

		Expect(repo.Enqueue(item("al", "e1", model.ArtworkPriorityScan))).To(Succeed())

		got, err := repo.DequeueBatch(10)
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(HaveLen(1))
		Expect(got[0].EnqueuedAt).To(BeTemporally("~", time.Now(), time.Minute),
			"a re-request must not inherit a spent 12h window and give up on its first attempt")
		Expect(got[0].Attempts).To(BeZero())
	})

	It("deletes on completion and counts", func() {
		Expect(repo.Enqueue(item("al", "c1", 0))).To(Succeed())
		n, _ := repo.Count()
		Expect(n).To(Equal(int64(1)))
		remove("al", "c1")
		n, _ = repo.Count()
		Expect(n).To(BeZero())
	})

	It("DeleteIfUnchanged deletes only while retry_at is unchanged", func() {
		Expect(repo.Enqueue(item("al", "d1", model.ArtworkPriorityScan))).To(Succeed())
		// Anchor retry_at in the past so it can never collide with the re-enqueue's now.
		backOff("al", "d1", time.Now().Add(-time.Hour))
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
		ids := slice.Map(got, func(it model.ArtworkQueueItem) string { return it.ItemID })
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
		Expect(awRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "al", ItemID: albumSgtPeppers.ID, ImageType: model.ImageTypePrimary, Hash: "hX", AttemptedAt: time.Now()})).To(Succeed())
		Expect(awRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "al", ItemID: albumAbbeyRoad.ID, ImageType: model.ImageTypePrimary, Hash: "", AttemptedAt: time.Now()})).To(Succeed())

		n, err := repo.EnqueueAllMissing(model.KindAlbumArtwork, model.ArtworkPriorityRecheck)
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

	It("EnqueueIfMissing skips items that already have an item_artwork row", func() {
		awRepo := NewArtworkRepository(context.Background(), GetDBXBuilder())
		Expect(awRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "al", ItemID: "resolved", ImageType: model.ImageTypePrimary, Hash: "hX", AttemptedAt: time.Now()})).To(Succeed())
		Expect(awRepo.PutItemArtwork(&model.ItemArtwork{ItemKind: "al", ItemID: "absent", ImageType: model.ImageTypePrimary, Hash: "", AttemptedAt: time.Now()})).To(Succeed())

		Expect(repo.EnqueueIfMissing(
			item("al", "resolved", model.ArtworkPriorityScan),
			item("al", "absent", model.ArtworkPriorityScan),
			item("al", "brandnew", model.ArtworkPriorityScan),
		)).To(Succeed())

		got, err := repo.DequeueBatch(100)
		Expect(err).ToNot(HaveOccurred())
		ids := slice.Map(got, func(it model.ArtworkQueueItem) string { return it.ItemID })
		Expect(ids).To(ConsistOf("brandnew"), "only an item with no state row may be enqueued")
	})

	It("EnqueueIfMissing leaves an already-queued row untouched", func() {
		Expect(repo.Enqueue(item("al", "queued", model.ArtworkPriorityBump))).To(Succeed())

		Expect(repo.EnqueueIfMissing(item("al", "queued", model.ArtworkPriorityScan))).To(Succeed())

		got, _ := repo.DequeueBatch(100)
		Expect(got).To(HaveLen(1))
		Expect(got[0].Priority).To(Equal(model.ArtworkPriorityBump), "the existing priority must survive")
	})

	Describe("EnqueueBySource", func() {
		BeforeEach(func() {
			artRepo := NewArtworkRepository(context.Background(), GetDBXBuilder())
			for _, ia := range []model.ItemArtwork{
				{ItemKind: "ar", ItemID: "ar1", ImageType: model.ImageTypePrimary, Hash: "h1", Source: "external:deezer"},
				{ItemKind: "ar", ItemID: "ar2", ImageType: model.ImageTypePrimary, Hash: "h2", Source: "external:lastfm"},
				{ItemKind: "ar", ItemID: "ar3", ImageType: model.ImageTypePrimary, Hash: "", Source: ""},
				{ItemKind: "al", ItemID: "al1", ImageType: model.ImageTypePrimary, Hash: "h4", Source: "external:deezer"},
			} {
				Expect(artRepo.PutItemArtwork(&ia)).To(Succeed())
			}
		})

		It("enqueues only the matching source within the kind", func() {
			n, err := repo.EnqueueBySource(model.KindArtistArtwork, []string{"external:deezer"}, model.ArtworkPriorityRecheck)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(int64(1)), "al1 is a different kind and must not be touched")

			got, err := repo.DequeueBatch(10)
			Expect(err).ToNot(HaveOccurred())
			Expect(slice.Map(got, func(it model.ArtworkQueueItem) string { return it.ItemID })).To(ConsistOf("ar1"))
		})

		It("treats the empty source as absent", func() {
			n, err := repo.EnqueueBySource(model.KindArtistArtwork, []string{""}, model.ArtworkPriorityRecheck)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(int64(1)))

			got, _ := repo.DequeueBatch(10)
			Expect(slice.Map(got, func(it model.ArtworkQueueItem) string { return it.ItemID })).To(ConsistOf("ar3"))
		})

		It("enqueues every source when none is given", func() {
			n, err := repo.EnqueueBySource(model.KindArtistArtwork, nil, model.ArtworkPriorityRecheck)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(int64(3)))
		})

		It("leaves the current artwork state in place", func() {
			_, err := repo.EnqueueBySource(model.KindArtistArtwork, []string{"external:deezer"}, model.ArtworkPriorityRecheck)
			Expect(err).ToNot(HaveOccurred())

			artRepo := NewArtworkRepository(context.Background(), GetDBXBuilder())
			ia, err := artRepo.GetItemArtwork(model.KindArtistArtwork, "ar1", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(ia.Hash).To(Equal("h1"), "the current image must survive until it is replaced")
			Expect(ia.Source).To(Equal("external:deezer"))
		})

		It("does not disturb an already-queued row", func() {
			Expect(repo.Enqueue(item("ar", "ar1", model.ArtworkPriorityBump))).To(Succeed())

			n, err := repo.EnqueueBySource(model.KindArtistArtwork, []string{"external:deezer"}, model.ArtworkPriorityRecheck)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(BeZero())

			got, _ := repo.DequeueBatch(10)
			Expect(got).To(HaveLen(1))
			Expect(got[0].Priority).To(Equal(model.ArtworkPriorityBump))
		})

		It("counts without enqueueing", func() {
			n, err := repo.CountBySource(model.KindArtistArtwork, []string{"external:deezer"})
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(int64(1)))

			queued, err := repo.Count()
			Expect(err).ToNot(HaveOccurred())
			Expect(queued).To(BeZero(), "CountBySource must not enqueue")
		})

		It("counts the absent source and every source", func() {
			Expect(repo.CountBySource(model.KindArtistArtwork, []string{""})).To(Equal(int64(1)))
			Expect(repo.CountBySource(model.KindArtistArtwork, nil)).To(Equal(int64(3)))
		})

		It("lists the distinct sources in use by a kind", func() {
			Expect(repo.SourcesInUse(model.KindArtistArtwork)).To(ConsistOf("", "external:deezer", "external:lastfm"))
			Expect(repo.SourcesInUse(model.KindAlbumArtwork)).To(ConsistOf("external:deezer"))
			Expect(repo.SourcesInUse(model.KindRadioArtwork)).To(BeEmpty())
		})
	})

	It("does not disturb an already-queued entity when enqueueing missing rows", func() {
		Expect(repo.Enqueue(item("al", albumRadioactivity.ID, model.ArtworkPriorityBump))).To(Succeed())

		_, err := repo.EnqueueAllMissing(model.KindAlbumArtwork, model.ArtworkPriorityRecheck)
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

	Describe("status counters", func() {
		It("groups queue rows by kind and priority", func() {
			Expect(repo.Enqueue(item("ar", "a1", model.ArtworkPriorityBackfill))).To(Succeed())
			Expect(repo.Enqueue(item("ar", "a2", model.ArtworkPriorityBackfill))).To(Succeed())
			Expect(repo.Enqueue(item("ar", "a3", model.ArtworkPriorityBump))).To(Succeed())
			Expect(repo.Enqueue(item("al", "b1", model.ArtworkPriorityScan))).To(Succeed())

			Expect(repo.CountByKindAndPriority()).To(ConsistOf(
				model.ArtworkQueueStat{ItemKind: "ar", Priority: model.ArtworkPriorityBackfill, Count: 2},
				model.ArtworkQueueStat{ItemKind: "ar", Priority: model.ArtworkPriorityBump, Count: 1},
				model.ArtworkQueueStat{ItemKind: "al", Priority: model.ArtworkPriorityScan, Count: 1},
			))
		})

		It("reports an empty queue as no rows", func() {
			Expect(repo.CountByKindAndPriority()).To(BeEmpty())
		})

		It("counts absent states and how many are due for recheck", func() {
			awRepo := NewArtworkRepository(context.Background(), GetDBXBuilder())
			old := time.Now().Add(-48 * time.Hour)
			for _, ia := range []model.ItemArtwork{
				{ItemKind: "ar", ItemID: "stale1", ImageType: model.ImageTypePrimary, Hash: "", AttemptedAt: old},
				{ItemKind: "ar", ItemID: "fresh1", ImageType: model.ImageTypePrimary, Hash: "", AttemptedAt: time.Now()},
				{ItemKind: "ar", ItemID: "found1", ImageType: model.ImageTypePrimary, Hash: "hX", AttemptedAt: old},
				{ItemKind: "al", ItemID: "stale2", ImageType: model.ImageTypePrimary, Hash: "", AttemptedAt: old},
			} {
				Expect(awRepo.PutItemArtwork(&ia)).To(Succeed())
			}

			Expect(repo.CountAbsent(model.KindArtistArtwork, time.Now().Add(-24*time.Hour))).
				To(Equal(model.ArtworkAbsentStat{Total: 2, Stale: 1}))
		})

		It("reports a kind with no absent state as zero, not as an error", func() {
			Expect(repo.CountAbsent(model.KindRadioArtwork, time.Now())).To(Equal(model.ArtworkAbsentStat{}))
		})
	})

	Describe("PurgeQueued", func() {
		queuedIDs := func() []string {
			GinkgoHelper()
			got, err := repo.DequeueBatch(100)
			Expect(err).ToNot(HaveOccurred())
			return slice.Map(got, func(it model.ArtworkQueueItem) string { return it.ItemID })
		}

		BeforeEach(func() {
			Expect(repo.Enqueue(
				item("ar", "ar-backfill", model.ArtworkPriorityBackfill),
				item("ar", "ar-bump", model.ArtworkPriorityBump),
				item("al", "al-backfill", model.ArtworkPriorityBackfill),
				item("mf", "mf-scan", model.ArtworkPriorityScan),
			)).To(Succeed())
		})

		It("deletes only the given kinds", func() {
			Expect(repo.PurgeQueued([]model.Kind{model.KindArtistArtwork}, nil)).To(BeNumerically("==", 2))
			Expect(queuedIDs()).To(ConsistOf("al-backfill", "mf-scan"))
		})

		It("deletes only the given priorities", func() {
			Expect(repo.PurgeQueued(nil, []int{model.ArtworkPriorityBackfill})).To(BeNumerically("==", 2))
			Expect(queuedIDs()).To(ConsistOf("ar-bump", "mf-scan"))
		})

		It("intersects kinds and priorities", func() {
			Expect(repo.PurgeQueued([]model.Kind{model.KindArtistArtwork}, []int{model.ArtworkPriorityBackfill})).
				To(BeNumerically("==", 1))
			Expect(queuedIDs()).To(ConsistOf("ar-bump", "al-backfill", "mf-scan"))
		})

		It("deletes every row when neither filter is given", func() {
			Expect(repo.PurgeQueued(nil, nil)).To(BeNumerically("==", 4))
			Expect(queuedIDs()).To(BeEmpty())
		})

		It("accepts several kinds and priorities at once", func() {
			Expect(repo.PurgeQueued(
				[]model.Kind{model.KindArtistArtwork, model.KindMediaFileArtwork},
				[]int{model.ArtworkPriorityBackfill, model.ArtworkPriorityScan},
			)).To(BeNumerically("==", 2))
			Expect(queuedIDs()).To(ConsistOf("ar-bump", "al-backfill"))
		})

		It("reports zero when nothing matches, and leaves the queue alone", func() {
			Expect(repo.PurgeQueued([]model.Kind{model.KindPlaylistArtwork}, nil)).To(BeNumerically("==", 0))
			Expect(queuedIDs()).To(HaveLen(4))
		})

		It("deletes a row that is still backing off", func() {
			backOff("ar", "ar-bump", time.Now().Add(time.Hour))

			Expect(repo.PurgeQueued([]model.Kind{model.KindArtistArtwork}, nil)).To(BeNumerically("==", 2))
			Expect(repo.Get(model.KindArtistArtwork, "ar-bump", model.ImageTypePrimary)).
				Error().To(MatchError(model.ErrNotFound))
		})
	})
})
