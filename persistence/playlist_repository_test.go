package persistence

import (
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
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

		// TODO Validate these tests
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

	Describe("Playlist Track Sorting", func() {
		var testPlaylistID string

		AfterEach(func() {
			if testPlaylistID != "" {
				Expect(repo.Delete(testPlaylistID)).To(BeNil())
				testPlaylistID = ""
			}
		})

		It("sorts tracks correctly by album (disc and track number)", func() {
			By("creating a playlist with multi-disc album tracks in arbitrary order")
			newPls := model.Playlist{Name: "Multi-Disc Test", OwnerID: "userid"}
			// Add tracks in intentionally scrambled order
			newPls.AddMediaFilesByID([]string{"2001", "2002", "2003", "2004"})
			Expect(repo.Put(&newPls)).To(Succeed())
			testPlaylistID = newPls.ID

			By("retrieving tracks sorted by album")
			tracksRepo := repo.Tracks(newPls.ID, false)
			tracks, err := tracksRepo.GetAll(model.QueryOptions{Sort: "album", Order: "asc"})
			Expect(err).ToNot(HaveOccurred())

			By("verifying tracks are sorted by disc number then track number")
			Expect(tracks).To(HaveLen(4))
			// Expected order: Disc 1 Track 1, Disc 1 Track 2, Disc 2 Track 1, Disc 2 Track 11
			Expect(tracks[0].MediaFileID).To(Equal("2002")) // Disc 1, Track 1
			Expect(tracks[1].MediaFileID).To(Equal("2004")) // Disc 1, Track 2
			Expect(tracks[2].MediaFileID).To(Equal("2003")) // Disc 2, Track 1
			Expect(tracks[3].MediaFileID).To(Equal("2001")) // Disc 2, Track 11
		})
	})

	Describe("Smart Playlists with Tag Criteria", func() {
		var mfRepo model.MediaFileRepository
		var testPlaylistID string
		var songWithGrouping, songWithoutGrouping model.MediaFile

		BeforeEach(func() {
			ctx := log.NewContext(GinkgoT().Context())
			ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: true})
			mfRepo = NewMediaFileRepository(ctx, GetDBXBuilder())

			// Register 'grouping' as a valid tag for smart playlists
			criteria.AddTagNames([]string{"grouping"})

			// Create a song with the grouping tag
			songWithGrouping = model.MediaFile{
				ID:       "test-grouping-1",
				Title:    "Song With Grouping",
				Artist:   "Test Artist",
				ArtistID: "1",
				Album:    "Test Album",
				AlbumID:  "101",
				Path:     "/test/grouping/song1.mp3",
				Tags: model.Tags{
					"grouping": []string{"My Crate"},
				},
				Participants: model.Participants{},
				LibraryID:    1,
				Lyrics:       "[]",
			}
			Expect(mfRepo.Put(&songWithGrouping)).To(Succeed())

			// Create a song without the grouping tag
			songWithoutGrouping = model.MediaFile{
				ID:           "test-grouping-2",
				Title:        "Song Without Grouping",
				Artist:       "Test Artist",
				ArtistID:     "1",
				Album:        "Test Album",
				AlbumID:      "101",
				Path:         "/test/grouping/song2.mp3",
				Tags:         model.Tags{},
				Participants: model.Participants{},
				LibraryID:    1,
				Lyrics:       "[]",
			}
			Expect(mfRepo.Put(&songWithoutGrouping)).To(Succeed())
		})

		AfterEach(func() {
			if testPlaylistID != "" {
				_ = repo.Delete(testPlaylistID)
				testPlaylistID = ""
			}
			// Clean up test media files
			_, _ = GetDBXBuilder().Delete("media_file", dbx.HashExp{"id": "test-grouping-1"}).Execute()
			_, _ = GetDBXBuilder().Delete("media_file", dbx.HashExp{"id": "test-grouping-2"}).Execute()
		})

		It("matches tracks with a tag value using 'contains' with empty string (issue #4728 workaround)", func() {
			By("creating a smart playlist that checks if grouping tag has any value")
			// This is the workaround for issue #4728: using 'contains' with empty string
			// generates SQL: value LIKE '%%' which matches any non-empty string
			rules := &criteria.Criteria{
				Expression: criteria.All{
					criteria.Contains{"grouping": ""},
				},
			}
			newPls := model.Playlist{Name: "Tracks with Grouping", OwnerID: "userid", Rules: rules}
			Expect(repo.Put(&newPls)).To(Succeed())
			testPlaylistID = newPls.ID

			By("refreshing the smart playlist")
			conf.Server.SmartPlaylistRefreshDelay = -1 * time.Second // Force refresh
			pls, err := repo.GetWithTracks(newPls.ID, true, false)
			Expect(err).ToNot(HaveOccurred())

			By("verifying only the track with grouping tag is matched")
			Expect(pls.Tracks).To(HaveLen(1))
			Expect(pls.Tracks[0].MediaFileID).To(Equal(songWithGrouping.ID))
		})

		It("excludes tracks with a tag value using 'notContains' with empty string", func() {
			By("creating a smart playlist that checks if grouping tag is NOT set")
			rules := &criteria.Criteria{
				Expression: criteria.All{
					criteria.NotContains{"grouping": ""},
				},
			}
			newPls := model.Playlist{Name: "Tracks without Grouping", OwnerID: "userid", Rules: rules}
			Expect(repo.Put(&newPls)).To(Succeed())
			testPlaylistID = newPls.ID

			By("refreshing the smart playlist")
			conf.Server.SmartPlaylistRefreshDelay = -1 * time.Second // Force refresh
			pls, err := repo.GetWithTracks(newPls.ID, true, false)
			Expect(err).ToNot(HaveOccurred())

			By("verifying the track with grouping is NOT in the playlist")
			for _, track := range pls.Tracks {
				Expect(track.MediaFileID).ToNot(Equal(songWithGrouping.ID))
			}

			By("verifying the track without grouping IS in the playlist")
			var foundWithoutGrouping bool
			for _, track := range pls.Tracks {
				if track.MediaFileID == songWithoutGrouping.ID {
					foundWithoutGrouping = true
					break
				}
			}
			Expect(foundWithoutGrouping).To(BeTrue())
		})
	})

	Describe("Smart Playlists Library Filtering", func() {
		var mfRepo model.MediaFileRepository
		var testPlaylistID string
		var lib2ID int
		var restrictedUserID string
		var uniqueLibPath string

		BeforeEach(func() {
			db := GetDBXBuilder()

			// Generate unique IDs for this test run
			uniqueSuffix := time.Now().Format("20060102150405.000")
			restrictedUserID = "restricted-user-" + uniqueSuffix
			uniqueLibPath = "/music/lib2-" + uniqueSuffix

			// Create a second library with unique name and path to avoid conflicts with other tests
			_, err := db.DB().Exec("INSERT INTO library (name, path, created_at, updated_at) VALUES (?, ?, datetime('now'), datetime('now'))", "Library 2-"+uniqueSuffix, uniqueLibPath)
			Expect(err).ToNot(HaveOccurred())
			err = db.DB().QueryRow("SELECT last_insert_rowid()").Scan(&lib2ID)
			Expect(err).ToNot(HaveOccurred())

			// Create a restricted user with access only to library 1
			_, err = db.DB().Exec("INSERT INTO user (id, user_name, name, is_admin, password, created_at, updated_at) VALUES (?, ?, 'Restricted User', false, 'pass', datetime('now'), datetime('now'))", restrictedUserID, restrictedUserID)
			Expect(err).ToNot(HaveOccurred())
			_, err = db.DB().Exec("INSERT INTO user_library (user_id, library_id) VALUES (?, 1)", restrictedUserID)
			Expect(err).ToNot(HaveOccurred())

			// Create test media files in each library
			ctx := log.NewContext(GinkgoT().Context())
			ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: true})
			mfRepo = NewMediaFileRepository(ctx, db)

			// Song in library 1 (accessible by restricted user)
			songLib1 := model.MediaFile{
				ID:           "lib1-song",
				Title:        "Song in Lib1",
				Artist:       "Test Artist",
				ArtistID:     "1",
				Album:        "Test Album",
				AlbumID:      "101",
				Path:         "/music/lib1/song.mp3",
				LibraryID:    1,
				Participants: model.Participants{},
				Tags:         model.Tags{},
				Lyrics:       "[]",
			}
			Expect(mfRepo.Put(&songLib1)).To(Succeed())

			// Song in library 2 (NOT accessible by restricted user)
			songLib2 := model.MediaFile{
				ID:           "lib2-song",
				Title:        "Song in Lib2",
				Artist:       "Test Artist",
				ArtistID:     "1",
				Album:        "Test Album",
				AlbumID:      "101",
				Path:         uniqueLibPath + "/song.mp3",
				LibraryID:    lib2ID,
				Participants: model.Participants{},
				Tags:         model.Tags{},
				Lyrics:       "[]",
			}
			Expect(mfRepo.Put(&songLib2)).To(Succeed())
		})

		AfterEach(func() {
			db := GetDBXBuilder()
			if testPlaylistID != "" {
				_ = repo.Delete(testPlaylistID)
				testPlaylistID = ""
			}
			// Clean up test data
			_, _ = db.Delete("media_file", dbx.HashExp{"id": "lib1-song"}).Execute()
			_, _ = db.Delete("media_file", dbx.HashExp{"id": "lib2-song"}).Execute()
			_, _ = db.Delete("user_library", dbx.HashExp{"user_id": restrictedUserID}).Execute()
			_, _ = db.Delete("user", dbx.HashExp{"id": restrictedUserID}).Execute()
			_, _ = db.DB().Exec("DELETE FROM library WHERE id = ?", lib2ID)
		})

		It("should only include tracks from libraries the user has access to (issue #4738)", func() {
			db := GetDBXBuilder()
			ctx := log.NewContext(GinkgoT().Context())

			// Create the smart playlist as the restricted user
			restrictedUser := model.User{ID: restrictedUserID, UserName: restrictedUserID, IsAdmin: false}
			ctx = request.WithUser(ctx, restrictedUser)
			restrictedRepo := NewPlaylistRepository(ctx, db)

			// Create a smart playlist that matches all songs
			rules := &criteria.Criteria{
				Expression: criteria.All{
					criteria.Gt{"playCount": -1}, // Matches everything
				},
			}
			newPls := model.Playlist{Name: "All Songs", OwnerID: restrictedUserID, Rules: rules}
			Expect(restrictedRepo.Put(&newPls)).To(Succeed())
			testPlaylistID = newPls.ID

			By("refreshing the smart playlist")
			conf.Server.SmartPlaylistRefreshDelay = -1 * time.Second // Force refresh
			pls, err := restrictedRepo.GetWithTracks(newPls.ID, true, false)
			Expect(err).ToNot(HaveOccurred())

			By("verifying only the track from library 1 is in the playlist")
			var foundLib1Song, foundLib2Song bool
			for _, track := range pls.Tracks {
				if track.MediaFileID == "lib1-song" {
					foundLib1Song = true
				}
				if track.MediaFileID == "lib2-song" {
					foundLib2Song = true
				}
			}
			Expect(foundLib1Song).To(BeTrue(), "Song from library 1 should be in the playlist")
			Expect(foundLib2Song).To(BeFalse(), "Song from library 2 should NOT be in the playlist")

			By("verifying playlist_tracks table only contains the accessible track")
			var playlistTracksCount int
			err = db.DB().QueryRow("SELECT count(*) FROM playlist_tracks WHERE playlist_id = ?", newPls.ID).Scan(&playlistTracksCount)
			Expect(err).ToNot(HaveOccurred())
			// Count should only include tracks visible to the user (lib1-song)
			// The count may include other test songs from library 1, but NOT lib2-song
			var lib2TrackCount int
			err = db.DB().QueryRow("SELECT count(*) FROM playlist_tracks WHERE playlist_id = ? AND media_file_id = 'lib2-song'", newPls.ID).Scan(&lib2TrackCount)
			Expect(err).ToNot(HaveOccurred())
			Expect(lib2TrackCount).To(Equal(0), "lib2-song should not be in playlist_tracks")

			By("verifying SongCount matches visible tracks")
			Expect(pls.SongCount).To(Equal(len(pls.Tracks)), "SongCount should match the number of visible tracks")
		})
	})
})
