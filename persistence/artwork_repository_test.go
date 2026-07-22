package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pocketbase/dbx"
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

	Context("image identity", func() {
		It("stores and retrieves an artwork by hash", func() {
			a := &model.Artwork{Hash: "abc123", Mime: "image/jpeg", Width: 500, Height: 500, SizeBytes: 1234, BlurHash: "LKO2?U%2Tw=w"}
			Expect(repo.PutImage(a)).To(Succeed())

			got, err := repo.GetImage("abc123")
			Expect(err).ToNot(HaveOccurred())
			Expect(got.Mime).To(Equal("image/jpeg"))
			Expect(got.BlurHash).To(Equal("LKO2?U%2Tw=w"))
			Expect(got.CreatedAt).ToNot(BeZero())
		})

		It("is idempotent on Put (upsert by hash)", func() {
			a := &model.Artwork{Hash: "dup1", Mime: "image/png"}
			Expect(repo.PutImage(a)).To(Succeed())
			a.BlurHash = "XYZ"
			Expect(repo.PutImage(a)).To(Succeed())
			got, _ := repo.GetImage("dup1")
			Expect(got.BlurHash).To(Equal("XYZ"))
		})

		It("refreshes created_at when reacquiring an existing hash", func() {
			Expect(repo.PutImage(&model.Artwork{Hash: "reacq", Mime: "image/jpeg"})).To(Succeed())
			_, err := GetDBXBuilder().NewQuery("UPDATE artwork SET created_at={:t} WHERE hash='reacq'").
				Bind(dbx.Params{"t": "2000-01-01 00:00:00"}).Execute()
			Expect(err).ToNot(HaveOccurred())

			Expect(repo.PutImage(&model.Artwork{Hash: "reacq", Mime: "image/png"})).To(Succeed())

			got, err := repo.GetImage("reacq")
			Expect(err).ToNot(HaveOccurred())
			Expect(got.CreatedAt).To(BeTemporally(">", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)))
		})

		It("returns ErrNotFound for a missing hash", func() {
			_, err := repo.GetImage("nope")
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("fetches a batch", func() {
			Expect(repo.PutImage(&model.Artwork{Hash: "b1", Mime: "image/jpeg"})).To(Succeed())
			Expect(repo.PutImage(&model.Artwork{Hash: "b2", Mime: "image/png"})).To(Succeed())
			got, err := repo.GetImages([]string{"b1", "b2", "missing"})
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(HaveLen(2))
			Expect(got["b2"].Mime).To(Equal("image/png"))
		})

		It("returns every stored hash with its current mime", func() {
			Expect(repo.PutImage(&model.Artwork{Hash: "all1", Mime: "image/jpeg"})).To(Succeed())
			Expect(repo.PutImage(&model.Artwork{Hash: "all2", Mime: "image/png"})).To(Succeed())
			mimes, err := repo.GetAllMimes()
			Expect(err).ToNot(HaveOccurred())
			Expect(mimes).To(HaveKeyWithValue("all1", "image/jpeg"))
			Expect(mimes).To(HaveKeyWithValue("all2", "image/png"))
		})

		It("finds orphans older than cutoff, honoring item_artwork references", func() {
			Expect(repo.PutImage(&model.Artwork{Hash: "orph1", Mime: "image/jpeg"})).To(Succeed())
			Expect(repo.PutImage(&model.Artwork{Hash: "ref1", Mime: "image/jpeg"})).To(Succeed())
			Expect(repo.PutItemArtwork(&model.ItemArtwork{ItemKind: "al", ItemID: "a1", ImageType: model.ImageTypePrimary, Hash: "ref1", Source: "folder"})).To(Succeed())

			orphans, err := repo.GetOrphanHashes(time.Now().Add(time.Minute))
			Expect(err).ToNot(HaveOccurred())
			Expect(orphans).To(ContainElement("orph1"))
			Expect(orphans).ToNot(ContainElement("ref1"))

			orphans, err = repo.GetOrphanHashes(time.Now().Add(-time.Hour))
			Expect(err).ToNot(HaveOccurred())
			Expect(orphans).To(BeEmpty())
		})

		It("deletes only unreferenced hashes older than the cutoff", func() {
			Expect(repo.PutImage(&model.Artwork{Hash: "d1", Mime: "image/jpeg"})).To(Succeed())
			Expect(repo.PutImage(&model.Artwork{Hash: "dref", Mime: "image/jpeg"})).To(Succeed())
			Expect(repo.PutItemArtwork(&model.ItemArtwork{ItemKind: "al", ItemID: "a1",
				ImageType: model.ImageTypePrimary, Hash: "dref", Source: "folder"})).To(Succeed())

			Expect(repo.DeleteOrphans(time.Now().Add(time.Minute), []string{"d1", "dref"})).To(Succeed())

			_, err := repo.GetImage("d1")
			Expect(err).To(MatchError(model.ErrNotFound))
			_, err = repo.GetImage("dref")
			Expect(err).ToNot(HaveOccurred())
		})

		It("spares an unreferenced hash younger than the cutoff", func() {
			Expect(repo.PutImage(&model.Artwork{Hash: "young", Mime: "image/jpeg"})).To(Succeed())
			Expect(repo.DeleteOrphans(time.Now().Add(-time.Hour), []string{"young"})).To(Succeed())
			_, err := repo.GetImage("young")
			Expect(err).ToNot(HaveOccurred())
		})

		It("fetches a batch larger than the SQL variable limit", func() {
			hashes := make([]string, 0, 250)
			for i := range 250 {
				h := fmt.Sprintf("big%03d", i)
				Expect(repo.PutImage(&model.Artwork{Hash: h, Mime: "image/jpeg"})).To(Succeed())
				hashes = append(hashes, h)
			}
			hashes = append(hashes, "absent1", "absent2")

			got, err := repo.GetImages(hashes)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(HaveLen(250))
		})
	})

	Context("item state", func() {
		It("upserts and reads state", func() {
			ia := &model.ItemArtwork{ItemKind: "al", ItemID: "al1", ImageType: model.ImageTypePrimary,
				Hash: "h1", Source: "folder", AttemptedAt: time.Now()}
			Expect(repo.PutItemArtwork(ia)).To(Succeed())
			ia.Source = "embedded"
			Expect(repo.PutItemArtwork(ia)).To(Succeed())

			got, err := repo.GetItemArtwork("al", "al1", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(got.Source).To(Equal("embedded"))
			Expect(got.UpdatedAt).ToNot(BeZero())
		})

		It("defaults attempted_at to now when unset", func() {
			before := time.Now().Add(-time.Second)
			Expect(repo.PutItemArtwork(&model.ItemArtwork{ItemKind: "ar", ItemID: "noattempt",
				ImageType: model.ImageTypePrimary, Hash: ""})).To(Succeed())
			got, err := repo.GetItemArtwork("ar", "noattempt", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(got.AttemptedAt).To(BeTemporally(">", before))
		})

		It("represents known-absent as empty hash", func() {
			Expect(repo.PutItemArtwork(&model.ItemArtwork{ItemKind: "ar", ItemID: "ar1",
				ImageType: model.ImageTypePrimary, Hash: "", AttemptedAt: time.Now()})).To(Succeed())
			got, err := repo.GetItemArtwork("ar", "ar1", model.ImageTypePrimary)
			Expect(err).ToNot(HaveOccurred())
			Expect(got.Hash).To(BeEmpty())
		})

		It("hydrates a page in one batch, including blurhash and absence", func() {
			Expect(repo.PutImage(&model.Artwork{Hash: "h9", Mime: "image/jpeg", BlurHash: "BH9"})).To(Succeed())
			Expect(repo.PutItemArtwork(&model.ItemArtwork{ItemKind: "al", ItemID: "x1", ImageType: model.ImageTypePrimary, Hash: "h9", Source: "folder"})).To(Succeed())
			Expect(repo.PutItemArtwork(&model.ItemArtwork{ItemKind: "al", ItemID: "x2", ImageType: model.ImageTypePrimary, Hash: "", Source: ""})).To(Succeed())

			info, err := repo.GetInfoForItems("al", []string{"x1", "x2", "x3"})
			Expect(err).ToNot(HaveOccurred())
			Expect(info).To(HaveLen(2))
			Expect(info["x1"].Hash).To(Equal("h9"))
			Expect(info["x1"].BlurHash).To(Equal("BH9"))
			Expect(info["x1"].Absent()).To(BeFalse())
			Expect(info["x2"].Absent()).To(BeTrue())
			_, unresolved := info["x3"]
			Expect(unresolved).To(BeFalse())
		})

		It("deletes all rows for an item", func() {
			Expect(repo.PutItemArtwork(&model.ItemArtwork{ItemKind: "pl", ItemID: "p1", ImageType: model.ImageTypePrimary, Hash: "h1"})).To(Succeed())
			Expect(repo.DeleteForItem("pl", "p1")).To(Succeed())
			_, err := repo.GetItemArtwork("pl", "p1", model.ImageTypePrimary)
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})
})
