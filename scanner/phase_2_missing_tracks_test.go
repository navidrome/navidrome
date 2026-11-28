package scanner

import (
	"context"
	"errors"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("phaseMissingTracks", func() {
	var (
		phase *phaseMissingTracks
		ctx   context.Context
		ds    model.DataStore
		state *scanState
	)

	// putTracks stores the tracks, in library 1 unless they name another one.
	putTracks := func(mfs ...*model.MediaFile) {
		GinkgoHelper()
		for _, mf := range mfs {
			if mf.LibraryID == 0 {
				mf.LibraryID = 1
			}
			Expect(ds.MediaFile().Put(ctx, mf)).To(Succeed())
		}
	}
	withStub := func(repos persistencetest.Repos) {
		phase = createPhaseMissingTracks(ctx, state, persistencetest.Stub(ds, repos))
	}
	albumStarred := func(albumID string) bool {
		GinkgoHelper()
		var n int
		Expect(db.Db().QueryRowContext(ctx, "SELECT count(*) FROM annotation WHERE item_type = 'album' AND item_id = ? AND starred",
			albumID).Scan(&n)).To(Succeed())
		return n > 0
	}
	missingCount := func() int64 {
		GinkgoHelper()
		n, err := ds.MediaFile().CountAll(ctx, model.QueryOptions{Filters: squirrel.Eq{"missing": true}})
		Expect(err).ToNot(HaveOccurred())
		return n
	}

	BeforeEach(func() {
		// The scanner runs as an admin, which purging missing files requires
		admin := model.User{ID: "admin-1", UserName: "admin", IsAdmin: true}
		ctx = request.WithUser(GinkgoT().Context(), admin)
		ds = persistence.New(db.Db())
		Expect(ds.User().Put(ctx, &admin)).To(Succeed())
		persistencetest.SetLibraries(ctx, ds, conf.Server().MusicFolder, "/lib2", "/lib3")
		_, err := db.Db().ExecContext(ctx, "UPDATE library SET last_scan_started_at = ? WHERE id = 1", time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC))
		Expect(err).ToNot(HaveOccurred())
		state = &scanState{
			libraries:         model.Libraries{{ID: 1, LastScanStartedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)}},
			totalLibraryCount: 1,
		}
		phase = createPhaseMissingTracks(ctx, state, ds)
	})

	Describe("produceMissingTracks", func() {
		var (
			put      func(tracks *missingTracks)
			produced []*missingTracks
		)

		BeforeEach(func() {
			produced = nil
			put = func(tracks *missingTracks) {
				produced = append(produced, tracks)
			}
		})

		When("there are no missing tracks", func() {
			It("should not call put", func() {
				putTracks(
					&model.MediaFile{ID: "1", PID: "A", Missing: false},
					&model.MediaFile{ID: "2", PID: "A", Missing: false},
				)

				err := phase.produce(put)
				Expect(err).ToNot(HaveOccurred())
				Expect(produced).To(BeEmpty())
			})
		})

		When("there are missing tracks", func() {
			It("should call put for any missing tracks with corresponding matches", func() {
				putTracks(
					&model.MediaFile{ID: "1", PID: "A", Missing: true, LibraryID: 1},
					&model.MediaFile{ID: "2", PID: "B", Missing: true, LibraryID: 1},
					&model.MediaFile{ID: "3", PID: "A", Missing: false, LibraryID: 1},
				)

				err := phase.produce(put)
				Expect(err).ToNot(HaveOccurred())
				Expect(produced).To(HaveLen(2))
				// PID A should have both missing and matched tracks
				var pidA *missingTracks
				for _, p := range produced {
					if p.pid == "A" {
						pidA = p
						break
					}
				}
				Expect(pidA).ToNot(BeNil())
				Expect(pidA.missing).To(HaveLen(1))
				Expect(pidA.matched).To(HaveLen(1))
				// PID B should have only missing tracks
				var pidB *missingTracks
				for _, p := range produced {
					if p.pid == "B" {
						pidB = p
						break
					}
				}
				Expect(pidB).ToNot(BeNil())
				Expect(pidB.missing).To(HaveLen(1))
				Expect(pidB.matched).To(BeEmpty())
			})
			It("should call put for any missing tracks even without matches", func() {
				putTracks(
					&model.MediaFile{ID: "1", PID: "A", Missing: true, LibraryID: 1},
					&model.MediaFile{ID: "2", PID: "B", Missing: true, LibraryID: 1},
					&model.MediaFile{ID: "3", PID: "C", Missing: false, LibraryID: 1},
				)

				err := phase.produce(put)
				Expect(err).ToNot(HaveOccurred())
				Expect(produced).To(HaveLen(2))
				// Both PID A and PID B should be produced even without matches
				var pidA, pidB *missingTracks
				for _, p := range produced {
					switch p.pid {
					case "A":
						pidA = p
					case "B":
						pidB = p
					}
				}
				Expect(pidA).ToNot(BeNil())
				Expect(pidA.missing).To(HaveLen(1))
				Expect(pidA.matched).To(BeEmpty())
				Expect(pidB).ToNot(BeNil())
				Expect(pidB.missing).To(HaveLen(1))
				Expect(pidB.matched).To(BeEmpty())
			})
		})
	})

	Describe("processMissingTracks", func() {
		It("should move the matched track when the missing track is the exact same", func() {
			missingTrack := model.MediaFile{ID: "1", PID: "A", Path: "dir1/path1.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}
			matchedTrack := model.MediaFile{ID: "2", PID: "A", Path: "dir2/path2.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}

			putTracks(&missingTrack)
			putTracks(&matchedTrack)

			in := &missingTracks{
				missing: []model.MediaFile{missingTrack},
				matched: []model.MediaFile{matchedTrack},
			}

			_, err := phase.processMissingTracks(in)
			Expect(err).ToNot(HaveOccurred())
			Expect(phase.totalMatched.Load()).To(Equal(uint32(1)))
			Expect(state.changesDetected.Load()).To(BeTrue())

			movedTrack, _ := ds.MediaFile().Get(ctx, "1")
			Expect(movedTrack.Path).To(Equal(matchedTrack.Path))
		})

		Context("claiming the album annotation reassignment", func() {
			var probe *probeTxDS
			missingTrack := model.MediaFile{ID: "1", PID: "A", AlbumID: "old-album", Path: "dir1/path1.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}
			matchedTrack := model.MediaFile{ID: "2", PID: "A", AlbumID: "new-album", Path: "dir2/path2.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}
			BeforeEach(func() {
				probe = &probeTxDS{DataStore: ds}
				phase = createPhaseMissingTracks(ctx, state, probe)
				putTracks(&missingTrack)
				putTracks(&matchedTrack)
			})

			It("claims the target album before the transaction, so a concurrent move skips it", func() {
				probe.during = func() {
					phase.annotationMutex.RLock()
					defer phase.annotationMutex.RUnlock()
					Expect(phase.processedAlbumAnnotations).To(HaveKeyWithValue("new-album", true))
				}
				Expect(phase.moveMatched(matchedTrack, missingTrack)).To(Succeed())
			})

			It("releases the claim when the move fails, so a later move can reassign", func() {
				probe.err = errors.New("boom")
				Expect(phase.moveMatched(matchedTrack, missingTrack)).To(MatchError("boom"))
				Expect(phase.processedAlbumAnnotations).ToNot(HaveKey("new-album"))
			})
		})

		Context("when the move transaction is rerun after a busy rollback", func() {
			var rerunDS *rerunTxDS
			BeforeEach(func() {
				rerunDS = &rerunTxDS{DataStore: ds}
				phase = createPhaseMissingTracks(ctx, state, rerunDS)
			})

			It("keeps the moved track", func() {
				missingTrack := model.MediaFile{ID: "1", PID: "A", Path: "dir1/path1.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}
				matchedTrack := model.MediaFile{ID: "2", PID: "A", Path: "dir2/path2.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}
				putTracks(&missingTrack)
				putTracks(&matchedTrack)

				_, err := phase.processMissingTracks(&missingTracks{
					missing: []model.MediaFile{missingTrack},
					matched: []model.MediaFile{matchedTrack},
				})
				Expect(err).ToNot(HaveOccurred())

				movedTrack, err := ds.MediaFile().Get(ctx, "1")
				Expect(err).ToNot(HaveOccurred())
				Expect(movedTrack.Path).To(Equal(matchedTrack.Path))
			})

			It("reassigns the album annotations in the attempt that commits", func() {
				Expect(ds.Album().SetStar(ctx, true, "old-album")).To(Succeed())
				missingTrack := model.MediaFile{ID: "1", PID: "A", AlbumID: "old-album", Path: "dir1/path1.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}
				matchedTrack := model.MediaFile{ID: "2", PID: "A", AlbumID: "new-album", Path: "dir2/path2.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}
				putTracks(&missingTrack)
				putTracks(&matchedTrack)

				_, err := phase.processMissingTracks(&missingTracks{
					missing: []model.MediaFile{missingTrack},
					matched: []model.MediaFile{matchedTrack},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(albumStarred("new-album")).To(BeTrue())
				Expect(albumStarred("old-album")).To(BeFalse())
			})
		})

		It("should move the matched track when the missing track has the same tags and filename", func() {
			missingTrack := model.MediaFile{ID: "1", PID: "A", Path: "path1.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}
			matchedTrack := model.MediaFile{ID: "2", PID: "A", Path: "path1.flac", Tags: model.Tags{"title": []string{"title1"}}, Size: 200}

			putTracks(&missingTrack)
			putTracks(&matchedTrack)

			in := &missingTracks{
				missing: []model.MediaFile{missingTrack},
				matched: []model.MediaFile{matchedTrack},
			}

			_, err := phase.processMissingTracks(in)
			Expect(err).ToNot(HaveOccurred())
			Expect(phase.totalMatched.Load()).To(Equal(uint32(1)))
			Expect(state.changesDetected.Load()).To(BeTrue())

			movedTrack, _ := ds.MediaFile().Get(ctx, "1")
			Expect(movedTrack.Path).To(Equal(matchedTrack.Path))
			Expect(movedTrack.Size).To(Equal(matchedTrack.Size))
		})

		It("should move the matched track when there's only one missing track and one matched track (same PID)", func() {
			missingTrack := model.MediaFile{ID: "1", PID: "A", Path: "dir1/path1.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}
			matchedTrack := model.MediaFile{ID: "2", PID: "A", Path: "dir2/path2.flac", Tags: model.Tags{"title": []string{"different title"}}, Size: 200}

			putTracks(&missingTrack)
			putTracks(&matchedTrack)

			in := &missingTracks{
				missing: []model.MediaFile{missingTrack},
				matched: []model.MediaFile{matchedTrack},
			}

			_, err := phase.processMissingTracks(in)
			Expect(err).ToNot(HaveOccurred())
			Expect(phase.totalMatched.Load()).To(Equal(uint32(1)))
			Expect(state.changesDetected.Load()).To(BeTrue())

			movedTrack, _ := ds.MediaFile().Get(ctx, "1")
			Expect(movedTrack.Path).To(Equal(matchedTrack.Path))
			Expect(movedTrack.Size).To(Equal(matchedTrack.Size))
		})

		It("should prioritize exact matches", func() {
			missingTrack := model.MediaFile{ID: "1", PID: "A", Path: "dir1/file1.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}
			matchedEquivalent := model.MediaFile{ID: "2", PID: "A", Path: "dir1/file1.flac", Tags: model.Tags{"title": []string{"title1"}}, Size: 200}
			matchedExact := model.MediaFile{ID: "3", PID: "A", Path: "dir2/file2.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}

			putTracks(&missingTrack)
			putTracks(&matchedEquivalent)
			putTracks(&matchedExact)

			in := &missingTracks{
				missing: []model.MediaFile{missingTrack},
				// Note that equivalent comes before the exact match
				matched: []model.MediaFile{matchedEquivalent, matchedExact},
			}

			_, err := phase.processMissingTracks(in)
			Expect(err).ToNot(HaveOccurred())
			Expect(phase.totalMatched.Load()).To(Equal(uint32(1)))
			Expect(state.changesDetected.Load()).To(BeTrue())

			movedTrack, _ := ds.MediaFile().Get(ctx, "1")
			Expect(movedTrack.Path).To(Equal(matchedExact.Path))
			Expect(movedTrack.Size).To(Equal(matchedExact.Size))
		})

		It("should not move anything if there's more than one match and they don't are not exact nor equivalent", func() {
			missingTrack := model.MediaFile{ID: "1", PID: "A", Path: "dir1/file1.mp3", Title: "title1", Size: 100}
			matched1 := model.MediaFile{ID: "2", PID: "A", Path: "dir1/file2.flac", Title: "another title", Size: 200}
			matched2 := model.MediaFile{ID: "3", PID: "A", Path: "dir2/file3.mp3", Title: "different title", Size: 100}

			putTracks(&missingTrack)
			putTracks(&matched1)
			putTracks(&matched2)

			in := &missingTracks{
				missing: []model.MediaFile{missingTrack},
				matched: []model.MediaFile{matched1, matched2},
			}

			_, err := phase.processMissingTracks(in)
			Expect(err).ToNot(HaveOccurred())
			Expect(phase.totalMatched.Load()).To(Equal(uint32(0)))
			Expect(state.changesDetected.Load()).To(BeFalse())

			// The missing track should still be the same
			movedTrack, _ := ds.MediaFile().Get(ctx, "1")
			Expect(movedTrack.Path).To(Equal(missingTrack.Path))
			Expect(movedTrack.Title).To(Equal(missingTrack.Title))
			Expect(movedTrack.Size).To(Equal(missingTrack.Size))
		})

		It("should not match the same target to multiple missing tracks (prevents duplicate paths)", func() {
			// Simulate a scenario where two missing tracks from different locations have the same
			// base filename and match the same newly imported track via IsEquivalent.
			// Without deduplication, both missing tracks would be "moved" to the same target,
			// creating two non-missing records with the same path.
			missingTrack1 := model.MediaFile{ID: "1", PID: "A", Path: "old_dir1/song.mp3", Title: "title1", Size: 100}
			missingTrack2 := model.MediaFile{ID: "2", PID: "A", Path: "old_dir2/song.mp3", Title: "title1", Size: 100}
			matchedTrack := model.MediaFile{ID: "3", PID: "A", Path: "new_dir/song.mp3", Title: "title1", Size: 200}

			putTracks(&missingTrack1)
			putTracks(&missingTrack2)
			putTracks(&matchedTrack)

			in := &missingTracks{
				missing: []model.MediaFile{missingTrack1, missingTrack2},
				matched: []model.MediaFile{matchedTrack},
			}

			_, err := phase.processMissingTracks(in)
			Expect(err).ToNot(HaveOccurred())
			// Only one of the missing tracks should be matched
			Expect(phase.totalMatched.Load()).To(Equal(uint32(1)))
			Expect(state.changesDetected.Load()).To(BeTrue())

			// The matched track should have been consumed by the first missing track
			movedTrack, _ := ds.MediaFile().Get(ctx, "1")
			Expect(movedTrack.Path).To(Equal(matchedTrack.Path))

			// The second missing track should remain unchanged
			unmatchedTrack, _ := ds.MediaFile().Get(ctx, "2")
			Expect(unmatchedTrack.Path).To(Equal(missingTrack2.Path))
		})

		It("should return an error when there's an error moving the matched track", func() {
			missingTrack := model.MediaFile{ID: "1", PID: "A", Path: "path1.mp3", Tags: model.Tags{"title": []string{"title1"}}}
			matchedTrack := model.MediaFile{ID: "2", PID: "A", Path: "path1.mp3", Tags: model.Tags{"title": []string{"title1"}}}

			putTracks(&missingTrack)
			putTracks(&matchedTrack)

			in := &missingTracks{
				missing: []model.MediaFile{missingTrack},
				matched: []model.MediaFile{matchedTrack},
			}

			withStub(failingMediaFiles(errors.New("db error")))

			_, err := phase.processMissingTracks(in)
			Expect(err).To(HaveOccurred())
			Expect(state.changesDetected.Load()).To(BeFalse())
		})
	})

	Describe("finalize", func() {
		It("should return nil if no error", func() {
			err := phase.finalize(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(state.changesDetected.Load()).To(BeFalse())
		})

		It("should return the error if provided", func() {
			err := phase.finalize(context.DeadlineExceeded)
			Expect(err).To(Equal(context.DeadlineExceeded))
			Expect(state.changesDetected.Load()).To(BeFalse())
		})

		When("PurgeMissing is 'always'", func() {
			BeforeEach(func() {
				conf.Update(func(c *conf.Options) { c.Scanner.PurgeMissing = consts.PurgeMissingAlways })
				putTracks(
					&model.MediaFile{ID: "m1", Path: "m1.mp3", Missing: true},
					&model.MediaFile{ID: "m2", Path: "m2.mp3", Missing: true},
					&model.MediaFile{ID: "m3", Path: "m3.mp3", Missing: true},
				)
			})
			It("should purge missing files", func() {
				Expect(state.changesDetected.Load()).To(BeFalse())
				err := phase.finalize(nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(state.changesDetected.Load()).To(BeTrue())
				Expect(missingCount()).To(BeZero())
			})
		})

		When("PurgeMissing is 'full'", func() {
			BeforeEach(func() {
				conf.Update(func(c *conf.Options) { c.Scanner.PurgeMissing = consts.PurgeMissingFull })
				putTracks(
					&model.MediaFile{ID: "m1", Path: "m1.mp3", Missing: true},
					&model.MediaFile{ID: "m2", Path: "m2.mp3", Missing: true},
				)
			})
			It("should not purge missing files if not a full scan", func() {
				state.fullScan = false
				err := phase.finalize(nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(state.changesDetected.Load()).To(BeFalse())
				Expect(missingCount()).To(Equal(int64(2)))
			})
			It("should purge missing files if full scan", func() {
				Expect(state.changesDetected.Load()).To(BeFalse())
				state.fullScan = true
				err := phase.finalize(nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(state.changesDetected.Load()).To(BeTrue())
				Expect(missingCount()).To(BeZero())
			})
		})

		When("PurgeMissing is 'never'", func() {
			BeforeEach(func() {
				conf.Update(func(c *conf.Options) { c.Scanner.PurgeMissing = consts.PurgeMissingNever })
				putTracks(&model.MediaFile{ID: "m1", Path: "m1.mp3", Missing: true})
			})
			It("should not purge missing files", func() {
				err := phase.finalize(nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(state.changesDetected.Load()).To(BeFalse())
				Expect(missingCount()).To(Equal(int64(1)))
			})
		})
	})

	Describe("processCrossLibraryMoves", func() {
		It("should skip processing if input is nil", func() {
			result, err := phase.processCrossLibraryMoves(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("should skip cross-library move detection when only one library is configured", func() {
			// Default BeforeEach sets up single library, so we just need to verify skip behavior
			Expect(state.totalLibraryCount).To(Equal(1))

			missingTrack := model.MediaFile{
				ID:                "missing1",
				LibraryID:         1,
				MbzReleaseTrackID: "mbz-track-123",
				Title:             "Test Track",
				Size:              1000,
				Suffix:            "mp3",
				Path:              "/lib1/track.mp3",
				Missing:           true,
				CreatedAt:         time.Now().Add(-30 * time.Minute),
			}

			in := &missingTracks{
				lib:     model.Library{ID: 1, Name: "Library 1"},
				missing: []model.MediaFile{missingTrack},
			}

			result, err := phase.processCrossLibraryMoves(in)
			Expect(err).ToNot(HaveOccurred())
			// Should return input unchanged (no processing done)
			Expect(result).To(Equal(in))
			// No matches should be found since cross-library search was skipped
			Expect(phase.totalMatched.Load()).To(Equal(uint32(0)))
			// No changes should be detected
			Expect(state.changesDetected.Load()).To(BeFalse())
		})

		Context("with multiple libraries", func() {
			BeforeEach(func() {
				// Set up multiple libraries for cross-library move tests
				state.libraries = model.Libraries{
					{ID: 1, LastScanStartedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
					{ID: 2, LastScanStartedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)},
				}
				state.totalLibraryCount = 2
			})

			It("should process cross-library moves using MusicBrainz Track ID", func() {
				scanStartTime := time.Now().Add(-1 * time.Hour)
				missingTrack := model.MediaFile{
					ID:                "missing1",
					LibraryID:         1,
					MbzReleaseTrackID: "mbz-track-123",
					Title:             "Test Track",
					Size:              1000,
					Suffix:            "mp3",
					Path:              "/lib1/track.mp3",
					Missing:           true,
					CreatedAt:         scanStartTime.Add(-30 * time.Minute),
				}

				movedTrack := model.MediaFile{
					ID:                "moved1",
					LibraryID:         2,
					MbzReleaseTrackID: "mbz-track-123",
					Title:             "Test Track",
					Size:              1000,
					Suffix:            "mp3",
					Path:              "/lib2/track.mp3",
					Missing:           false,
					CreatedAt:         scanStartTime.Add(-10 * time.Minute),
				}

				putTracks(&missingTrack)
				putTracks(&movedTrack)

				in := &missingTracks{
					lib:     model.Library{ID: 1, Name: "Library 1"},
					missing: []model.MediaFile{missingTrack},
				}

				result, err := phase.processCrossLibraryMoves(in)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(in))
				Expect(phase.totalMatched.Load()).To(Equal(uint32(1)))
				Expect(state.changesDetected.Load()).To(BeTrue())

				// Verify the move was performed
				updatedTrack, _ := ds.MediaFile().Get(ctx, "missing1")
				Expect(updatedTrack.Path).To(Equal("/lib2/track.mp3"))
				Expect(updatedTrack.LibraryID).To(Equal(2))
			})

			It("should fall back to intrinsic properties when MBZ Track ID is empty", func() {
				scanStartTime := time.Now().Add(-1 * time.Hour)
				missingTrack := model.MediaFile{
					ID:                "missing2",
					LibraryID:         1,
					MbzReleaseTrackID: "",
					Title:             "Test Track 2",
					Size:              2000,
					Suffix:            "flac",
					DiscNumber:        1,
					TrackNumber:       1,
					Album:             "Test Album",
					Path:              "/lib1/track2.flac",
					Missing:           true,
					CreatedAt:         scanStartTime.Add(-30 * time.Minute),
				}

				movedTrack := model.MediaFile{
					ID:                "moved2",
					LibraryID:         2,
					MbzReleaseTrackID: "",
					Title:             "Test Track 2",
					Size:              2000,
					Suffix:            "flac",
					DiscNumber:        1,
					TrackNumber:       1,
					Album:             "Test Album",
					Path:              "/lib2/track2.flac",
					Missing:           false,
					CreatedAt:         scanStartTime.Add(-10 * time.Minute),
				}

				putTracks(&missingTrack)
				putTracks(&movedTrack)

				in := &missingTracks{
					lib:     model.Library{ID: 1, Name: "Library 1"},
					missing: []model.MediaFile{missingTrack},
				}

				result, err := phase.processCrossLibraryMoves(in)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(in))
				Expect(phase.totalMatched.Load()).To(Equal(uint32(1)))
				Expect(state.changesDetected.Load()).To(BeTrue())

				// Verify the move was performed
				updatedTrack, _ := ds.MediaFile().Get(ctx, "missing2")
				Expect(updatedTrack.Path).To(Equal("/lib2/track2.flac"))
				Expect(updatedTrack.LibraryID).To(Equal(2))
			})

			It("should not match files in the same library", func() {
				scanStartTime := time.Now().Add(-1 * time.Hour)
				missingTrack := model.MediaFile{
					ID:                "missing3",
					LibraryID:         1,
					MbzReleaseTrackID: "mbz-track-456",
					Title:             "Test Track 3",
					Size:              3000,
					Suffix:            "mp3",
					Path:              "/lib1/track3.mp3",
					Missing:           true,
					CreatedAt:         scanStartTime.Add(-30 * time.Minute),
				}

				sameLibTrack := model.MediaFile{
					ID:                "same1",
					LibraryID:         1, // Same library
					MbzReleaseTrackID: "mbz-track-456",
					Title:             "Test Track 3",
					Size:              3000,
					Suffix:            "mp3",
					Path:              "/lib1/other/track3.mp3",
					Missing:           false,
					CreatedAt:         scanStartTime.Add(-10 * time.Minute),
				}

				putTracks(&missingTrack)
				putTracks(&sameLibTrack)

				in := &missingTracks{
					lib:     model.Library{ID: 1, Name: "Library 1"},
					missing: []model.MediaFile{missingTrack},
				}

				result, err := phase.processCrossLibraryMoves(in)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(in))
				Expect(phase.totalMatched.Load()).To(Equal(uint32(0)))
				Expect(state.changesDetected.Load()).To(BeFalse())
			})

			It("should prioritize MBZ Track ID over intrinsic properties", func() {
				scanStartTime := time.Now().Add(-1 * time.Hour)
				missingTrack := model.MediaFile{
					ID:                "missing4",
					LibraryID:         1,
					MbzReleaseTrackID: "mbz-track-789",
					Title:             "Test Track 4",
					Size:              4000,
					Suffix:            "mp3",
					Path:              "/lib1/track4.mp3",
					Missing:           true,
					CreatedAt:         scanStartTime.Add(-30 * time.Minute),
				}

				// Track with same MBZ ID
				mbzTrack := model.MediaFile{
					ID:                "mbz1",
					LibraryID:         2,
					MbzReleaseTrackID: "mbz-track-789",
					Title:             "Test Track 4",
					Size:              4000,
					Suffix:            "mp3",
					Path:              "/lib2/track4.mp3",
					Missing:           false,
					CreatedAt:         scanStartTime.Add(-10 * time.Minute),
				}

				// Track with same intrinsic properties but no MBZ ID
				intrinsicTrack := model.MediaFile{
					ID:                "intrinsic1",
					LibraryID:         3,
					MbzReleaseTrackID: "",
					Title:             "Test Track 4",
					Size:              4000,
					Suffix:            "mp3",
					DiscNumber:        1,
					TrackNumber:       1,
					Album:             "Test Album",
					Path:              "/lib3/track4.mp3",
					Missing:           false,
					CreatedAt:         scanStartTime.Add(-5 * time.Minute),
				}

				putTracks(&missingTrack)
				putTracks(&mbzTrack)
				putTracks(&intrinsicTrack)

				in := &missingTracks{
					lib:     model.Library{ID: 1, Name: "Library 1"},
					missing: []model.MediaFile{missingTrack},
				}

				result, err := phase.processCrossLibraryMoves(in)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(in))
				Expect(phase.totalMatched.Load()).To(Equal(uint32(1)))
				Expect(state.changesDetected.Load()).To(BeTrue())

				// Verify the MBZ track was chosen (not the intrinsic one)
				updatedTrack, _ := ds.MediaFile().Get(ctx, "missing4")
				Expect(updatedTrack.Path).To(Equal("/lib2/track4.mp3"))
				Expect(updatedTrack.LibraryID).To(Equal(2))
			})

			It("should handle equivalent matches correctly", func() {
				scanStartTime := time.Now().Add(-1 * time.Hour)
				missingTrack := model.MediaFile{
					ID:                "missing5",
					LibraryID:         1,
					MbzReleaseTrackID: "",
					Title:             "Test Track 5",
					Size:              5000,
					Suffix:            "mp3",
					Path:              "/lib1/path/track5.mp3",
					Missing:           true,
					CreatedAt:         scanStartTime.Add(-30 * time.Minute),
				}

				// Equivalent match (same filename, different directory)
				equivalentTrack := model.MediaFile{
					ID:                "equiv1",
					LibraryID:         2,
					MbzReleaseTrackID: "",
					Title:             "Test Track 5",
					Size:              5000,
					Suffix:            "mp3",
					Path:              "/lib2/different/track5.mp3",
					Missing:           false,
					CreatedAt:         scanStartTime.Add(-10 * time.Minute),
				}

				putTracks(&missingTrack)
				putTracks(&equivalentTrack)

				in := &missingTracks{
					lib:     model.Library{ID: 1, Name: "Library 1"},
					missing: []model.MediaFile{missingTrack},
				}

				result, err := phase.processCrossLibraryMoves(in)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(in))
				Expect(phase.totalMatched.Load()).To(Equal(uint32(1)))
				Expect(state.changesDetected.Load()).To(BeTrue())

				// Verify the equivalent match was accepted
				updatedTrack, _ := ds.MediaFile().Get(ctx, "missing5")
				Expect(updatedTrack.Path).To(Equal("/lib2/different/track5.mp3"))
				Expect(updatedTrack.LibraryID).To(Equal(2))
			})

			It("should skip matching when multiple matches are found but none are exact", func() {
				scanStartTime := time.Now().Add(-1 * time.Hour)
				missingTrack := model.MediaFile{
					ID:                "missing6",
					LibraryID:         1,
					MbzReleaseTrackID: "",
					Title:             "Test Track 6",
					Size:              6000,
					Suffix:            "mp3",
					DiscNumber:        1,
					TrackNumber:       1,
					Album:             "Test Album",
					Path:              "/lib1/track6.mp3",
					Missing:           true,
					CreatedAt:         scanStartTime.Add(-30 * time.Minute),
				}

				// Multiple matches with different metadata (not exact matches)
				match1 := model.MediaFile{
					ID:                "match1",
					LibraryID:         2,
					MbzReleaseTrackID: "",
					Title:             "Test Track 6",
					Size:              6000,
					Suffix:            "mp3",
					DiscNumber:        1,
					TrackNumber:       1,
					Album:             "Test Album",
					Path:              "/lib2/different_track.mp3",
					Artist:            "Different Artist", // This makes it non-exact
					Missing:           false,
					CreatedAt:         scanStartTime.Add(-10 * time.Minute),
				}

				match2 := model.MediaFile{
					ID:                "match2",
					LibraryID:         3,
					MbzReleaseTrackID: "",
					Title:             "Test Track 6",
					Size:              6000,
					Suffix:            "mp3",
					DiscNumber:        1,
					TrackNumber:       1,
					Album:             "Test Album",
					Path:              "/lib3/another_track.mp3",
					Artist:            "Another Artist", // This makes it non-exact
					Missing:           false,
					CreatedAt:         scanStartTime.Add(-5 * time.Minute),
				}

				putTracks(&missingTrack)
				putTracks(&match1)
				putTracks(&match2)

				in := &missingTracks{
					lib:     model.Library{ID: 1, Name: "Library 1"},
					missing: []model.MediaFile{missingTrack},
				}

				result, err := phase.processCrossLibraryMoves(in)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(in))
				Expect(phase.totalMatched.Load()).To(Equal(uint32(0)))
				Expect(state.changesDetected.Load()).To(BeFalse())

				// Verify no move was performed
				unchangedTrack, _ := ds.MediaFile().Get(ctx, "missing6")
				Expect(unchangedTrack.Path).To(Equal("/lib1/track6.mp3"))
				Expect(unchangedTrack.LibraryID).To(Equal(1))
			})

			It("should handle errors gracefully", func() {
				withStub(failingMediaFiles(errors.New("db error")))

				missingTrack := model.MediaFile{
					ID:                "missing7",
					LibraryID:         1,
					MbzReleaseTrackID: "mbz-track-error",
					Title:             "Test Track 7",
					Size:              7000,
					Suffix:            "mp3",
					Path:              "/lib1/track7.mp3",
					Missing:           true,
					CreatedAt:         time.Now().Add(-30 * time.Minute),
				}

				in := &missingTracks{
					lib:     model.Library{ID: 1, Name: "Library 1"},
					missing: []model.MediaFile{missingTrack},
				}

				// Should not fail completely, just skip the problematic file
				result, err := phase.processCrossLibraryMoves(in)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(in))
				Expect(phase.totalMatched.Load()).To(Equal(uint32(0)))
				Expect(state.changesDetected.Load()).To(BeFalse())
			})
		}) // End of Context "with multiple libraries"
	})

	Describe("CreatedAt preservation (#5050)", func() {
		var spy *albumSpy

		BeforeEach(func() {
			spy = &albumSpy{reassigned: map[string]string{}, copied: map[string]string{}}
			withStub(spy.repos())
		})

		It("should preserve the missing track's created_at when moving within a library", func() {
			originalTime := time.Date(2020, 3, 15, 10, 0, 0, 0, time.UTC)
			missingTrack := model.MediaFile{
				ID: "1", PID: "A", Path: "old/song.mp3",
				AlbumID:   "album-1",
				LibraryID: 1,
				CreatedAt: originalTime,
				Tags:      model.Tags{"title": []string{"My Song"}},
				Size:      100,
			}
			matchedTrack := model.MediaFile{
				ID: "2", PID: "A", Path: "new/song.mp3",
				AlbumID:   "album-1", // Same album
				LibraryID: 1,
				CreatedAt: time.Now(), // Much newer
				Tags:      model.Tags{"title": []string{"My Song"}},
				Size:      100,
			}

			putTracks(&missingTrack)
			putTracks(&matchedTrack)

			in := &missingTracks{
				missing: []model.MediaFile{missingTrack},
				matched: []model.MediaFile{matchedTrack},
			}

			_, err := phase.processMissingTracks(in)
			Expect(err).ToNot(HaveOccurred())

			movedTrack, _ := ds.MediaFile().Get(ctx, "1")
			Expect(movedTrack.Path).To(Equal("new/song.mp3"))
			Expect(movedTrack.CreatedAt).To(Equal(originalTime))
		})

		It("should preserve created_at during cross-library moves with album change", func() {
			originalTime := time.Date(2019, 6, 1, 12, 0, 0, 0, time.UTC)
			missingTrack := model.MediaFile{
				ID: "missing-ca", PID: "B", Path: "lib1/song.mp3",
				AlbumID:   "old-album",
				LibraryID: 1,
				CreatedAt: originalTime,
			}
			matchedTrack := model.MediaFile{
				ID: "matched-ca", PID: "B", Path: "lib2/song.mp3",
				AlbumID:   "new-album",
				LibraryID: 2,
				CreatedAt: time.Now(),
			}

			// Set up albums so CopyAttributes can find them
			Expect(ds.Album().Put(ctx, &model.Album{ID: "old-album", LibraryID: 1, CreatedAt: originalTime})).To(Succeed())
			Expect(ds.Album().Put(ctx, &model.Album{ID: "new-album", LibraryID: 2, CreatedAt: time.Now()})).To(Succeed())

			putTracks(&missingTrack)
			putTracks(&matchedTrack)

			err := phase.moveMatched(matchedTrack, missingTrack)
			Expect(err).ToNot(HaveOccurred())

			// Track's created_at should be preserved from the missing file
			movedTrack, _ := ds.MediaFile().Get(ctx, "missing-ca")
			Expect(movedTrack.CreatedAt).To(Equal(originalTime))

			// Verify the new album's CreatedAt was copied from the old one
			newAlbum, err := ds.Album().Get(ctx, "new-album")
			Expect(err).ToNot(HaveOccurred())
			Expect(newAlbum.CreatedAt).To(Equal(originalTime))
		})

		It("should not copy album created_at when album ID does not change", func() {
			originalTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
			missingTrack := model.MediaFile{
				ID: "missing-same", PID: "C", Path: "dir1/song.mp3",
				AlbumID:   "same-album",
				LibraryID: 1,
				CreatedAt: originalTime,
			}
			matchedTrack := model.MediaFile{
				ID: "matched-same", PID: "C", Path: "dir2/song.mp3",
				AlbumID:   "same-album", // Same album
				LibraryID: 1,
				CreatedAt: time.Now(),
			}

			putTracks(&missingTrack)
			putTracks(&matchedTrack)

			err := phase.moveMatched(matchedTrack, missingTrack)
			Expect(err).ToNot(HaveOccurred())

			// Track's created_at should still be preserved
			movedTrack, _ := ds.MediaFile().Get(ctx, "missing-same")
			Expect(movedTrack.CreatedAt).To(Equal(originalTime))

			// CopyAttributes should NOT have been called (same album)
			Expect(spy.copied).To(BeEmpty())
		})
	})

	Describe("Album Annotation Reassignment", func() {
		var (
			spy          *albumSpy
			missingTrack model.MediaFile
			matchedTrack model.MediaFile
			oldAlbumID   string
			newAlbumID   string
		)

		BeforeEach(func() {
			spy = &albumSpy{reassigned: map[string]string{}, copied: map[string]string{}}
			withStub(spy.repos())

			oldAlbumID = "old-album-id"
			newAlbumID = "new-album-id"

			missingTrack = model.MediaFile{
				ID:        "missing-track-id",
				PID:       "same-pid",
				Path:      "old/path.mp3",
				AlbumID:   oldAlbumID,
				LibraryID: 1,
				Missing:   true,
				PlayCount: 5,
				Rating:    4,
				Starred:   true,
			}

			matchedTrack = model.MediaFile{
				ID:        "matched-track-id",
				PID:       "same-pid",
				Path:      "new/path.mp3",
				AlbumID:   newAlbumID,
				LibraryID: 2, // Different library
				Missing:   false,
				PlayCount: 2,
				Rating:    3,
				Starred:   false,
			}

			// Store both tracks in the database
			putTracks(&missingTrack)
			putTracks(&matchedTrack)
		})

		When("album ID changes during cross-library move", func() {
			It("should reassign album annotations when AlbumID changes", func() {
				Expect(ds.Album().SetStar(ctx, true, oldAlbumID)).To(Succeed())

				err := phase.moveMatched(matchedTrack, missingTrack)
				Expect(err).ToNot(HaveOccurred())

				Expect(albumStarred(newAlbumID)).To(BeTrue())
				Expect(albumStarred(oldAlbumID)).To(BeFalse())
			})

			It("should not reassign annotations when AlbumID is the same", func() {
				missingTrack.AlbumID = newAlbumID // Same album

				err := phase.moveMatched(matchedTrack, missingTrack)
				Expect(err).ToNot(HaveOccurred())

				// Verify that ReassignAnnotation was NOT called
				Expect(spy.reassigned).To(BeEmpty())
			})
		})

		When("error handling", func() {
			It("should handle ReassignAnnotation errors gracefully", func() {
				// Make the album repo return an error
				spy.err = errors.New("db error")

				// The move should still succeed even if annotation reassignment fails
				err := phase.moveMatched(matchedTrack, missingTrack)
				Expect(err).ToNot(HaveOccurred())

				// Verify that the track was still moved (ID should be updated)
				movedTrack, err := ds.MediaFile().Get(ctx, missingTrack.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(movedTrack.Path).To(Equal(matchedTrack.Path))
			})
		})
	})
})

// rerunTxDS runs every WithTxRetry block twice, rolling back the first attempt as a busy retry would.
type rerunTxDS struct {
	model.DataStore
}

func (d *rerunTxDS) WithTxRetry(ctx context.Context, block func(context.Context, model.DataStore) error, label ...string) error {
	rolledBack := errors.New("rolled back")
	err := d.DataStore.WithTxRetry(ctx, func(ctx context.Context, tx model.DataStore) error {
		if err := block(ctx, tx); err != nil {
			return err
		}
		return rolledBack
	}, label...)
	if !errors.Is(err, rolledBack) {
		return err
	}
	return d.DataStore.WithTxRetry(ctx, block, label...)
}

// probeTxDS runs a hook inside each WithTxRetry block, and can fail the transaction after it.
type probeTxDS struct {
	model.DataStore
	during func()
	err    error
}

func (d *probeTxDS) WithTxRetry(ctx context.Context, block func(context.Context, model.DataStore) error, label ...string) error {
	return d.DataStore.WithTxRetry(ctx, func(ctx context.Context, tx model.DataStore) error {
		if err := block(ctx, tx); err != nil {
			return err
		}
		if d.during != nil {
			d.during()
		}
		return d.err
	}, label...)
}

// failingMediaFiles fails media file deletes and cross-library match lookups.
func failingMediaFiles(err error) persistencetest.Repos {
	return persistencetest.Repos{MediaFile: func(r model.MediaFileRepository) model.MediaFileRepository {
		return &failingMediaFileRepo{MediaFileRepository: r, err: err}
	}}
}

type failingMediaFileRepo struct {
	model.MediaFileRepository
	err error
}

func (r *failingMediaFileRepo) Delete(context.Context, string) error {
	return r.err
}

func (r *failingMediaFileRepo) FindRecentFilesByMBZTrackID(context.Context, model.MediaFile, time.Time) (model.MediaFiles, error) {
	return nil, r.err
}

func (r *failingMediaFileRepo) FindRecentFilesByProperties(context.Context, model.MediaFile, time.Time) (model.MediaFiles, error) {
	return nil, r.err
}

// albumSpy records album annotation reassignments and attribute copies (from -> to), and can fail them.
type albumSpy struct {
	reassigned map[string]string
	copied     map[string]string
	err        error
}

func (s *albumSpy) repos() persistencetest.Repos {
	return persistencetest.Repos{Album: func(r model.AlbumRepository) model.AlbumRepository {
		return &albumSpyRepo{AlbumRepository: r, spy: s}
	}}
}

type albumSpyRepo struct {
	model.AlbumRepository
	spy *albumSpy
}

func (r *albumSpyRepo) ReassignAnnotation(ctx context.Context, prevID, newID string) error {
	r.spy.reassigned[prevID] = newID
	if r.spy.err != nil {
		return r.spy.err
	}
	return r.AlbumRepository.ReassignAnnotation(ctx, prevID, newID)
}

func (r *albumSpyRepo) CopyAttributes(ctx context.Context, fromID, toID string, columns ...string) error {
	r.spy.copied[fromID] = toID
	if r.spy.err != nil {
		return r.spy.err
	}
	return r.AlbumRepository.CopyAttributes(ctx, fromID, toID, columns...)
}
