package persistence

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pocketbase/dbx"
)

var _ = Describe("ScrobbleRepository", func() {
	var repo model.ScrobbleRepository
	var rawRepo sqlRepository
	var ctx context.Context
	var fileID string
	var userID string

	BeforeEach(func() {
		fileID = id.NewRandom()
		userID = id.NewRandom()
		ctx = request.WithUser(log.NewContext(GinkgoT().Context()), model.User{ID: userID, UserName: "johndoe", IsAdmin: true})
		db := GetDBXBuilder()
		repo = NewScrobbleRepository(ctx, db)

		rawRepo = sqlRepository{
			ctx:       ctx,
			tableName: "scrobbles",
			db:        db,
		}
	})

	AfterEach(func() {
		_, _ = rawRepo.db.Delete("scrobbles", dbx.HashExp{"media_file_id": fileID}).Execute()
		_, _ = rawRepo.db.Delete("media_file", dbx.HashExp{"id": fileID}).Execute()
		_, _ = rawRepo.db.Delete("user", dbx.HashExp{"id": userID}).Execute()
	})

	Describe("RecordScrobble", func() {
		It("records a scrobble event and returns an ID", func() {
			submissionTime := time.Now().UTC()

			// Insert User
			_, err := rawRepo.db.Insert("user", dbx.Params{
				"id":         userID,
				"user_name":  "user",
				"password":   "pw",
				"created_at": time.Now(),
				"updated_at": time.Now(),
			}).Execute()
			Expect(err).ToNot(HaveOccurred())

			// Insert MediaFile
			_, err = rawRepo.db.Insert("media_file", dbx.Params{
				"id":         fileID,
				"path":       "path",
				"created_at": time.Now(),
				"updated_at": time.Now(),
			}).Execute()
			Expect(err).ToNot(HaveOccurred())

			scrobbleID, err := repo.RecordScrobble(fileID, submissionTime, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(scrobbleID).ToNot(BeEmpty())

			// Verify insertion
			var scrobble struct {
				ID             string `db:"id"`
				MediaFileID    string `db:"media_file_id"`
				UserID         string `db:"user_id"`
				SubmissionTime int64  `db:"submission_time"`
				Duration       *int   `db:"duration"`
			}
			err = rawRepo.db.Select("*").From("scrobbles").
				Where(dbx.HashExp{"id": scrobbleID}).
				One(&scrobble)
			Expect(err).ToNot(HaveOccurred())
			Expect(scrobble.ID).To(Equal(scrobbleID))
			Expect(scrobble.MediaFileID).To(Equal(fileID))
			Expect(scrobble.UserID).To(Equal(userID))
			Expect(scrobble.SubmissionTime).To(Equal(submissionTime.Unix()))
			Expect(scrobble.Duration).To(BeNil())
		})

		It("records a scrobble event with initial duration", func() {
			submissionTime := time.Now().UTC()
			duration := 180

			// Insert User
			_, err := rawRepo.db.Insert("user", dbx.Params{
				"id":         userID,
				"user_name":  "user",
				"password":   "pw",
				"created_at": time.Now(),
				"updated_at": time.Now(),
			}).Execute()
			Expect(err).ToNot(HaveOccurred())

			// Insert MediaFile
			_, err = rawRepo.db.Insert("media_file", dbx.Params{
				"id":         fileID,
				"path":       "path",
				"created_at": time.Now(),
				"updated_at": time.Now(),
			}).Execute()
			Expect(err).ToNot(HaveOccurred())

			scrobbleID, err := repo.RecordScrobble(fileID, submissionTime, &duration)
			Expect(err).ToNot(HaveOccurred())
			Expect(scrobbleID).ToNot(BeEmpty())

			// Verify insertion
			var scrobble struct {
				ID             string `db:"id"`
				MediaFileID    string `db:"media_file_id"`
				UserID         string `db:"user_id"`
				SubmissionTime int64  `db:"submission_time"`
				Duration       *int   `db:"duration"`
			}
			err = rawRepo.db.Select("*").From("scrobbles").
				Where(dbx.HashExp{"id": scrobbleID}).
				One(&scrobble)
			Expect(err).ToNot(HaveOccurred())
			Expect(scrobble.Duration).ToNot(BeNil())
			Expect(*scrobble.Duration).To(Equal(180))
		})
	})

	Describe("UpdateDuration", func() {
		It("updates the duration of an existing scrobble", func() {
			submissionTime := time.Now().UTC()

			// Insert User
			_, err := rawRepo.db.Insert("user", dbx.Params{
				"id":         userID,
				"user_name":  "user",
				"password":   "pw",
				"created_at": time.Now(),
				"updated_at": time.Now(),
			}).Execute()
			Expect(err).ToNot(HaveOccurred())

			// Insert MediaFile
			_, err = rawRepo.db.Insert("media_file", dbx.Params{
				"id":         fileID,
				"path":       "path",
				"created_at": time.Now(),
				"updated_at": time.Now(),
			}).Execute()
			Expect(err).ToNot(HaveOccurred())

			// Create scrobble with nil duration
			scrobbleID, err := repo.RecordScrobble(fileID, submissionTime, nil)
			Expect(err).ToNot(HaveOccurred())

			// Update duration
			err = repo.UpdateDuration(scrobbleID, 240)
			Expect(err).ToNot(HaveOccurred())

			// Verify update
			var scrobble struct {
				Duration *int `db:"duration"`
			}
			err = rawRepo.db.Select("duration").From("scrobbles").
				Where(dbx.HashExp{"id": scrobbleID}).
				One(&scrobble)
			Expect(err).ToNot(HaveOccurred())
			Expect(scrobble.Duration).ToNot(BeNil())
			Expect(*scrobble.Duration).To(Equal(240))
		})
	})
})
