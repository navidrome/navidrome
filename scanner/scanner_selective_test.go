package scanner_test

import (
	"context"
	"path/filepath"
	"testing/fstest"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/core/storage/storagetest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/slice"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ScanFolders", Ordered, func() {
	var ctx context.Context
	var lib model.Library
	var ds model.DataStore
	var s model.Scanner
	var fsys storagetest.FakeFS

	BeforeAll(func() {
		ctx = request.WithUser(GinkgoT().Context(), model.User{ID: "123", IsAdmin: true})
		tmpDir := GinkgoT().TempDir()
		conf.Server.DbPath = filepath.Join(tmpDir, "test-selective-scan.db?_journal_mode=WAL")
		log.Warn("Using DB at " + conf.Server.DbPath)
		db.Db().SetMaxOpenConns(1)
	})

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.MusicFolder = "fake:///music"
		conf.Server.DevExternalScanner = false

		db.Init(ctx)
		DeferCleanup(func() {
			Expect(tests.ClearDB()).To(Succeed())
		})

		ds = persistence.New(db.Db())

		// Create the admin user in the database to match the context
		adminUser := model.User{
			ID:          "123",
			UserName:    "admin",
			Name:        "Admin User",
			IsAdmin:     true,
			NewPassword: "password",
		}
		Expect(ds.User(ctx).Put(&adminUser)).To(Succeed())

		s = scanner.New(ctx, ds, artwork.NoopCacheWarmer(), events.NoopBroker(),
			core.NewPlaylists(ds), metrics.NewNoopInstance())

		lib = model.Library{ID: 1, Name: "Fake Library", Path: "fake:///music"}
		Expect(ds.Library(ctx).Put(&lib)).To(Succeed())

		// Initialize fake filesystem
		fsys = storagetest.FakeFS{}
		storagetest.Register("fake", &fsys)
	})

	Describe("Adding tracks to the library", func() {
		It("scans specified folders recursively including all subdirectories", func() {
			rock := template(_t{"albumartist": "Rock Artist", "album": "Rock Album"})
			jazz := template(_t{"albumartist": "Jazz Artist", "album": "Jazz Album"})
			pop := template(_t{"albumartist": "Pop Artist", "album": "Pop Album"})
			createFS(fstest.MapFS{
				"rock/track1.mp3":        rock(track(1, "Rock Track 1")),
				"rock/track2.mp3":        rock(track(2, "Rock Track 2")),
				"rock/subdir/track3.mp3": rock(track(3, "Rock Track 3")),
				"jazz/track4.mp3":        jazz(track(1, "Jazz Track 1")),
				"jazz/subdir/track5.mp3": jazz(track(2, "Jazz Track 2")),
				"pop/track6.mp3":         pop(track(1, "Pop Track 1")),
			})

			// Scan only the "rock" and "jazz" folders (including their subdirectories)
			targets := []model.ScanTarget{
				{LibraryID: lib.ID, FolderPath: "rock"},
				{LibraryID: lib.ID, FolderPath: "jazz"},
			}

			warnings, err := s.ScanFolders(ctx, false, targets)
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeEmpty())

			// Verify all tracks in rock and jazz folders (including subdirectories) were imported
			allFiles, err := ds.MediaFile(ctx).GetAll()
			Expect(err).ToNot(HaveOccurred())

			// Should have 5 tracks (all rock and jazz tracks including subdirectories)
			Expect(allFiles).To(HaveLen(5))

			// Get the file paths
			paths := slice.Map(allFiles, func(mf model.MediaFile) string {
				return filepath.ToSlash(mf.Path)
			})

			// Verify the correct files were scanned (including subdirectories)
			Expect(paths).To(ContainElements(
				"rock/track1.mp3",
				"rock/track2.mp3",
				"rock/subdir/track3.mp3",
				"jazz/track4.mp3",
				"jazz/subdir/track5.mp3",
			))

			// Verify files in the pop folder were NOT scanned
			Expect(paths).ToNot(ContainElement("pop/track6.mp3"))
		})
	})

	Describe("Deleting folders", func() {
		Context("when a child folder is deleted", func() {
			var (
				revolver, help func(...map[string]any) *fstest.MapFile
				artistFolderID string
				album1FolderID string
				album2FolderID string
				album1TrackIDs []string
				album2TrackIDs []string
			)

			BeforeEach(func() {
				// Setup template functions for creating test files
				revolver = storagetest.Template(_t{"albumartist": "The Beatles", "album": "Revolver", "year": 1966})
				help = storagetest.Template(_t{"albumartist": "The Beatles", "album": "Help!", "year": 1965})

				// Initial filesystem with nested folders
				fsys.SetFiles(fstest.MapFS{
					"The Beatles/Revolver/01 - Taxman.mp3":        revolver(storagetest.Track(1, "Taxman")),
					"The Beatles/Revolver/02 - Eleanor Rigby.mp3": revolver(storagetest.Track(2, "Eleanor Rigby")),
					"The Beatles/Help!/01 - Help!.mp3":            help(storagetest.Track(1, "Help!")),
					"The Beatles/Help!/02 - The Night Before.mp3": help(storagetest.Track(2, "The Night Before")),
				})

				// First scan - import everything
				_, err := s.ScanAll(ctx, true)
				Expect(err).ToNot(HaveOccurred())

				// Verify initial state - all folders exist
				folders, err := ds.Folder(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"library_id": lib.ID}})
				Expect(err).ToNot(HaveOccurred())
				Expect(folders).To(HaveLen(4)) // root, Artist, Album1, Album2

				// Store folder IDs for later verification
				for _, f := range folders {
					switch f.Name {
					case "The Beatles":
						artistFolderID = f.ID
					case "Revolver":
						album1FolderID = f.ID
					case "Help!":
						album2FolderID = f.ID
					}
				}

				// Verify all tracks exist
				allTracks, err := ds.MediaFile(ctx).GetAll()
				Expect(err).ToNot(HaveOccurred())
				Expect(allTracks).To(HaveLen(4))

				// Store track IDs for later verification
				for _, t := range allTracks {
					if t.Album == "Revolver" {
						album1TrackIDs = append(album1TrackIDs, t.ID)
					} else if t.Album == "Help!" {
						album2TrackIDs = append(album2TrackIDs, t.ID)
					}
				}

				// Verify no tracks are missing initially
				for _, t := range allTracks {
					Expect(t.Missing).To(BeFalse())
				}
			})

			It("should mark child folder and its tracks as missing when parent is scanned", func() {
				// Delete the child folder (Help!) from the filesystem
				fsys.SetFiles(fstest.MapFS{
					"The Beatles/Revolver/01 - Taxman.mp3":        revolver(storagetest.Track(1, "Taxman")),
					"The Beatles/Revolver/02 - Eleanor Rigby.mp3": revolver(storagetest.Track(2, "Eleanor Rigby")),
					// "The Beatles/Help!" folder and its contents are DELETED
				})

				// Run selective scan on the parent folder (Artist)
				// This simulates what the watcher does when a child folder is deleted
				_, err := s.ScanFolders(ctx, false, []model.ScanTarget{
					{LibraryID: lib.ID, FolderPath: "The Beatles"},
				})
				Expect(err).ToNot(HaveOccurred())

				// Verify the deleted child folder is now marked as missing
				deletedFolder, err := ds.Folder(ctx).Get(album2FolderID)
				Expect(err).ToNot(HaveOccurred())
				Expect(deletedFolder.Missing).To(BeTrue(), "Deleted child folder should be marked as missing")

				// Verify the deleted folder's tracks are marked as missing
				for _, trackID := range album2TrackIDs {
					track, err := ds.MediaFile(ctx).Get(trackID)
					Expect(err).ToNot(HaveOccurred())
					Expect(track.Missing).To(BeTrue(), "Track in deleted folder should be marked as missing")
				}

				// Verify the parent folder is still present and not marked as missing
				parentFolder, err := ds.Folder(ctx).Get(artistFolderID)
				Expect(err).ToNot(HaveOccurred())
				Expect(parentFolder.Missing).To(BeFalse(), "Parent folder should not be marked as missing")

				// Verify the sibling folder and its tracks are still present and not missing
				siblingFolder, err := ds.Folder(ctx).Get(album1FolderID)
				Expect(err).ToNot(HaveOccurred())
				Expect(siblingFolder.Missing).To(BeFalse(), "Sibling folder should not be marked as missing")

				for _, trackID := range album1TrackIDs {
					track, err := ds.MediaFile(ctx).Get(trackID)
					Expect(err).ToNot(HaveOccurred())
					Expect(track.Missing).To(BeFalse(), "Track in sibling folder should not be marked as missing")
				}
			})

			It("should mark deeply nested child folders as missing", func() {
				// Add a deeply nested folder structure
				fsys.SetFiles(fstest.MapFS{
					"The Beatles/Revolver/01 - Taxman.mp3":               revolver(storagetest.Track(1, "Taxman")),
					"The Beatles/Revolver/02 - Eleanor Rigby.mp3":        revolver(storagetest.Track(2, "Eleanor Rigby")),
					"The Beatles/Help!/01 - Help!.mp3":                   help(storagetest.Track(1, "Help!")),
					"The Beatles/Help!/02 - The Night Before.mp3":        help(storagetest.Track(2, "The Night Before")),
					"The Beatles/Help!/Bonus/01 - Bonus Track.mp3":       help(storagetest.Track(99, "Bonus Track")),
					"The Beatles/Help!/Bonus/Nested/01 - Deep Track.mp3": help(storagetest.Track(100, "Deep Track")),
				})

				// Rescan to import the new nested structure
				_, err := s.ScanAll(ctx, true)
				Expect(err).ToNot(HaveOccurred())

				// Verify nested folders were created
				allFolders, err := ds.Folder(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"library_id": lib.ID}})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(allFolders)).To(BeNumerically(">", 4), "Should have more folders with nested structure")

				// Now delete the entire Help! folder including nested children
				fsys.SetFiles(fstest.MapFS{
					"The Beatles/Revolver/01 - Taxman.mp3":        revolver(storagetest.Track(1, "Taxman")),
					"The Beatles/Revolver/02 - Eleanor Rigby.mp3": revolver(storagetest.Track(2, "Eleanor Rigby")),
					// All Help! subfolders are deleted
				})

				// Run selective scan on parent
				_, err = s.ScanFolders(ctx, false, []model.ScanTarget{
					{LibraryID: lib.ID, FolderPath: "The Beatles"},
				})
				Expect(err).ToNot(HaveOccurred())

				// Verify all Help! folders (including nested ones) are marked as missing
				missingFolders, err := ds.Folder(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.And{
						squirrel.Eq{"library_id": lib.ID},
						squirrel.Eq{"missing": true},
					},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(missingFolders)).To(BeNumerically(">", 0), "At least one folder should be marked as missing")

				// Verify all tracks in deleted folders are marked as missing
				allTracks, err := ds.MediaFile(ctx).GetAll()
				Expect(err).ToNot(HaveOccurred())
				Expect(allTracks).To(HaveLen(6))

				for _, track := range allTracks {
					if track.Album == "Help!" {
						Expect(track.Missing).To(BeTrue(), "All tracks in deleted Help! folder should be marked as missing")
					} else if track.Album == "Revolver" {
						Expect(track.Missing).To(BeFalse(), "Tracks in Revolver folder should not be marked as missing")
					}
				}
			})
		})
	})
})
