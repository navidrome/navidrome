package persistence

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
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
	var ctx context.Context

	Describe("RecordScrobble", func() {
		var fileID string
		var userID string
		var rawRepo sqlRepository

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

		It("records a scrobble event", func() {
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

			err = repo.RecordScrobble(fileID, submissionTime)
			Expect(err).ToNot(HaveOccurred())

			// Verify insertion
			var scrobble struct {
				MediaFileID    string `db:"media_file_id"`
				UserID         string `db:"user_id"`
				SubmissionTime int64  `db:"submission_time"`
			}
			err = rawRepo.db.Select("*").From("scrobbles").
				Where(dbx.HashExp{"media_file_id": fileID, "user_id": userID}).
				One(&scrobble)
			Expect(err).ToNot(HaveOccurred())
			Expect(scrobble.MediaFileID).To(Equal(fileID))
			Expect(scrobble.UserID).To(Equal(userID))
			Expect(scrobble.SubmissionTime).To(Equal(submissionTime.Unix()))
		})
	})

	Context("admin user (id userid)", func() {
		BeforeEach(func() {
			ctx = request.WithUser(log.NewContext(context.TODO()), adminUser)
			repo = NewScrobbleRepository(ctx, GetDBXBuilder())
		})

		Describe("Count", func() {
			It("Returns the number of scrobbles in the DB for admin user", func() {
				Expect(repo.CountAll()).To(Equal(int64(2)))
			})

			It("returns scrobbles in a range", func() {
				Expect(repo.CountAll(model.QueryOptions{Filters: squirrel.LtOrEq{"submission_time": 1}})).To(Equal(int64(1)))
			})
		})

		Describe("Get", func() {
			It("returns an existing scrobble for the user", func() {
				scrobble, err := repo.Get("1")
				Expect(err).To(BeNil())
				Expect(scrobble.ID).To(Equal(int64(1)))
				Expect(scrobble.MediaFileID).To(Equal("1001"))
				Expect(scrobble.SubmissionTime).To(Equal(firstScrobble.SubmissionTime))

			})

			It("does not return a scrobble that exists for another user", func() {
				_, err := repo.Get("2")
				Expect(err).To(MatchError(model.ErrNotFound))
			})

			It("does not return a scrobble that does not exist", func() {
				_, err := repo.Get("444")
				Expect(err).To(MatchError(model.ErrNotFound))
			})
		})

		Describe("GetAll", func() {
			It("returns all scrobbles in reverse order", func() {
				scrobbles, err := repo.GetAll(model.QueryOptions{
					Sort:  "submission_time",
					Order: "DESC",
				})
				Expect(err).To(BeNil())
				Expect(scrobbles).To(HaveLen(2))

				Expect(scrobbles[0].ID).To(Equal(int64(3)))
				Expect(scrobbles[0].MediaFileID).To(Equal("1002"))
				Expect(scrobbles[0].SubmissionTime).To(Equal(thirdScrobble.SubmissionTime))

				Expect(scrobbles[1].ID).To(Equal(int64(1)))
				Expect(scrobbles[1].MediaFileID).To(Equal("1001"))
				Expect(scrobbles[1].SubmissionTime).To(Equal(firstScrobble.SubmissionTime))
			})

			It("returns scrobbles in a range", func() {
				scrobbles, err := repo.GetAll(model.QueryOptions{
					Filters: squirrel.GtOrEq{"submission_time": 1}})

				Expect(err).To(BeNil())
				Expect(scrobbles).To(HaveLen(1))

				Expect(scrobbles[0].ID).To(Equal(int64(3)))
				Expect(scrobbles[0].MediaFileID).To(Equal("1002"))
				Expect(scrobbles[0].SubmissionTime).To(Equal(thirdScrobble.SubmissionTime))
			})
		})
	})

	Context("non-admin user", func() {
		BeforeEach(func() {
			ctx = request.WithUser(log.NewContext(context.TODO()), regularUser)
			repo = NewScrobbleRepository(ctx, GetDBXBuilder())
		})

		Describe("Count", func() {
			It("Returns the number of scrobbles in the DB for admin user", func() {
				Expect(repo.CountAll()).To(Equal(int64(1)))
			})

			It("returns scrobbles in a range", func() {
				Expect(repo.CountAll(model.QueryOptions{Filters: squirrel.LtOrEq{"submission_time": 1}})).To(Equal(int64(0)))
			})
		})

		Describe("Get", func() {
			It("returns an existing scrobble for the user", func() {
				scrobble, err := repo.Get("2")
				Expect(err).To(BeNil())
				Expect(scrobble.ID).To(Equal(int64(2)))
				Expect(scrobble.MediaFileID).To(Equal("1003"))
				Expect(scrobble.SubmissionTime).To(Equal(secondScrobble.SubmissionTime))
			})

			It("does not return a scrobble that exists for another user", func() {
				_, err := repo.Get("1")
				Expect(err).To(MatchError(model.ErrNotFound))
			})

			It("does not return a scrobble that does not exist", func() {
				_, err := repo.Get("444")
				Expect(err).To(MatchError(model.ErrNotFound))
			})
		})

		Describe("GetAll", func() {
			It("returns all scrobbles in reverse order", func() {
				scrobbles, err := repo.GetAll(model.QueryOptions{
					Sort:  "submission_time",
					Order: "DESC",
				})
				Expect(err).To(BeNil())
				Expect(scrobbles).To(HaveLen(1))

				Expect(scrobbles[0].ID).To(Equal(int64(2)))
				Expect(scrobbles[0].MediaFileID).To(Equal("1003"))
				Expect(scrobbles[0].SubmissionTime).To(Equal(secondScrobble.SubmissionTime))
			})

			It("returns scrobbles in a range", func() {
				scrobbles, err := repo.GetAll(model.QueryOptions{
					Filters: squirrel.GtOrEq{"submission_time": 1}})

				Expect(err).To(BeNil())
				Expect(scrobbles).To(HaveLen(1))

				Expect(scrobbles[0].ID).To(Equal(int64(2)))
				Expect(scrobbles[0].MediaFileID).To(Equal("1003"))
				Expect(scrobbles[0].SubmissionTime).To(Equal(secondScrobble.SubmissionTime))
			})
		})
	})
})
