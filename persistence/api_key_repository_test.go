package persistence

import (
	"context"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("APIKeyRepository", func() {
	var repo model.APIKeyRepository

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: true})
		repo = NewAPIKeyRepository(ctx, GetDBXBuilder())
	})

	Describe("Put", func() {
		It("sets an ID if it is not set", func() {
			apiKey := &model.APIKey{
				UserID: "userid",
				Name:   "Test API Key",
				Key:    "test-key",
			}

			err := repo.Put(apiKey)

			Expect(err).ToNot(HaveOccurred())
			Expect(apiKey.ID).ToNot(BeEmpty())
			Expect(apiKey.CreatedAt).ToNot(BeZero())
		})

		It("keeps existing values", func() {
			apiKey := &model.APIKey{
				ID:     "existing-id",
				UserID: "userid",
				Name:   "Test API Key 2",
				Key:    "test-key-2",
			}

			err := repo.Put(apiKey)

			Expect(err).ToNot(HaveOccurred())
			Expect(apiKey.ID).To(Equal("existing-id"))
			Expect(apiKey.CreatedAt).ToNot(BeZero())
		})
	})

	Describe("FindByKey", func() {
		It("returns the API key with matching key", func() {
			apiKey := &model.APIKey{
				UserID: "userid",
				Name:   "Unique API Key",
				Key:    "unique-test-key",
			}

			err := repo.Put(apiKey)
			Expect(err).ToNot(HaveOccurred())

			result, err := repo.FindByKey("unique-test-key")

			Expect(err).ToNot(HaveOccurred())
			Expect(result.ID).To(Equal(apiKey.ID))
			Expect(result.Key).To(Equal("unique-test-key"))
		})

		It("returns error when key not found", func() {
			_, err := repo.FindByKey("non-existent-key")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Save", func() {
		It("creates a new API key with a generated key", func() {
			apiKey := &model.APIKey{
				Name: "Test API Key Save",
			}

			id, err := repo.Save(apiKey)

			Expect(err).ToNot(HaveOccurred())
			Expect(id).ToNot(BeEmpty())
			Expect(apiKey.Key).To(HavePrefix("nav_"))
			Expect(apiKey.UserID).To(Equal("userid"))
		})
	})

	Describe("Update", func() {
		It("only updates the name field", func() {
			apiKey := &model.APIKey{
				UserID: "userid",
				Name:   "Original Name",
				Key:    "test-key-for-update",
			}

			err := repo.Put(apiKey)
			Expect(err).ToNot(HaveOccurred())

			updateKey := &model.APIKey{
				Name:   "Updated Name",
				Key:    "should-not-change",
				UserID: "2222",
			}

			err = repo.Update(apiKey.ID, updateKey)
			Expect(err).ToNot(HaveOccurred())

			result, err := repo.Get(apiKey.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Name).To(Equal("Updated Name"))
			Expect(result.Key).To(Equal("test-key-for-update"))
			Expect(result.UserID).To(Equal("userid"))
		})

		It("returns error when attempting to update non-existent key", func() {
			err := repo.Update("non-existent-id", &model.APIKey{Name: "Updated Name"})
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Delete", func() {
		It("deletes an existing API key", func() {
			apiKey := &model.APIKey{
				UserID: "userid",
				Name:   "API Key to Delete",
				Key:    "key-to-delete",
			}

			err := repo.Put(apiKey)
			Expect(err).ToNot(HaveOccurred())

			err = repo.Delete(apiKey.ID)
			Expect(err).ToNot(HaveOccurred())

			_, err = repo.Get(apiKey.ID)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("User permissions", func() {
		var nonAdminCtx context.Context
		var nonAdminRepo model.APIKeyRepository
		var adminKey model.APIKey

		BeforeEach(func() {
			nonAdminCtx = log.NewContext(context.TODO())
			nonAdminCtx = context.WithValue(nonAdminCtx, "user", model.User{ID: "2222", UserName: "user", IsAdmin: false})
			nonAdminRepo = NewAPIKeyRepository(nonAdminCtx, GetDBXBuilder())

			cleanupKeys := func(key string) {
				foundKey, err := repo.FindByKey(key)
				if err == nil {
					_ = repo.Delete(foundKey.ID)
				}
			}
			cleanupKeys("admin-key")
			cleanupKeys("user-key")

			tmpAdminKey := &model.APIKey{
				UserID: "userid",
				Name:   "Admin's API Key",
				Key:    "admin-key",
			}
			err := repo.Put(tmpAdminKey)
			Expect(err).ToNot(HaveOccurred())
			adminKey = *tmpAdminKey

			userKey := &model.APIKey{
				UserID: "2222",
				Name:   "User's API Key",
				Key:    "user-key",
			}
			err = repo.Put(userKey)
			Expect(err).ToNot(HaveOccurred())
		})

		It("non-admin users can only see their own API keys", func() {
			results, err := nonAdminRepo.GetAll()
			Expect(err).ToNot(HaveOccurred())

			for _, key := range results {
				Expect(key.UserID).To(Equal("2222"))
			}
		})

		It("admin users can see all API keys", func() {
			results, err := repo.GetAll()
			Expect(err).ToNot(HaveOccurred())

			userIds := make(map[string]bool)
			for _, key := range results {
				userIds[key.UserID] = true
			}

			Expect(userIds).To(HaveKey("userid"))
			Expect(userIds).To(HaveKey("2222"))
		})

		It("a user cannot view/delete/update another user's key", func() {
			result, err := nonAdminRepo.Read(adminKey.ID)
			Expect(result).To(BeNil())
			Expect(err).To(MatchError(rest.ErrPermissionDenied))

			updatedKey := &model.APIKey{Name: "new admin key name"}
			err = nonAdminRepo.Update(adminKey.ID, updatedKey)
			Expect(err).To(MatchError(rest.ErrPermissionDenied))

			err = nonAdminRepo.Delete(adminKey.ID)
			Expect(err).To(MatchError(rest.ErrPermissionDenied))
		})
	})
})
