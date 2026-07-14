package persistence

import (
	"slices"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pocketbase/dbx"
)

var _ = Describe("PlaylistRepository", func() {
	var repo model.PlaylistRepository

	BeforeEach(func() {
		ctx := log.NewContext(GinkgoT().Context())
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

	Describe("Annotations", func() {
		var plsID string

		BeforeEach(func() {
			pls := model.Playlist{Name: "Annotated", OwnerID: "userid"}
			Expect(repo.Put(&pls)).To(Succeed())
			plsID = pls.ID
		})

		countAnnotations := func() int {
			var count int
			Expect(GetDBXBuilder().NewQuery(
				"SELECT count(*) FROM annotation WHERE item_type = 'playlist' AND item_id = {:id}").
				Bind(dbx.Params{"id": plsID}).Row(&count)).To(Succeed())
			return count
		}

		It("stores and reads back starred", func() {
			Expect(repo.SetStar(true, plsID)).To(Succeed())

			p, err := repo.Get(plsID)
			Expect(err).ToNot(HaveOccurred())
			Expect(p.Starred).To(BeTrue())
			Expect(p.StarredAt).ToNot(BeNil())
		})

		It("stores and reads back rating and average_rating", func() {
			Expect(repo.SetRating(4, plsID)).To(Succeed())

			p, err := repo.Get(plsID)
			Expect(err).ToNot(HaveOccurred())
			Expect(p.Rating).To(Equal(4))
			Expect(p.RatedAt).ToNot(BeNil())
			Expect(p.AverageRating).To(Equal(4.0))
		})

		It("keeps annotations isolated per user", func() {
			Expect(repo.SetStar(true, plsID)).To(Succeed())

			otherCtx := request.WithUser(log.NewContext(GinkgoT().Context()),
				model.User{ID: "otheruser", UserName: "otheruser", IsAdmin: true})
			otherRepo := NewPlaylistRepository(otherCtx, GetDBXBuilder())

			p, err := otherRepo.Get(plsID)
			Expect(err).ToNot(HaveOccurred())
			Expect(p.Starred).To(BeFalse())
		})

		It("reads starred back through GetAll", func() {
			Expect(repo.SetStar(true, plsID)).To(Succeed())

			all, err := repo.GetAll()
			Expect(err).ToNot(HaveOccurred())
			idx := slices.IndexFunc(all, func(p model.Playlist) bool { return p.ID == plsID })
			Expect(idx).To(BeNumerically(">=", 0))
			Expect(all[idx].Starred).To(BeTrue())
		})

		It("counts playlists using annotation filters", func() {
			Expect(repo.SetStar(true, plsID)).To(Succeed())

			options := model.QueryOptions{Filters: squirrel.Eq{"starred": true}}
			starred, err := repo.GetAll(options)
			Expect(err).ToNot(HaveOccurred())
			Expect(starred).To(ContainElement(HaveField("ID", plsID)))

			count, err := repo.CountAll(options)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(int64(len(starred))))
		})

		It("does not leak an annotation row of another item_type sharing the playlist id", func() {
			// Older builds (and the star fallthrough) can leave a media_file-typed row
			// under a playlist id; the item_type-scoped join must not surface or dupe it.
			_, err := GetDBXBuilder().NewQuery(
				"INSERT INTO annotation (user_id, item_id, item_type, starred) VALUES ({:uid}, {:id}, 'media_file', 1)").
				Bind(dbx.Params{"uid": "userid", "id": plsID}).Execute()
			Expect(err).ToNot(HaveOccurred())

			p, err := repo.Get(plsID)
			Expect(err).ToNot(HaveOccurred())
			Expect(p.Starred).To(BeFalse())

			all, err := repo.GetAll()
			Expect(err).ToNot(HaveOccurred())
			matches := 0
			for _, pl := range all {
				if pl.ID == plsID {
					matches++
				}
			}
			Expect(matches).To(Equal(1))
		})

		It("relies on the annotation sweep, not Delete, to clean up annotations", func() {
			Expect(repo.SetStar(true, plsID)).To(Succeed())

			Expect(repo.Delete(plsID)).To(Succeed())
			Expect(countAnnotations()).To(Equal(1))

			Expect(repo.(*playlistRepository).cleanAnnotations()).To(Succeed())
			Expect(countAnnotations()).To(Equal(0))
		})
	})

	It("Put/Exists/Delete", func() {
		By("saves the playlist to the DB")
		newPls := model.Playlist{Name: "Great!", OwnerID: "userid"}
		newPls.AddMediaFilesByID([]string{"1004", "1003"})

		By("saves the playlist to the DB")
		Expect(repo.Put(&newPls)).To(BeNil())

		By("adds repeated songs to a playlist and keeps the order")
		newPls.AddMediaFilesByID([]string{"1004"})
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

	Describe("GetPlaylists", func() {
		It("returns playlists for a track", func() {
			pls, err := repo.GetPlaylists(songRadioactivity.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls).To(HaveLen(1))
			Expect(pls[0].ID).To(Equal(plsBest.ID))
		})

		It("returns empty when none", func() {
			pls, err := repo.GetPlaylists("9999")
			Expect(err).ToNot(HaveOccurred())
			Expect(pls).To(HaveLen(0))
		})
	})

	Describe("Track Deletion and Renumbering", func() {
		var testPlaylistID string

		AfterEach(func() {
			if testPlaylistID != "" {
				Expect(repo.Delete(testPlaylistID)).To(BeNil())
				testPlaylistID = ""
			}
		})

		// helper to get track positions and media file IDs
		getTrackInfo := func(playlistID string) (ids []string, mediaFileIDs []string) {
			pls, err := repo.GetWithTracks(playlistID, false, false)
			Expect(err).ToNot(HaveOccurred())
			for _, t := range pls.Tracks {
				ids = append(ids, t.ID)
				mediaFileIDs = append(mediaFileIDs, t.MediaFileID)
			}
			return
		}

		It("renumbers correctly after deleting a track from the middle", func() {
			By("creating a playlist with 4 tracks")
			newPls := model.Playlist{Name: "Renumber Test Middle", OwnerID: "userid"}
			newPls.AddMediaFilesByID([]string{"1001", "1002", "1003", "1004"})
			Expect(repo.Put(&newPls)).To(Succeed())
			testPlaylistID = newPls.ID

			By("deleting the second track (position 2)")
			tracksRepo := repo.Tracks(newPls.ID, false)
			Expect(tracksRepo.Delete("2")).To(Succeed())

			By("verifying remaining tracks are renumbered sequentially")
			ids, mediaFileIDs := getTrackInfo(newPls.ID)
			Expect(ids).To(Equal([]string{"1", "2", "3"}))
			Expect(mediaFileIDs).To(Equal([]string{"1001", "1003", "1004"}))
		})

		It("renumbers correctly after deleting the first track", func() {
			By("creating a playlist with 3 tracks")
			newPls := model.Playlist{Name: "Renumber Test First", OwnerID: "userid"}
			newPls.AddMediaFilesByID([]string{"1001", "1002", "1003"})
			Expect(repo.Put(&newPls)).To(Succeed())
			testPlaylistID = newPls.ID

			By("deleting the first track (position 1)")
			tracksRepo := repo.Tracks(newPls.ID, false)
			Expect(tracksRepo.Delete("1")).To(Succeed())

			By("verifying remaining tracks are renumbered sequentially")
			ids, mediaFileIDs := getTrackInfo(newPls.ID)
			Expect(ids).To(Equal([]string{"1", "2"}))
			Expect(mediaFileIDs).To(Equal([]string{"1002", "1003"}))
		})

		It("renumbers correctly after deleting the last track", func() {
			By("creating a playlist with 3 tracks")
			newPls := model.Playlist{Name: "Renumber Test Last", OwnerID: "userid"}
			newPls.AddMediaFilesByID([]string{"1001", "1002", "1003"})
			Expect(repo.Put(&newPls)).To(Succeed())
			testPlaylistID = newPls.ID

			By("deleting the last track (position 3)")
			tracksRepo := repo.Tracks(newPls.ID, false)
			Expect(tracksRepo.Delete("3")).To(Succeed())

			By("verifying remaining tracks are renumbered sequentially")
			ids, mediaFileIDs := getTrackInfo(newPls.ID)
			Expect(ids).To(Equal([]string{"1", "2"}))
			Expect(mediaFileIDs).To(Equal([]string{"1001", "1002"}))
		})
	})
})
