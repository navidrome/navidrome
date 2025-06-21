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
)

var _ = Describe("ScrobbleBufferRepository", func() {
	var scrobble model.ScrobbleBufferRepository
	var rawRepo sqlRepository

	enqueueTime := time.Date(2025, 01, 01, 00, 00, 00, 00, time.Local)
	var ids []string

	var insertManually = func(service, userId, mediaFileId string, playTime time.Time) {
		id := id.NewRandom()
		ids = append(ids, id)

		ins := squirrel.Insert("scrobble_buffer").SetMap(map[string]interface{}{
			"id":            id,
			"user_id":       userId,
			"service":       service,
			"media_file_id": mediaFileId,
			"play_time":     playTime,
			"enqueue_time":  enqueueTime,
		})
		_, err := rawRepo.executeSQL(ins)
		Expect(err).ToNot(HaveOccurred())
	}

	BeforeEach(func() {
		ctx := request.WithUser(log.NewContext(context.TODO()), model.User{ID: "userid", UserName: "johndoe", IsAdmin: true})
		db := GetDBXBuilder()
		scrobble = NewScrobbleBufferRepository(ctx, db)

		rawRepo = sqlRepository{
			ctx:       ctx,
			tableName: "scrobble_buffer",
			db:        db,
		}
		ids = []string{}
	})

	AfterEach(func() {
		del := squirrel.Delete(rawRepo.tableName)
		_, err := rawRepo.executeSQL(del)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("Without data", func() {
		Describe("Count", func() {
			It("returns zero when empty", func() {
				count, err := scrobble.Length()
				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(BeZero())
			})
		})

		Describe("Dequeue", func() {
			It("is a no-op when deleting a nonexistent item", func() {
				err := scrobble.Dequeue(&model.ScrobbleEntry{ID: "fake"})
				Expect(err).ToNot(HaveOccurred())

				count, err := scrobble.Length()
				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(Equal(int64(0)))
			})
		})

		Describe("Next", func() {
			It("should not fail with no item for the service", func() {
				entry, err := scrobble.Next("fake", "userid")
				Expect(entry).To(BeNil())
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Describe("UserIds", func() {
			It("should return empty list with no data", func() {
				ids, err := scrobble.UserIDs("service")
				Expect(err).ToNot(HaveOccurred())
				Expect(ids).To(BeEmpty())
			})
		})
	})

	Describe("With data", func() {
		timeA := enqueueTime.Add(24 * time.Hour)
		timeB := enqueueTime.Add(48 * time.Hour)
		timeC := enqueueTime.Add(72 * time.Hour)
		timeD := enqueueTime.Add(96 * time.Hour)

		BeforeEach(func() {
			insertManually("a", "userid", "1001", timeB)
			insertManually("a", "userid", "1002", timeA)
			insertManually("a", "2222", "1003", timeC)
			insertManually("b", "2222", "1004", timeD)
		})

		Describe("Count", func() {
			It("Returns count when populated", func() {
				count, err := scrobble.Length()
				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(Equal(int64(4)))
			})
		})

		Describe("Dequeue", func() {
			It("is a no-op when deleting a nonexistent item", func() {
				err := scrobble.Dequeue(&model.ScrobbleEntry{ID: "fake"})
				Expect(err).ToNot(HaveOccurred())

				count, err := scrobble.Length()
				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(Equal(int64(4)))
			})

			It("deletes an item when specified properly", func() {
				err := scrobble.Dequeue(&model.ScrobbleEntry{ID: ids[3]})
				Expect(err).ToNot(HaveOccurred())

				count, err := scrobble.Length()
				Expect(err).ToNot(HaveOccurred())
				Expect(count).To(Equal(int64(3)))

				entry, err := scrobble.Next("b", "2222")
				Expect(err).ToNot(HaveOccurred())
				Expect(entry).To(BeNil())
			})
		})

		Describe("Enqueue", func() {
			DescribeTable("enqueues an item properly",
				func(service, userId, fileId string, playTime time.Time) {
					now := time.Now()
					err := scrobble.Enqueue(service, userId, fileId, playTime)
					Expect(err).ToNot(HaveOccurred())

					count, err := scrobble.Length()
					Expect(err).ToNot(HaveOccurred())
					Expect(count).To(Equal(int64(5)))

					entry, err := scrobble.Next(service, userId)
					Expect(err).ToNot(HaveOccurred())
					Expect(entry).ToNot(BeNil())

					Expect(entry.EnqueueTime).To(BeTemporally("~", now, 100*time.Millisecond))
					Expect(entry.MediaFileID).To(Equal(fileId))
					Expect(entry.PlayTime).To(BeTemporally("==", playTime))
				},
				Entry("to an existing service with multiple values", "a", "userid", "1004", enqueueTime),
				Entry("to a new service", "c", "2222", "1001", timeD),
				Entry("to an existing service as new user", "b", "userid", "1003", timeC),
			)
		})

		Describe("Next", func() {
			DescribeTable("Returns the next item when populated",
				func(service, id string, playTime time.Time, fileId, artistId string) {
					entry, err := scrobble.Next(service, id)
					Expect(err).ToNot(HaveOccurred())
					Expect(entry).ToNot(BeNil())

					Expect(entry.Service).To(Equal(service))
					Expect(entry.UserID).To(Equal(id))
					Expect(entry.PlayTime).To(BeTemporally("==", playTime))
					Expect(entry.EnqueueTime).To(BeTemporally("==", enqueueTime))
					Expect(entry.MediaFileID).To(Equal(fileId))

					Expect(entry.MediaFile.Participants).To(HaveLen(1))

					artists, ok := entry.MediaFile.Participants[model.RoleArtist]
					Expect(ok).To(BeTrue(), "no artist role in participants")

					Expect(artists).To(HaveLen(1))
					Expect(artists[0].ID).To(Equal(artistId))
				},

				Entry("Service with multiple values for one user", "a", "userid", timeA, "1002", "3"),
				Entry("Service with users", "a", "2222", timeC, "1003", "2"),
				Entry("Service with one user", "b", "2222", timeD, "1004", "2"),
			)

		})

		Describe("UserIds", func() {
			It("should return ordered list for services", func() {
				ids, err := scrobble.UserIDs("a")
				Expect(err).ToNot(HaveOccurred())
				Expect(ids).To(Equal([]string{"2222", "userid"}))
			})

			It("should return for a different service", func() {
				ids, err := scrobble.UserIDs("b")
				Expect(err).ToNot(HaveOccurred())
				Expect(ids).To(Equal([]string{"2222"}))
			})
		})
	})
})
