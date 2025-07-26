package persistence

import (
	"context"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pocketbase/dbx"
)

var _ = Describe("APIKeyRepository", func() {
	var repo model.APIKeyRepository
	var playerRepo model.PlayerRepository
	var database *dbx.DB

	var (
		adminPlayer   = model.Player{ID: "1", Name: "NavidromeUI [Firefox/Linux]", UserAgent: "Firefox/Linux", UserId: adminUser.ID, Username: adminUser.UserName, Client: "NavidromeUI", IP: "127.0.0.1", ReportRealPath: true, ScrobbleEnabled: true}
		regularPlayer = model.Player{ID: "3", Name: "NavidromeUI [Safari/macOS]", UserAgent: "Safari/macOS", UserId: regularUser.ID, Username: regularUser.UserName, Client: "NavidromeUI", ReportRealPath: true, ScrobbleEnabled: false}

		players = model.Players{adminPlayer, regularPlayer}
	)

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, adminUser)
		database = GetDBXBuilder()

		playerRepo = NewPlayerRepository(ctx, database)
		for idx := range players {
			err := playerRepo.Put(&players[idx])
			Expect(err).To(BeNil())
		}
		repo = NewAPIKeyRepository(ctx, database)
	})

	Describe("FindByKey", func() {
		It("returns the API key with matching key", func() {
			apiKey := &model.APIKey{
				PlayerID: adminPlayer.ID,
				Name:     "Unique API Key",
			}
			apiKeyId, err := repo.Save(apiKey)
			Expect(err).ToNot(HaveOccurred())
			apiKey, err = repo.Get(apiKeyId)
			Expect(err).ToNot(HaveOccurred())

			result, err := repo.FindByKey(apiKey.Key)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.ID).To(Equal(apiKey.ID))
			Expect(result.Key).To(Equal(apiKey.Key))
		})

		It("returns error when key not found", func() {
			_, err := repo.FindByKey("non-existent-key")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Save", func() {
		It("creates a new API key with a generated key", func() {
			apiKey := &model.APIKey{
				Name:     "Test API Key Save",
				PlayerID: adminPlayer.ID,
			}

			id, err := repo.Save(apiKey)

			Expect(err).ToNot(HaveOccurred())
			Expect(id).ToNot(BeEmpty())
			Expect(apiKey.Key).To(HavePrefix("nav_"))
			Expect(apiKey.PlayerID).To(Equal(adminPlayer.ID))
			Expect(apiKey.Name).To(Equal("Test API Key Save"))
		})
	})

	Describe("Update", func() {
		It("only updates the name field", func() {
			apiKey := &model.APIKey{
				PlayerID: adminPlayer.ID,
				Name:     "Original Name",
			}

			_, err := repo.Save(apiKey)
			Expect(err).ToNot(HaveOccurred())

			updateKey := &model.APIKey{
				Name:     "Updated Name",
				PlayerID: regularPlayer.ID,
			}

			err = repo.Update(apiKey.ID, updateKey)
			Expect(err).ToNot(HaveOccurred())

			result, err := repo.Get(apiKey.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Name).To(Equal("Updated Name"))
			Expect(result.Key).To(Equal(apiKey.Key))
			Expect(result.PlayerID).To(Equal(adminPlayer.ID))
		})

		It("returns error when attempting to update non-existent key", func() {
			err := repo.Update("non-existent-id", &model.APIKey{Name: "Updated Name"})
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Delete", func() {
		It("deletes an existing API key", func() {
			apiKey := &model.APIKey{
				PlayerID: adminPlayer.ID,
				Name:     "API Key to Delete",
			}

			_, err := repo.Save(apiKey)
			Expect(err).ToNot(HaveOccurred())

			err = repo.Delete(apiKey.ID)
			Expect(err).ToNot(HaveOccurred())

			_, err = repo.Get(apiKey.ID)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("RefreshKey", func() {
		It("generates a new key for an existing API key", func() {
			apiKey := &model.APIKey{
				PlayerID: adminPlayer.ID,
				Name:     "Test Refresh",
			}
			_, err := repo.Save(apiKey)
			Expect(err).ToNot(HaveOccurred())

			originalKey := apiKey.Key

			newKey, err := repo.RefreshKey(apiKey.ID)

			Expect(err).ToNot(HaveOccurred())
			Expect(newKey).ToNot(BeEmpty())
			Expect(newKey).ToNot(Equal(originalKey))
			Expect(newKey).To(HavePrefix("nav_"))

			refreshed, err := repo.Get(apiKey.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(refreshed.Key).To(Equal(newKey))
		})

		It("returns an error for non-existent API key", func() {
			_, err := repo.RefreshKey("non-existent-id")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(rest.ErrNotFound))
		})

		It("enforces user permissions", func() {
			apiKey := &model.APIKey{
				PlayerID: adminPlayer.ID,
				Name:     "Test Permission",
			}
			_, err := repo.Save(apiKey)
			Expect(err).ToNot(HaveOccurred())

			nonAdminCtx := log.NewContext(context.TODO())
			nonAdminCtx = request.WithUser(nonAdminCtx, regularUser)
			nonAdminRepo := NewAPIKeyRepository(nonAdminCtx, database)

			_, err = nonAdminRepo.RefreshKey(apiKey.ID)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(rest.ErrPermissionDenied))
		})
	})

	Describe("User permissions", func() {
		var nonAdminCtx context.Context
		var nonAdminRepo model.APIKeyRepository
		var adminKey model.APIKey

		BeforeEach(func() {
			nonAdminCtx = log.NewContext(context.TODO())
			nonAdminCtx = request.WithUser(nonAdminCtx, regularUser)
			nonAdminRepo = NewAPIKeyRepository(nonAdminCtx, database)

			cleanupKeys := func(key string) {
				foundKey, err := repo.FindByKey(key)
				if err == nil {
					_ = repo.Delete(foundKey.ID)
				}
			}
			cleanupKeys("admin-key")
			cleanupKeys("user-key")

			tmpAdminKey := &model.APIKey{
				PlayerID: adminPlayer.ID,
				Name:     "Admin's API Key",
			}
			_, err := repo.Save(tmpAdminKey)
			Expect(err).ToNot(HaveOccurred())
			adminKey = *tmpAdminKey

			userKey := &model.APIKey{
				PlayerID: regularPlayer.ID,
				Name:     "User's API Key",
			}
			_, err = repo.Save(userKey)
			Expect(err).ToNot(HaveOccurred())
		})

		It("non-admin users can only see their own API keys", func() {
			results, err := nonAdminRepo.GetAll()
			Expect(err).ToNot(HaveOccurred())

			for _, key := range results {
				Expect(key.PlayerID).To(Equal(regularPlayer.ID))
			}
		})

		It("admin users can see all API keys", func() {
			results, err := repo.GetAll()
			Expect(err).ToNot(HaveOccurred())

			userIds := make(map[string]bool)
			for _, key := range results {
				userIds[key.PlayerID] = true
			}

			Expect(userIds).To(HaveKey(adminPlayer.ID))
			Expect(userIds).To(HaveKey(regularPlayer.ID))
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
