package persistence

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ShareRepository", func() {
	var repo model.ShareRepository
	var ctx context.Context
	var adminUser = model.User{ID: "admin", UserName: "admin", IsAdmin: true}

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = request.WithUser(log.NewContext(context.TODO()), adminUser)
		repo = NewShareRepository(ctx, GetDBXBuilder())

		// Insert the admin user into the database (required for foreign key constraint)
		ur := NewUserRepository(ctx, GetDBXBuilder())
		err := ur.Put(&adminUser)
		Expect(err).ToNot(HaveOccurred())

		// Clean up shares
		db := GetDBXBuilder()
		_, err = db.NewQuery("DELETE FROM share").Execute()
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("Headless Access", func() {
		Context("Repository creation and basic operations", func() {
			It("should create repository successfully with no user context", func() {
				// Create repository with no user context (headless)
				headlessRepo := NewShareRepository(context.Background(), GetDBXBuilder())
				Expect(headlessRepo).ToNot(BeNil())
			})

			It("should handle GetAll for headless processes", func() {
				// Create a simple share directly in database
				shareID := "headless-test-share"
				_, err := GetDBXBuilder().NewQuery(`
					INSERT INTO share (id, user_id, description, resource_type, resource_ids, created_at, updated_at) 
					VALUES ({:id}, {:user}, {:desc}, {:type}, {:ids}, {:created}, {:updated})
				`).Bind(map[string]interface{}{
					"id":      shareID,
					"user":    adminUser.ID,
					"desc":    "Headless Test Share",
					"type":    "song",
					"ids":     "song-1",
					"created": time.Now(),
					"updated": time.Now(),
				}).Execute()
				Expect(err).ToNot(HaveOccurred())

				// Headless process should see all shares
				headlessRepo := NewShareRepository(context.Background(), GetDBXBuilder())
				shares, err := headlessRepo.GetAll()
				Expect(err).ToNot(HaveOccurred())

				found := false
				for _, s := range shares {
					if s.ID == shareID {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue(), "Headless process should see all shares")
			})

			It("should handle individual share retrieval for headless processes", func() {
				// Create a simple share
				shareID := "headless-get-share"
				_, err := GetDBXBuilder().NewQuery(`
					INSERT INTO share (id, user_id, description, resource_type, resource_ids, created_at, updated_at) 
					VALUES ({:id}, {:user}, {:desc}, {:type}, {:ids}, {:created}, {:updated})
				`).Bind(map[string]interface{}{
					"id":      shareID,
					"user":    adminUser.ID,
					"desc":    "Headless Get Share",
					"type":    "song",
					"ids":     "song-2",
					"created": time.Now(),
					"updated": time.Now(),
				}).Execute()
				Expect(err).ToNot(HaveOccurred())

				// Headless process should be able to get the share
				headlessRepo := NewShareRepository(context.Background(), GetDBXBuilder())
				share, err := headlessRepo.Get(shareID)
				Expect(err).ToNot(HaveOccurred())
				Expect(share.ID).To(Equal(shareID))
				Expect(share.Description).To(Equal("Headless Get Share"))
			})
		})
	})

	Describe("SQL ambiguity fix verification", func() {
		It("should handle share operations without SQL ambiguity errors", func() {
			// This test verifies that the loadMedia function doesn't cause SQL ambiguity
			// The key fix was using "album.id" instead of "id" in the album query filters

			// Create a share that would trigger the loadMedia function
			shareID := "sql-test-share"
			_, err := GetDBXBuilder().NewQuery(`
				INSERT INTO share (id, user_id, description, resource_type, resource_ids, created_at, updated_at) 
				VALUES ({:id}, {:user}, {:desc}, {:type}, {:ids}, {:created}, {:updated})
			`).Bind(map[string]interface{}{
				"id":      shareID,
				"user":    adminUser.ID,
				"desc":    "SQL Test Share",
				"type":    "album",
				"ids":     "non-existent-album", // Won't find albums, but shouldn't cause SQL errors
				"created": time.Now(),
				"updated": time.Now(),
			}).Execute()
			Expect(err).ToNot(HaveOccurred())

			// The Get operation should work without SQL ambiguity errors
			// even if no albums are found
			share, err := repo.Get(shareID)
			Expect(err).ToNot(HaveOccurred())
			Expect(share.ID).To(Equal(shareID))
			// Albums array should be empty since we used non-existent album ID
			Expect(share.Albums).To(BeEmpty())
		})
	})
})
