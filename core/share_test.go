package core

import (
	"context"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Share", func() {
	var ds model.DataStore
	var share Share
	var ctx context.Context

	putTrack := func(id, title string) {
		GinkgoHelper()
		Expect(ds.MediaFile().Put(ctx, &model.MediaFile{ID: id, LibraryID: model.DefaultLibraryID, Path: id + ".mp3", Title: title})).To(Succeed())
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
		for _, id := range []string{"user-1", "logged-in-user", "victim-user"} {
			Expect(ds.User().Put(ctx, &model.User{ID: id, UserName: id})).To(Succeed())
		}
		ctx = request.WithUser(ctx, model.User{ID: "user-1", UserName: "user-1"})
		share = NewShare(ds)
	})

	Describe("NewRepository", func() {
		var repo rest.Persistable[model.Share]

		BeforeEach(func() {
			repo = share.Repository().(rest.Persistable[model.Share])
			Expect(ds.Album().Put(ctx, &model.Album{ID: "123", Name: "Album", LibraryID: model.DefaultLibraryID})).To(Succeed())
		})

		Describe("Save", func() {
			It("it sets a random ID", func() {
				entity := &model.Share{Description: "test", ResourceIDs: "123"}
				id, err := repo.Save(ctx, entity)
				Expect(err).ToNot(HaveOccurred())
				Expect(id).ToNot(BeEmpty())
				Expect(entity.ID).To(Equal(id))
			})

			It("assigns the logged-in user as owner, ignoring a client-supplied UserID", func() {
				loggedInCtx := request.WithUser(ctx, model.User{ID: "logged-in-user"})
				repo := share.Repository().(rest.Persistable[model.Share])
				entity := &model.Share{Description: "test", ResourceIDs: "123", UserID: "victim-user"}
				_, err := repo.Save(loggedInCtx, entity)
				Expect(err).ToNot(HaveOccurred())
				Expect(entity.UserID).To(Equal("logged-in-user"))
				stored, err := ds.Share().Get(loggedInCtx, entity.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(stored.UserID).To(Equal("logged-in-user"))
			})

			It("does not truncate ASCII labels shorter than 30 characters", func() {
				putTrack("456", "Example Media File")
				entity := &model.Share{Description: "test", ResourceIDs: "456"}
				_, err := repo.Save(ctx, entity)
				Expect(err).ToNot(HaveOccurred())
				Expect(entity.Contents).To(Equal("Example Media File"))
			})

			It("truncates ASCII labels longer than 30 characters", func() {
				putTrack("789", "Example Media File But The Title Is Really Long For Testing Purposes")
				entity := &model.Share{Description: "test", ResourceIDs: "789"}
				_, err := repo.Save(ctx, entity)
				Expect(err).ToNot(HaveOccurred())
				Expect(entity.Contents).To(Equal("Example Media File But The ..."))
			})

			It("does not truncate CJK labels shorter than 30 runes", func() {
				putTrack("456", "青春コンプレックス")
				entity := &model.Share{Description: "test", ResourceIDs: "456"}
				_, err := repo.Save(ctx, entity)
				Expect(err).ToNot(HaveOccurred())
				Expect(entity.Contents).To(Equal("青春コンプレックス"))
			})

			It("truncates CJK labels longer than 30 runes", func() {
				putTrack("789", "私の中の幻想的世界観及びその顕現を想起させたある現実での出来事に関する一考察")
				entity := &model.Share{Description: "test", ResourceIDs: "789"}
				_, err := repo.Save(ctx, entity)
				Expect(err).ToNot(HaveOccurred())
				Expect(entity.Contents).To(Equal("私の中の幻想的世界観及びその顕現を想起させたある現実で..."))
			})

			It("fails when any of the resource IDs does not exist", func() {
				entity := &model.Share{Description: "test", ResourceIDs: "123,missing"}
				_, err := repo.Save(ctx, entity)
				Expect(err).To(MatchError(model.ErrNotFound))
			})

			It("fails when the resource IDs are of mixed types", func() {
				putTrack("456", "Example Media File")
				entity := &model.Share{Description: "test", ResourceIDs: "123,456"}
				_, err := repo.Save(ctx, entity)
				Expect(err).To(HaveOccurred())
			})
		})

		Describe("Update", func() {
			It("filters out read-only fields", func() {
				id, err := repo.Save(ctx, &model.Share{Description: "old", ResourceIDs: "123"})
				Expect(err).ToNot(HaveOccurred())
				before, err := ds.Share().Get(ctx, id)
				Expect(err).ToNot(HaveOccurred())

				err = repo.Update(ctx, id, model.Share{Description: "new", Downloadable: true, Contents: "changed", ResourceIDs: "456", VisitCount: 99})
				Expect(err).ToNot(HaveOccurred())

				after, err := ds.Share().Get(ctx, id)
				Expect(err).ToNot(HaveOccurred())
				Expect(after.Description).To(Equal("new"))
				Expect(after.Downloadable).To(BeTrue())
				Expect(after.Contents).To(Equal(before.Contents))
				Expect(after.ResourceIDs).To(Equal("123"))
				Expect(after.VisitCount).To(BeZero())
			})
		})
	})
})
