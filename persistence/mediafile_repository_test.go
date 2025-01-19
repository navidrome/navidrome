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

var _ = Describe("MediaRepository", func() {
	var mr model.MediaFileRepository

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, model.User{ID: "userid"})
		mr = NewMediaFileRepository(ctx, GetDBXBuilder())
	})

	It("gets mediafile from the DB", func() {
		actual, err := mr.Get("1004")
		Expect(err).ToNot(HaveOccurred())
		actual.CreatedAt = time.Time{}
		Expect(actual).To(Equal(&songAntenna))
	})

	It("returns ErrNotFound", func() {
		_, err := mr.Get("56")
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("counts the number of mediafiles in the DB", func() {
		Expect(mr.CountAll()).To(Equal(int64(4)))
	})

	It("checks existence of mediafiles in the DB", func() {
		Expect(mr.Exists(songAntenna.ID)).To(BeTrue())
		Expect(mr.Exists("666")).To(BeFalse())
	})

	It("delete tracks by id", func() {
		newID := id.NewRandom()
		Expect(mr.Put(&model.MediaFile{LibraryID: 1, ID: newID})).To(BeNil())

		Expect(mr.Delete(newID)).To(BeNil())

		_, err := mr.Get(newID)
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	XIt("filters by genre", func() {
		Expect(mr.GetAll(model.QueryOptions{
			Sort:    "genre.name asc, title asc",
			Filters: squirrel.Eq{"genre.name": "Rock"},
		})).To(Equal(model.MediaFiles{
			songDayInALife,
			songAntenna,
			songComeTogether,
		}))
	})

	Context("Annotations", func() {
		It("increments play count when the tracks does not have annotations", func() {
			id := "incplay.firsttime"
			Expect(mr.Put(&model.MediaFile{LibraryID: 1, ID: id})).To(BeNil())
			playDate := time.Now()
			Expect(mr.IncPlayCount(id, playDate)).To(BeNil())

			mf, err := mr.Get(id)
			Expect(err).To(BeNil())

			Expect(mf.PlayDate.Unix()).To(Equal(playDate.Unix()))
			Expect(mf.PlayCount).To(Equal(int64(1)))
		})

		It("preserves play date if and only if provided date is older", func() {
			id := "incplay.playdate"
			Expect(mr.Put(&model.MediaFile{LibraryID: 1, ID: id})).To(BeNil())
			playDate := time.Now()
			Expect(mr.IncPlayCount(id, playDate)).To(BeNil())
			mf, err := mr.Get(id)
			Expect(err).To(BeNil())
			Expect(mf.PlayDate.Unix()).To(Equal(playDate.Unix()))
			Expect(mf.PlayCount).To(Equal(int64(1)))

			playDateLate := playDate.AddDate(0, 0, 1)
			Expect(mr.IncPlayCount(id, playDateLate)).To(BeNil())
			mf, err = mr.Get(id)
			Expect(err).To(BeNil())
			Expect(mf.PlayDate.Unix()).To(Equal(playDateLate.Unix()))
			Expect(mf.PlayCount).To(Equal(int64(2)))

			playDateEarly := playDate.AddDate(0, 0, -1)
			Expect(mr.IncPlayCount(id, playDateEarly)).To(BeNil())
			mf, err = mr.Get(id)
			Expect(err).To(BeNil())
			Expect(mf.PlayDate.Unix()).To(Equal(playDateLate.Unix()))
			Expect(mf.PlayCount).To(Equal(int64(3)))
		})

		It("increments play count on newly starred items", func() {
			id := "star.incplay"
			Expect(mr.Put(&model.MediaFile{LibraryID: 1, ID: id})).To(BeNil())
			Expect(mr.SetStar(true, id)).To(BeNil())
			playDate := time.Now()
			Expect(mr.IncPlayCount(id, playDate)).To(BeNil())

			mf, err := mr.Get(id)
			Expect(err).To(BeNil())

			Expect(mf.PlayDate.Unix()).To(Equal(playDate.Unix()))
			Expect(mf.PlayCount).To(Equal(int64(1)))
		})
	})
})
