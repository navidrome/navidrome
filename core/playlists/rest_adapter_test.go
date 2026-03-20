package playlists_test

import (
	"context"
	"time"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
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
			ps = playlists.NewPlaylists(ds, core.NewImageUploadService())
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

			It("clears server-managed fields to prevent injection via REST API", func() {
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.NewRepository(ctx).(rest.Persistable)
				now := time.Now()
				pls := &model.Playlist{
					Name:             "Legit Playlist",
					Comment:          "A comment",
					Public:           true,
					Rules:            &criteria.Criteria{Expression: criteria.Contains{"title": "test"}},
					Path:             "/some/path/playlist.m3u",
					Sync:             true,
					UploadedImage:    "injected-image-path",
					ExternalImageURL: "http://evil.example.com/ssrf",
					EvaluatedAt:      &now,
				}
				_, err := repo.Save(pls)
				Expect(err).ToNot(HaveOccurred())

				saved := mockPlsRepo.Last
				// User-settable fields are preserved
				Expect(saved.Name).To(Equal("Legit Playlist"))
				Expect(saved.Comment).To(Equal("A comment"))
				Expect(saved.Public).To(BeTrue())
				Expect(saved.Rules).ToNot(BeNil())
				// Server-managed fields are cleared
				Expect(saved.Path).To(BeEmpty())
				Expect(saved.Sync).To(BeFalse())
				Expect(saved.UploadedImage).To(BeEmpty())
				Expect(saved.ExternalImageURL).To(BeEmpty())
				Expect(saved.EvaluatedAt).To(BeNil())
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

			It("updates smart playlist rules", func() {
				mockPlsRepo.Data["smart-1"] = &model.Playlist{
					ID:      "smart-1",
					Name:    "Smart Playlist",
					OwnerID: "user-1",
					Rules:   &criteria.Criteria{Expression: criteria.Contains{"title": "old"}},
				}
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.NewRepository(ctx).(rest.Persistable)
				newRules := &criteria.Criteria{Expression: criteria.Contains{"title": "new"}}
				pls := &model.Playlist{Name: "Smart Playlist", Rules: newRules}
				err := repo.Update("smart-1", pls)
				Expect(err).ToNot(HaveOccurred())
				Expect(mockPlsRepo.Last.Rules).To(Equal(newRules))
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
