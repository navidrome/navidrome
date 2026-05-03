package playlists_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
)

var _ = Describe("Playlists Permissions", func() {
	var ds *tests.MockDataStore
	var ps playlists.Playlists
	var mockUserRepo *tests.MockedUserRepo
	var mockPlsRepo *tests.MockPlaylistRepo
	var mockPermissions *tests.MockPlaylistPermissionRepo
	ctx := context.Background()

	BeforeEach(func() {
		mockUserRepo = tests.CreateMockUserRepo()
		mockPlsRepo = tests.CreateMockPlaylistRepo()
		ds = &tests.MockDataStore{
			MockedUser:     mockUserRepo,
			MockedPlaylist: mockPlsRepo,
		}
		ctx = request.WithUser(ctx, model.User{ID: "123"})

		mockUserRepo.Data = map[string]*model.User{
			"user-1":     {ID: "user-1"},
			"other-user": {ID: "other-user"},
		}

		mockPlsRepo.Data = map[string]*model.Playlist{
			"pls-1": {ID: "pls-1", Name: "My Playlist", OwnerID: "user-1"},
		}
		mockPermissions = tests.CreateMockPlaylistPermissionRepo()
		mockPlsRepo.PermissionsRepo = mockPermissions
		ps = playlists.NewPlaylists(ds, core.NewImageUploadService())
	})

	Describe("GetPermissionsForPlaylist", func() {
		It("allows owner to get permissions", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			perms, err := ps.GetPermissionsForPlaylist(ctx, "pls-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(Equal(model.PlaylistPermissions{}))
		})
		It("allows admin to get permissions", func() {
			ctx = request.WithUser(ctx, model.User{ID: "admin", IsAdmin: true})
			perms, err := ps.GetPermissionsForPlaylist(ctx, "pls-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(perms).To(Equal(model.PlaylistPermissions{}))
		})
		It("denies non-owner or non-admin from getting permissions", func() {
			ctx = request.WithUser(ctx, model.User{ID: "other-user", IsAdmin: false})
			perms, err := ps.GetPermissionsForPlaylist(ctx, "pls-1")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(model.ErrNotAuthorized))
			Expect(perms).To(BeNil())
		})
		It("retruns error when playlist not found", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			perms, err := ps.GetPermissionsForPlaylist(ctx, "non-existend-pls")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(model.ErrNotFound))
			Expect(perms).To(BeNil())
		})
		Context("with existing permissions", func() {
			BeforeEach(func() {
				mockPermissions.Data = map[string]*model.PlaylistPermission{
					"ignored-1": {Permission: model.PermissionEditor, UserID: "some-user-1", PlaylistID: "pls-1"},
					"ignored-2": {Permission: model.PermissionViewer, UserID: "some-user-2", PlaylistID: "pls-1"},
				}
			})
			It("returns correct list of permissions", func() {
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				perms, err := ps.GetPermissionsForPlaylist(ctx, "pls-1")
				Expect(err).ToNot(HaveOccurred())
				Expect(perms).To(ConsistOf(
					model.PlaylistPermission{PlaylistID: "pls-1", UserID: "some-user-1", Permission: model.PermissionEditor},
					model.PlaylistPermission{PlaylistID: "pls-1", UserID: "some-user-2", Permission: model.PermissionViewer},
				))
			})
		})
	})

	Describe("AddPermission", func() {
		It("allows owner to add permissions", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.AddPermission(ctx, "pls-1", "other-user", model.PermissionEditor)
			Expect(err).ToNot(HaveOccurred())
			Expect(mockPermissions.Data).To(Equal(map[string]*model.PlaylistPermission{"other-user": {Permission: model.PermissionEditor}}))
		})
		It("allows admin to add permissions", func() {
			ctx = request.WithUser(ctx, model.User{ID: "admin", IsAdmin: true})
			err := ps.AddPermission(ctx, "pls-1", "other-user", model.PermissionEditor)
			Expect(err).ToNot(HaveOccurred())
			Expect(mockPermissions.Data).To(Equal(map[string]*model.PlaylistPermission{"other-user": {Permission: model.PermissionEditor}}))
		})
		It("denies non-owner or non-admin from adding permissions", func() {
			ctx = request.WithUser(ctx, model.User{ID: "other-user", IsAdmin: false})
			err := ps.AddPermission(ctx, "pls-1", "other-user", model.PermissionEditor)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(model.ErrNotAuthorized))
		})
		It("retruns error when playlist not found", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.AddPermission(ctx, "non-existend-pls", "other-user", model.PermissionEditor)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(model.ErrNotFound))
		})
		It("retruns error when user to add not found", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.AddPermission(ctx, "pls-1", "non-existent-user", model.PermissionEditor)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(model.ErrNotFound))
			Expect(err).To(MatchError(ContainSubstring("validating existence of user with ID %q", "non-existent-user")))
		})
		It("retruns error when permission to add not found", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.AddPermission(ctx, "pls-1", "other-user", "non-existent-perm")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(model.ErrValidation))
			Expect(err).To(MatchError(ContainSubstring("permission %q not supported", "non-existent-perm")))
		})
	})

	Describe("RemovePermission", func() {
		It("allows owner to remove permissions", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.RemovePermission(ctx, "pls-1", "other-user")
			Expect(err).ToNot(HaveOccurred())
			Expect(mockPermissions.Data).To(BeEmpty())
		})
		It("allows admin to remove permissions", func() {
			ctx = request.WithUser(ctx, model.User{ID: "admin", IsAdmin: true})
			err := ps.RemovePermission(ctx, "pls-1", "other-user")
			Expect(err).ToNot(HaveOccurred())
			Expect(mockPermissions.Data).To(BeEmpty())
		})
		It("denies non-owner or non-admin from removing permissions", func() {
			ctx = request.WithUser(ctx, model.User{ID: "other-user", IsAdmin: false})
			err := ps.RemovePermission(ctx, "pls-1", "other-user")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(model.ErrNotAuthorized))
		})
		It("retruns error when playlist not found", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.RemovePermission(ctx, "non-existend-pls", "other-user")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(model.ErrNotFound))
		})
		It("retruns error when user to add not found", func() {
			ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
			err := ps.RemovePermission(ctx, "pls-1", "non-existent-user")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})
})
