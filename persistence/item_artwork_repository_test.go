package persistence

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ItemArtworkRepository", func() {
	var repo model.ItemArtworkRepository
	var awRepo model.ArtworkRepository

	BeforeEach(func() {
		clearArtworkTables()
		repo = NewItemArtworkRepository(context.Background(), GetDBXBuilder())
		awRepo = NewArtworkRepository(context.Background(), GetDBXBuilder())
	})

	It("upserts and reads state", func() {
		ia := &model.ItemArtwork{ItemKind: "al", ItemID: "al1", ImageType: model.ImageTypePrimary,
			Hash: "h1", Source: "folder", AttemptedAt: time.Now()}
		Expect(repo.Put(ia)).To(Succeed())
		ia.Source = "embedded"
		Expect(repo.Put(ia)).To(Succeed())

		got, err := repo.Get("al", "al1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Source).To(Equal("embedded"))
		Expect(got.UpdatedAt).ToNot(BeZero())
	})

	It("represents known-absent as empty hash", func() {
		Expect(repo.Put(&model.ItemArtwork{ItemKind: "ar", ItemID: "ar1",
			ImageType: model.ImageTypePrimary, Hash: "", AttemptedAt: time.Now()})).To(Succeed())
		got, err := repo.Get("ar", "ar1", model.ImageTypePrimary)
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Hash).To(BeEmpty())
	})

	It("hydrates a page in one batch, including blurhash and absence", func() {
		Expect(awRepo.Put(&model.Artwork{Hash: "h9", Mime: "image/jpeg", BlurHash: "BH9"})).To(Succeed())
		Expect(repo.Put(&model.ItemArtwork{ItemKind: "al", ItemID: "x1", ImageType: model.ImageTypePrimary, Hash: "h9", Source: "folder"})).To(Succeed())
		Expect(repo.Put(&model.ItemArtwork{ItemKind: "al", ItemID: "x2", ImageType: model.ImageTypePrimary, Hash: "", Source: ""})).To(Succeed())

		info, err := repo.GetInfoForItems("al", []string{"x1", "x2", "x3"})
		Expect(err).ToNot(HaveOccurred())
		Expect(info).To(HaveLen(2))
		Expect(info["x1"].Hash).To(Equal("h9"))
		Expect(info["x1"].BlurHash).To(Equal("BH9"))
		Expect(info["x1"].Absent).To(BeFalse())
		Expect(info["x2"].Absent).To(BeTrue())
		_, unresolved := info["x3"]
		Expect(unresolved).To(BeFalse())
	})

	It("deletes all rows for an item", func() {
		Expect(repo.Put(&model.ItemArtwork{ItemKind: "pl", ItemID: "p1", ImageType: model.ImageTypePrimary, Hash: "h1"})).To(Succeed())
		Expect(repo.DeleteForItem("pl", "p1")).To(Succeed())
		_, err := repo.Get("pl", "p1", model.ImageTypePrimary)
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("enqueues stale absent states for recheck", func() {
		old := time.Now().Add(-48 * time.Hour)
		Expect(repo.Put(&model.ItemArtwork{ItemKind: "ar", ItemID: "stale1", ImageType: model.ImageTypePrimary, Hash: "", AttemptedAt: old})).To(Succeed())
		Expect(repo.Put(&model.ItemArtwork{ItemKind: "ar", ItemID: "fresh1", ImageType: model.ImageTypePrimary, Hash: "", AttemptedAt: time.Now()})).To(Succeed())
		Expect(repo.Put(&model.ItemArtwork{ItemKind: "ar", ItemID: "found1", ImageType: model.ImageTypePrimary, Hash: "hX", AttemptedAt: old})).To(Succeed())

		n, err := repo.EnqueueStaleAbsent("ar", time.Now().Add(-24*time.Hour))
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(int64(1)))

		qRepo := NewArtworkQueueRepository(context.Background(), GetDBXBuilder())
		items, err := qRepo.DequeueBatch(10)
		Expect(err).ToNot(HaveOccurred())
		Expect(items).To(HaveLen(1))
		Expect(items[0].ItemID).To(Equal("stale1"))
		Expect(items[0].Priority).To(Equal(model.ArtworkPriorityRecheck))
	})
})
