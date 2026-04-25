package persistence

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

var _ = Describe("PlaylistPermissionRepository", Ordered, func() {
	var (
		playlistRepo model.PlaylistRepository
		repo         model.PlaylistPermissionRepository
	)

	BeforeEach(func() {
		ctx := log.NewContext(context.Background())
		ctx = request.WithUser(ctx, adminUser)
		playlistRepo = NewPlaylistRepository(ctx, GetDBXBuilder())

		repo = playlistRepo.Permissions(testPlaylists[0].ID)
	})

	Describe("constructor", func() {
		It("should return nil", func() {
			Expect(playlistRepo.Permissions("non-existent-playlist")).To(BeNil())
		})
	})

	Describe("working repo", func() {
		Describe("No permissions exist yet", func() {
			It("should return nothing", func() {
				perms, err := repo.GetAll()
				Expect(err).ToNot(HaveOccurred())
				Expect(perms).To(BeEmpty())
			})
		})
		Describe("Create permission", func() {
			It("should create the permission", func() {
				Expect(repo.Put(adminUser.ID, model.PermissionEditor)).To(Succeed())
			})
		})
		Describe("There now is a permission", func() {
			It("should return the permissions", func() {
				perms, err := repo.GetAll()
				Expect(err).ToNot(HaveOccurred())
				Expect(perms).To(Equal(model.PlaylistPermissions{{
					PlaylistID: testPlaylists[0].ID, UserID: adminUser.ID, Permission: model.PermissionEditor,
				}}))
			})
		})
		Describe("Test if user is allowed", func() {
			It("should return true for admin with editor", func() {
				ok, err := repo.IsUserAllowed(adminUser.ID, []model.Permission{model.PermissionEditor})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
			It("should return false for admin with viewer", func() {
				ok, err := repo.IsUserAllowed(adminUser.ID, []model.Permission{model.PermissionViewer})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
			It("should return false for a different user", func() {
				ok, err := repo.IsUserAllowed("1234567", []model.Permission{model.PermissionEditor})
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
		Describe("Delete permission", func() {
			It("should succeed", func() {
				Expect(repo.Delete(adminUser.ID)).To(Succeed())
			})
		})
		Describe("Permissions are gone", func() {
			It("should return nothing", func() {
				perms, err := repo.GetAll()
				Expect(err).ToNot(HaveOccurred())
				Expect(perms).To(BeEmpty())
			})
		})
	})

})
