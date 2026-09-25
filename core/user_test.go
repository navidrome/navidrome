package core_test

import (
	"context"
	"errors"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("User Service", func() {
	var service core.User
	var ds *tests.MockDataStore
	var userRepo *tests.MockedUserRepo
	var pluginManager *mockPluginUnloader
	var ctx context.Context

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		userRepo = tests.CreateMockUserRepo()
		ds.MockedUser = userRepo
		pluginManager = &mockPluginUnloader{}
		service = core.NewUser(ds, pluginManager)
		ctx = GinkgoT().Context()
	})

	Describe("NewRepository", func() {
		It("returns a rest.Persistable[model.User]", func() {
			repo := service.Repository()
			_, ok := repo.(rest.Persistable[model.User])
			Expect(ok).To(BeTrue())
		})
	})

	Describe("Delete", func() {
		var repo rest.Persistable[model.User]

		BeforeEach(func() {
			r := service.Repository()
			repo = r.(rest.Persistable[model.User])

			// Add a test user
			user := &model.User{
				ID:       "user-123",
				UserName: "testuser",
				IsAdmin:  false,
			}
			user.NewPassword = "password"
			Expect(userRepo.Put(ctx, user)).To(Succeed())
		})

		It("deletes the user successfully", func() {
			err := repo.Delete(ctx, "user-123")
			Expect(err).NotTo(HaveOccurred())

			// Verify user is deleted
			_, err = userRepo.Get(ctx, "user-123")
			Expect(err).To(Equal(model.ErrNotFound))
		})

		It("calls UnloadDisabledPlugins after successful deletion", func() {
			err := repo.Delete(ctx, "user-123")
			Expect(err).NotTo(HaveOccurred())
			Expect(pluginManager.unloadCalls).To(Equal(1))
		})

		It("still calls UnloadDisabledPlugins when deletion fails", func() {
			err := repo.Delete(ctx, "non-existent")
			Expect(err).To(MatchError(model.ErrNotFound))
			Expect(pluginManager.unloadCalls).To(Equal(1))
		})

		It("unloads plugins when a bulk delete fails after removing earlier users", func() {
			err := repo.Delete(ctx, "user-123", "non-existent")
			Expect(err).To(MatchError(model.ErrNotFound))

			_, err = userRepo.Get(ctx, "user-123")
			Expect(err).To(Equal(model.ErrNotFound))
			Expect(pluginManager.unloadCalls).To(Equal(1))
		})

		It("returns error when repository fails", func() {
			userRepo.Error = errors.New("database error")
			err := repo.Delete(ctx, "user-123")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("database error"))
			Expect(pluginManager.unloadCalls).To(Equal(1))
		})
	})
})
