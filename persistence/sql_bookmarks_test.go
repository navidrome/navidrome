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
		mr = NewMediaFileRepository(ctx, GetDBXBuilder())
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

	Describe("library access", func() {
		var otherLib model.Library
		var restrictedUser model.User
		var adminCtx context.Context
		var userMr model.MediaFileRepository

		BeforeEach(func() {
			adminCtx, otherLib, restrictedUser = restrictedFixture("bmk")

			adminMr := NewMediaFileRepository(adminCtx, GetDBXBuilder())
			Expect(adminMr.Put(&model.MediaFile{
				ID: "bmk-otherlib-track", LibraryID: otherLib.ID,
				Path: "hidden/bookmarked.mp3", Title: "Hidden Bookmarked",
			})).To(Succeed())
			DeferCleanup(func() { _ = adminMr.Delete("bmk-otherlib-track") })

			userCtx := request.WithUser(log.NewContext(GinkgoT().Context()), restrictedUser)
			userMr = NewMediaFileRepository(userCtx, GetDBXBuilder())
		})

		It("does not return bookmarks for tracks outside the user's libraries", func() {
			Expect(userMr.AddBookmark("bmk-otherlib-track", "sneaky", 1)).To(Succeed())

			Expect(userMr.GetBookmarks()).To(BeEmpty())
		})

		It("still returns the bookmark for an admin", func() {
			adminMr := NewMediaFileRepository(adminCtx, GetDBXBuilder())
			Expect(adminMr.AddBookmark("bmk-otherlib-track", "mine", 1)).To(Succeed())
			DeferCleanup(func() { _ = adminMr.DeleteBookmark("bmk-otherlib-track") })

			bms, err := adminMr.GetBookmarks()
			Expect(err).ToNot(HaveOccurred())
			Expect(bms).To(HaveLen(1))
			Expect(bms[0].Item.ID).To(Equal("bmk-otherlib-track"))
		})

		It("keeps returning bookmarks for tracks inside the user's libraries", func() {
			Expect(userMr.AddBookmark(songAntenna.ID, "allowed", 5)).To(Succeed())
			DeferCleanup(func() { _ = userMr.DeleteBookmark(songAntenna.ID) })

			bms, err := userMr.GetBookmarks()
			Expect(err).ToNot(HaveOccurred())
			Expect(bms).To(HaveLen(1))
			Expect(bms[0].Item.ID).To(Equal(songAntenna.ID))
		})
	})
})
