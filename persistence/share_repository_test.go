package persistence

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
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
		ctx = request.WithUser(log.NewContext(GinkgoT().Context()), adminUser)
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
				headlessRepo := NewShareRepository(GinkgoT().Context(), GetDBXBuilder())
				Expect(headlessRepo).ToNot(BeNil())
			})

			It("should handle GetAll for headless processes", func() {
				// Create a simple share directly in database
				shareID := "headless-test-share"
				_, err := GetDBXBuilder().NewQuery(`
					INSERT INTO share (id, user_id, description, resource_type, resource_ids, created_at, updated_at) 
					VALUES ({:id}, {:user}, {:desc}, {:type}, {:ids}, {:created}, {:updated})
				`).Bind(map[string]any{
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
				headlessRepo := NewShareRepository(GinkgoT().Context(), GetDBXBuilder())
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
				`).Bind(map[string]any{
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
				headlessRepo := NewShareRepository(GinkgoT().Context(), GetDBXBuilder())
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
			`).Bind(map[string]any{
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

	Describe("Playlist share library scoping", func() {
		var otherLib model.Library
		var owner model.User
		var plsID string

		BeforeEach(func() {
			adminCtx := request.WithUser(log.NewContext(GinkgoT().Context()), adminUser)

			// A second library the owner has no access to, plus a track in it
			lr := NewLibraryRepository(adminCtx, GetDBXBuilder())
			otherLib = model.Library{ID: 0, Name: "Share Other Library", Path: "/share/other/lib"}
			Expect(lr.Put(&otherLib)).To(Succeed())
			mr := NewMediaFileRepository(adminCtx, GetDBXBuilder())
			Expect(mr.Put(&model.MediaFile{ID: "share-other", LibraryID: otherLib.ID, Path: "s/other.mp3", Title: "ShareOther"})).To(Succeed())
			Expect(mr.Put(&model.MediaFile{ID: "share-ok", LibraryID: 1, Path: "s/ok.mp3", Title: "ShareOK"})).To(Succeed())

			// Non-admin owner with access to library 1 only
			owner = createUserWithLibraries("share-owner", []int{1})
			ur := NewUserRepository(adminCtx, GetDBXBuilder())
			Expect(ur.Put(&owner)).To(Succeed())
			Expect(ur.SetUserLibraries(owner.ID, []int{1})).To(Succeed())

			// Owner-owned playlist containing tracks from both libraries
			plsID = "share-scope-pls"
			ownerCtx := request.WithUser(log.NewContext(GinkgoT().Context()), owner)
			pr := NewPlaylistRepository(ownerCtx, GetDBXBuilder())
			pls := &model.Playlist{ID: plsID, Name: "Scope Test", OwnerID: owner.ID}
			pls.AddMediaFiles(model.MediaFiles{{ID: "share-ok"}, {ID: "share-other"}})
			Expect(pr.Put(pls)).To(Succeed())

			// Share row owned by the non-admin owner
			_, err := GetDBXBuilder().NewQuery(`
				INSERT INTO share (id, user_id, description, resource_type, resource_ids, created_at, updated_at)
				VALUES ({:id}, {:user}, {:desc}, {:type}, {:ids}, {:created}, {:updated})
			`).Bind(map[string]any{
				"id": "share-scope", "user": owner.ID, "desc": "Scope test share",
				"type": "playlist", "ids": plsID, "created": time.Now(), "updated": time.Now(),
			}).Execute()
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			adminCtx := request.WithUser(log.NewContext(GinkgoT().Context()), adminUser)
			b := GetDBXBuilder()
			_, _ = b.NewQuery(`DELETE FROM share WHERE id = 'share-scope'`).Execute()
			pr := NewPlaylistRepository(adminCtx, b)
			_ = pr.Delete(plsID)
			mr := NewMediaFileRepository(adminCtx, b).(*mediaFileRepository)
			_, _ = mr.executeSQL(squirrel.Delete("media_file").Where(squirrel.Eq{"id": []string{"share-other", "share-ok"}}))
			lr := NewLibraryRepository(adminCtx, b).(*libraryRepository)
			_ = lr.delete(squirrel.Eq{"id": otherLib.ID})
			_ = NewUserRepository(adminCtx, b).Delete(owner.ID)
		})

		It("excludes tracks the owner cannot access from the shared playlist", func() {
			// Read the share as admin (mimics the public-share render path, which uses
			// the share repository's own context). loadMedia must scope to the owner.
			adminRepo := NewShareRepository(request.WithUser(log.NewContext(GinkgoT().Context()), adminUser), GetDBXBuilder())
			share, err := adminRepo.Get("share-scope")
			Expect(err).ToNot(HaveOccurred())

			Expect(share.Tracks).To(ContainElement(HaveField("ID", "share-ok")))
			Expect(share.Tracks).ToNot(ContainElement(HaveField("ID", "share-other")),
				"a track outside the owner's libraries must not appear in the share")
		})

		It("returns no tracks when the playlist is not visible to the owner", func() {
			// A private playlist owned by someone else: the share owner can no longer
			// see it, so Tracks() returns nil. The share must render with no tracks
			// instead of panicking.
			privatePlsID := "private-pls"
			adminCtx := request.WithUser(log.NewContext(GinkgoT().Context()), adminUser)
			pr := NewPlaylistRepository(adminCtx, GetDBXBuilder())
			privatePls := &model.Playlist{ID: privatePlsID, Name: "Private", OwnerID: adminUser.ID, Public: false}
			privatePls.AddMediaFiles(model.MediaFiles{{ID: "share-ok"}})
			Expect(pr.Put(privatePls)).To(Succeed())
			DeferCleanup(func() { _ = pr.Delete(privatePlsID) })

			_, err := GetDBXBuilder().NewQuery(`
				INSERT INTO share (id, user_id, description, resource_type, resource_ids, created_at, updated_at)
				VALUES ({:id}, {:user}, {:desc}, {:type}, {:ids}, {:created}, {:updated})
			`).Bind(map[string]any{
				"id": "share-private", "user": owner.ID, "desc": "Private share",
				"type": "playlist", "ids": privatePlsID, "created": time.Now(), "updated": time.Now(),
			}).Execute()
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(func() { _, _ = GetDBXBuilder().NewQuery(`DELETE FROM share WHERE id = 'share-private'`).Execute() })

			adminRepo := NewShareRepository(adminCtx, GetDBXBuilder())
			share, err := adminRepo.Get("share-private")
			Expect(err).ToNot(HaveOccurred())
			Expect(share.Tracks).To(BeEmpty())
		})
	})

	Describe("Ownership Checks", func() {
		var ownerUser = model.User{ID: "2222", UserName: "regular-user"}
		var otherUser = model.User{ID: "3333", UserName: "third-user"}

		insertShare := func(shareID, userID string) {
			_, err := GetDBXBuilder().NewQuery(`
				INSERT INTO share (id, user_id, description, resource_type, resource_ids, created_at, updated_at)
				VALUES ({:id}, {:user}, {:desc}, {:type}, {:ids}, {:created}, {:updated})
			`).Bind(map[string]any{
				"id":      shareID,
				"user":    userID,
				"desc":    "Test Share",
				"type":    "media_file",
				"ids":     "1001",
				"created": time.Now(),
				"updated": time.Now(),
			}).Execute()
			Expect(err).ToNot(HaveOccurred())
		}

		Describe("Delete", func() {
			It("allows a non-admin user to delete their own share", func() {
				insertShare("own-share-del", ownerUser.ID)
				ctx := request.WithUser(log.NewContext(GinkgoT().Context()), ownerUser)
				repo := NewShareRepository(ctx, GetDBXBuilder())
				err := repo.(rest.Persistable).Delete("own-share-del")
				Expect(err).ToNot(HaveOccurred())
			})

			It("denies a non-admin user from deleting another user's share", func() {
				insertShare("other-share-del", ownerUser.ID)
				ctx := request.WithUser(log.NewContext(GinkgoT().Context()), otherUser)
				repo := NewShareRepository(ctx, GetDBXBuilder())
				err := repo.(rest.Persistable).Delete("other-share-del")
				Expect(err).To(Equal(rest.ErrPermissionDenied))

				// The share was not deleted: the owner can still read it.
				ownerCtx := request.WithUser(log.NewContext(GinkgoT().Context()), ownerUser)
				ownerRepo := NewShareRepository(ownerCtx, GetDBXBuilder())
				_, err = ownerRepo.(rest.Repository).Read("other-share-del")
				Expect(err).ToNot(HaveOccurred())
			})

			It("allows an admin to delete any user's share", func() {
				insertShare("admin-del-share", ownerUser.ID)
				ctx := request.WithUser(log.NewContext(GinkgoT().Context()), adminUser)
				repo := NewShareRepository(ctx, GetDBXBuilder())
				err := repo.(rest.Persistable).Delete("admin-del-share")
				Expect(err).ToNot(HaveOccurred())
			})

			It("allows headless context (no user) to delete a share", func() {
				insertShare("headless-del-share", ownerUser.ID)
				repo := NewShareRepository(GinkgoT().Context(), GetDBXBuilder())
				err := repo.(rest.Persistable).Delete("headless-del-share")
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Describe("Update", func() {
			It("allows a non-admin user to update their own share", func() {
				insertShare("own-share-upd", ownerUser.ID)
				ctx := request.WithUser(log.NewContext(GinkgoT().Context()), ownerUser)
				repo := NewShareRepository(ctx, GetDBXBuilder())
				err := repo.(rest.Persistable).Update("own-share-upd", &model.Share{Description: "Updated"}, "description")
				Expect(err).ToNot(HaveOccurred())
			})

			It("denies a non-admin user from updating another user's share", func() {
				insertShare("other-share-upd", ownerUser.ID)
				ctx := request.WithUser(log.NewContext(GinkgoT().Context()), otherUser)
				repo := NewShareRepository(ctx, GetDBXBuilder())
				err := repo.(rest.Persistable).Update("other-share-upd", &model.Share{Description: "Hacked"}, "description")
				Expect(err).To(Equal(rest.ErrPermissionDenied))
			})

			It("allows an admin to update any user's share", func() {
				insertShare("admin-upd-share", ownerUser.ID)
				ctx := request.WithUser(log.NewContext(GinkgoT().Context()), adminUser)
				repo := NewShareRepository(ctx, GetDBXBuilder())
				err := repo.(rest.Persistable).Update("admin-upd-share", &model.Share{Description: "Admin Updated"}, "description")
				Expect(err).ToNot(HaveOccurred())
			})

			It("allows headless context (no user) to update a share", func() {
				insertShare("headless-upd-share", ownerUser.ID)
				repo := NewShareRepository(GinkgoT().Context(), GetDBXBuilder())
				err := repo.(rest.Persistable).Update("headless-upd-share", &model.Share{Description: "Headless"}, "description")
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns not found when updating a nonexistent share", func() {
				ctx := request.WithUser(log.NewContext(context.TODO()), ownerUser)
				repo := NewShareRepository(ctx, GetDBXBuilder())
				err := repo.(rest.Persistable).Update("does-not-exist", &model.Share{Description: "Ghost"}, "description")
				Expect(err).To(Equal(rest.ErrNotFound))
			})

			It("updates all columns when no specific columns are given", func() {
				insertShare("all-cols-share", ownerUser.ID)
				ctx := request.WithUser(log.NewContext(context.TODO()), ownerUser)
				repo := NewShareRepository(ctx, GetDBXBuilder())
				// No cols: the update must write every column, not just updated_at.
				err := repo.(rest.Persistable).Update("all-cols-share",
					&model.Share{Description: "All Updated", MaxBitRate: 192, ResourceType: "album", ResourceIDs: "2002"})
				Expect(err).ToNot(HaveOccurred())

				got, err := repo.(rest.Repository).Read("all-cols-share")
				Expect(err).ToNot(HaveOccurred())
				share := got.(*model.Share)
				Expect(share.Description).To(Equal("All Updated"))
				Expect(share.MaxBitRate).To(Equal(192))
				Expect(share.ResourceType).To(Equal("album"))
			})

			It("does not let an owner reassign their share to another user", func() {
				insertShare("reassign-share", ownerUser.ID)
				ctx := request.WithUser(log.NewContext(context.TODO()), ownerUser)
				repo := NewShareRepository(ctx, GetDBXBuilder())
				err := repo.(rest.Persistable).Update("reassign-share",
					&model.Share{UserID: otherUser.ID, Description: "Given away"}, "user_id", "description")
				Expect(err).ToNot(HaveOccurred())

				// Ownership must not have moved, even though user_id was passed in the body and cols.
				got, err := repo.(rest.Repository).Read("reassign-share")
				Expect(err).ToNot(HaveOccurred())
				Expect(got.(*model.Share).UserID).To(Equal(ownerUser.ID))
			})
		})

		Describe("Read scoping", func() {
			BeforeEach(func() {
				// Persist owner/other users so the JOIN in selectShare resolves.
				ur := NewUserRepository(ctx, GetDBXBuilder())
				Expect(ur.Put(&ownerUser)).To(Succeed())
				Expect(ur.Put(&otherUser)).To(Succeed())

				insertShare("share-owner-1", ownerUser.ID)
				insertShare("share-owner-2", ownerUser.ID)
				insertShare("share-other-1", otherUser.ID)
			})

			Context("non-admin user", func() {
				var nonAdminRepo model.ShareRepository
				var nonAdminRest rest.Repository

				BeforeEach(func() {
					nonAdminCtx := request.WithUser(log.NewContext(GinkgoT().Context()), ownerUser)
					nonAdminRepo = NewShareRepository(nonAdminCtx, GetDBXBuilder())
					nonAdminRest = nonAdminRepo.(rest.Repository)
				})

				It("GetAll returns only own shares", func() {
					shares, err := nonAdminRepo.GetAll()
					Expect(err).ToNot(HaveOccurred())
					ids := make([]string, len(shares))
					for i, s := range shares {
						ids[i] = s.ID
					}
					Expect(ids).To(ConsistOf("share-owner-1", "share-owner-2"))
				})

				It("ReadAll returns only own shares", func() {
					res, err := nonAdminRest.ReadAll()
					Expect(err).ToNot(HaveOccurred())
					shares := res.(model.Shares)
					ids := make([]string, len(shares))
					for i, s := range shares {
						ids[i] = s.ID
					}
					Expect(ids).To(ConsistOf("share-owner-1", "share-owner-2"))
				})

				It("Get returns own share", func() {
					s, err := nonAdminRepo.Get("share-owner-1")
					Expect(err).ToNot(HaveOccurred())
					Expect(s.ID).To(Equal("share-owner-1"))
				})

				It("Get returns ErrNotFound for another user's share", func() {
					_, err := nonAdminRepo.Get("share-other-1")
					Expect(err).To(MatchError(model.ErrNotFound))
				})

				It("Read returns ErrNotFound for another user's share", func() {
					_, err := nonAdminRest.Read("share-other-1")
					Expect(err).To(MatchError(model.ErrNotFound))
				})

				It("Exists returns true for own share", func() {
					exists, err := nonAdminRepo.Exists("share-owner-1")
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeTrue())
				})

				It("Exists returns false for another user's share", func() {
					exists, err := nonAdminRepo.Exists("share-other-1")
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeFalse())
				})

				It("CountAll counts only own shares", func() {
					count, err := nonAdminRepo.CountAll()
					Expect(err).ToNot(HaveOccurred())
					Expect(count).To(BeNumerically("==", 2))
				})

				It("Count (rest) counts only own shares", func() {
					count, err := nonAdminRest.Count()
					Expect(err).ToNot(HaveOccurred())
					Expect(count).To(BeNumerically("==", 2))
				})
			})

			Context("admin user", func() {
				It("GetAll returns all shares", func() {
					adminCtx := request.WithUser(log.NewContext(GinkgoT().Context()), adminUser)
					adminRepo := NewShareRepository(adminCtx, GetDBXBuilder())
					shares, err := adminRepo.GetAll()
					Expect(err).ToNot(HaveOccurred())
					ids := make([]string, len(shares))
					for i, s := range shares {
						ids[i] = s.ID
					}
					Expect(ids).To(ConsistOf("share-owner-1", "share-owner-2", "share-other-1"))
				})

				It("CountAll counts all shares", func() {
					adminCtx := request.WithUser(log.NewContext(GinkgoT().Context()), adminUser)
					adminRepo := NewShareRepository(adminCtx, GetDBXBuilder())
					count, err := adminRepo.CountAll()
					Expect(err).ToNot(HaveOccurred())
					Expect(count).To(BeNumerically("==", 3))
				})
			})

			Context("headless context (public share route)", func() {
				It("GetAll returns all shares", func() {
					headlessRepo := NewShareRepository(GinkgoT().Context(), GetDBXBuilder())
					shares, err := headlessRepo.GetAll()
					Expect(err).ToNot(HaveOccurred())
					Expect(shares).To(HaveLen(3))
				})

				It("Get returns another user's share", func() {
					headlessRepo := NewShareRepository(GinkgoT().Context(), GetDBXBuilder())
					s, err := headlessRepo.Get("share-other-1")
					Expect(err).ToNot(HaveOccurred())
					Expect(s.ID).To(Equal("share-other-1"))
				})

				It("Exists returns true for any share", func() {
					headlessRepo := NewShareRepository(GinkgoT().Context(), GetDBXBuilder())
					exists, err := headlessRepo.Exists("share-other-1")
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeTrue())
				})
			})
		})
	})
})
