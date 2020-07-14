package persistence

import (
	"context"
	"time"

	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/model/request"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaRepository", func() {
	var mr model.MediaFileRepository

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, model.User{ID: "userid"})
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

	It("finds tracks by path when using wildcards chars", func() {
		Expect(mr.Put(&model.MediaFile{ID: "7001", Path: P("/Find:By'Path/_/123.mp3")})).To(BeNil())
		Expect(mr.Put(&model.MediaFile{ID: "7002", Path: P("/Find:By'Path/1/123.mp3")})).To(BeNil())

		found, err := mr.FindByPath(P("/Find:By'Path/_/"))
		Expect(err).To(BeNil())
		Expect(found).To(HaveLen(1))
		Expect(found[0].ID).To(Equal("7001"))
	})

	It("finds tracks by path case sensitively", func() {
		Expect(mr.Put(&model.MediaFile{ID: "7003", Path: P("/Casesensitive/file1.mp3")})).To(BeNil())
		Expect(mr.Put(&model.MediaFile{ID: "7004", Path: P("/casesensitive/file2.mp3")})).To(BeNil())

		found, err := mr.FindByPath(P("/Casesensitive"))
		Expect(err).To(BeNil())
		Expect(found).To(HaveLen(1))
		Expect(found[0].ID).To(Equal("7003"))

		found, err = mr.FindByPath(P("/casesensitive/"))
		Expect(err).To(BeNil())
		Expect(found).To(HaveLen(1))
		Expect(found[0].ID).To(Equal("7004"))
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
		Expect(mr.Put(&model.MediaFile{ID: id3, Path: P("/ab_/" + id3 + ".mp3")})).To(BeNil())
		id4 := "4444"
		Expect(mr.Put(&model.MediaFile{ID: id4, Path: P("/abc/" + id4 + ".mp3")})).To(BeNil())

		Expect(mr.DeleteByPath(P("/ab_"))).To(Equal(int64(1)))

		Expect(mr.Get(id1)).ToNot(BeNil())
		Expect(mr.Get(id2)).ToNot(BeNil())
		Expect(mr.Get(id4)).ToNot(BeNil())
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
			Expect(mf.PlayCount).To(Equal(int64(1)))
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
			Expect(mf.PlayCount).To(Equal(int64(1)))
		})
	})
})
