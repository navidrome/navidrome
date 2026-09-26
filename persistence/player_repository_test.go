package persistence

import (
	"context"
	"errors"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pocketbase/dbx"
)

const testAPIKey = "nds_0123456789abcdefghijkl"

func expectAPIKeyError(err error, msg string) {
	var verr *rest.ValidationError
	ExpectWithOffset(1, errors.As(err, &verr)).To(BeTrue())
	ExpectWithOffset(1, verr.Errors).To(HaveKeyWithValue("apiKey", msg))
}

var _ = Describe("PlayerRepository", func() {
	var adminRepo *playerRepository
	var database *dbx.DB
	var ctx context.Context

	var (
		adminPlayer1  = model.Player{ID: "1", Name: "NavidromeUI [Firefox/Linux]", UserAgent: "Firefox/Linux", UserId: adminUser.ID, Username: adminUser.UserName, Client: "NavidromeUI", IP: "127.0.0.1", ReportRealPath: true, ScrobbleEnabled: true}
		adminPlayer2  = model.Player{ID: "2", Name: "GenericClient [Chrome/Windows]", IP: "192.168.0.5", UserAgent: "Chrome/Windows", UserId: adminUser.ID, Username: adminUser.UserName, Client: "GenericClient", MaxBitRate: 128}
		regularPlayer = model.Player{ID: "3", Name: "NavidromeUI [Safari/macOS]", UserAgent: "Safari/macOS", UserId: regularUser.ID, Username: regularUser.UserName, Client: "NavidromeUI", ReportRealPath: true, ScrobbleEnabled: false}

		players = model.Players{adminPlayer1, adminPlayer2, regularPlayer}
	)

	BeforeEach(func() {
		ctx = request.WithUser(log.NewContext(GinkgoT().Context()), adminUser)

		database = GetDBXBuilder()
		adminRepo = NewPlayerRepository(database).(*playerRepository)

		for idx := range players {
			err := adminRepo.Put(ctx, &players[idx])
			Expect(err).To(BeNil())
		}
	})

	AfterEach(func() {
		players, err := adminRepo.ReadAll(ctx)
		Expect(err).To(BeNil())
		for i := range players {
			err = adminRepo.Delete(ctx, players[i].ID)
			Expect(err).To(BeNil())
		}
	})

	Describe("FindMatch", func() {
		It("finds existing match", func() {
			player, err := adminRepo.FindMatch(ctx, adminUser.ID, "NavidromeUI", "Firefox/Linux")
			Expect(err).To(BeNil())
			Expect(*player).To(Equal(adminPlayer1))
		})

		It("doesn't find bad match", func() {
			_, err := adminRepo.FindMatch(ctx, regularUser.ID, "NavidromeUI", "Firefox/Linux")
			Expect(err).To(Equal(model.ErrNotFound))
		})
	})

	Describe("Get", func() {
		It("Gets an existing item from user", func() {
			player, err := adminRepo.Get(ctx, adminPlayer1.ID)
			Expect(err).To(BeNil())
			Expect(*player).To(Equal(adminPlayer1))
		})

		It("Gets an existing item from another user", func() {
			player, err := adminRepo.Get(ctx, regularPlayer.ID)
			Expect(err).To(BeNil())
			Expect(*player).To(Equal(regularPlayer))
		})

		It("does not get nonexistent item", func() {
			_, err := adminRepo.Get(ctx, "i don't exist")
			Expect(err).To(Equal(model.ErrNotFound))
		})
	})

	DescribeTableSubtree("per context", func(admin bool, players model.Players, userPlayer model.Player, otherPlayer model.Player) {
		var repo *playerRepository
		var repoCtx context.Context

		BeforeEach(func() {
			repoCtx = ctx
			if admin {
				repo = adminRepo
			} else {
				repoCtx = request.WithUser(ctx, regularUser)
				repo = NewPlayerRepository(database).(*playerRepository)
			}
		})

		baseCount := int64(len(players))

		Describe("Count", func() {
			It("should return all", func() {
				count, err := repo.Count(repoCtx)
				Expect(err).To(BeNil())
				Expect(count).To(Equal(baseCount))
			})
		})

		Describe("Delete", func() {
			It("deletes a player owned by the current user", func() {
				err := repo.Delete(repoCtx, userPlayer.ID)
				Expect(err).To(BeNil())

				count, err := repo.Count(repoCtx)
				Expect(err).To(BeNil())
				Expect(count).To(Equal(baseCount - 1))

				_, err = repo.Get(repoCtx, userPlayer.ID)
				Expect(err).To(Equal(model.ErrNotFound))
			})

			It("does not delete another user's player when not admin", func() {
				err := repo.Delete(repoCtx, otherPlayer.ID)

				if admin {
					// Admins may delete any player.
					Expect(err).To(BeNil())
					Expect(repo.Count(repoCtx)).To(Equal(baseCount - 1))
					_, err = repo.Get(repoCtx, otherPlayer.ID)
					Expect(err).To(Equal(model.ErrNotFound))
				} else {
					// The ownership-restricted delete matches no owned row, so it reports
					// permission-denied and leaves the other user's player untouched.
					Expect(err).To(Equal(rest.ErrPermissionDenied))
					Expect(repo.Count(repoCtx)).To(Equal(baseCount))
					item, err := repo.Get(repoCtx, otherPlayer.ID)
					Expect(err).To(BeNil())
					Expect(*item).To(Equal(otherPlayer))
				}
			})

			It("returns not-found for a nonexistent player", func() {
				err := repo.Delete(repoCtx, "i don't exist")
				Expect(err).To(Equal(rest.ErrNotFound))
				Expect(repo.Count(repoCtx)).To(Equal(baseCount))
			})
		})

		Describe("Read", func() {
			It("can read from current user", func() {
				player, err := repo.Read(repoCtx, userPlayer.ID)
				Expect(err).To(BeNil())
				Expect(player).To(Equal(&userPlayer))
			})

			It("can read from other user or fail if not admin", func() {
				player, err := repo.Read(repoCtx, otherPlayer.ID)
				if admin {
					Expect(err).To(BeNil())
					Expect(player).To(Equal(&otherPlayer))
				} else {
					Expect(err).To(Equal(model.ErrNotFound))
				}
			})

			It("does not get nonexistent item", func() {
				_, err := repo.Read(repoCtx, "i don't exist")
				Expect(err).To(Equal(model.ErrNotFound))
			})
		})

		Describe("ReadAll", func() {
			It("should get all items", func() {
				data, err := repo.ReadAll(repoCtx)
				Expect(err).To(BeNil())
				Expect(model.Players(data)).To(Equal(players))
			})
		})

		Describe("Save", func() {
			DescribeTable("item type", func(player model.Player) {
				clone := player
				clone.ID = ""
				clone.IP = "192.168.1.1"
				clone.APIKey = new(testAPIKey)
				id, err := repo.Save(repoCtx, &clone)

				if clone.UserId == "" {
					Expect(err).To(HaveOccurred())
				} else if player.UserId != userPlayer.UserId {
					Expect(err).To(Equal(rest.ErrPermissionDenied))
					clone.UserId = ""
				} else {
					Expect(err).To(BeNil())
					Expect(id).ToNot(BeEmpty())
				}

				count, err := repo.Count(repoCtx)
				Expect(err).To(BeNil())

				clone.ID = id
				newItem, err := repo.Get(repoCtx, id)

				if clone.UserId == "" {
					Expect(count).To(Equal(baseCount))
					Expect(err).To(Equal(model.ErrNotFound))
				} else {
					Expect(count).To(Equal(baseCount + 1))
					Expect(err).To(BeNil())
					clone.APIKey = nil
					clone.HasAPIKey = true
					Expect(*newItem).To(Equal(clone))
				}
			},
				Entry("same user", userPlayer),
				Entry("other item", otherPlayer),
			)
		})

		Describe("Update", func() {
			DescribeTable("item type", func(player model.Player) {
				clone := player
				clone.IP = "192.168.1.1"
				clone.MaxBitRate = 10000
				err := repo.Update(repoCtx, clone.ID, clone, "ip")

				if player.UserId == "" {
					Expect(err).To(HaveOccurred())
				} else if !admin && player.Username == adminPlayer1.Username {
					// A non-admin cannot target another user's player: the ownership-restricted
					// update matches no owned row, so it reports permission-denied rather than
					// touching it.
					Expect(err).To(Equal(rest.ErrPermissionDenied))
					clone.IP = player.IP
				} else {
					Expect(err).To(BeNil())
				}

				clone.MaxBitRate = player.MaxBitRate
				newItem, err := repo.Get(repoCtx, clone.ID)

				if player.UserId == "" {
					Expect(err).To(Equal(model.ErrNotFound))
				} else if !admin && player.UserId == adminUser.ID {
					Expect(*newItem).To(Equal(player))
				} else {
					Expect(*newItem).To(Equal(clone))
				}
			},
				Entry("same user", userPlayer),
				Entry("other item", otherPlayer),
				Entry("fake item", model.Player{}),
			)
		})
	},
		Entry("admin context", true, players, adminPlayer1, regularPlayer),
		Entry("regular context", false, model.Players{regularPlayer}, regularPlayer, adminPlayer1),
	)

	Describe("API keys", func() {
		const key = testAPIKey
		const otherKey = "nds_ABCDEFGHIJKLMNOPQRSTUV"
		var ownerCtx, otherCtx context.Context

		BeforeEach(func() {
			ownerCtx = request.WithUser(log.NewContext(GinkgoT().Context()), regularUser)
			otherCtx = request.WithUser(log.NewContext(GinkgoT().Context()), thirdUser)
		})

		storedHash := func(id string) string {
			var row struct {
				Hash string `db:"api_key_hash"`
			}
			Expect(database.NewQuery("select coalesce(api_key_hash, '') as api_key_hash from player where id = {:id}").
				Bind(dbx.Params{"id": id}).One(&row)).To(Succeed())
			return row.Hash
		}

		Describe("SetAPIKey", func() {
			It("stores only the hash and finds the player by the key", func() {
				Expect(adminRepo.SetAPIKey(ownerCtx, regularPlayer.ID, key)).To(Succeed())

				Expect(storedHash(regularPlayer.ID)).To(Equal(hashAPIKey(key)))
				plr, err := adminRepo.FindByAPIKey(ctx, key)
				Expect(err).ToNot(HaveOccurred())
				Expect(plr.ID).To(Equal(regularPlayer.ID))
				Expect(plr.HasAPIKey).To(BeTrue())
				Expect(plr.APIKey).To(BeNil())
			})

			It("replaces the previous key", func() {
				Expect(adminRepo.SetAPIKey(ownerCtx, regularPlayer.ID, key)).To(Succeed())
				Expect(adminRepo.SetAPIKey(ownerCtx, regularPlayer.ID, otherKey)).To(Succeed())

				_, err := adminRepo.FindByAPIKey(ctx, key)
				Expect(err).To(MatchError(model.ErrNotFound))
				_, err = adminRepo.FindByAPIKey(ctx, otherKey)
				Expect(err).ToNot(HaveOccurred())
			})

			DescribeTable("rejects malformed keys",
				func(bad string) {
					err := adminRepo.SetAPIKey(ownerCtx, regularPlayer.ID, bad)
					expectAPIKeyError(err, "resources.player.validation.apiKeyFormat")
					Expect(storedHash(regularPlayer.ID)).To(BeEmpty())
				},
				Entry("no prefix", "0123456789abcdefghijklmn"),
				Entry("too short", "nds_short"),
				Entry("too long", key+"x"),
				Entry("bad chars", "nds_0123456789abcdefghij-!"),
			)

			It("revokes with an empty key, by the owner or an admin", func() {
				Expect(adminRepo.SetAPIKey(ownerCtx, regularPlayer.ID, key)).To(Succeed())
				Expect(adminRepo.SetAPIKey(ownerCtx, regularPlayer.ID, "")).To(Succeed())
				Expect(storedHash(regularPlayer.ID)).To(BeEmpty())

				Expect(adminRepo.SetAPIKey(ownerCtx, regularPlayer.ID, key)).To(Succeed())
				Expect(adminRepo.SetAPIKey(ctx, regularPlayer.ID, "")).To(Succeed())
				Expect(storedHash(regularPlayer.ID)).To(BeEmpty())
			})

			It("accepts revoking a player that has no key", func() {
				Expect(adminRepo.SetAPIKey(ownerCtx, regularPlayer.ID, "")).To(Succeed())
			})

			It("does not let an admin set a key on another user's player", func() {
				Expect(adminRepo.SetAPIKey(ctx, regularPlayer.ID, key)).To(MatchError(rest.ErrPermissionDenied))
				Expect(storedHash(regularPlayer.ID)).To(BeEmpty())
			})

			It("does not let another user set or revoke", func() {
				Expect(adminRepo.SetAPIKey(ownerCtx, regularPlayer.ID, key)).To(Succeed())
				Expect(adminRepo.SetAPIKey(otherCtx, regularPlayer.ID, otherKey)).To(MatchError(rest.ErrPermissionDenied))
				Expect(adminRepo.SetAPIKey(otherCtx, regularPlayer.ID, "")).To(MatchError(rest.ErrPermissionDenied))
				Expect(storedHash(regularPlayer.ID)).To(Equal(hashAPIKey(key)))
			})

			It("returns not found for a missing player", func() {
				Expect(adminRepo.SetAPIKey(ownerCtx, "missing", key)).To(MatchError(rest.ErrNotFound))
				Expect(adminRepo.SetAPIKey(ownerCtx, "missing", "")).To(MatchError(rest.ErrNotFound))
			})

			It("does not find unknown or empty keys", func() {
				_, err := adminRepo.FindByAPIKey(ctx, otherKey)
				Expect(err).To(MatchError(model.ErrNotFound))
				_, err = adminRepo.FindByAPIKey(ctx, "")
				Expect(err).To(MatchError(model.ErrNotFound))
			})

			It("drops the key with the player", func() {
				Expect(adminRepo.SetAPIKey(ownerCtx, regularPlayer.ID, key)).To(Succeed())
				Expect(adminRepo.Delete(ownerCtx, regularPlayer.ID)).To(Succeed())
				_, err := adminRepo.FindByAPIKey(ctx, key)
				Expect(err).To(MatchError(model.ErrNotFound))
			})
		})

		Describe("Save (create)", func() {
			It("creates the player with the key, owned by the logged-in user", func() {
				id, err := adminRepo.Save(ownerCtx, &model.Player{Name: "Manual player", APIKey: new(key)})
				Expect(err).ToNot(HaveOccurred())

				plr, err := adminRepo.FindByAPIKey(ctx, key)
				Expect(err).ToNot(HaveOccurred())
				Expect(plr.ID).To(Equal(id))
				Expect(plr.UserId).To(Equal(regularUser.ID))
			})

			It("requires a key", func() {
				count, _ := adminRepo.CountAll(ctx)
				_, err := adminRepo.Save(ownerCtx, &model.Player{Name: "No key"})
				expectAPIKeyError(err, "ra.validation.required")

				_, err = adminRepo.Save(ownerCtx, &model.Player{Name: "Empty key", APIKey: new("")})
				expectAPIKeyError(err, "ra.validation.required")
				Expect(adminRepo.CountAll(ctx)).To(Equal(count))
			})

			It("rejects a malformed key without creating the player", func() {
				count, _ := adminRepo.CountAll(ctx)
				_, err := adminRepo.Save(ownerCtx, &model.Player{Name: "Bad", APIKey: new("nds_bad")})
				expectAPIKeyError(err, "resources.player.validation.apiKeyFormat")
				Expect(adminRepo.CountAll(ctx)).To(Equal(count))
			})

			It("does not let an admin create a keyed player for another user", func() {
				count, _ := adminRepo.CountAll(ctx)
				_, err := adminRepo.Save(ctx, &model.Player{Name: "For someone", UserId: regularUser.ID, APIKey: new(key)})
				Expect(err).To(MatchError(rest.ErrPermissionDenied))
				_, err = adminRepo.Save(ctx, &model.Player{Name: "For someone", UserId: regularUser.ID})
				Expect(err).To(MatchError(rest.ErrPermissionDenied))
				Expect(adminRepo.CountAll(ctx)).To(Equal(count))
			})

			It("rejects a key already used by another player without creating the player", func() {
				Expect(adminRepo.SetAPIKey(ctx, adminPlayer1.ID, key)).To(Succeed())
				count, _ := adminRepo.CountAll(ctx)
				_, err := adminRepo.Save(ownerCtx, &model.Player{Name: "Duplicate", APIKey: new(key)})
				expectAPIKeyError(err, "ra.validation.unique")
				Expect(adminRepo.CountAll(ctx)).To(Equal(count))
			})
		})

		Describe("Update (edit)", func() {
			It("rolls back the key change when the rest of the edit fails", func() {
				_, err := database.NewQuery(`create trigger fail_player_rename before update of name on player
					when new.name = 'boom' begin select raise(abort, 'boom'); end`).Execute()
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(func() {
					_, _ = database.NewQuery("drop trigger if exists fail_player_rename").Execute()
				})

				plr := regularPlayer
				plr.Name = "boom"
				plr.APIKey = new(key)
				Expect(adminRepo.Update(ownerCtx, plr.ID, plr, "name", "apiKey")).ToNot(Succeed())
				Expect(storedHash(regularPlayer.ID)).To(BeEmpty())
			})

			It("keeps the key when apiKey is absent (a normal edit)", func() {
				Expect(adminRepo.SetAPIKey(ownerCtx, regularPlayer.ID, key)).To(Succeed())

				plr := regularPlayer
				plr.Name = "Renamed"
				Expect(adminRepo.Update(ownerCtx, plr.ID, plr, "name", "hasApiKey")).To(Succeed())
				Expect(adminRepo.Update(ownerCtx, plr.ID, plr)).To(Succeed())

				found, err := adminRepo.FindByAPIKey(ctx, key)
				Expect(err).ToNot(HaveOccurred())
				Expect(found.Name).To(Equal("Renamed"))
			})

			It("sets a new key when apiKey has a value", func() {
				plr := regularPlayer
				plr.APIKey = new(key)
				Expect(adminRepo.Update(ownerCtx, plr.ID, plr, "name", "apiKey")).To(Succeed())
				Expect(storedHash(regularPlayer.ID)).To(Equal(hashAPIKey(key)))
			})

			It("revokes the key when apiKey is empty", func() {
				Expect(adminRepo.SetAPIKey(ownerCtx, regularPlayer.ID, key)).To(Succeed())
				plr := regularPlayer
				plr.APIKey = new("")
				Expect(adminRepo.Update(ownerCtx, plr.ID, plr, "apiKey")).To(Succeed())
				Expect(storedHash(regularPlayer.ID)).To(BeEmpty())
			})

			It("lets an admin edit another user's keyed player without touching the key", func() {
				Expect(adminRepo.SetAPIKey(ownerCtx, regularPlayer.ID, key)).To(Succeed())
				plr := regularPlayer
				plr.MaxBitRate = 192
				Expect(adminRepo.Update(ctx, plr.ID, plr, "maxBitRate", "hasApiKey")).To(Succeed())
				Expect(storedHash(regularPlayer.ID)).To(Equal(hashAPIKey(key)))
			})

			It("refuses an admin setting a key on another user's player and leaves other columns alone", func() {
				plr := regularPlayer
				plr.Name = "Hijacked"
				plr.APIKey = new(key)
				Expect(adminRepo.Update(ctx, plr.ID, plr, "name", "apiKey")).To(MatchError(rest.ErrPermissionDenied))

				got, err := adminRepo.Get(ctx, regularPlayer.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(got.Name).To(Equal(regularPlayer.Name))
				Expect(storedHash(regularPlayer.ID)).To(BeEmpty())
			})

			It("refuses a key already used by another player and leaves other columns alone", func() {
				Expect(adminRepo.SetAPIKey(ctx, adminPlayer1.ID, key)).To(Succeed())
				plr := regularPlayer
				plr.Name = "Renamed"
				plr.APIKey = new(key)
				err := adminRepo.Update(ownerCtx, plr.ID, plr, "name", "apiKey")
				expectAPIKeyError(err, "ra.validation.unique")

				got, err := adminRepo.Get(ctx, regularPlayer.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(got.Name).To(Equal(regularPlayer.Name))
				Expect(storedHash(regularPlayer.ID)).To(BeEmpty())
				Expect(storedHash(adminPlayer1.ID)).To(Equal(hashAPIKey(key)))
			})
		})
	})

	Describe("Ownership enforcement (cross-tenant write protection)", func() {
		var regularRepo *playerRepository
		var regularCtx context.Context

		BeforeEach(func() {
			regularCtx = request.WithUser(ctx, regularUser)
			regularRepo = NewPlayerRepository(database).(*playerRepository)
		})

		It("does not let a regular user hijack another user's player by spoofing userId in the body", func() {
			// Attacker (regularUser) targets the victim's (adminUser) player by URL id,
			// but sets userId in the body to their own id to try to pass the permission check.
			spoofed := model.Player{
				ID:         adminPlayer1.ID,
				Name:       "HIJACKED",
				UserId:     regularUser.ID, // attacker's own id, spoofed in the body
				MaxBitRate: 1,
			}

			// The ownership-restricted update matches no row owned by the attacker, so the write
			// targets nothing and reports permission-denied rather than overwriting the victim's row.
			err := regularRepo.Update(regularCtx, adminPlayer1.ID, spoofed, "name", "user_id", "max_bit_rate")
			Expect(err).To(Equal(rest.ErrPermissionDenied))

			// The victim's player must remain untouched.
			stored, err := adminRepo.Get(ctx, adminPlayer1.ID)
			Expect(err).To(BeNil())
			Expect(*stored).To(Equal(adminPlayer1))
		})

		It("does not let a regular user overwrite another user's player via Save with a spoofed id", func() {
			spoofed := model.Player{
				ID:             adminPlayer1.ID,
				Name:           "HIJACKED",
				UserId:         regularUser.ID,
				ReportRealPath: true,
				APIKey:         new(testAPIKey),
			}

			id, err := regularRepo.Save(regularCtx, &spoofed)
			Expect(err).To(BeNil())
			Expect(id).ToNot(Equal(adminPlayer1.ID))

			stored, err := adminRepo.Get(ctx, adminPlayer1.ID)
			Expect(err).To(BeNil())
			Expect(*stored).To(Equal(adminPlayer1))

			created, err := adminRepo.Get(ctx, id)
			Expect(err).To(BeNil())
			Expect(created.UserId).To(Equal(regularUser.ID))
		})

		It("does not let a regular user reassign their own player to another user", func() {
			// Owner updates their own player but tries to give it away to the admin. The update
			// succeeds for the other fields, but user_id is never written, so ownership stays put.
			reassign := regularPlayer
			reassign.UserId = adminUser.ID
			reassign.Name = "given-away"

			err := regularRepo.Update(regularCtx, regularPlayer.ID, reassign, "name", "user_id")
			Expect(err).To(BeNil())

			// Ownership must not have changed.
			stored, err := adminRepo.Get(ctx, regularPlayer.ID)
			Expect(err).To(BeNil())
			Expect(stored.UserId).To(Equal(regularUser.ID))
		})

		It("does not let an admin reassign a player to another user", func() {
			// Even an admin cannot change a player's owner via update.
			reassign := regularPlayer
			reassign.UserId = adminUser.ID
			reassign.Name = "admin-renamed"

			err := adminRepo.Update(regularCtx, regularPlayer.ID, reassign, "name", "user_id")
			Expect(err).To(BeNil())

			// The name change applies, but ownership must not have moved.
			stored, err := adminRepo.Get(ctx, regularPlayer.ID)
			Expect(err).To(BeNil())
			Expect(stored.Name).To(Equal("admin-renamed"))
			Expect(stored.UserId).To(Equal(regularUser.ID))
		})

		It("lets the owner update their own player", func() {
			update := regularPlayer
			update.Name = "renamed-by-owner"

			err := regularRepo.Update(regularCtx, regularPlayer.ID, update, "name")
			Expect(err).To(BeNil())

			stored, err := adminRepo.Get(ctx, regularPlayer.ID)
			Expect(err).To(BeNil())
			Expect(stored.Name).To(Equal("renamed-by-owner"))
			Expect(stored.UserId).To(Equal(regularUser.ID))
		})

		It("returns not found when updating a nonexistent player", func() {
			ghost := model.Player{ID: "does-not-exist", Name: "ghost", UserId: regularUser.ID}
			err := regularRepo.Update(regularCtx, "does-not-exist", ghost, "name")
			Expect(err).To(Equal(rest.ErrNotFound))
		})
	})
})
