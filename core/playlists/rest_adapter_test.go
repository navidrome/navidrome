package playlists_test

import (
	"context"
	"time"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("REST Adapter", func() {
	var ds model.DataStore
	var ps playlists.Playlists
	var ctx context.Context

	stored := func(id string) *model.Playlist {
		return storedPlaylist(ctx, ds, id)
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
		putUsers(ctx, ds, "123", "user-1", "other-user")
		ctx = request.WithUser(ctx, model.User{ID: "123"})
	})

	Describe("NewRepository", func() {
		var repo rest.Persistable[model.Playlist]

		BeforeEach(func() {
			putPlaylists(ctx, ds, &model.Playlist{ID: "pls-1", Name: "My Playlist", OwnerID: "user-1", Public: true})
			ps = playlists.NewPlaylists(ds, artwork.NewUploader(ds))
		})

		Describe("Save", func() {
			It("sets the owner from the context user", func() {
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.Repository().(rest.Persistable[model.Playlist])
				pls := &model.Playlist{Name: "New Playlist"}
				id, err := repo.Save(ctx, pls)
				Expect(err).ToNot(HaveOccurred())
				Expect(id).ToNot(BeEmpty())
				Expect(pls.OwnerID).To(Equal("user-1"))
			})

			It("forces a new creation by clearing ID", func() {
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.Repository().(rest.Persistable[model.Playlist])
				pls := &model.Playlist{ID: "should-be-cleared", Name: "New"}
				_, err := repo.Save(ctx, pls)
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.ID).ToNot(Equal("should-be-cleared"))
			})

			It("clears server-managed fields to prevent injection via REST API", func() {
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.Repository().(rest.Persistable[model.Playlist])
				pls := &model.Playlist{
					Name:             "Legit Playlist",
					Comment:          "A comment",
					Public:           true,
					Rules:            &criteria.Criteria{Expression: criteria.All{criteria.Contains{"title": "test"}}},
					Path:             "/some/path/playlist.m3u",
					Sync:             true,
					UploadedImage:    "injected-image-path",
					ExternalImageURL: "http://evil.example.com/ssrf",
					EvaluatedAt:      new(time.Now()),
				}
				id, err := repo.Save(ctx, pls)
				Expect(err).ToNot(HaveOccurred())

				saved := stored(id)
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
				repo = ps.Repository().(rest.Persistable[model.Playlist])
				pls := &model.Playlist{Name: "Updated"}
				err := repo.Update(ctx, "pls-1", *pls)
				Expect(err).ToNot(HaveOccurred())
				Expect(stored("pls-1").Name).To(Equal("Updated"))
			})

			It("allows admin to update any playlist", func() {
				ctx = request.WithUser(ctx, model.User{ID: "admin-1", IsAdmin: true})
				repo = ps.Repository().(rest.Persistable[model.Playlist])
				pls := &model.Playlist{Name: "Updated"}
				err := repo.Update(ctx, "pls-1", *pls)
				Expect(err).ToNot(HaveOccurred())
				Expect(stored("pls-1").Name).To(Equal("Updated"))
			})

			It("denies non-owner, non-admin", func() {
				ctx = request.WithUser(ctx, model.User{ID: "other-user", IsAdmin: false})
				repo = ps.Repository().(rest.Persistable[model.Playlist])
				pls := &model.Playlist{Name: "Updated"}
				err := repo.Update(ctx, "pls-1", *pls)
				Expect(err).To(Equal(rest.ErrPermissionDenied))
				Expect(stored("pls-1").Name).To(Equal("My Playlist"))
			})

			It("denies regular user from changing ownership", func() {
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.Repository().(rest.Persistable[model.Playlist])
				pls := &model.Playlist{Name: "Updated", OwnerID: "other-user"}
				err := repo.Update(ctx, "pls-1", *pls)
				Expect(err).To(Equal(rest.ErrPermissionDenied))
				Expect(stored("pls-1").OwnerID).To(Equal("user-1"))
			})

			DescribeTable("denies regular user from changing ownership under any case-variant JSON key",
				func(colName string) {
					// rest.Put's field-name extraction is case-sensitive, but Go's
					// json decoder is case-insensitive on struct fields, so any
					// {"OwnerId":"x"} / {"OWNERID":"x"} / {"ownerid":"x"} populates
					// entity.OwnerID. sentFields normalizes both sides so the
					// permission gate fires regardless of casing.
					ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
					repo = ps.Repository().(rest.Persistable[model.Playlist])
					pls := &model.Playlist{OwnerID: "other-user"}
					err := repo.Update(ctx, "pls-1", *pls, colName)
					Expect(err).To(Equal(rest.ErrPermissionDenied))
				},
				Entry("canonical camelCase", "ownerId"),
				Entry("PascalCase", "OwnerId"),
				Entry("all upper", "OWNERID"),
				Entry("all lower", "ownerid"),
			)

			It("updates smart playlist rules", func() {
				putPlaylists(ctx, ds, &model.Playlist{
					ID:      "smart-1",
					Name:    "Smart Playlist",
					OwnerID: "user-1",
					Rules:   &criteria.Criteria{Expression: criteria.All{criteria.Contains{"title": "old"}}},
				})
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.Repository().(rest.Persistable[model.Playlist])
				newRules := &criteria.Criteria{Expression: criteria.All{criteria.Contains{"title": "new"}}}
				pls := &model.Playlist{Name: "Smart Playlist", Rules: newRules}
				err := repo.Update(ctx, "smart-1", *pls)
				Expect(err).ToNot(HaveOccurred())
				Expect(stored("smart-1").Rules).To(Equal(newRules))
			})

			It("invalidates the imported hash when rules change, so the next scan re-syncs the file", func() {
				putPlaylists(ctx, ds, &model.Playlist{
					ID:           "smart-1",
					Name:         "Smart Playlist",
					OwnerID:      "user-1",
					Path:         "/music/smart.nsp",
					Sync:         true,
					ImportedHash: hashOf("file content"),
					Rules:        &criteria.Criteria{Expression: criteria.All{criteria.Contains{"title": "old"}}},
				})
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.Repository().(rest.Persistable[model.Playlist])
				newRules := &criteria.Criteria{Expression: criteria.All{criteria.Contains{"title": "new"}}}
				pls := &model.Playlist{Rules: newRules}
				err := repo.Update(ctx, "smart-1", *pls, "rules")
				Expect(err).ToNot(HaveOccurred())
				Expect(stored("smart-1").ImportedHash).To(BeEmpty())
			})

			It("allows toggling sync for file-backed playlists", func() {
				originalTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
				putPlaylists(ctx, ds, &model.Playlist{
					ID:      "file-pls",
					Name:    "File Playlist",
					OwnerID: "user-1",
					Path:    "/music/playlist.m3u",
					Sync:    true,
				})
				backdate("file-pls", originalTime)
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.Repository().(rest.Persistable[model.Playlist])
				pls := &model.Playlist{Name: "File Playlist", Sync: false}
				err := repo.Update(ctx, "file-pls", *pls)
				Expect(err).ToNot(HaveOccurred())
				Expect(stored("file-pls").Sync).To(BeFalse())
				Expect(stored("file-pls").UpdatedAt).To(BeTemporally("==", originalTime))
			})

			It("does not allow setting sync on non-file-backed playlists", func() {
				putPlaylists(ctx, ds, &model.Playlist{
					ID:      "manual-pls",
					Name:    "Manual Playlist",
					OwnerID: "user-1",
					Path:    "",
					Sync:    false,
				})
				originalTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
				backdate("manual-pls", originalTime)
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.Repository().(rest.Persistable[model.Playlist])
				pls := &model.Playlist{Name: "Manual Playlist", Sync: true}
				err := repo.Update(ctx, "manual-pls", *pls)
				Expect(err).ToNot(HaveOccurred())
				Expect(stored("manual-pls").Sync).To(BeFalse())
				Expect(stored("manual-pls").UpdatedAt).To(BeTemporally("==", originalTime))
			})

			It("does not bump updatedAt when only public changes", func() {
				originalTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
				putPlaylists(ctx, ds, &model.Playlist{
					ID:      "pls-pub",
					Name:    "My Playlist",
					OwnerID: "user-1",
					Public:  false,
				})
				backdate("pls-pub", originalTime)
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.Repository().(rest.Persistable[model.Playlist])
				pls := &model.Playlist{Name: "My Playlist", Public: true}
				err := repo.Update(ctx, "pls-pub", *pls)
				Expect(err).ToNot(HaveOccurred())
				Expect(stored("pls-pub").Public).To(BeTrue())
				Expect(stored("pls-pub").UpdatedAt).To(BeTemporally("==", originalTime))
			})

			It("bumps updatedAt when name changes along with sync", func() {
				putPlaylists(ctx, ds, &model.Playlist{
					ID:      "file-pls2",
					Name:    "Old Name",
					OwnerID: "user-1",
					Path:    "/music/playlist.m3u",
					Sync:    true,
				})
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.Repository().(rest.Persistable[model.Playlist])
				pls := &model.Playlist{Name: "New Name", Sync: false}
				err := repo.Update(ctx, "file-pls2", *pls)
				Expect(err).ToNot(HaveOccurred())
				Expect(stored("file-pls2").Name).To(Equal("New Name"))
				Expect(stored("file-pls2").Sync).To(BeFalse())
			})

			It("returns rest.ErrNotFound when playlist doesn't exist", func() {
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.Repository().(rest.Persistable[model.Playlist])
				pls := &model.Playlist{Name: "Updated"}
				err := repo.Update(ctx, "nonexistent", *pls)
				Expect(err).To(Equal(rest.ErrNotFound))
			})

			// Regression tests for #5541: partial REST updates (e.g. bulk "Make Public")
			// must only touch the fields the client actually sent. The cols list from
			// rest.Put names those fields; fields outside it must be left alone, even
			// when the deserialized entity has zero values for them.
			Context("with partial updates (cols)", func() {
				BeforeEach(func() {
					ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
					putPlaylists(ctx, ds, &model.Playlist{
						ID:      "partial",
						Name:    "Original Name",
						Comment: "Original comment",
						OwnerID: "user-1",
						Public:  false,
					})
				})

				It("preserves name and comment when only public is sent (bulk Make Public)", func() {
					repo = ps.Repository().(rest.Persistable[model.Playlist])
					err := repo.Update(ctx, "partial", model.Playlist{Public: true}, "public")
					Expect(err).ToNot(HaveOccurred())
					Expect(stored("partial").Name).To(Equal("Original Name"))
					Expect(stored("partial").Comment).To(Equal("Original comment"))
					Expect(stored("partial").Public).To(BeTrue())
				})

				It("preserves name when only sync is sent for a file-backed playlist", func() {
					putPlaylists(ctx, ds, &model.Playlist{
						ID:      "file-partial",
						Name:    "Keep Me",
						OwnerID: "user-1",
						Path:    "/music/p.m3u",
						Sync:    true,
					})
					repo = ps.Repository().(rest.Persistable[model.Playlist])
					err := repo.Update(ctx, "file-partial", model.Playlist{Sync: false}, "sync")
					Expect(err).ToNot(HaveOccurred())
					Expect(stored("file-partial").Name).To(Equal("Keep Me"))
					Expect(stored("file-partial").Sync).To(BeFalse())
				})

				It("renames the playlist when only name is sent", func() {
					repo = ps.Repository().(rest.Persistable[model.Playlist])
					err := repo.Update(ctx, "partial", model.Playlist{Name: "Renamed"}, "name")
					Expect(err).ToNot(HaveOccurred())
					Expect(stored("partial").Name).To(Equal("Renamed"))
					Expect(stored("partial").Comment).To(Equal("Original comment"))
					Expect(stored("partial").Public).To(BeFalse())
				})

				It("clears the comment when an empty comment is sent explicitly", func() {
					repo = ps.Repository().(rest.Persistable[model.Playlist])
					err := repo.Update(ctx, "partial", model.Playlist{Comment: ""}, "comment")
					Expect(err).ToNot(HaveOccurred())
					Expect(stored("partial").Comment).To(BeEmpty())
					Expect(stored("partial").Name).To(Equal("Original Name"))
				})

				It("updates rules-only on a smart playlist (Feishin-style edit)", func() {
					putPlaylists(ctx, ds, &model.Playlist{
						ID:      "smart-partial",
						Name:    "Smart Original",
						Comment: "smart comment",
						OwnerID: "user-1",
						Public:  true,
						Rules:   &criteria.Criteria{Expression: criteria.All{criteria.Is{"genre": "Rock"}}},
					})
					repo = ps.Repository().(rest.Persistable[model.Playlist])
					newRules := &criteria.Criteria{Expression: criteria.All{criteria.Is{"genre": "Jazz"}}, Sort: "year DESC"}
					err := repo.Update(ctx, "smart-partial", model.Playlist{Rules: newRules}, "rules")
					Expect(err).ToNot(HaveOccurred())
					Expect(stored("smart-partial").Rules).To(Equal(newRules))
					Expect(stored("smart-partial").Name).To(Equal("Smart Original"))
					Expect(stored("smart-partial").Comment).To(Equal("smart comment"))
					Expect(stored("smart-partial").Public).To(BeTrue())
				})

				It("resets EvaluatedAt when rules change", func() {
					evaluatedAt := time.Now().Add(-1 * time.Hour)
					putPlaylists(ctx, ds, &model.Playlist{
						ID:          "smart-reset",
						Name:        "Smart",
						OwnerID:     "user-1",
						Rules:       &criteria.Criteria{Expression: criteria.All{criteria.Is{"genre": "Rock"}}},
						EvaluatedAt: &evaluatedAt,
					})
					repo = ps.Repository().(rest.Persistable[model.Playlist])
					newRules := &criteria.Criteria{Expression: criteria.All{criteria.Is{"genre": "Jazz"}}}
					err := repo.Update(ctx, "smart-reset", model.Playlist{Rules: newRules}, "rules")
					Expect(err).ToNot(HaveOccurred())
					Expect(stored("smart-reset").EvaluatedAt).To(BeNil())
				})

				It("keeps EvaluatedAt when rules are not changed", func() {
					evaluatedAt := time.Now().Add(-1 * time.Hour)
					putPlaylists(ctx, ds, &model.Playlist{
						ID:          "smart-keep",
						Name:        "Smart",
						OwnerID:     "user-1",
						Rules:       &criteria.Criteria{Expression: criteria.All{criteria.Is{"genre": "Rock"}}},
						EvaluatedAt: &evaluatedAt,
					})
					repo = ps.Repository().(rest.Persistable[model.Playlist])
					err := repo.Update(ctx, "smart-keep", model.Playlist{Name: "Renamed Smart"}, "name")
					Expect(err).ToNot(HaveOccurred())
					Expect(stored("smart-keep").EvaluatedAt).ToNot(BeNil())
					Expect(*stored("smart-keep").EvaluatedAt).To(BeTemporally("~", evaluatedAt, time.Second))
				})

				It("updates name and rules together (smart-playlist Edit form)", func() {
					putPlaylists(ctx, ds, &model.Playlist{
						ID:      "smart-edit",
						Name:    "Smart Original",
						Comment: "smart comment",
						OwnerID: "user-1",
						Rules:   &criteria.Criteria{Expression: criteria.All{criteria.Is{"genre": "Rock"}}},
					})
					repo = ps.Repository().(rest.Persistable[model.Playlist])
					newRules := &criteria.Criteria{Expression: criteria.All{criteria.Is{"artist": "Miles Davis"}}, Sort: "album"}
					err := repo.Update(ctx, "smart-edit",
						model.Playlist{Name: "Smart Renamed", Rules: newRules},
						"name", "rules")
					Expect(err).ToNot(HaveOccurred())
					Expect(stored("smart-edit").Name).To(Equal("Smart Renamed"))
					Expect(stored("smart-edit").Rules).To(Equal(newRules))
					Expect(stored("smart-edit").Comment).To(Equal("smart comment"))
				})

				It("does not bump the saved rules on an idempotent rules-only PUT", func() {
					rules := &criteria.Criteria{Expression: criteria.All{criteria.Is{"genre": "Rock"}}}
					putPlaylists(ctx, ds, &model.Playlist{
						ID:      "smart-idempotent",
						Name:    "Smart Idempotent",
						OwnerID: "user-1",
						Rules:   rules,
					})
					originalTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
					backdate("smart-idempotent", originalTime)
					repo = ps.Repository().(rest.Persistable[model.Playlist])
					// Same rules sent back — rulesEqual should report no change and
					// the request should no-op (no Put call).
					sameRules := &criteria.Criteria{Expression: criteria.All{criteria.Is{"genre": "Rock"}}}
					err := repo.Update(ctx, "smart-idempotent", model.Playlist{Rules: sameRules}, "rules")
					Expect(err).ToNot(HaveOccurred())
					Expect(stored("smart-idempotent").UpdatedAt).To(BeTemporally("==", originalTime))
				})

				It("preserves rules when only public is sent (smart playlist + bulk Make Public)", func() {
					rules := &criteria.Criteria{Expression: criteria.All{criteria.Is{"genre": "Rock"}}}
					putPlaylists(ctx, ds, &model.Playlist{
						ID:      "smart-public",
						Name:    "Smart Public",
						OwnerID: "user-1",
						Public:  false,
						Rules:   rules,
					})
					repo = ps.Repository().(rest.Persistable[model.Playlist])
					err := repo.Update(ctx, "smart-public", model.Playlist{Public: true}, "public")
					Expect(err).ToNot(HaveOccurred())
					Expect(stored("smart-public").Public).To(BeTrue())
					Expect(stored("smart-public").Rules).To(Equal(rules))
					Expect(stored("smart-public").Name).To(Equal("Smart Public"))
				})

				It("does not treat a missing ownerId as an ownership transfer attempt", func() {
					// A non-admin user sending only {public:true} should not be blocked
					// just because OwnerID is the zero value in the deserialized entity.
					repo = ps.Repository().(rest.Persistable[model.Playlist])
					err := repo.Update(ctx, "partial", model.Playlist{Public: true}, "public")
					Expect(err).ToNot(HaveOccurred())
				})

				It("matches cols case-insensitively (mirrors json decoder behavior)", func() {
					// Go's json decoder populates struct fields from case-variant keys
					// like {"Name":"x"}, but rest.Put's field-name extraction is
					// case-sensitive. sentFields normalizes both sides so a request
					// with {"Name":"Renamed"} is honored, not silently ignored.
					repo = ps.Repository().(rest.Persistable[model.Playlist])
					err := repo.Update(ctx, "partial", model.Playlist{Name: "Renamed"}, "Name")
					Expect(err).ToNot(HaveOccurred())
					Expect(stored("partial").Name).To(Equal("Renamed"))
					Expect(stored("partial").Comment).To(Equal("Original comment"))
				})
			})
		})

		Describe("Delete", func() {
			It("delegates to service Delete with permission checks", func() {
				ctx = request.WithUser(ctx, model.User{ID: "user-1", IsAdmin: false})
				repo = ps.Repository().(rest.Persistable[model.Playlist])
				err := repo.Delete(ctx, "pls-1")
				Expect(err).ToNot(HaveOccurred())
				Expect(ds.Playlist().Exists(asAdmin(ctx), "pls-1")).To(BeFalse())
			})

			It("denies non-owner", func() {
				ctx = request.WithUser(ctx, model.User{ID: "other-user", IsAdmin: false})
				repo = ps.Repository().(rest.Persistable[model.Playlist])
				err := repo.Delete(ctx, "pls-1")
				Expect(err).To(Equal(rest.ErrPermissionDenied))
				Expect(ds.Playlist().Exists(asAdmin(ctx), "pls-1")).To(BeTrue())
			})
		})
	})
})
