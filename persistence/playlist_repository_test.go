package persistence

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PlaylistRepository", func() {
	var repo model.PlaylistRepository

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: true})
		repo = NewPlaylistRepository(ctx, GetDBXBuilder())
	})

	Describe("Count", func() {
		It("returns the number of playlists in the DB", func() {
			Expect(repo.CountAll()).To(Equal(int64(2)))
		})
	})

	Describe("Exists", func() {
		It("returns true for an existing playlist", func() {
			Expect(repo.Exists(plsCool.ID)).To(BeTrue())
		})
		It("returns false for a non-existing playlist", func() {
			Expect(repo.Exists("666")).To(BeFalse())
		})
	})

	Describe("Get", func() {
		It("returns an existing playlist", func() {
			p, err := repo.Get(plsBest.ID)
			Expect(err).To(BeNil())
			// Compare all but Tracks and timestamps
			p2 := *p
			p2.Tracks = plsBest.Tracks
			p2.UpdatedAt = plsBest.UpdatedAt
			p2.CreatedAt = plsBest.CreatedAt
			Expect(p2).To(Equal(plsBest))
			// Compare tracks
			for i := range p.Tracks {
				Expect(p.Tracks[i].ID).To(Equal(plsBest.Tracks[i].ID))
			}
		})
		It("returns ErrNotFound for a non-existing playlist", func() {
			_, err := repo.Get("666")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
		It("returns all tracks", func() {
			pls, err := repo.GetWithTracks(plsBest.ID, true, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.Name).To(Equal(plsBest.Name))
			Expect(pls.Tracks).To(HaveLen(2))
			Expect(pls.Tracks[0].ID).To(Equal("1"))
			Expect(pls.Tracks[0].PlaylistID).To(Equal(plsBest.ID))
			Expect(pls.Tracks[0].MediaFileID).To(Equal(songDayInALife.ID))
			Expect(pls.Tracks[0].MediaFile.ID).To(Equal(songDayInALife.ID))
			Expect(pls.Tracks[1].ID).To(Equal("2"))
			Expect(pls.Tracks[1].PlaylistID).To(Equal(plsBest.ID))
			Expect(pls.Tracks[1].MediaFileID).To(Equal(songRadioactivity.ID))
			Expect(pls.Tracks[1].MediaFile.ID).To(Equal(songRadioactivity.ID))
			mfs := pls.MediaFiles()
			Expect(mfs).To(HaveLen(2))
			Expect(mfs[0].ID).To(Equal(songDayInALife.ID))
			Expect(mfs[1].ID).To(Equal(songRadioactivity.ID))
		})
	})

	It("Put/Exists/Delete", func() {
		By("saves the playlist to the DB")
		newPls := model.Playlist{Name: "Great!", OwnerID: "userid"}
		newPls.AddTracks([]string{"1004", "1003"})

		By("saves the playlist to the DB")
		Expect(repo.Put(&newPls)).To(BeNil())

		By("adds repeated songs to a playlist and keeps the order")
		newPls.AddTracks([]string{"1004"})
		Expect(repo.Put(&newPls)).To(BeNil())
		saved, _ := repo.GetWithTracks(newPls.ID, true, false)
		Expect(saved.Tracks).To(HaveLen(3))
		Expect(saved.Tracks[0].MediaFileID).To(Equal("1004"))
		Expect(saved.Tracks[1].MediaFileID).To(Equal("1003"))
		Expect(saved.Tracks[2].MediaFileID).To(Equal("1004"))

		By("returns the newly created playlist")
		Expect(repo.Exists(newPls.ID)).To(BeTrue())

		By("returns deletes the playlist")
		Expect(repo.Delete(newPls.ID)).To(BeNil())

		By("returns error if tries to retrieve the deleted playlist")
		Expect(repo.Exists(newPls.ID)).To(BeFalse())
	})

	Describe("GetAll", func() {
		It("returns all playlists from DB", func() {
			all, err := repo.GetAll()
			Expect(err).To(BeNil())
			Expect(all[0].ID).To(Equal(plsBest.ID))
			Expect(all[1].ID).To(Equal(plsCool.ID))
		})
	})

	Context("Smart Playlists", func() {
		var rules *criteria.Criteria
		BeforeEach(func() {
			rules = &criteria.Criteria{
				Expression: criteria.All{
					criteria.Contains{"title": "love"},
				},
			}
		})
		Context("valid rules", func() {
			Specify("Put/Get", func() {
				newPls := model.Playlist{Name: "Great!", OwnerID: "userid", Rules: rules}
				Expect(repo.Put(&newPls)).To(Succeed())

				savedPls, err := repo.Get(newPls.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(savedPls.Rules).To(Equal(rules))
			})
		})

		Context("invalid rules", func() {
			It("fails to Put it in the DB", func() {
				rules = &criteria.Criteria{
					// This is invalid because "contains" cannot have multiple fields
					Expression: criteria.All{
						criteria.Contains{"genre": "Hardcore", "filetype": "mp3"},
					},
				}
				newPls := model.Playlist{Name: "Great!", OwnerID: "userid", Rules: rules}
				Expect(repo.Put(&newPls)).To(MatchError(ContainSubstring("invalid criteria expression")))
			})
		})

		// BFR Validate these tests
		XContext("child smart playlists", func() {
			When("refresh day has expired", func() {
				It("should refresh tracks for smart playlist referenced in parent smart playlist criteria", func() {
					conf.Server.SmartPlaylistRefreshDelay = -1 * time.Second

					nestedPls := model.Playlist{Name: "Nested", OwnerID: "userid", Rules: rules}
					Expect(repo.Put(&nestedPls)).To(Succeed())

					parentPls := model.Playlist{Name: "Parent", OwnerID: "userid", Rules: &criteria.Criteria{
						Expression: criteria.All{
							criteria.InPlaylist{"id": nestedPls.ID},
						},
					}}
					Expect(repo.Put(&parentPls)).To(Succeed())

					nestedPlsRead, err := repo.Get(nestedPls.ID)
					Expect(err).ToNot(HaveOccurred())

					_, err = repo.GetWithTracks(parentPls.ID, true, false)
					Expect(err).ToNot(HaveOccurred())

					// Check that the nested playlist was refreshed by parent get by verifying evaluatedAt is updated since first nestedPls get
					nestedPlsAfterParentGet, err := repo.Get(nestedPls.ID)
					Expect(err).ToNot(HaveOccurred())

					Expect(*nestedPlsAfterParentGet.EvaluatedAt).To(BeTemporally(">", *nestedPlsRead.EvaluatedAt))
				})
			})

			When("refresh day has not expired", func() {
				It("should NOT refresh tracks for smart playlist referenced in parent smart playlist criteria", func() {
					conf.Server.SmartPlaylistRefreshDelay = 1 * time.Hour

					nestedPls := model.Playlist{Name: "Nested", OwnerID: "userid", Rules: rules}
					Expect(repo.Put(&nestedPls)).To(Succeed())

					parentPls := model.Playlist{Name: "Parent", OwnerID: "userid", Rules: &criteria.Criteria{
						Expression: criteria.All{
							criteria.InPlaylist{"id": nestedPls.ID},
						},
					}}
					Expect(repo.Put(&parentPls)).To(Succeed())

					nestedPlsRead, err := repo.Get(nestedPls.ID)
					Expect(err).ToNot(HaveOccurred())

					_, err = repo.GetWithTracks(parentPls.ID, true, false)
					Expect(err).ToNot(HaveOccurred())

					// Check that the nested playlist was not refreshed by parent get by verifying evaluatedAt is not updated since first nestedPls get
					nestedPlsAfterParentGet, err := repo.Get(nestedPls.ID)
					Expect(err).ToNot(HaveOccurred())

					Expect(*nestedPlsAfterParentGet.EvaluatedAt).To(Equal(*nestedPlsRead.EvaluatedAt))
				})
			})
		})
	})
})
