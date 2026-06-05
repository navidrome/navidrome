package persistence

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PlayQueueRepository", func() {
	var repo model.PlayQueueRepository
	var ctx context.Context

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: true})
		repo = NewPlayQueueRepository(ctx, GetDBXBuilder())
	})

	Describe("Store", func() {
		It("stores a complete playqueue", func() {
			expected := aPlayQueue("userid", 1, 123, songComeTogether, songDayInALife)
			Expect(repo.Store(expected)).To(Succeed())

			actual, err := repo.RetrieveWithMediaFiles("userid")
			Expect(err).ToNot(HaveOccurred())
			AssertPlayQueue(expected, actual)
			Expect(countPlayQueues(repo, "userid")).To(Equal(1))
		})

		It("replaces existing playqueue when storing without column names", func() {
			By("Storing initial playqueue")
			initial := aPlayQueue("userid", 0, 100, songComeTogether)
			Expect(repo.Store(initial)).To(Succeed())

			By("Storing replacement playqueue")
			replacement := aPlayQueue("userid", 1, 200, songDayInALife, songAntenna)
			Expect(repo.Store(replacement)).To(Succeed())

			actual, err := repo.RetrieveWithMediaFiles("userid")
			Expect(err).ToNot(HaveOccurred())
			AssertPlayQueue(replacement, actual)
			Expect(countPlayQueues(repo, "userid")).To(Equal(1))
		})

		It("clears playqueue when storing empty items", func() {
			By("Storing initial playqueue")
			initial := aPlayQueue("userid", 0, 100, songComeTogether)
			Expect(repo.Store(initial)).To(Succeed())

			By("Storing empty playqueue")
			empty := aPlayQueue("userid", 0, 0)
			Expect(repo.Store(empty)).To(Succeed())

			By("Verifying playqueue is cleared")
			_, err := repo.Retrieve("userid")
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("updates only current field when specified", func() {
			By("Storing initial playqueue")
			initial := aPlayQueue("userid", 0, 100, songComeTogether, songDayInALife)
			Expect(repo.Store(initial)).To(Succeed())

			By("Getting the existing playqueue to obtain its ID")
			existing, err := repo.Retrieve("userid")
			Expect(err).ToNot(HaveOccurred())

			By("Updating only current field")
			update := &model.PlayQueue{
				ID:        existing.ID, // Use existing ID for partial update
				UserID:    "userid",
				Current:   1,
				ChangedBy: "test-update",
			}
			Expect(repo.Store(update, "current")).To(Succeed())

			By("Verifying only current was updated")
			actual, err := repo.Retrieve("userid")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.Current).To(Equal(1))
			Expect(actual.Position).To(Equal(int64(100))) // Should remain unchanged
			Expect(actual.Items).To(HaveLen(2))           // Should remain unchanged
		})

		It("updates only position field when specified", func() {
			By("Storing initial playqueue")
			initial := aPlayQueue("userid", 1, 100, songComeTogether, songDayInALife)
			Expect(repo.Store(initial)).To(Succeed())

			By("Getting the existing playqueue to obtain its ID")
			existing, err := repo.Retrieve("userid")
			Expect(err).ToNot(HaveOccurred())

			By("Updating only position field")
			update := &model.PlayQueue{
				ID:        existing.ID, // Use existing ID for partial update
				UserID:    "userid",
				Position:  500,
				ChangedBy: "test-update",
			}
			Expect(repo.Store(update, "position")).To(Succeed())

			By("Verifying only position was updated")
			actual, err := repo.Retrieve("userid")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.Position).To(Equal(int64(500)))
			Expect(actual.Current).To(Equal(1)) // Should remain unchanged
			Expect(actual.Items).To(HaveLen(2)) // Should remain unchanged
		})

		It("updates multiple specified fields", func() {
			By("Storing initial playqueue")
			initial := aPlayQueue("userid", 0, 100, songComeTogether)
			Expect(repo.Store(initial)).To(Succeed())

			By("Getting the existing playqueue to obtain its ID")
			existing, err := repo.Retrieve("userid")
			Expect(err).ToNot(HaveOccurred())

			By("Updating current and position fields")
			update := &model.PlayQueue{
				ID:        existing.ID, // Use existing ID for partial update
				UserID:    "userid",
				Current:   1,
				Position:  300,
				ChangedBy: "test-update",
			}
			Expect(repo.Store(update, "current", "position")).To(Succeed())

			By("Verifying both fields were updated")
			actual, err := repo.Retrieve("userid")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.Current).To(Equal(1))
			Expect(actual.Position).To(Equal(int64(300)))
			Expect(actual.Items).To(HaveLen(1)) // Should remain unchanged
		})

		It("preserves existing data when updating with empty items list and column names", func() {
			By("Storing initial playqueue")
			initial := aPlayQueue("userid", 0, 100, songComeTogether, songDayInALife)
			Expect(repo.Store(initial)).To(Succeed())

			By("Getting the existing playqueue to obtain its ID")
			existing, err := repo.Retrieve("userid")
			Expect(err).ToNot(HaveOccurred())

			By("Updating only position with empty items")
			update := &model.PlayQueue{
				ID:        existing.ID, // Use existing ID for partial update
				UserID:    "userid",
				Position:  200,
				ChangedBy: "test-update",
				Items:     []model.MediaFile{}, // Empty items
			}
			Expect(repo.Store(update, "position")).To(Succeed())

			By("Verifying items are preserved")
			actual, err := repo.Retrieve("userid")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.Position).To(Equal(int64(200)))
			Expect(actual.Items).To(HaveLen(2)) // Should remain unchanged
		})

		It("ensures only one record per user by reusing existing record ID", func() {
			By("Storing initial playqueue")
			initial := aPlayQueue("userid", 0, 100, songComeTogether)
			Expect(repo.Store(initial)).To(Succeed())
			initialCount := countPlayQueues(repo, "userid")
			Expect(initialCount).To(Equal(1))

			By("Storing another playqueue with different ID but same user")
			different := aPlayQueue("userid", 1, 200, songDayInALife)
			different.ID = "different-id" // Force a different ID
			Expect(repo.Store(different)).To(Succeed())

			By("Verifying only one record exists for the user")
			finalCount := countPlayQueues(repo, "userid")
			Expect(finalCount).To(Equal(1))

			By("Verifying the record was updated, not duplicated")
			actual, err := repo.Retrieve("userid")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.Current).To(Equal(1))           // Should be updated value
			Expect(actual.Position).To(Equal(int64(200))) // Should be updated value
			Expect(actual.Items).To(HaveLen(1))           // Should be new items
			Expect(actual.Items[0].ID).To(Equal(songDayInALife.ID))
		})

		It("ensures only one record per user even with partial updates", func() {
			By("Storing initial playqueue")
			initial := aPlayQueue("userid", 0, 100, songComeTogether, songDayInALife)
			Expect(repo.Store(initial)).To(Succeed())
			initialCount := countPlayQueues(repo, "userid")
			Expect(initialCount).To(Equal(1))

			By("Storing partial update with different ID but same user")
			partialUpdate := &model.PlayQueue{
				ID:        "completely-different-id", // Use a completely different ID
				UserID:    "userid",
				Current:   1,
				ChangedBy: "test-partial",
			}
			Expect(repo.Store(partialUpdate, "current")).To(Succeed())

			By("Verifying only one record still exists for the user")
			finalCount := countPlayQueues(repo, "userid")
			Expect(finalCount).To(Equal(1))

			By("Verifying the existing record was updated with new current value")
			actual, err := repo.Retrieve("userid")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.Current).To(Equal(1))           // Should be updated value
			Expect(actual.Position).To(Equal(int64(100))) // Should remain unchanged
			Expect(actual.Items).To(HaveLen(2))           // Should remain unchanged
		})
	})

	Describe("Retrieve", func() {
		It("returns notfound error if there's no playqueue for the user", func() {
			_, err := repo.Retrieve("user999")
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("retrieves the playqueue with only track IDs (no full MediaFile data)", func() {
			By("Storing a playqueue for the user")

			expected := aPlayQueue("userid", 1, 123, songComeTogether, songDayInALife)
			Expect(repo.Store(expected)).To(Succeed())

			actual, err := repo.Retrieve("userid")
			Expect(err).ToNot(HaveOccurred())

			// Basic playqueue properties should match
			Expect(actual.ID).To(Equal(expected.ID))
			Expect(actual.UserID).To(Equal(expected.UserID))
			Expect(actual.Current).To(Equal(expected.Current))
			Expect(actual.Position).To(Equal(expected.Position))
			Expect(actual.ChangedBy).To(Equal(expected.ChangedBy))
			Expect(actual.Items).To(HaveLen(len(expected.Items)))

			// Items should only contain IDs, not full MediaFile data
			for i, item := range actual.Items {
				Expect(item.ID).To(Equal(expected.Items[i].ID))
				// These fields should be empty since we're not loading full MediaFiles
				Expect(item.Title).To(BeEmpty())
				Expect(item.Path).To(BeEmpty())
				Expect(item.Album).To(BeEmpty())
				Expect(item.Artist).To(BeEmpty())
			}
		})

		It("returns items with IDs even when some tracks don't exist in the DB", func() {
			// Add a new song to the DB
			newSong := songRadioactivity
			newSong.ID = "temp-track"
			newSong.Path = "/new-path"
			mfRepo := NewMediaFileRepository(ctx, GetDBXBuilder())

			Expect(mfRepo.Put(&newSong)).To(Succeed())

			// Create a playqueue with the new song
			pq := aPlayQueue("userid", 0, 0, newSong, songAntenna)
			Expect(repo.Store(pq)).To(Succeed())

			// Delete the new song from the database
			Expect(mfRepo.Delete("temp-track")).To(Succeed())

			// Retrieve the playqueue with Retrieve method
			actual, err := repo.Retrieve("userid")
			Expect(err).ToNot(HaveOccurred())

			// The playqueue should still contain both track IDs (including the deleted one)
			Expect(actual.Items).To(HaveLen(2))
			Expect(actual.Items[0].ID).To(Equal("temp-track"))
			Expect(actual.Items[1].ID).To(Equal(songAntenna.ID))

			// Items should only contain IDs, no other data
			for _, item := range actual.Items {
				Expect(item.Title).To(BeEmpty())
				Expect(item.Path).To(BeEmpty())
				Expect(item.Album).To(BeEmpty())
				Expect(item.Artist).To(BeEmpty())
			}
		})
	})

	Describe("RetrieveWithMediaFiles", func() {
		It("returns notfound error if there's no playqueue for the user", func() {
			_, err := repo.RetrieveWithMediaFiles("user999")
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("retrieves the playqueue with full MediaFile data", func() {
			By("Storing a playqueue for the user")

			expected := aPlayQueue("userid", 1, 123, songComeTogether, songDayInALife)
			Expect(repo.Store(expected)).To(Succeed())

			actual, err := repo.RetrieveWithMediaFiles("userid")
			Expect(err).ToNot(HaveOccurred())

			AssertPlayQueue(expected, actual)
		})

		It("does not return tracks if they don't exist in the DB", func() {
			// Add a new song to the DB
			newSong := songRadioactivity
			newSong.ID = "temp-track"
			newSong.Path = "/new-path"
			mfRepo := NewMediaFileRepository(ctx, GetDBXBuilder())

			Expect(mfRepo.Put(&newSong)).To(Succeed())

			// Create a playqueue with the new song
			pq := aPlayQueue("userid", 0, 0, newSong, songAntenna)
			Expect(repo.Store(pq)).To(Succeed())

			// Retrieve the playqueue
			actual, err := repo.RetrieveWithMediaFiles("userid")
			Expect(err).ToNot(HaveOccurred())

			// The playqueue should contain both tracks
			AssertPlayQueue(pq, actual)

			// Delete the new song
			Expect(mfRepo.Delete("temp-track")).To(Succeed())

			// Retrieve the playqueue
			actual, err = repo.RetrieveWithMediaFiles("userid")
			Expect(err).ToNot(HaveOccurred())

			// The playqueue should not contain the deleted track
			Expect(actual.Items).To(HaveLen(1))
			Expect(actual.Items[0].ID).To(Equal(songAntenna.ID))
		})
	})

	Describe("Clear", func() {
		It("clears an existing playqueue", func() {
			By("Storing a playqueue")
			expected := aPlayQueue("userid", 1, 123, songComeTogether, songDayInALife)
			Expect(repo.Store(expected)).To(Succeed())

			By("Verifying playqueue exists")
			_, err := repo.Retrieve("userid")
			Expect(err).ToNot(HaveOccurred())

			By("Clearing the playqueue")
			Expect(repo.Clear("userid")).To(Succeed())

			By("Verifying playqueue is cleared")
			_, err = repo.Retrieve("userid")
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("does not error when clearing non-existent playqueue", func() {
			// Clear should not error even if no playqueue exists
			Expect(repo.Clear("nonexistent-user")).To(Succeed())
		})

		It("only clears the specified user's playqueue", func() {
			By("Creating users in the database to avoid foreign key constraints")
			userRepo := NewUserRepository(ctx, GetDBXBuilder())
			user1 := &model.User{ID: "user1", UserName: "user1", Name: "User 1", Email: "user1@test.com"}
			user2 := &model.User{ID: "user2", UserName: "user2", Name: "User 2", Email: "user2@test.com"}
			Expect(userRepo.Put(user1)).To(Succeed())
			Expect(userRepo.Put(user2)).To(Succeed())

			By("Storing playqueues for two users")
			user1Queue := aPlayQueue("user1", 0, 100, songComeTogether)
			user2Queue := aPlayQueue("user2", 1, 200, songDayInALife)
			Expect(repo.Store(user1Queue)).To(Succeed())
			Expect(repo.Store(user2Queue)).To(Succeed())

			By("Clearing only user1's playqueue")
			Expect(repo.Clear("user1")).To(Succeed())

			By("Verifying user1's playqueue is cleared")
			_, err := repo.Retrieve("user1")
			Expect(err).To(MatchError(model.ErrNotFound))

			By("Verifying user2's playqueue still exists")
			actual, err := repo.Retrieve("user2")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.UserID).To(Equal("user2"))
			Expect(actual.Current).To(Equal(1))
			Expect(actual.Position).To(Equal(int64(200)))
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

func aPlayQueue(userId string, current int, position int64, items ...model.MediaFile) *model.PlayQueue {
	createdAt := time.Now()
	updatedAt := createdAt.Add(time.Minute)
	return &model.PlayQueue{
		ID:        id.NewRandom(),
		UserID:    userId,
		Current:   current,
		Position:  position,
		ChangedBy: "test",
		Items:     items,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}
