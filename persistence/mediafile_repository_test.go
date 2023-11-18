package persistence

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/beego/beego/v2/client/orm"
	"github.com/google/uuid"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaRepository", func() {
	var mr model.MediaFileRepository
	var mf model.DirectoryEntryRepository

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, model.User{ID: "userid"})
		orm := orm.NewOrm()
		mr = NewMediaFileRepository(ctx, orm)
		mf = NewDirectoryEntryRepository(ctx, orm)
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

	It("finds tracks by path when using wildcards chars", func() {
		Expect(mr.Put(&model.MediaFile{ID: "7001", Path: P("/Find:By'Path/_/123.mp3")})).To(BeNil())
		Expect(mr.Put(&model.MediaFile{ID: "7002", Path: P("/Find:By'Path/1/123.mp3")})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{Path: "/Find:By'Path", ID: "1234"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{Path: "/Find:By'Path/_/", ID: "12345", ParentId: "1234"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{Path: "/Find:By'Path/1/", ID: "12346", ParentId: "1235"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{ID: "7001", ParentId: "12345"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{ID: "7002", ParentId: "12346"})).To(BeNil())

		found, err := mr.FindAllByParent("12345")
		Expect(err).To(BeNil())
		Expect(found).To(HaveLen(1))
		Expect(found[0].ID).To(Equal("7001"))
	})

	It("finds tracks by path when using UTF8 chars", func() {
		Expect(mr.Put(&model.MediaFile{ID: "7010", Path: P("/Пётр Ильич Чайковский/123.mp3")})).To(BeNil())
		Expect(mr.Put(&model.MediaFile{ID: "7011", Path: P("/Пётр Ильич Чайковский/222.mp3")})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{Path: "/Пётр Ильич Чайковский", ID: "2345"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{ID: "7010", ParentId: "2345"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{ID: "7011", ParentId: "2345"})).To(BeNil())

		found, err := mr.FindAllByParent("2345")
		Expect(err).To(BeNil())
		Expect(found).To(HaveLen(2))
	})

	It("finds tracks by path case sensitively", func() {
		Expect(mr.Put(&model.MediaFile{ID: "7003", Path: P("/Casesensitive/file1.mp3")})).To(BeNil())
		Expect(mr.Put(&model.MediaFile{ID: "7004", Path: P("/casesensitive/file2.mp3")})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{Path: "/Casesensitive", ID: "3456"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{Path: "/casesensitive", ID: "3457"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{ID: "7003", ParentId: "3456"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{ID: "7004", ParentId: "3457"})).To(BeNil())

		found, err := mr.FindAllByParent("3456")
		Expect(err).To(BeNil())
		Expect(found).To(HaveLen(1))
		Expect(found[0].ID).To(Equal("7003"))

		found, err = mr.FindAllByParent("3457")
		Expect(err).To(BeNil())
		Expect(found).To(HaveLen(1))
		Expect(found[0].ID).To(Equal("7004"))
	})

	It("delete tracks by id", func() {
		id := uuid.NewString()
		Expect(mr.Put(&model.MediaFile{ID: id})).To(BeNil())

		Expect(mr.Delete(id)).To(BeNil())

		_, err := mr.Get(id)
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("delete tracks by path", func() {
		id1 := "6001"
		Expect(mr.Put(&model.MediaFile{ID: id1, Path: P("/abc/123/" + id1 + ".mp3")})).To(BeNil())
		id2 := "6002"
		Expect(mr.Put(&model.MediaFile{ID: id2, Path: P("/abc/123/" + id2 + ".mp3")})).To(BeNil())
		id3 := "6003"
		Expect(mr.Put(&model.MediaFile{ID: id3, Path: P("/ab_/" + id3 + ".mp3")})).To(BeNil())
		id4 := "6004"
		Expect(mr.Put(&model.MediaFile{ID: id4, Path: P("/abc/" + id4 + ".mp3")})).To(BeNil())
		id5 := "6005"
		Expect(mr.Put(&model.MediaFile{ID: id5, Path: P("/Ab_/" + id5 + ".mp3")})).To(BeNil())

		Expect(mf.Put(&model.DirectoryEntry{Path: "/", ID: "4567"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{Path: "/abc", ID: "45678", ParentId: "4567"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{Path: "/ab_", ID: "45679", ParentId: "4567"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{Path: "/Ab_", ID: "45670", ParentId: "4567"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{Path: "/abc/123", ID: "456789", ParentId: "45678"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{ID: id1, ParentId: "456789"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{ID: id2, ParentId: "456789"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{ID: id3, ParentId: "45679"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{ID: id4, ParentId: "45678"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{ID: id5, ParentId: "45670"})).To(BeNil())

		Expect(mr.DeleteByPath(P("/ab_"))).To(Equal(int64(1)))

		Expect(mr.Get(id1)).ToNot(BeNil())
		Expect(mr.Get(id2)).ToNot(BeNil())
		Expect(mr.Get(id4)).ToNot(BeNil())
		Expect(mr.Get(id5)).ToNot(BeNil())
		_, err := mr.Get(id3)
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("delete tracks by path containing UTF8 chars", func() {
		id1 := "6011"
		Expect(mr.Put(&model.MediaFile{ID: id1, Path: P("/Legião Urbana/" + id1 + ".mp3")})).To(BeNil())
		id2 := "6012"
		Expect(mr.Put(&model.MediaFile{ID: id2, Path: P("/Legião Urbana/" + id2 + ".mp3")})).To(BeNil())
		id3 := "6013"
		Expect(mr.Put(&model.MediaFile{ID: id3, Path: P("/Legião Urbana/" + id3 + ".mp3")})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{ID: "5678", Path: "/Legião Urbana"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{ID: id1, ParentId: "5678"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{ID: id2, ParentId: "5678"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{ID: id3, ParentId: "5678"})).To(BeNil())

		Expect(mr.FindAllByParent("5678")).To(HaveLen(3))
		Expect(mr.DeleteByPath(P("/Legião Urbana"))).To(Equal(int64(3)))
		Expect(mr.FindAllByParent("5678")).To(HaveLen(0))
	})

	It("only deletes tracks that match exact path", func() {
		id1 := "6021"
		Expect(mr.Put(&model.MediaFile{ID: id1, Path: P("/music/overlap/Ella Fitzgerald/" + id1 + ".mp3")})).To(BeNil())
		id2 := "6022"
		Expect(mr.Put(&model.MediaFile{ID: id2, Path: P("/music/overlap/Ella Fitzgerald/" + id2 + ".mp3")})).To(BeNil())
		id3 := "6023"
		Expect(mr.Put(&model.MediaFile{ID: id3, Path: P("/music/overlap/Ella Fitzgerald & Louis Armstrong - They Can't Take That Away From Me.mp3")})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{ID: "6789", Path: "/music/overlap"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{ID: "67890", Path: "/music/overlap/Ella Fitzgerald", ParentId: "6789"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{ID: "6021", ParentId: "67890"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{ID: "6022", ParentId: "67890"})).To(BeNil())
		Expect(mf.Put(&model.DirectoryEntry{ID: "6023", ParentId: "6789"})).To(BeNil())

		Expect(mr.FindAllByParent("67890")).To(HaveLen(2))
		Expect(mr.DeleteByPath(P("/music/overlap/Ella Fitzgerald"))).To(Equal(int64(2)))
		Expect(mr.FindAllByParent("6789")).To(HaveLen(1))
	})

	It("filters by genre", func() {
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
			Expect(mr.Put(&model.MediaFile{ID: id})).To(BeNil())
			playDate := time.Now()
			Expect(mr.IncPlayCount(id, playDate)).To(BeNil())

			mf, err := mr.Get(id)
			Expect(err).To(BeNil())

			Expect(mf.PlayDate.Unix()).To(Equal(playDate.Unix()))
			Expect(mf.PlayCount).To(Equal(int64(1)))
		})

		It("preserves play date if and only if provided date is older", func() {
			id := "incplay.playdate"
			Expect(mr.Put(&model.MediaFile{ID: id})).To(BeNil())
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
