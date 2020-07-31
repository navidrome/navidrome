package persistence

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/model/request"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PlayQueueRepository", func() {
	var repo model.PlayQueueRepository

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, model.User{ID: "user1", UserName: "user1", IsAdmin: true})
		repo = NewPlayQueueRepository(ctx, orm.NewOrm())
	})

	Describe("PlayQueues", func() {
		It("returns notfound error if there's no playqueue for the user", func() {
			_, err := repo.Retrieve("user999")
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("stores and retrieves the playqueue for the user", func() {
			By("Storing a playqueue for the user")

			expected := aPlayQueue("user1", songDayInALife.ID, 123, songComeTogether, songDayInALife)
			Expect(repo.Store(expected)).To(BeNil())

			actual, err := repo.Retrieve("user1")
			Expect(err).To(BeNil())

			AssertPlayQueue(expected, actual)

			By("Storing a new playqueue for the same user")

			new := aPlayQueue("user1", songRadioactivity.ID, 321, songAntenna, songRadioactivity)
			Expect(repo.Store(new)).To(BeNil())

			actual, err = repo.Retrieve("user1")
			Expect(err).To(BeNil())

			AssertPlayQueue(new, actual)
			Expect(countPlayQueues(repo, "user1")).To(Equal(1))
		})
	})

	Describe("Bookmarks", func() {
		It("returns an empty collection if there are no bookmarks", func() {
			Expect(repo.GetBookmarks("user999")).To(BeEmpty())
		})

		It("saves and overrides bookmarks", func() {
			By("Saving the bookmark")
			Expect(repo.AddBookmark("user5", songAntenna.ID, "this is a comment", 123)).To(BeNil())

			bms, err := repo.GetBookmarks("user5")
			Expect(err).To(BeNil())

			Expect(bms).To(HaveLen(1))
			Expect(bms[0].ID).To(Equal(songAntenna.ID))
			Expect(bms[0].Comment).To(Equal("this is a comment"))
			Expect(bms[0].Position).To(Equal(int64(123)))

			By("Overriding the bookmark")
			Expect(repo.AddBookmark("user5", songAntenna.ID, "another comment", 333)).To(BeNil())

			bms, err = repo.GetBookmarks("user5")
			Expect(err).To(BeNil())

			Expect(bms[0].ID).To(Equal(songAntenna.ID))
			Expect(bms[0].Comment).To(Equal("another comment"))
			Expect(bms[0].Position).To(Equal(int64(333)))

			By("Saving another bookmark")
			Expect(repo.AddBookmark("user5", songComeTogether.ID, "one more comment", 444)).To(BeNil())
			bms, err = repo.GetBookmarks("user5")
			Expect(err).To(BeNil())
			Expect(bms).To(HaveLen(2))

			By("Delete bookmark")
			Expect(repo.DeleteBookmark("user5", songAntenna.ID))
			bms, err = repo.GetBookmarks("user5")
			Expect(err).To(BeNil())
			Expect(bms).To(HaveLen(1))
			Expect(bms[0].ID).To(Equal(songComeTogether.ID))
		})
	})
})

func countPlayQueues(repo model.PlayQueueRepository, userId string) int {
	r := repo.(*playQueueRepository)
	c, err := r.count(squirrel.Select().Where(squirrel.Eq{"user_id": userId}))
	if err != nil {
		panic(err)
	}
	return int(c)
}

func AssertPlayQueue(expected, actual *model.PlayQueue) {
	Expect(actual.ID).To(Equal(expected.ID))
	Expect(actual.UserID).To(Equal(expected.UserID))
	Expect(actual.Comment).To(Equal(expected.Comment))
	Expect(actual.Current).To(Equal(expected.Current))
	Expect(actual.Position).To(Equal(expected.Position))
	Expect(actual.ChangedBy).To(Equal(expected.ChangedBy))
	Expect(actual.Items).To(HaveLen(len(expected.Items)))
	for i, item := range actual.Items {
		Expect(item.Title).To(Equal(expected.Items[i].Title))
	}
}

func aPlayQueue(userId, current string, position int64, items ...model.MediaFile) *model.PlayQueue {
	createdAt := time.Now()
	updatedAt := createdAt.Add(time.Minute)
	id, _ := uuid.NewRandom()
	return &model.PlayQueue{
		ID:        id.String(),
		UserID:    userId,
		Comment:   "no_comments",
		Current:   current,
		Position:  position,
		ChangedBy: "test",
		Items:     items,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}
