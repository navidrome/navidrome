package persistence

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/google/uuid"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
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

			another := aPlayQueue("user1", songRadioactivity.ID, 321, songAntenna, songRadioactivity)
			Expect(repo.Store(another)).To(BeNil())

			actual, err = repo.Retrieve("user1")
			Expect(err).To(BeNil())

			AssertPlayQueue(another, actual)
			Expect(countPlayQueues(repo, "user1")).To(Equal(1))
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
	return &model.PlayQueue{
		ID:        uuid.NewString(),
		UserID:    userId,
		Current:   current,
		Position:  position,
		ChangedBy: "test",
		Items:     items,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}
