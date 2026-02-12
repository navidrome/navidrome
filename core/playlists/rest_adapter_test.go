package playlists_test

import (
	"context"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("REST Adapter", func() {
	var ds *tests.MockDataStore
	var ps playlists.Playlists
	var mockPlsRepo *tests.MockPlaylistRepo
	ctx := context.Background()

	BeforeEach(func() {
		mockPlsRepo = tests.CreateMockPlaylistRepo()
		ds = &tests.MockDataStore{
			MockedPlaylist: mockPlsRepo,
			MockedLibrary:  &tests.MockLibraryRepo{},
		}
		ctx = request.WithUser(ctx, model.User{ID: "123"})
	})

	Describe("NewRepository", func() {
		var repo rest.Persistable

		BeforeEach(func() {
			mockPlsRepo.Data = map[string]*model.Playlist{
				"pls-1": {ID: "pls-1", Name: "My Playlist", OwnerID: "user-1"},
			}
			ps = playlists.NewPlaylists(ds)
		})

		Describe("Save", func() {
			It("sets the owner from the context user", func() {
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.NewRepository(ctx).(rest.Persistable)
				pls := &model.Playlist{Name: "New Playlist"}
				id, err := repo.Save(pls)
				Expect(err).ToNot(HaveOccurred())
				Expect(id).ToNot(BeEmpty())
				Expect(pls.OwnerID).To(Equal("user-1"))
			})

			It("forces a new creation by clearing ID", func() {
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.NewRepository(ctx).(rest.Persistable)
				pls := &model.Playlist{ID: "should-be-cleared", Name: "New"}
				_, err := repo.Save(pls)
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.ID).ToNot(Equal("should-be-cleared"))
			})
		})

		Describe("Update", func() {
			It("allows owner to update their playlist", func() {
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.NewRepository(ctx).(rest.Persistable)
				pls := &model.Playlist{Name: "Updated"}
				err := repo.Update("pls-1", pls)
				Expect(err).ToNot(HaveOccurred())
			})

			It("allows admin to update any playlist", func() {
				ctx = request.WithUser(ctx, model.User{ID: "admin-1", IsAdmin: true})
				repo = ps.NewRepository(ctx).(rest.Persistable)
				pls := &model.Playlist{Name: "Updated"}
				err := repo.Update("pls-1", pls)
				Expect(err).ToNot(HaveOccurred())
			})

			It("denies non-owner, non-admin", func() {
				ctx = request.WithUser(ctx, model.User{ID: "other-user", IsAdmin: false})
				repo = ps.NewRepository(ctx).(rest.Persistable)
				pls := &model.Playlist{Name: "Updated"}
				err := repo.Update("pls-1", pls)
				Expect(err).To(Equal(rest.ErrPermissionDenied))
			})

			It("denies regular user from changing ownership", func() {
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.NewRepository(ctx).(rest.Persistable)
				pls := &model.Playlist{Name: "Updated", OwnerID: "other-user"}
				err := repo.Update("pls-1", pls)
				Expect(err).To(Equal(rest.ErrPermissionDenied))
			})

			It("returns rest.ErrNotFound when playlist doesn't exist", func() {
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.NewRepository(ctx).(rest.Persistable)
				pls := &model.Playlist{Name: "Updated"}
				err := repo.Update("nonexistent", pls)
				Expect(err).To(Equal(rest.ErrNotFound))
			})
		})

		Describe("Delete", func() {
			It("delegates to service Delete with permission checks", func() {
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.NewRepository(ctx).(rest.Persistable)
				err := repo.Delete("pls-1")
				Expect(err).ToNot(HaveOccurred())
				Expect(mockPlsRepo.Deleted).To(ContainElement("pls-1"))
			})

			It("denies non-owner", func() {
				ctx = request.WithUser(ctx, model.User{ID: "other-user", IsAdmin: false})
				repo = ps.NewRepository(ctx).(rest.Persistable)
				err := repo.Delete("pls-1")
				Expect(err).To(Equal(rest.ErrPermissionDenied))
			})
		})
	})
})
