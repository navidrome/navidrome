package persistence

import (
	"context"

	"github.com/beego/beego/v2/client/orm"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaFolderRepository", func() {
	var mf model.MediaFolderRepository
	var o orm.Ormer

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, model.User{ID: "userid"})
		o = orm.NewOrm()
		mf = NewMediaFolderRepository(ctx, o)

		mfs, err := mf.GetAllDirectories()
		Expect(err).To(BeNil())
		for _, folder := range mfs {
			if folder.ID != "0" {
				err = mf.Delete(folder.ID)
				Expect(err).To((BeNil()))
			}
		}

		for i := range testFolders {
			err = mf.Put(&testFolders[i])
			Expect(err).To(BeNil())
		}

		_, err = o.Raw("PRAGMA foreign_keys = ON;").Exec()
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		_, err := o.Raw("PRAGMA foreign_keys = OFF;").Exec()
		Expect(err).To(BeNil())
	})

	It("gets mediafolder from the DB", func() {
		Expect(mf.Get("0")).To(Equal(&rootDir))
	})

	It("returns ErrNotFound", func() {
		_, err := mf.Get("12521")
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("returns the root directories", func() {
		mfs, err := mf.GetDbRoot()
		Expect(err).To(BeNil())
		Expect(mfs).To(Equal(model.MediaFolders{rootDir}))
	})

	It("returns the root directories (hardcoded)", func() {
		mfs, err := mf.GetRoot()
		Expect(err).To(BeNil())
		Expect(mfs).To(Equal(model.MediaFolders{model.MediaFolder{
			ID: conf.Server.MusicFolderId, Path: conf.Server.MusicFolder,
			Name: "Music Library",
		}}))
	})

	It("Deletes all directories on root", func() {
		// this requires SQL foreign keys
		Expect(mf.Delete("0")).To(BeNil())
		mfs, err := mf.GetAllDirectories()
		Expect(err).To(BeNil())
		Expect(mfs).To(Equal(model.MediaFolders{}))
	})
})
