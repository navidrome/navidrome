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
				pls := &model.Playlist{
					Name:             "Legit Playlist",
					Comment:          "A comment",
					Public:           true,
					Rules:            &criteria.Criteria{Expression: criteria.Contains{"title": "test"}},
					Path:             "/some/path/playlist.m3u",
					Sync:             true,
					UploadedImage:    "injected-image-path",
					ExternalImageURL: "http://evil.example.com/ssrf",
					EvaluatedAt:      new(time.Now()),
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

			It("allows toggling sync for file-backed playlists", func() {
				originalTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
				mockPlsRepo.Data["file-pls"] = &model.Playlist{
					ID:        "file-pls",
					Name:      "File Playlist",
					OwnerID:   "user-1",
					Path:      "/music/playlist.m3u",
					Sync:      true,
					UpdatedAt: originalTime,
				}
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.NewRepository(ctx).(rest.Persistable)
				pls := &model.Playlist{Name: "File Playlist", Sync: false}
				err := repo.Update("file-pls", pls)
				Expect(err).ToNot(HaveOccurred())
				Expect(mockPlsRepo.Last.Sync).To(BeFalse())
				Expect(mockPlsRepo.Last.UpdatedAt).To(Equal(originalTime))
			})

			It("does not allow setting sync on non-file-backed playlists", func() {
				mockPlsRepo.Data["manual-pls"] = &model.Playlist{
					ID:      "manual-pls",
					Name:    "Manual Playlist",
					OwnerID: "user-1",
					Path:    "",
					Sync:    false,
				}
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.NewRepository(ctx).(rest.Persistable)
				pls := &model.Playlist{Name: "Manual Playlist", Sync: true}
				err := repo.Update("manual-pls", pls)
				Expect(err).ToNot(HaveOccurred())
				Expect(mockPlsRepo.Last).To(BeNil())
			})

			It("does not bump updatedAt when only public changes", func() {
				originalTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
				mockPlsRepo.Data["pls-pub"] = &model.Playlist{
					ID:        "pls-pub",
					Name:      "My Playlist",
					OwnerID:   "user-1",
					Public:    false,
					UpdatedAt: originalTime,
				}
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.NewRepository(ctx).(rest.Persistable)
				pls := &model.Playlist{Name: "My Playlist", Public: true}
				err := repo.Update("pls-pub", pls)
				Expect(err).ToNot(HaveOccurred())
				Expect(mockPlsRepo.Last.Public).To(BeTrue())
				Expect(mockPlsRepo.Last.UpdatedAt).To(Equal(originalTime))
			})

			It("bumps updatedAt when name changes along with sync", func() {
				mockPlsRepo.Data["file-pls2"] = &model.Playlist{
					ID:      "file-pls2",
					Name:    "Old Name",
					OwnerID: "user-1",
					Path:    "/music/playlist.m3u",
					Sync:    true,
				}
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.NewRepository(ctx).(rest.Persistable)
				pls := &model.Playlist{Name: "New Name", Sync: false}
				err := repo.Update("file-pls2", pls)
				Expect(err).ToNot(HaveOccurred())
				Expect(mockPlsRepo.Last.Name).To(Equal("New Name"))
				Expect(mockPlsRepo.Last.Sync).To(BeFalse())
			})

			It("returns rest.ErrNotFound when playlist doesn't exist", func() {
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.NewRepository(ctx).(rest.Persistable)
				pls := &model.Playlist{Name: "Updated"}
				err := repo.Update("nonexistent", pls)
				Expect(err).To(Equal(rest.ErrNotFound))
			})

			// Regression tests for #5541: partial REST updates (e.g. bulk "Make Public")
			// must only touch the fields the client actually sent. The cols list from
			// rest.Put names those fields; fields outside it must be left alone, even
			// when the deserialized entity has zero values for them.
			Context("with partial updates (cols)", func() {
				BeforeEach(func() {
					ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
					mockPlsRepo.Data["partial"] = &model.Playlist{
						ID:      "partial",
						Name:    "Original Name",
						Comment: "Original comment",
						OwnerID: "user-1",
						Public:  false,
					}
				})

				It("preserves name and comment when only public is sent (bulk Make Public)", func() {
					repo = ps.NewRepository(ctx).(rest.Persistable)
					err := repo.Update("partial", &model.Playlist{Public: true}, "public")
					Expect(err).ToNot(HaveOccurred())
					Expect(mockPlsRepo.Last.Name).To(Equal("Original Name"))
					Expect(mockPlsRepo.Last.Comment).To(Equal("Original comment"))
					Expect(mockPlsRepo.Last.Public).To(BeTrue())
				})

				It("preserves name when only sync is sent for a file-backed playlist", func() {
					mockPlsRepo.Data["file-partial"] = &model.Playlist{
						ID:      "file-partial",
						Name:    "Keep Me",
						OwnerID: "user-1",
						Path:    "/music/p.m3u",
						Sync:    true,
					}
					repo = ps.NewRepository(ctx).(rest.Persistable)
					err := repo.Update("file-partial", &model.Playlist{Sync: false}, "sync")
					Expect(err).ToNot(HaveOccurred())
					Expect(mockPlsRepo.Last.Name).To(Equal("Keep Me"))
					Expect(mockPlsRepo.Last.Sync).To(BeFalse())
				})

				It("renames the playlist when only name is sent", func() {
					repo = ps.NewRepository(ctx).(rest.Persistable)
					err := repo.Update("partial", &model.Playlist{Name: "Renamed"}, "name")
					Expect(err).ToNot(HaveOccurred())
					Expect(mockPlsRepo.Last.Name).To(Equal("Renamed"))
					Expect(mockPlsRepo.Last.Comment).To(Equal("Original comment"))
					Expect(mockPlsRepo.Last.Public).To(BeFalse())
				})

				It("clears the comment when an empty comment is sent explicitly", func() {
					repo = ps.NewRepository(ctx).(rest.Persistable)
					err := repo.Update("partial", &model.Playlist{Comment: ""}, "comment")
					Expect(err).ToNot(HaveOccurred())
					Expect(mockPlsRepo.Last.Comment).To(BeEmpty())
					Expect(mockPlsRepo.Last.Name).To(Equal("Original Name"))
				})

				It("does not treat a missing ownerId as an ownership transfer attempt", func() {
					// A non-admin user sending only {public:true} should not be blocked
					// just because OwnerID is the zero value in the deserialized entity.
					repo = ps.NewRepository(ctx).(rest.Persistable)
					err := repo.Update("partial", &model.Playlist{Public: true}, "public")
					Expect(err).ToNot(HaveOccurred())
				})
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
