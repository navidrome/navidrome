package scanner

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("phaseMissingTracks", func() {
	var (
		phase *phaseMissingTracks
		ctx   context.Context
		ds    model.DataStore
		mr    *tests.MockMediaFileRepo
		lr    *tests.MockLibraryRepo
		state *scanState
	)

	BeforeEach(func() {
		ctx = context.Background()
		mr = tests.CreateMockMediaFileRepo()
		lr = &tests.MockLibraryRepo{}
		lr.SetData(model.Libraries{{ID: 1, LastScanStartedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)}})
		ds = &tests.MockDataStore{MockedMediaFile: mr, MockedLibrary: lr}
		state = &scanState{
			libraries: model.Libraries{{ID: 1, LastScanStartedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)}},
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
				mr.SetData(model.MediaFiles{
					{ID: "1", PID: "A", Missing: false},
					{ID: "2", PID: "A", Missing: false},
				})

				err := phase.produce(put)
				Expect(err).ToNot(HaveOccurred())
				Expect(produced).To(BeEmpty())
			})
		})

		When("there are missing tracks", func() {
			It("should call put for any missing tracks with corresponding matches", func() {
				mr.SetData(model.MediaFiles{
					{ID: "1", PID: "A", Missing: true, LibraryID: 1},
					{ID: "2", PID: "B", Missing: true, LibraryID: 1},
					{ID: "3", PID: "A", Missing: false, LibraryID: 1},
				})

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
				Expect(pidB.matched).To(HaveLen(0))
			})
			It("should call put for any missing tracks even without matches", func() {
				mr.SetData(model.MediaFiles{
					{ID: "1", PID: "A", Missing: true, LibraryID: 1},
					{ID: "2", PID: "B", Missing: true, LibraryID: 1},
					{ID: "3", PID: "C", Missing: false, LibraryID: 1},
				})

				err := phase.produce(put)
				Expect(err).ToNot(HaveOccurred())
				Expect(produced).To(HaveLen(2))
				// Both PID A and PID B should be produced even without matches
				var pidA, pidB *missingTracks
				for _, p := range produced {
					if p.pid == "A" {
						pidA = p
					} else if p.pid == "B" {
						pidB = p
					}
				}
				Expect(pidA).ToNot(BeNil())
				Expect(pidA.missing).To(HaveLen(1))
				Expect(pidA.matched).To(HaveLen(0))
				Expect(pidB).ToNot(BeNil())
				Expect(pidB.missing).To(HaveLen(1))
				Expect(pidB.matched).To(HaveLen(0))
			})
		})
	})

	Describe("processMissingTracks", func() {
		It("should move the matched track when the missing track is the exact same", func() {
			missingTrack := model.MediaFile{ID: "1", PID: "A", Path: "dir1/path1.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}
			matchedTrack := model.MediaFile{ID: "2", PID: "A", Path: "dir2/path2.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}

			_ = ds.MediaFile(ctx).Put(&missingTrack)
			_ = ds.MediaFile(ctx).Put(&matchedTrack)

			in := &missingTracks{
				missing: []model.MediaFile{missingTrack},
				matched: []model.MediaFile{matchedTrack},
			}

			_, err := phase.processMissingTracks(in)
			Expect(err).ToNot(HaveOccurred())
			Expect(phase.totalMatched.Load()).To(Equal(uint32(1)))
			Expect(state.changesDetected.Load()).To(BeTrue())

			movedTrack, _ := ds.MediaFile(ctx).Get("1")
			Expect(movedTrack.Path).To(Equal(matchedTrack.Path))
		})

		It("should move the matched track when the missing track has the same tags and filename", func() {
			missingTrack := model.MediaFile{ID: "1", PID: "A", Path: "path1.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}
			matchedTrack := model.MediaFile{ID: "2", PID: "A", Path: "path1.flac", Tags: model.Tags{"title": []string{"title1"}}, Size: 200}

			_ = ds.MediaFile(ctx).Put(&missingTrack)
			_ = ds.MediaFile(ctx).Put(&matchedTrack)

			in := &missingTracks{
				missing: []model.MediaFile{missingTrack},
				matched: []model.MediaFile{matchedTrack},
			}

			_, err := phase.processMissingTracks(in)
			Expect(err).ToNot(HaveOccurred())
			Expect(phase.totalMatched.Load()).To(Equal(uint32(1)))
			Expect(state.changesDetected.Load()).To(BeTrue())

			movedTrack, _ := ds.MediaFile(ctx).Get("1")
			Expect(movedTrack.Path).To(Equal(matchedTrack.Path))
			Expect(movedTrack.Size).To(Equal(matchedTrack.Size))
		})

		It("should move the matched track when there's only one missing track and one matched track (same PID)", func() {
			missingTrack := model.MediaFile{ID: "1", PID: "A", Path: "dir1/path1.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}
			matchedTrack := model.MediaFile{ID: "2", PID: "A", Path: "dir2/path2.flac", Tags: model.Tags{"title": []string{"different title"}}, Size: 200}

			_ = ds.MediaFile(ctx).Put(&missingTrack)
			_ = ds.MediaFile(ctx).Put(&matchedTrack)

			in := &missingTracks{
				missing: []model.MediaFile{missingTrack},
				matched: []model.MediaFile{matchedTrack},
			}

			_, err := phase.processMissingTracks(in)
			Expect(err).ToNot(HaveOccurred())
			Expect(phase.totalMatched.Load()).To(Equal(uint32(1)))
			Expect(state.changesDetected.Load()).To(BeTrue())

			movedTrack, _ := ds.MediaFile(ctx).Get("1")
			Expect(movedTrack.Path).To(Equal(matchedTrack.Path))
			Expect(movedTrack.Size).To(Equal(matchedTrack.Size))
		})

		It("should prioritize exact matches", func() {
			missingTrack := model.MediaFile{ID: "1", PID: "A", Path: "dir1/file1.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}
			matchedEquivalent := model.MediaFile{ID: "2", PID: "A", Path: "dir1/file1.flac", Tags: model.Tags{"title": []string{"title1"}}, Size: 200}
			matchedExact := model.MediaFile{ID: "3", PID: "A", Path: "dir2/file2.mp3", Tags: model.Tags{"title": []string{"title1"}}, Size: 100}

			_ = ds.MediaFile(ctx).Put(&missingTrack)
			_ = ds.MediaFile(ctx).Put(&matchedEquivalent)
			_ = ds.MediaFile(ctx).Put(&matchedExact)

			in := &missingTracks{
				missing: []model.MediaFile{missingTrack},
				// Note that equivalent comes before the exact match
				matched: []model.MediaFile{matchedEquivalent, matchedExact},
			}

			_, err := phase.processMissingTracks(in)
			Expect(err).ToNot(HaveOccurred())
			Expect(phase.totalMatched.Load()).To(Equal(uint32(1)))
			Expect(state.changesDetected.Load()).To(BeTrue())

			movedTrack, _ := ds.MediaFile(ctx).Get("1")
			Expect(movedTrack.Path).To(Equal(matchedExact.Path))
			Expect(movedTrack.Size).To(Equal(matchedExact.Size))
		})

		It("should not move anything if there's more than one match and they don't are not exact nor equivalent", func() {
			missingTrack := model.MediaFile{ID: "1", PID: "A", Path: "dir1/file1.mp3", Title: "title1", Size: 100}
			matched1 := model.MediaFile{ID: "2", PID: "A", Path: "dir1/file2.flac", Title: "another title", Size: 200}
			matched2 := model.MediaFile{ID: "3", PID: "A", Path: "dir2/file3.mp3", Title: "different title", Size: 100}

			_ = ds.MediaFile(ctx).Put(&missingTrack)
			_ = ds.MediaFile(ctx).Put(&matched1)
			_ = ds.MediaFile(ctx).Put(&matched2)

			in := &missingTracks{
				missing: []model.MediaFile{missingTrack},
				matched: []model.MediaFile{matched1, matched2},
			}

			_, err := phase.processMissingTracks(in)
			Expect(err).ToNot(HaveOccurred())
			Expect(phase.totalMatched.Load()).To(Equal(uint32(0)))
			Expect(state.changesDetected.Load()).To(BeFalse())

			// The missing track should still be the same
			movedTrack, _ := ds.MediaFile(ctx).Get("1")
			Expect(movedTrack.Path).To(Equal(missingTrack.Path))
			Expect(movedTrack.Title).To(Equal(missingTrack.Title))
			Expect(movedTrack.Size).To(Equal(missingTrack.Size))
		})

		It("should return an error when there's an error moving the matched track", func() {
			missingTrack := model.MediaFile{ID: "1", PID: "A", Path: "path1.mp3", Tags: model.Tags{"title": []string{"title1"}}}
			matchedTrack := model.MediaFile{ID: "2", PID: "A", Path: "path1.mp3", Tags: model.Tags{"title": []string{"title1"}}}

			_ = ds.MediaFile(ctx).Put(&missingTrack)
			_ = ds.MediaFile(ctx).Put(&matchedTrack)

			in := &missingTracks{
				missing: []model.MediaFile{missingTrack},
				matched: []model.MediaFile{matchedTrack},
			}

			// Simulate an error when moving the matched track by deleting the track from the DB
			_ = ds.MediaFile(ctx).Delete("2")

			_, err := phase.processMissingTracks(in)
			Expect(err).To(HaveOccurred())
			Expect(state.changesDetected.Load()).To(BeFalse())
		})
	})

	Describe("finalize", func() {
		It("should return nil if no error", func() {
			err := phase.finalize(nil)
			Expect(err).To(BeNil())
			Expect(state.changesDetected.Load()).To(BeFalse())
		})

		It("should return the error if provided", func() {
			err := phase.finalize(context.DeadlineExceeded)
			Expect(err).To(Equal(context.DeadlineExceeded))
			Expect(state.changesDetected.Load()).To(BeFalse())
		})

		When("PurgeMissing is 'always'", func() {
			BeforeEach(func() {
				conf.Server.Scanner.PurgeMissing = consts.PurgeMissingAlways
				mr.CountAllValue = 3
				mr.DeleteAllMissingValue = 3
			})
			It("should purge missing files", func() {
				Expect(state.changesDetected.Load()).To(BeFalse())
				err := phase.finalize(nil)
				Expect(err).To(BeNil())
				Expect(state.changesDetected.Load()).To(BeTrue())
			})
		})

		When("PurgeMissing is 'full'", func() {
			BeforeEach(func() {
				conf.Server.Scanner.PurgeMissing = consts.PurgeMissingFull
				mr.CountAllValue = 2
				mr.DeleteAllMissingValue = 2
			})
			It("should not purge missing files if not a full scan", func() {
				state.fullScan = false
				err := phase.finalize(nil)
				Expect(err).To(BeNil())
				Expect(state.changesDetected.Load()).To(BeFalse())
			})
			It("should purge missing files if full scan", func() {
				Expect(state.changesDetected.Load()).To(BeFalse())
				state.fullScan = true
				err := phase.finalize(nil)
				Expect(err).To(BeNil())
				Expect(state.changesDetected.Load()).To(BeTrue())
			})
		})

		When("PurgeMissing is 'never'", func() {
			BeforeEach(func() {
				conf.Server.Scanner.PurgeMissing = consts.PurgeMissingNever
				mr.CountAllValue = 1
				mr.DeleteAllMissingValue = 1
			})
			It("should not purge missing files", func() {
				err := phase.finalize(nil)
				Expect(err).To(BeNil())
				Expect(state.changesDetected.Load()).To(BeFalse())
			})
		})
	})

	Describe("processCrossLibraryMoves", func() {
		It("should skip processing if input is nil", func() {
			result, err := phase.processCrossLibraryMoves(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeNil())
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

			_ = ds.MediaFile(ctx).Put(&missingTrack)
			_ = ds.MediaFile(ctx).Put(&movedTrack)

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
			updatedTrack, _ := ds.MediaFile(ctx).Get("missing1")
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

			_ = ds.MediaFile(ctx).Put(&missingTrack)
			_ = ds.MediaFile(ctx).Put(&movedTrack)

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
			updatedTrack, _ := ds.MediaFile(ctx).Get("missing2")
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

			_ = ds.MediaFile(ctx).Put(&missingTrack)
			_ = ds.MediaFile(ctx).Put(&sameLibTrack)

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

			_ = ds.MediaFile(ctx).Put(&missingTrack)
			_ = ds.MediaFile(ctx).Put(&mbzTrack)
			_ = ds.MediaFile(ctx).Put(&intrinsicTrack)

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
			updatedTrack, _ := ds.MediaFile(ctx).Get("missing4")
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

			_ = ds.MediaFile(ctx).Put(&missingTrack)
			_ = ds.MediaFile(ctx).Put(&equivalentTrack)

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
			updatedTrack, _ := ds.MediaFile(ctx).Get("missing5")
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

			_ = ds.MediaFile(ctx).Put(&missingTrack)
			_ = ds.MediaFile(ctx).Put(&match1)
			_ = ds.MediaFile(ctx).Put(&match2)

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
			unchangedTrack, _ := ds.MediaFile(ctx).Get("missing6")
			Expect(unchangedTrack.Path).To(Equal("/lib1/track6.mp3"))
			Expect(unchangedTrack.LibraryID).To(Equal(1))
		})

		It("should handle errors gracefully", func() {
			// Set up mock to return error
			mr.Err = true

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
	})

	Describe("Album Annotation Reassignment", func() {
		var (
			albumRepo    *tests.MockAlbumRepo
			missingTrack model.MediaFile
			matchedTrack model.MediaFile
			oldAlbumID   string
			newAlbumID   string
		)

		BeforeEach(func() {
			albumRepo = ds.Album(ctx).(*tests.MockAlbumRepo)
			albumRepo.ReassignAnnotationCalls = make(map[string]string)

			oldAlbumID = "old-album-id"
			newAlbumID = "new-album-id"

			missingTrack = model.MediaFile{
				ID:        "missing-track-id",
				PID:       "same-pid",
				Path:      "old/path.mp3",
				AlbumID:   oldAlbumID,
				LibraryID: 1,
				Missing:   true,
				Annotations: model.Annotations{
					PlayCount: 5,
					Rating:    4,
					Starred:   true,
				},
			}

			matchedTrack = model.MediaFile{
				ID:        "matched-track-id",
				PID:       "same-pid",
				Path:      "new/path.mp3",
				AlbumID:   newAlbumID,
				LibraryID: 2, // Different library
				Missing:   false,
				Annotations: model.Annotations{
					PlayCount: 2,
					Rating:    3,
					Starred:   false,
				},
			}

			// Store both tracks in the database
			_ = ds.MediaFile(ctx).Put(&missingTrack)
			_ = ds.MediaFile(ctx).Put(&matchedTrack)
		})

		When("album ID changes during cross-library move", func() {
			It("should reassign album annotations when AlbumID changes", func() {
				err := phase.moveMatched(matchedTrack, missingTrack)
				Expect(err).ToNot(HaveOccurred())

				// Verify that ReassignAnnotation was called
				Expect(albumRepo.ReassignAnnotationCalls).To(HaveKeyWithValue(oldAlbumID, newAlbumID))
			})

			It("should not reassign annotations when AlbumID is the same", func() {
				missingTrack.AlbumID = newAlbumID // Same album

				err := phase.moveMatched(matchedTrack, missingTrack)
				Expect(err).ToNot(HaveOccurred())

				// Verify that ReassignAnnotation was NOT called
				Expect(albumRepo.ReassignAnnotationCalls).To(BeEmpty())
			})
		})

		When("error handling", func() {
			It("should handle ReassignAnnotation errors gracefully", func() {
				// Make the album repo return an error
				albumRepo.SetError(true)

				// The move should still succeed even if annotation reassignment fails
				err := phase.moveMatched(matchedTrack, missingTrack)
				Expect(err).ToNot(HaveOccurred())

				// Verify that the track was still moved (ID should be updated)
				movedTrack, err := ds.MediaFile(ctx).Get(missingTrack.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(movedTrack.Path).To(Equal(matchedTrack.Path))
			})
		})
	})
})
