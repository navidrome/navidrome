package persistence

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
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
		ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: true})
		repo = NewPlayQueueRepository(ctx, getDBXBuilder())
	})

	Describe("PlayQueues", func() {
		It("returns notfound error if there's no playqueue for the user", func() {
			_, err := repo.Retrieve("user999")
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("stores and retrieves the playqueue for the user", func() {
			By("Storing a playqueue for the user")

			expected := aPlayQueue("userid", songDayInALife.ID, 123, songComeTogether, songDayInALife)
			Expect(repo.Store(expected)).To(Succeed())

			actual, err := repo.Retrieve("userid")
			Expect(err).ToNot(HaveOccurred())

			AssertPlayQueue(expected, actual)

			By("Storing a new playqueue for the same user")

			another := aPlayQueue("userid", songRadioactivity.ID, 321, songAntenna, songRadioactivity)
			Expect(repo.Store(another)).To(Succeed())

			actual, err = repo.Retrieve("userid")
			Expect(err).ToNot(HaveOccurred())

			AssertPlayQueue(another, actual)
			Expect(countPlayQueues(repo, "userid")).To(Equal(1))
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
