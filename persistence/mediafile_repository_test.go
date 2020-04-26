package persistence

import (
	"context"
	"time"

	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaRepository", func() {
	var mr model.MediaFileRepository

	BeforeEach(func() {
		ctx := context.WithValue(log.NewContext(context.TODO()), "user", model.User{ID: "userid"})
		mr = NewMediaFileRepository(ctx, orm.NewOrm())
	})

	It("gets mediafile from the DB", func() {
		Expect(mr.Get("1004")).To(Equal(&songAntenna))
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

	It("find mediafiles by album", func() {
		Expect(mr.FindByAlbum("103")).To(Equal(model.MediaFiles{
			songRadioactivity,
			songAntenna,
		}))
	})

	It("returns empty array when no tracks are found", func() {
		Expect(mr.FindByAlbum("67")).To(Equal(model.MediaFiles{}))
	})

	It("finds tracks by path", func() {
		Expect(mr.FindByPath(P("/beatles/1/sgt"))).To(Equal(model.MediaFiles{
			songDayInALife,
		}))
	})

	It("returns starred tracks", func() {
		Expect(mr.GetStarred()).To(Equal(model.MediaFiles{
			songComeTogether,
		}))
	})

	It("delete tracks by id", func() {
		random, _ := uuid.NewRandom()
		id := random.String()
		Expect(mr.Put(&model.MediaFile{ID: id})).To(BeNil())

		Expect(mr.Delete(id)).To(BeNil())

		_, err := mr.Get(id)
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("delete tracks by path", func() {
		id1 := "1111"
		Expect(mr.Put(&model.MediaFile{ID: id1, Path: P("/abc/123/" + id1 + ".mp3")})).To(BeNil())
		id2 := "2222"
		Expect(mr.Put(&model.MediaFile{ID: id2, Path: P("/abc/123/" + id2 + ".mp3")})).To(BeNil())
		id3 := "3333"
		Expect(mr.Put(&model.MediaFile{ID: id3, Path: P("/abc/" + id3 + ".mp3")})).To(BeNil())

		Expect(mr.DeleteByPath(P("/abc"))).To(BeNil())

		Expect(mr.Get(id1)).ToNot(BeNil())
		Expect(mr.Get(id2)).ToNot(BeNil())
		_, err := mr.Get(id3)
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	Context("Annotations", func() {
		It("increments play count when the tracks does not have annotations", func() {
			id := "incplay.firsttime"
			Expect(mr.Put(&model.MediaFile{ID: id})).To(BeNil())
			playDate := time.Now()
			Expect(mr.IncPlayCount(id, playDate)).To(BeNil())

			mf, err := mr.Get(id)
			Expect(err).To(BeNil())

			Expect(mf.PlayDate.Unix()).To(Equal(playDate.Unix()))
			Expect(mf.PlayCount).To(Equal(1))
		})

		It("increments play count on newly starred items", func() {
			id := "star.incplay"
			Expect(mr.Put(&model.MediaFile{ID: id})).To(BeNil())
			Expect(mr.SetStar(true, id)).To(BeNil())
			playDate := time.Now()
			Expect(mr.IncPlayCount(id, playDate)).To(BeNil())

			mf, err := mr.Get(id)
			Expect(err).To(BeNil())

			Expect(mf.PlayDate.Unix()).To(Equal(playDate.Unix()))
			Expect(mf.PlayCount).To(Equal(1))
		})
	})
})
