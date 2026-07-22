package persistence

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// clearArtworkTables resets the shared test DB's artwork tables so specs don't leak state.
func clearArtworkTables() {
	db := GetDBXBuilder()
	for _, t := range []string{"artwork_queue", "item_artwork", "artwork"} {
		_, err := db.NewQuery("DELETE FROM " + t).Execute()
		Expect(err).ToNot(HaveOccurred())
	}
}

var _ = Describe("ArtworkRepository", func() {
	var repo model.ArtworkRepository

	BeforeEach(func() {
		clearArtworkTables()
		repo = NewArtworkRepository(context.Background(), GetDBXBuilder())
	})

	It("stores and retrieves an artwork by hash", func() {
		a := &model.Artwork{Hash: "abc123", Mime: "image/jpeg", Width: 500, Height: 500, SizeBytes: 1234, BlurHash: "LKO2?U%2Tw=w"}
		Expect(repo.Put(a)).To(Succeed())

		got, err := repo.Get("abc123")
		Expect(err).ToNot(HaveOccurred())
		Expect(got.Mime).To(Equal("image/jpeg"))
		Expect(got.BlurHash).To(Equal("LKO2?U%2Tw=w"))
		Expect(got.CreatedAt).ToNot(BeZero())
	})

	It("is idempotent on Put (upsert by hash)", func() {
		a := &model.Artwork{Hash: "dup1", Mime: "image/png"}
		Expect(repo.Put(a)).To(Succeed())
		a.BlurHash = "XYZ"
		Expect(repo.Put(a)).To(Succeed())
		got, _ := repo.Get("dup1")
		Expect(got.BlurHash).To(Equal("XYZ"))
	})

	It("returns ErrNotFound for a missing hash", func() {
		_, err := repo.Get("nope")
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("fetches a batch", func() {
		Expect(repo.Put(&model.Artwork{Hash: "b1", Mime: "image/jpeg"})).To(Succeed())
		Expect(repo.Put(&model.Artwork{Hash: "b2", Mime: "image/png"})).To(Succeed())
		got, err := repo.GetBatch([]string{"b1", "b2", "missing"})
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(HaveLen(2))
		Expect(got["b2"].Mime).To(Equal("image/png"))
	})

	It("finds orphans older than cutoff, honoring item_artwork references", func() {
		Expect(repo.Put(&model.Artwork{Hash: "orph1", Mime: "image/jpeg"})).To(Succeed())
		Expect(repo.Put(&model.Artwork{Hash: "ref1", Mime: "image/jpeg"})).To(Succeed())
		iaRepo := NewItemArtworkRepository(context.Background(), GetDBXBuilder())
		Expect(iaRepo.Put(&model.ItemArtwork{ItemKind: "al", ItemID: "a1", ImageType: model.ImageTypePrimary, Hash: "ref1", Source: "folder"})).To(Succeed())

		orphans, err := repo.GetOrphanHashes(time.Now().Add(time.Minute))
		Expect(err).ToNot(HaveOccurred())
		Expect(orphans).To(ContainElement("orph1"))
		Expect(orphans).ToNot(ContainElement("ref1"))

		orphans, err = repo.GetOrphanHashes(time.Now().Add(-time.Hour))
		Expect(err).ToNot(HaveOccurred())
		Expect(orphans).To(BeEmpty())
	})

	It("deletes by hashes", func() {
		Expect(repo.Put(&model.Artwork{Hash: "d1", Mime: "image/jpeg"})).To(Succeed())
		Expect(repo.Delete("d1")).To(Succeed())
		_, err := repo.Get("d1")
		Expect(err).To(MatchError(model.ErrNotFound))
	})
})
