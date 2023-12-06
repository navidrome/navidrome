package persistence

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sqlBookmarks", func() {
	var mr model.MediaFileRepository

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, model.User{ID: "userid"})
		mr = NewMediaFileRepository(ctx, getDBXBuilder())
	})

	Describe("Bookmarks", func() {
		It("returns an empty collection if there are no bookmarks", func() {
			Expect(mr.GetBookmarks()).To(BeEmpty())
		})

		It("saves and overrides bookmarks", func() {
			By("Saving the bookmark")
			Expect(mr.AddBookmark(songAntenna.ID, "this is a comment", 123)).To(BeNil())

			bms, err := mr.GetBookmarks()
			Expect(err).ToNot(HaveOccurred())

			Expect(bms).To(HaveLen(1))
			Expect(bms[0].Item.ID).To(Equal(songAntenna.ID))
			Expect(bms[0].Item.Title).To(Equal(songAntenna.Title))
			Expect(bms[0].Comment).To(Equal("this is a comment"))
			Expect(bms[0].Position).To(Equal(int64(123)))
			created := bms[0].CreatedAt
			updated := bms[0].UpdatedAt
			Expect(created.IsZero()).To(BeFalse())
			Expect(updated).To(BeTemporally(">=", created))

			By("Overriding the bookmark")
			Expect(mr.AddBookmark(songAntenna.ID, "another comment", 333)).To(BeNil())

			bms, err = mr.GetBookmarks()
			Expect(err).ToNot(HaveOccurred())

			Expect(bms[0].Item.ID).To(Equal(songAntenna.ID))
			Expect(bms[0].Comment).To(Equal("another comment"))
			Expect(bms[0].Position).To(Equal(int64(333)))
			Expect(bms[0].CreatedAt).To(Equal(created))
			Expect(bms[0].UpdatedAt).To(BeTemporally(">=", updated))

			By("Saving another bookmark")
			Expect(mr.AddBookmark(songComeTogether.ID, "one more comment", 444)).To(BeNil())
			bms, err = mr.GetBookmarks()
			Expect(err).ToNot(HaveOccurred())
			Expect(bms).To(HaveLen(2))

			By("Delete bookmark")
			Expect(mr.DeleteBookmark(songAntenna.ID)).To(Succeed())
			bms, err = mr.GetBookmarks()
			Expect(err).ToNot(HaveOccurred())
			Expect(bms).To(HaveLen(1))
			Expect(bms[0].Item.ID).To(Equal(songComeTogether.ID))
			Expect(bms[0].Item.Title).To(Equal(songComeTogether.Title))

			Expect(mr.DeleteBookmark(songComeTogether.ID)).To(Succeed())
			Expect(mr.GetBookmarks()).To(BeEmpty())
		})
	})
})
