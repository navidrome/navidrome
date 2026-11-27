package core_test

import (
	"context"
	"errors"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("User Service", func() {
	var service core.User
	var ds model.DataStore
	var pluginManager *mockPluginUnloader
	var ctx context.Context

	BeforeEach(func() {
		ctx = request.WithUser(GinkgoT().Context(), model.User{ID: "admin-1", IsAdmin: true})
		ds = persistence.New(db.Db())
		pluginManager = &mockPluginUnloader{}
		service = core.NewUser(ds, pluginManager)
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
			Expect(ds.User().Put(ctx, user)).To(Succeed())
		})

		It("deletes the user successfully", func() {
			err := repo.Delete(ctx, "user-123")
			Expect(err).NotTo(HaveOccurred())

			// Verify user is deleted
			_, err = ds.User().Get(ctx, "user-123")
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

			_, err = ds.User().Get(ctx, "user-123")
			Expect(err).To(Equal(model.ErrNotFound))
			Expect(pluginManager.unloadCalls).To(Equal(1))
		})

		It("returns error when repository fails", func() {
			failing := persistencetest.Stub(ds, persistencetest.Repos{User: func(r model.UserRepository) model.UserRepository {
				return &failingUserDeleteRepo{UserRepository: r}
			}})
			repo := core.NewUser(failing, pluginManager).Repository().(rest.Persistable[model.User])
			err := repo.Delete(ctx, "user-123")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("database error"))
			Expect(pluginManager.unloadCalls).To(Equal(1))
		})
	})
})

type failingUserDeleteRepo struct{ model.UserRepository }

func (r *failingUserDeleteRepo) Delete(context.Context, ...string) error {
	return errors.New("database error")
}
