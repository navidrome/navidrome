package persistence

import (
	"context"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pocketbase/dbx"
)

var _ = Describe("MediaRepository", func() {
	var mr model.MediaFileRepository

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, model.User{ID: "userid"})
		mr = NewMediaFileRepository(ctx, GetDBXBuilder())
	})

	It("gets mediafile from the DB", func() {
		actual, err := mr.Get("1004")
		Expect(err).ToNot(HaveOccurred())
		actual.CreatedAt = time.Time{}
		Expect(actual).To(Equal(&songAntenna))
	})

	It("returns ErrNotFound", func() {
		_, err := mr.Get("56")
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("counts the number of mediafiles in the DB", func() {
		Expect(mr.CountAll()).To(Equal(int64(6)))
	})

	It("returns songs ordered by lyrics with a specific title/artist", func() {
		// attempt to mimic filters.SongsByArtistTitleWithLyricsFirst, except we want all items
		results, err := mr.GetAll(model.QueryOptions{
			Sort:  "lyrics, updated_at",
			Order: "desc",
			Filters: squirrel.And{
				squirrel.Eq{"title": "Antenna"},
				squirrel.Or{
					Exists("json_tree(participants, '$.albumartist')", squirrel.Eq{"value": "Kraftwerk"}),
					Exists("json_tree(participants, '$.artist')", squirrel.Eq{"value": "Kraftwerk"}),
				},
			},
		})

		Expect(err).To(BeNil())
		Expect(results).To(HaveLen(3))
		Expect(results[0].Lyrics).To(Equal(`[{"lang":"xxx","line":[{"value":"This is a set of lyrics"}],"synced":false}]`))
		for _, item := range results[1:] {
			Expect(item.Lyrics).To(Equal("[]"))
			Expect(item.Title).To(Equal("Antenna"))
			Expect(item.Participants[model.RoleArtist][0].Name).To(Equal("Kraftwerk"))
		}
	})

	It("checks existence of mediafiles in the DB", func() {
		Expect(mr.Exists(songAntenna.ID)).To(BeTrue())
		Expect(mr.Exists("666")).To(BeFalse())
	})

	It("delete tracks by id", func() {
		newID := id.NewRandom()
		Expect(mr.Put(&model.MediaFile{LibraryID: 1, ID: newID})).To(Succeed())

		Expect(mr.Delete(newID)).To(Succeed())

		_, err := mr.Get(newID)
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("deletes all missing files", func() {
		new1 := model.MediaFile{ID: id.NewRandom(), LibraryID: 1}
		new2 := model.MediaFile{ID: id.NewRandom(), LibraryID: 1}
		Expect(mr.Put(&new1)).To(Succeed())
		Expect(mr.Put(&new2)).To(Succeed())
		Expect(mr.MarkMissing(true, &new1, &new2)).To(Succeed())

		adminCtx := request.WithUser(log.NewContext(context.TODO()), model.User{ID: "userid", IsAdmin: true})
		adminRepo := NewMediaFileRepository(adminCtx, GetDBXBuilder())

		// Ensure the files are marked as missing and we have 2 of them
		count, err := adminRepo.CountAll(model.QueryOptions{Filters: squirrel.Eq{"missing": true}})
		Expect(count).To(BeNumerically("==", 2))
		Expect(err).ToNot(HaveOccurred())

		count, err = adminRepo.DeleteAllMissing()
		Expect(err).ToNot(HaveOccurred())
		Expect(count).To(BeNumerically("==", 2))

		_, err = mr.Get(new1.ID)
		Expect(err).To(MatchError(model.ErrNotFound))
		_, err = mr.Get(new2.ID)
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	Context("Annotations", func() {
		It("increments play count when the tracks does not have annotations", func() {
			id := "incplay.firsttime"
			Expect(mr.Put(&model.MediaFile{LibraryID: 1, ID: id})).To(BeNil())
			playDate := time.Now()
			Expect(mr.IncPlayCount(id, playDate)).To(BeNil())

			mf, err := mr.Get(id)
			Expect(err).To(BeNil())

			Expect(mf.PlayDate.Unix()).To(Equal(playDate.Unix()))
			Expect(mf.PlayCount).To(Equal(int64(1)))
		})

		It("preserves play date if and only if provided date is older", func() {
			id := "incplay.playdate"
			Expect(mr.Put(&model.MediaFile{LibraryID: 1, ID: id})).To(BeNil())
			playDate := time.Now()
			Expect(mr.IncPlayCount(id, playDate)).To(BeNil())
			mf, err := mr.Get(id)
			Expect(err).To(BeNil())
			Expect(mf.PlayDate.Unix()).To(Equal(playDate.Unix()))
			Expect(mf.PlayCount).To(Equal(int64(1)))

			playDateLate := playDate.AddDate(0, 0, 1)
			Expect(mr.IncPlayCount(id, playDateLate)).To(BeNil())
			mf, err = mr.Get(id)
			Expect(err).To(BeNil())
			Expect(mf.PlayDate.Unix()).To(Equal(playDateLate.Unix()))
			Expect(mf.PlayCount).To(Equal(int64(2)))

			playDateEarly := playDate.AddDate(0, 0, -1)
			Expect(mr.IncPlayCount(id, playDateEarly)).To(BeNil())
			mf, err = mr.Get(id)
			Expect(err).To(BeNil())
			Expect(mf.PlayDate.Unix()).To(Equal(playDateLate.Unix()))
			Expect(mf.PlayCount).To(Equal(int64(3)))
		})

		It("increments play count on newly starred items", func() {
			id := "star.incplay"
			Expect(mr.Put(&model.MediaFile{LibraryID: 1, ID: id})).To(BeNil())
			Expect(mr.SetStar(true, id)).To(BeNil())
			playDate := time.Now()
			Expect(mr.IncPlayCount(id, playDate)).To(BeNil())

			mf, err := mr.Get(id)
			Expect(err).To(BeNil())

			Expect(mf.PlayDate.Unix()).To(Equal(playDate.Unix()))
			Expect(mf.PlayCount).To(Equal(int64(1)))
		})
	})

	Context("Sort options", func() {
		Context("recently_added sort", func() {
			var testMediaFiles []model.MediaFile

			BeforeEach(func() {
				DeferCleanup(configtest.SetupConfig())

				// Create test media files with specific timestamps
				testMediaFiles = []model.MediaFile{
					{
						ID:        id.NewRandom(),
						LibraryID: 1,
						Title:     "Old Song",
						Path:      "/test/old.mp3",
					},
					{
						ID:        id.NewRandom(),
						LibraryID: 1,
						Title:     "Middle Song",
						Path:      "/test/middle.mp3",
					},
					{
						ID:        id.NewRandom(),
						LibraryID: 1,
						Title:     "New Song",
						Path:      "/test/new.mp3",
					},
				}

				// Insert test data first
				for i := range testMediaFiles {
					Expect(mr.Put(&testMediaFiles[i])).To(Succeed())
				}

				// Then manually update timestamps using direct SQL to bypass the repository logic
				db := GetDBXBuilder()

				// Set specific timestamps for testing
				oldTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
				middleTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
				newTime := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)

				// Update "Old Song": created long ago, updated recently
				_, err := db.Update("media_file",
					map[string]interface{}{
						"created_at": oldTime,
						"updated_at": newTime,
					},
					dbx.HashExp{"id": testMediaFiles[0].ID}).Execute()
				Expect(err).ToNot(HaveOccurred())

				// Update "Middle Song": created and updated at the same middle time
				_, err = db.Update("media_file",
					map[string]interface{}{
						"created_at": middleTime,
						"updated_at": middleTime,
					},
					dbx.HashExp{"id": testMediaFiles[1].ID}).Execute()
				Expect(err).ToNot(HaveOccurred())

				// Update "New Song": created recently, updated long ago
				_, err = db.Update("media_file",
					map[string]interface{}{
						"created_at": newTime,
						"updated_at": oldTime,
					},
					dbx.HashExp{"id": testMediaFiles[2].ID}).Execute()
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				// Clean up test data
				for _, mf := range testMediaFiles {
					_ = mr.Delete(mf.ID)
				}
			})

			When("RecentlyAddedByModTime is false", func() {
				var testRepo model.MediaFileRepository

				BeforeEach(func() {
					conf.Server.RecentlyAddedByModTime = false
					// Create repository AFTER setting config
					ctx := log.NewContext(GinkgoT().Context())
					ctx = request.WithUser(ctx, model.User{ID: "userid"})
					testRepo = NewMediaFileRepository(ctx, GetDBXBuilder())
				})

				It("sorts by created_at", func() {
					// Get results sorted by recently_added (should use created_at)
					results, err := testRepo.GetAll(model.QueryOptions{
						Sort:    "recently_added",
						Order:   "desc",
						Filters: squirrel.Eq{"media_file.id": []string{testMediaFiles[0].ID, testMediaFiles[1].ID, testMediaFiles[2].ID}},
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(results).To(HaveLen(3))

					// Verify sorting by created_at (newest first in descending order)
					Expect(results[0].Title).To(Equal("New Song"))    // created 2022
					Expect(results[1].Title).To(Equal("Middle Song")) // created 2021
					Expect(results[2].Title).To(Equal("Old Song"))    // created 2020
				})

				It("sorts in ascending order when specified", func() {
					// Get results sorted by recently_added in ascending order
					results, err := testRepo.GetAll(model.QueryOptions{
						Sort:    "recently_added",
						Order:   "asc",
						Filters: squirrel.Eq{"media_file.id": []string{testMediaFiles[0].ID, testMediaFiles[1].ID, testMediaFiles[2].ID}},
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(results).To(HaveLen(3))

					// Verify sorting by created_at (oldest first)
					Expect(results[0].Title).To(Equal("Old Song"))    // created 2020
					Expect(results[1].Title).To(Equal("Middle Song")) // created 2021
					Expect(results[2].Title).To(Equal("New Song"))    // created 2022
				})
			})

			When("RecentlyAddedByModTime is true", func() {
				var testRepo model.MediaFileRepository

				BeforeEach(func() {
					conf.Server.RecentlyAddedByModTime = true
					// Create repository AFTER setting config
					ctx := log.NewContext(GinkgoT().Context())
					ctx = request.WithUser(ctx, model.User{ID: "userid"})
					testRepo = NewMediaFileRepository(ctx, GetDBXBuilder())
				})

				It("sorts by updated_at", func() {
					// Get results sorted by recently_added (should use updated_at)
					results, err := testRepo.GetAll(model.QueryOptions{
						Sort:    "recently_added",
						Order:   "desc",
						Filters: squirrel.Eq{"media_file.id": []string{testMediaFiles[0].ID, testMediaFiles[1].ID, testMediaFiles[2].ID}},
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(results).To(HaveLen(3))

					// Verify sorting by updated_at (newest first in descending order)
					Expect(results[0].Title).To(Equal("Old Song"))    // updated 2022
					Expect(results[1].Title).To(Equal("Middle Song")) // updated 2021
					Expect(results[2].Title).To(Equal("New Song"))    // updated 2020
				})
			})

		})
	})

	Describe("Search", func() {
		Context("text search", func() {
			It("finds media files by title", func() {
				results, err := mr.Search("Antenna", 0, 10)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(HaveLen(3)) // songAntenna, songAntennaWithLyrics, songAntenna2
				for _, result := range results {
					Expect(result.Title).To(Equal("Antenna"))
				}
			})

			It("finds media files case insensitively", func() {
				results, err := mr.Search("antenna", 0, 10)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(HaveLen(3))
				for _, result := range results {
					Expect(result.Title).To(Equal("Antenna"))
				}
			})

			It("returns empty result when no matches found", func() {
				results, err := mr.Search("nonexistent", 0, 10)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(BeEmpty())
			})
		})

		Context("MBID search", func() {
			var mediaFileWithMBID model.MediaFile
			var raw *mediaFileRepository

			BeforeEach(func() {
				raw = mr.(*mediaFileRepository)
				// Create a test media file with MBID
				mediaFileWithMBID = model.MediaFile{
					ID:                "test-mbid-mediafile",
					Title:             "Test MBID MediaFile",
					MbzRecordingID:    "550e8400-e29b-41d4-a716-446655440020", // Valid UUID v4
					MbzReleaseTrackID: "550e8400-e29b-41d4-a716-446655440021", // Valid UUID v4
					LibraryID:         1,
					Path:              "/test/path/test.mp3",
				}

				// Insert the test media file into the database
				err := mr.Put(&mediaFileWithMBID)
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				// Clean up test data using direct SQL
				_, _ = raw.executeSQL(squirrel.Delete(raw.tableName).Where(squirrel.Eq{"id": mediaFileWithMBID.ID}))
			})

			It("finds media file by mbz_recording_id", func() {
				results, err := mr.Search("550e8400-e29b-41d4-a716-446655440020", 0, 10)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(HaveLen(1))
				Expect(results[0].ID).To(Equal("test-mbid-mediafile"))
				Expect(results[0].Title).To(Equal("Test MBID MediaFile"))
			})

			It("finds media file by mbz_release_track_id", func() {
				results, err := mr.Search("550e8400-e29b-41d4-a716-446655440021", 0, 10)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(HaveLen(1))
				Expect(results[0].ID).To(Equal("test-mbid-mediafile"))
				Expect(results[0].Title).To(Equal("Test MBID MediaFile"))
			})

			It("returns empty result when MBID is not found", func() {
				results, err := mr.Search("550e8400-e29b-41d4-a716-446655440099", 0, 10)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(BeEmpty())
			})

			It("missing media files are never returned by search", func() {
				// Create a missing media file with MBID
				missingMediaFile := model.MediaFile{
					ID:             "test-missing-mbid-mediafile",
					Title:          "Test Missing MBID MediaFile",
					MbzRecordingID: "550e8400-e29b-41d4-a716-446655440022",
					LibraryID:      1,
					Path:           "/test/path/missing.mp3",
					Missing:        true,
				}

				err := mr.Put(&missingMediaFile)
				Expect(err).ToNot(HaveOccurred())

				// Search never returns missing media files (hardcoded behavior)
				results, err := mr.Search("550e8400-e29b-41d4-a716-446655440022", 0, 10)
				Expect(err).ToNot(HaveOccurred())
				Expect(results).To(BeEmpty())

				// Clean up
				_, _ = raw.executeSQL(squirrel.Delete(raw.tableName).Where(squirrel.Eq{"id": missingMediaFile.ID}))
			})
		})
	})
})
