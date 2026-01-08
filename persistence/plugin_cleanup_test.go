package persistence

import (
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin Cleanup", func() {
	var pluginRepo model.PluginRepository
	var userRepo model.UserRepository
	var libraryRepo model.LibraryRepository

	BeforeEach(func() {
		ctx := GinkgoT().Context()
		ctx = request.WithUser(ctx, model.User{ID: "admin", UserName: "admin", IsAdmin: true})
		db := GetDBXBuilder()
		pluginRepo = NewPluginRepository(ctx, db)
		userRepo = NewUserRepository(ctx, db)
		libraryRepo = NewLibraryRepository(ctx, db)

		// Clean up any existing plugins
		all, _ := pluginRepo.GetAll()
		for _, p := range all {
			_ = pluginRepo.Delete(p.ID)
		}
	})

	AfterEach(func() {
		// Clean up after tests
		all, _ := pluginRepo.GetAll()
		for _, p := range all {
			_ = pluginRepo.Delete(p.ID)
		}
	})

	Describe("cleanupPluginUserReferences", func() {
		It("removes user ID from plugin users array", func() {
			// Create a plugin with multiple users
			plugin := &model.Plugin{
				ID:       "test-plugin",
				Path:     "/plugins/test.wasm",
				Manifest: `{"name":"test"}`,
				SHA256:   "abc123",
				Users:    `["user1","user2","user3"]`,
				Enabled:  true,
			}
			Expect(pluginRepo.Put(plugin)).To(Succeed())

			// Clean up user2 reference
			db := GetDBXBuilder()
			Expect(cleanupPluginUserReferences(db, "user2")).To(Succeed())

			// Verify user2 was removed
			updated, err := pluginRepo.Get("test-plugin")
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Users).To(Equal(`["user1","user3"]`))
			Expect(updated.Enabled).To(BeTrue()) // Still has users, should remain enabled
		})

		It("auto-disables plugin when last permitted user is removed", func() {
			// Create a plugin that requires users permission with only one user
			plugin := &model.Plugin{
				ID:       "user-plugin",
				Path:     "/plugins/user.wasm",
				Manifest: `{"name":"user-plugin","permissions":{"users":{}}}`,
				SHA256:   "def456",
				Users:    `["only-user"]`,
				AllUsers: false,
				Enabled:  true,
			}
			Expect(pluginRepo.Put(plugin)).To(Succeed())

			// Remove the only user
			db := GetDBXBuilder()
			Expect(cleanupPluginUserReferences(db, "only-user")).To(Succeed())

			// Verify plugin was auto-disabled
			updated, err := pluginRepo.Get("user-plugin")
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Users).To(Equal(`[]`))
			Expect(updated.Enabled).To(BeFalse())
		})

		It("does not disable plugin when allUsers is true", func() {
			plugin := &model.Plugin{
				ID:       "all-users-plugin",
				Path:     "/plugins/all.wasm",
				Manifest: `{"name":"all-users","permissions":{"users":{}}}`,
				SHA256:   "ghi789",
				Users:    `["user1"]`,
				AllUsers: true,
				Enabled:  true,
			}
			Expect(pluginRepo.Put(plugin)).To(Succeed())

			// Remove the user (but allUsers is true)
			db := GetDBXBuilder()
			Expect(cleanupPluginUserReferences(db, "user1")).To(Succeed())

			// Plugin should still be enabled because allUsers is true
			updated, err := pluginRepo.Get("all-users-plugin")
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Enabled).To(BeTrue())
		})

		It("does not affect plugins without users permission requirement", func() {
			plugin := &model.Plugin{
				ID:       "no-users-perm",
				Path:     "/plugins/noperm.wasm",
				Manifest: `{"name":"no-perm"}`, // No permissions.users in manifest
				SHA256:   "jkl012",
				Users:    `["user1"]`,
				Enabled:  true,
			}
			Expect(pluginRepo.Put(plugin)).To(Succeed())

			// Remove the user
			db := GetDBXBuilder()
			Expect(cleanupPluginUserReferences(db, "user1")).To(Succeed())

			// Plugin should still be enabled (no users permission requirement)
			updated, err := pluginRepo.Get("no-users-perm")
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Users).To(Equal(`[]`))
			Expect(updated.Enabled).To(BeTrue())
		})
	})

	Describe("cleanupPluginLibraryReferences", func() {
		It("removes library ID from plugin libraries array", func() {
			// Create a plugin with multiple libraries
			plugin := &model.Plugin{
				ID:        "lib-plugin",
				Path:      "/plugins/lib.wasm",
				Manifest:  `{"name":"lib-plugin"}`,
				SHA256:    "mno345",
				Libraries: `[1,2,3]`,
				Enabled:   true,
			}
			Expect(pluginRepo.Put(plugin)).To(Succeed())

			// Clean up library 2 reference
			db := GetDBXBuilder()
			Expect(cleanupPluginLibraryReferences(db, 2)).To(Succeed())

			// Verify library 2 was removed
			updated, err := pluginRepo.Get("lib-plugin")
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Libraries).To(Equal(`[1,3]`))
		})

		It("auto-disables plugin when last permitted library is removed", func() {
			// Create a plugin that requires library permission with only one library
			plugin := &model.Plugin{
				ID:           "lib-only-plugin",
				Path:         "/plugins/libonly.wasm",
				Manifest:     `{"name":"lib-only","permissions":{"library":{}}}`,
				SHA256:       "pqr678",
				Libraries:    `[99]`,
				AllLibraries: false,
				Enabled:      true,
			}
			Expect(pluginRepo.Put(plugin)).To(Succeed())

			// Remove the only library
			db := GetDBXBuilder()
			Expect(cleanupPluginLibraryReferences(db, 99)).To(Succeed())

			// Verify plugin was auto-disabled
			updated, err := pluginRepo.Get("lib-only-plugin")
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Libraries).To(Equal(`[]`))
			Expect(updated.Enabled).To(BeFalse())
		})

		It("does not disable plugin when allLibraries is true", func() {
			plugin := &model.Plugin{
				ID:           "all-libs-plugin",
				Path:         "/plugins/alllibs.wasm",
				Manifest:     `{"name":"all-libs","permissions":{"library":{}}}`,
				SHA256:       "stu901",
				Libraries:    `[1]`,
				AllLibraries: true,
				Enabled:      true,
			}
			Expect(pluginRepo.Put(plugin)).To(Succeed())

			// Remove the library (but allLibraries is true)
			db := GetDBXBuilder()
			Expect(cleanupPluginLibraryReferences(db, 1)).To(Succeed())

			// Plugin should still be enabled
			updated, err := pluginRepo.Get("all-libs-plugin")
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Enabled).To(BeTrue())
		})
	})

	Describe("User Delete integration", func() {
		It("cleans up plugin references when user is deleted", func() {
			// Create a test user
			user := &model.User{
				ID:       "test-delete-user",
				UserName: "plugin-cleanup-test-user",
				IsAdmin:  false,
			}
			user.NewPassword = "password123"
			Expect(userRepo.Put(user)).To(Succeed())

			// Create a plugin referencing this user
			plugin := &model.Plugin{
				ID:       "user-ref-plugin",
				Path:     "/plugins/userref.wasm",
				Manifest: `{"name":"user-ref"}`,
				SHA256:   "xyz123",
				Users:    `["test-delete-user","other-user"]`,
				Enabled:  true,
			}
			Expect(pluginRepo.Put(plugin)).To(Succeed())

			// Delete the user
			Expect(userRepo.Delete("test-delete-user")).To(Succeed())

			// Verify user was removed from plugin
			updated, err := pluginRepo.Get("user-ref-plugin")
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Users).To(Equal(`["other-user"]`))
		})
	})

	Describe("Library Delete integration", func() {
		It("cleans up plugin references when library is deleted", func() {
			// Create a test library (ID > 1 since ID 1 cannot be deleted)
			library := &model.Library{
				ID:   99,
				Name: "Test Library",
				Path: "/tmp/test-lib",
			}
			Expect(libraryRepo.Put(library)).To(Succeed())

			// Create a plugin referencing this library
			plugin := &model.Plugin{
				ID:        "lib-ref-plugin",
				Path:      "/plugins/libref.wasm",
				Manifest:  `{"name":"lib-ref"}`,
				SHA256:    "abc789",
				Libraries: `[99,1]`,
				Enabled:   true,
			}
			Expect(pluginRepo.Put(plugin)).To(Succeed())

			// Delete the library
			Expect(libraryRepo.Delete(99)).To(Succeed())

			// Verify library was removed from plugin
			updated, err := pluginRepo.Get("lib-ref-plugin")
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Libraries).To(Equal(`[1]`))
		})
	})
})
