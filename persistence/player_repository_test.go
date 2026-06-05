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

var _ = Describe("PlayerRepository", func() {
	var adminRepo *playerRepository
	var database *dbx.DB

	var (
		adminPlayer1  = model.Player{ID: "1", Name: "NavidromeUI [Firefox/Linux]", UserAgent: "Firefox/Linux", UserId: adminUser.ID, Username: adminUser.UserName, Client: "NavidromeUI", IP: "127.0.0.1", ReportRealPath: true, ScrobbleEnabled: true}
		adminPlayer2  = model.Player{ID: "2", Name: "GenericClient [Chrome/Windows]", IP: "192.168.0.5", UserAgent: "Chrome/Windows", UserId: adminUser.ID, Username: adminUser.UserName, Client: "GenericClient", MaxBitRate: 128}
		regularPlayer = model.Player{ID: "3", Name: "NavidromeUI [Safari/macOS]", UserAgent: "Safari/macOS", UserId: regularUser.ID, Username: regularUser.UserName, Client: "NavidromeUI", ReportRealPath: true, ScrobbleEnabled: false}

		players = model.Players{adminPlayer1, adminPlayer2, regularPlayer}
	)

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, adminUser)

		database = GetDBXBuilder()
		adminRepo = NewPlayerRepository(ctx, database).(*playerRepository)

		for idx := range players {
			err := adminRepo.Put(&players[idx])
			Expect(err).To(BeNil())
		}
	})

	AfterEach(func() {
		items, err := adminRepo.ReadAll()
		Expect(err).To(BeNil())
		players, ok := items.(model.Players)
		Expect(ok).To(BeTrue())
		for i := range players {
			err = adminRepo.Delete(players[i].ID)
			Expect(err).To(BeNil())
		}
	})

	Describe("EntityName", func() {
		It("returns the right name", func() {
			Expect(adminRepo.EntityName()).To(Equal("player"))
		})
	})

	Describe("FindMatch", func() {
		It("finds existing match", func() {
			player, err := adminRepo.FindMatch(adminUser.ID, "NavidromeUI", "Firefox/Linux")
			Expect(err).To(BeNil())
			Expect(*player).To(Equal(adminPlayer1))
		})

		It("doesn't find bad match", func() {
			_, err := adminRepo.FindMatch(regularUser.ID, "NavidromeUI", "Firefox/Linux")
			Expect(err).To(Equal(model.ErrNotFound))
		})
	})

	Describe("Get", func() {
		It("Gets an existing item from user", func() {
			player, err := adminRepo.Get(adminPlayer1.ID)
			Expect(err).To(BeNil())
			Expect(*player).To(Equal(adminPlayer1))
		})

		It("Gets an existing item from another user", func() {
			player, err := adminRepo.Get(regularPlayer.ID)
			Expect(err).To(BeNil())
			Expect(*player).To(Equal(regularPlayer))
		})

		It("does not get nonexistent item", func() {
			_, err := adminRepo.Get("i don't exist")
			Expect(err).To(Equal(model.ErrNotFound))
		})
	})

	DescribeTableSubtree("per context", func(admin bool, players model.Players, userPlayer model.Player, otherPlayer model.Player) {
		var repo *playerRepository

		BeforeEach(func() {
			if admin {
				repo = adminRepo
			} else {
				ctx := log.NewContext(context.TODO())
				ctx = request.WithUser(ctx, regularUser)
				repo = NewPlayerRepository(ctx, database).(*playerRepository)
			}
		})

		baseCount := int64(len(players))

		Describe("Count", func() {
			It("should return all", func() {
				count, err := repo.Count()
				Expect(err).To(BeNil())
				Expect(count).To(Equal(baseCount))
			})
		})

		Describe("Delete", func() {
			It("deletes a player owned by the current user", func() {
				err := repo.Delete(userPlayer.ID)
				Expect(err).To(BeNil())

				count, err := repo.Count()
				Expect(err).To(BeNil())
				Expect(count).To(Equal(baseCount - 1))

				_, err = repo.Get(userPlayer.ID)
				Expect(err).To(Equal(model.ErrNotFound))
			})

			It("does not delete another user's player when not admin", func() {
				err := repo.Delete(otherPlayer.ID)

				if admin {
					// Admins may delete any player.
					Expect(err).To(BeNil())
					Expect(repo.Count()).To(Equal(baseCount - 1))
					_, err = repo.Get(otherPlayer.ID)
					Expect(err).To(Equal(model.ErrNotFound))
				} else {
					// The ownership-restricted delete matches no owned row, so it reports
					// permission-denied and leaves the other user's player untouched.
					Expect(err).To(Equal(rest.ErrPermissionDenied))
					Expect(repo.Count()).To(Equal(baseCount))
					item, err := repo.Get(otherPlayer.ID)
					Expect(err).To(BeNil())
					Expect(*item).To(Equal(otherPlayer))
				}
			})

			It("returns not-found for a nonexistent player", func() {
				err := repo.Delete("i don't exist")
				Expect(err).To(Equal(rest.ErrNotFound))
				Expect(repo.Count()).To(Equal(baseCount))
			})
		})

		Describe("Read", func() {
			It("can read from current user", func() {
				player, err := repo.Read(userPlayer.ID)
				Expect(err).To(BeNil())
				Expect(player).To(Equal(&userPlayer))
			})

			It("can read from other user or fail if not admin", func() {
				player, err := repo.Read(otherPlayer.ID)
				if admin {
					Expect(err).To(BeNil())
					Expect(player).To(Equal(&otherPlayer))
				} else {
					Expect(err).To(Equal(model.ErrNotFound))
				}
			})

			It("does not get nonexistent item", func() {
				_, err := repo.Read("i don't exist")
				Expect(err).To(Equal(model.ErrNotFound))
			})
		})

		Describe("ReadAll", func() {
			It("should get all items", func() {
				data, err := repo.ReadAll()
				Expect(err).To(BeNil())
				Expect(data).To(Equal(players))
			})
		})

		Describe("Save", func() {
			DescribeTable("item type", func(player model.Player) {
				clone := player
				clone.ID = ""
				clone.IP = "192.168.1.1"
				id, err := repo.Save(&clone)

				if clone.UserId == "" {
					Expect(err).To(HaveOccurred())
				} else if !admin && player.Username == adminPlayer1.Username {
					Expect(err).To(Equal(rest.ErrPermissionDenied))
					clone.UserId = ""
				} else {
					Expect(err).To(BeNil())
					Expect(id).ToNot(BeEmpty())
				}

				count, err := repo.Count()
				Expect(err).To(BeNil())

				clone.ID = id
				newItem, err := repo.Get(id)

				if clone.UserId == "" {
					Expect(count).To(Equal(baseCount))
					Expect(err).To(Equal(model.ErrNotFound))
				} else {
					Expect(count).To(Equal(baseCount + 1))
					Expect(err).To(BeNil())
					Expect(*newItem).To(Equal(clone))
				}
			},
				Entry("same user", userPlayer),
				Entry("other item", otherPlayer),
				Entry("fake item", model.Player{}),
			)
		})

		Describe("Update", func() {
			DescribeTable("item type", func(player model.Player) {
				clone := player
				clone.IP = "192.168.1.1"
				clone.MaxBitRate = 10000
				err := repo.Update(clone.ID, &clone, "ip")

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
				newItem, err := repo.Get(clone.ID)

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

	Describe("Ownership enforcement (cross-tenant write protection)", func() {
		var regularRepo *playerRepository

		BeforeEach(func() {
			ctx := log.NewContext(context.TODO())
			ctx = request.WithUser(ctx, regularUser)
			regularRepo = NewPlayerRepository(ctx, database).(*playerRepository)
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
			err := regularRepo.Update(adminPlayer1.ID, &spoofed, "name", "user_id", "max_bit_rate")
			Expect(err).To(Equal(rest.ErrPermissionDenied))

			// The victim's player must remain untouched.
			stored, err := adminRepo.Get(adminPlayer1.ID)
			Expect(err).To(BeNil())
			Expect(*stored).To(Equal(adminPlayer1))
		})

		It("does not let a regular user reassign their own player to another user", func() {
			// Owner updates their own player but tries to give it away to the admin. The update
			// succeeds for the other fields, but user_id is never written, so ownership stays put.
			reassign := regularPlayer
			reassign.UserId = adminUser.ID
			reassign.Name = "given-away"

			err := regularRepo.Update(regularPlayer.ID, &reassign, "name", "user_id")
			Expect(err).To(BeNil())

			// Ownership must not have changed.
			stored, err := adminRepo.Get(regularPlayer.ID)
			Expect(err).To(BeNil())
			Expect(stored.UserId).To(Equal(regularUser.ID))
		})

		It("does not let an admin reassign a player to another user", func() {
			// Even an admin cannot change a player's owner via update.
			reassign := regularPlayer
			reassign.UserId = adminUser.ID
			reassign.Name = "admin-renamed"

			err := adminRepo.Update(regularPlayer.ID, &reassign, "name", "user_id")
			Expect(err).To(BeNil())

			// The name change applies, but ownership must not have moved.
			stored, err := adminRepo.Get(regularPlayer.ID)
			Expect(err).To(BeNil())
			Expect(stored.Name).To(Equal("admin-renamed"))
			Expect(stored.UserId).To(Equal(regularUser.ID))
		})

		It("lets the owner update their own player", func() {
			update := regularPlayer
			update.Name = "renamed-by-owner"

			err := regularRepo.Update(regularPlayer.ID, &update, "name")
			Expect(err).To(BeNil())

			stored, err := adminRepo.Get(regularPlayer.ID)
			Expect(err).To(BeNil())
			Expect(stored.Name).To(Equal("renamed-by-owner"))
			Expect(stored.UserId).To(Equal(regularUser.ID))
		})

		It("returns not found when updating a nonexistent player", func() {
			ghost := model.Player{ID: "does-not-exist", Name: "ghost", UserId: regularUser.ID}
			err := regularRepo.Update("does-not-exist", &ghost, "name")
			Expect(err).To(Equal(rest.ErrNotFound))
		})
	})
})
