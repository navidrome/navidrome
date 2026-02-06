package e2e

import (
	"fmt"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bookmark and PlayQueue Endpoints", Ordered, func() {
	BeforeAll(func() {
		setupTestDB()
	})

	Describe("Bookmark Endpoints", Ordered, func() {
		var trackID string

		BeforeAll(func() {
			// Get a media file ID from the database to use for bookmarks
			mfs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{Max: 1})
			Expect(err).ToNot(HaveOccurred())
			Expect(mfs).ToNot(BeEmpty())
			trackID = mfs[0].ID
		})

		It("getBookmarks returns empty initially", func() {
			r := newReq("getBookmarks")
			resp, err := router.GetBookmarks(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Bookmarks).ToNot(BeNil())
			Expect(resp.Bookmarks.Bookmark).To(BeEmpty())
		})

		It("createBookmark creates a bookmark with position", func() {
			r := newReq("createBookmark", "id", trackID, "position", "12345", "comment", "test bookmark")
			resp, err := router.CreateBookmark(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		It("getBookmarks shows the created bookmark", func() {
			r := newReq("getBookmarks")
			resp, err := router.GetBookmarks(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Bookmarks).ToNot(BeNil())
			Expect(resp.Bookmarks.Bookmark).To(HaveLen(1))

			bmk := resp.Bookmarks.Bookmark[0]
			Expect(bmk.Entry.Id).To(Equal(trackID))
			Expect(bmk.Position).To(Equal(int64(12345)))
			Expect(bmk.Comment).To(Equal("test bookmark"))
			Expect(bmk.Username).To(Equal(adminUser.UserName))
		})

		It("deleteBookmark removes the bookmark", func() {
			r := newReq("deleteBookmark", "id", trackID)
			resp, err := router.DeleteBookmark(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))

			// Verify it's gone
			r = newReq("getBookmarks")
			resp, err = router.GetBookmarks(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Bookmarks.Bookmark).To(BeEmpty())
		})
	})

	Describe("PlayQueue Endpoints", Ordered, func() {
		var trackIDs []string

		BeforeAll(func() {
			// Get multiple media file IDs from the database
			mfs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{Max: 3, Sort: "title"})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(mfs)).To(BeNumerically(">=", 2))
			for _, mf := range mfs {
				trackIDs = append(trackIDs, mf.ID)
			}
		})

		It("getPlayQueue returns empty when nothing saved", func() {
			r := newReq("getPlayQueue")
			resp, err := router.GetPlayQueue(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			// When no play queue exists, PlayQueue should be nil (no entry returned)
			Expect(resp.PlayQueue).To(BeNil())
		})

		It("savePlayQueue stores current play queue", func() {
			r := newReq("savePlayQueue",
				"id", trackIDs[0],
				"id", trackIDs[1],
				"current", trackIDs[1],
				"position", "5000",
			)
			resp, err := router.SavePlayQueue(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		It("getPlayQueue returns saved queue with tracks", func() {
			r := newReq("getPlayQueue")
			resp, err := router.GetPlayQueue(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.PlayQueue).ToNot(BeNil())
			Expect(resp.PlayQueue.Entry).To(HaveLen(2))
			Expect(resp.PlayQueue.Current).To(Equal(trackIDs[1]))
			Expect(resp.PlayQueue.Position).To(Equal(int64(5000)))
			Expect(resp.PlayQueue.Username).To(Equal(adminUser.UserName))
			Expect(resp.PlayQueue.ChangedBy).To(Equal("test-client"))
		})

		It("getPlayQueueByIndex returns data with current index", func() {
			r := newReq("getPlayQueueByIndex")
			resp, err := router.GetPlayQueueByIndex(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.PlayQueueByIndex).ToNot(BeNil())
			Expect(resp.PlayQueueByIndex.Entry).To(HaveLen(2))
			Expect(resp.PlayQueueByIndex.CurrentIndex).ToNot(BeNil())
			Expect(*resp.PlayQueueByIndex.CurrentIndex).To(Equal(1))
			Expect(resp.PlayQueueByIndex.Position).To(Equal(int64(5000)))
		})

		It("savePlayQueueByIndex stores queue by index", func() {
			r := newReq("savePlayQueueByIndex",
				"id", trackIDs[0],
				"id", trackIDs[1],
				"id", trackIDs[2],
				"currentIndex", fmt.Sprintf("%d", 0),
				"position", "9999",
			)
			resp, err := router.SavePlayQueueByIndex(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))

			// Verify with getPlayQueueByIndex
			r = newReq("getPlayQueueByIndex")
			resp, err = router.GetPlayQueueByIndex(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.PlayQueueByIndex).ToNot(BeNil())
			Expect(resp.PlayQueueByIndex.Entry).To(HaveLen(3))
			Expect(resp.PlayQueueByIndex.CurrentIndex).ToNot(BeNil())
			Expect(*resp.PlayQueueByIndex.CurrentIndex).To(Equal(0))
			Expect(resp.PlayQueueByIndex.Position).To(Equal(int64(9999)))
		})
	})
})
