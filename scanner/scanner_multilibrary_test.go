package scanner_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing/fstest"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
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

var _ = Describe("Scanner - Multi-Library", Ordered, func() {
	var ctx context.Context
	var lib1, lib2 model.Library
	var ds *tests.MockDataStore
	var s scanner.Scanner

	createFS := func(path string, files fstest.MapFS) storagetest.FakeFS {
		fs := storagetest.FakeFS{}
		fs.SetFiles(files)
		storagetest.Register(path, &fs)
		return fs
	}

	BeforeAll(func() {
		ctx = request.WithUser(GinkgoT().Context(), model.User{ID: "123", IsAdmin: true})
		tmpDir := GinkgoT().TempDir()
		conf.Server.DbPath = filepath.Join(tmpDir, "test-scanner-multilibrary.db?_journal_mode=WAL")
		log.Warn("Using DB at " + conf.Server.DbPath)
		db.Db().SetMaxOpenConns(1)
	})

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DevExternalScanner = false

		db.Init(ctx)
		DeferCleanup(func() {
			Expect(tests.ClearDB()).To(Succeed())
		})

		ds = &tests.MockDataStore{RealDS: persistence.New(db.Db())}

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

		// Create two test libraries (let DB auto-assign IDs)
		lib1 = model.Library{Name: "Rock Collection", Path: "rock:///music"}
		lib2 = model.Library{Name: "Jazz Collection", Path: "jazz:///music"}
		Expect(ds.Library(ctx).Put(&lib1)).To(Succeed())
		Expect(ds.Library(ctx).Put(&lib2)).To(Succeed())
	})

	runScanner := func(ctx context.Context, fullScan bool) error {
		_, err := s.ScanAll(ctx, fullScan)
		return err
	}

	Context("Two Libraries with Different Content", func() {
		BeforeEach(func() {
			// Rock library content
			beatles := template(_t{"albumartist": "The Beatles", "album": "Abbey Road", "year": 1969, "genre": "Rock"})
			zeppelin := template(_t{"albumartist": "Led Zeppelin", "album": "IV", "year": 1971, "genre": "Rock"})

			_ = createFS("rock", fstest.MapFS{
				"The Beatles/Abbey Road/01 - Come Together.mp3": beatles(track(1, "Come Together")),
				"The Beatles/Abbey Road/02 - Something.mp3":     beatles(track(2, "Something")),
				"Led Zeppelin/IV/01 - Black Dog.mp3":            zeppelin(track(1, "Black Dog")),
				"Led Zeppelin/IV/02 - Rock and Roll.mp3":        zeppelin(track(2, "Rock and Roll")),
			})

			// Jazz library content
			miles := template(_t{"albumartist": "Miles Davis", "album": "Kind of Blue", "year": 1959, "genre": "Jazz"})
			coltrane := template(_t{"albumartist": "John Coltrane", "album": "Giant Steps", "year": 1960, "genre": "Jazz"})

			_ = createFS("jazz", fstest.MapFS{
				"Miles Davis/Kind of Blue/01 - So What.mp3":            miles(track(1, "So What")),
				"Miles Davis/Kind of Blue/02 - Freddie Freeloader.mp3": miles(track(2, "Freddie Freeloader")),
				"John Coltrane/Giant Steps/01 - Giant Steps.mp3":       coltrane(track(1, "Giant Steps")),
				"John Coltrane/Giant Steps/02 - Cousin Mary.mp3":       coltrane(track(2, "Cousin Mary")),
			})
		})

		When("scanning both libraries", func() {
			It("should import files with correct library_id", func() {
				Expect(runScanner(ctx, true)).To(Succeed())

				// Check Rock library media files
				rockFiles, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"library_id": lib1.ID},
					Sort:    "title",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(rockFiles).To(HaveLen(4))

				rockTitles := slice.Map(rockFiles, func(f model.MediaFile) string { return f.Title })
				Expect(rockTitles).To(ContainElements("Come Together", "Something", "Black Dog", "Rock and Roll"))

				// Verify all rock files have correct library_id
				for _, mf := range rockFiles {
					Expect(mf.LibraryID).To(Equal(lib1.ID), "Rock file %s should have library_id %d", mf.Title, lib1.ID)
				}

				// Check Jazz library media files
				jazzFiles, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"library_id": lib2.ID},
					Sort:    "title",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(jazzFiles).To(HaveLen(4))

				jazzTitles := slice.Map(jazzFiles, func(f model.MediaFile) string { return f.Title })
				Expect(jazzTitles).To(ContainElements("So What", "Freddie Freeloader", "Giant Steps", "Cousin Mary"))

				// Verify all jazz files have correct library_id
				for _, mf := range jazzFiles {
					Expect(mf.LibraryID).To(Equal(lib2.ID), "Jazz file %s should have library_id %d", mf.Title, lib2.ID)
				}
			})

			It("should create albums with correct library_id", func() {
				Expect(runScanner(ctx, true)).To(Succeed())

				// Check Rock library albums
				rockAlbums, err := ds.Album(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"library_id": lib1.ID},
					Sort:    "name",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(rockAlbums).To(HaveLen(2))
				Expect(rockAlbums[0].Name).To(Equal("Abbey Road"))
				Expect(rockAlbums[0].LibraryID).To(Equal(lib1.ID))
				Expect(rockAlbums[0].SongCount).To(Equal(2))
				Expect(rockAlbums[1].Name).To(Equal("IV"))
				Expect(rockAlbums[1].LibraryID).To(Equal(lib1.ID))
				Expect(rockAlbums[1].SongCount).To(Equal(2))

				// Check Jazz library albums
				jazzAlbums, err := ds.Album(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"library_id": lib2.ID},
					Sort:    "name",
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(jazzAlbums).To(HaveLen(2))
				Expect(jazzAlbums[0].Name).To(Equal("Giant Steps"))
				Expect(jazzAlbums[0].LibraryID).To(Equal(lib2.ID))
				Expect(jazzAlbums[0].SongCount).To(Equal(2))
				Expect(jazzAlbums[1].Name).To(Equal("Kind of Blue"))
				Expect(jazzAlbums[1].LibraryID).To(Equal(lib2.ID))
				Expect(jazzAlbums[1].SongCount).To(Equal(2))
			})

			It("should create folders with correct library_id", func() {
				Expect(runScanner(ctx, true)).To(Succeed())

				// Check Rock library folders
				rockFolders, err := ds.Folder(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"library_id": lib1.ID},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(rockFolders).To(HaveLen(5)) // ., The Beatles, Led Zeppelin, Abbey Road, IV

				for _, folder := range rockFolders {
					Expect(folder.LibraryID).To(Equal(lib1.ID), "Rock folder %s should have library_id %d", folder.Name, lib1.ID)
				}

				// Check Jazz library folders
				jazzFolders, err := ds.Folder(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"library_id": lib2.ID},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(jazzFolders).To(HaveLen(5)) // ., Miles Davis, John Coltrane, Kind of Blue, Giant Steps

				for _, folder := range jazzFolders {
					Expect(folder.LibraryID).To(Equal(lib2.ID), "Jazz folder %s should have library_id %d", folder.Name, lib2.ID)
				}
			})

			It("should create library-artist associations correctly", func() {
				Expect(runScanner(ctx, true)).To(Succeed())

				// Check library-artist associations

				// Get all artists and check library associations
				allArtists, err := ds.Artist(ctx).GetAll()
				Expect(err).ToNot(HaveOccurred())

				rockArtistNames := []string{}
				jazzArtistNames := []string{}

				for _, artist := range allArtists {
					// Check if artist is associated with rock library
					var count int64
					err := db.Db().QueryRow(
						"SELECT COUNT(*) FROM library_artist WHERE library_id = ? AND artist_id = ?",
						lib1.ID, artist.ID,
					).Scan(&count)
					Expect(err).ToNot(HaveOccurred())
					if count > 0 {
						rockArtistNames = append(rockArtistNames, artist.Name)
					}

					// Check if artist is associated with jazz library
					err = db.Db().QueryRow(
						"SELECT COUNT(*) FROM library_artist WHERE library_id = ? AND artist_id = ?",
						lib2.ID, artist.ID,
					).Scan(&count)
					Expect(err).ToNot(HaveOccurred())
					if count > 0 {
						jazzArtistNames = append(jazzArtistNames, artist.Name)
					}
				}

				Expect(rockArtistNames).To(ContainElements("The Beatles", "Led Zeppelin"))
				Expect(jazzArtistNames).To(ContainElements("Miles Davis", "John Coltrane"))

				// Artists should not be shared between libraries (except [Unknown Artist])
				for _, name := range rockArtistNames {
					if name != "[Unknown Artist]" {
						Expect(jazzArtistNames).ToNot(ContainElement(name))
					}
				}
			})

			It("should update library statistics correctly", func() {
				Expect(runScanner(ctx, true)).To(Succeed())

				// Check Rock library stats
				rockLib, err := ds.Library(ctx).Get(lib1.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(rockLib.TotalSongs).To(Equal(4))
				Expect(rockLib.TotalAlbums).To(Equal(2))

				Expect(rockLib.TotalArtists).To(Equal(3)) // The Beatles, Led Zeppelin, [Unknown Artist]
				Expect(rockLib.TotalFolders).To(Equal(2)) // Abbey Road, IV (only folders with audio files)

				// Check Jazz library stats
				jazzLib, err := ds.Library(ctx).Get(lib2.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(jazzLib.TotalSongs).To(Equal(4))
				Expect(jazzLib.TotalAlbums).To(Equal(2))
				Expect(jazzLib.TotalArtists).To(Equal(3)) // Miles Davis, John Coltrane, [Unknown Artist]
				Expect(jazzLib.TotalFolders).To(Equal(2)) // Kind of Blue, Giant Steps (only folders with audio files)
			})
		})

		When("libraries have different content", func() {
			It("should maintain separate statistics per library", func() {
				Expect(runScanner(ctx, true)).To(Succeed())

				// Verify rock library stats
				rockLib, err := ds.Library(ctx).Get(lib1.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(rockLib.TotalSongs).To(Equal(4))
				Expect(rockLib.TotalAlbums).To(Equal(2))

				// Verify jazz library stats
				jazzLib, err := ds.Library(ctx).Get(lib2.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(jazzLib.TotalSongs).To(Equal(4))
				Expect(jazzLib.TotalAlbums).To(Equal(2))

				// Verify that libraries don't interfere with each other
				rockFiles, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"library_id": lib1.ID},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(rockFiles).To(HaveLen(4))

				jazzFiles, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"library_id": lib2.ID},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(jazzFiles).To(HaveLen(4))
			})
		})

		When("verifying library isolation", func() {
			It("should keep library data completely separate", func() {
				Expect(runScanner(ctx, true)).To(Succeed())

				// Verify that rock library only contains rock content
				rockAlbums, err := ds.Album(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"library_id": lib1.ID},
				})
				Expect(err).ToNot(HaveOccurred())
				rockAlbumNames := slice.Map(rockAlbums, func(a model.Album) string { return a.Name })
				Expect(rockAlbumNames).To(ContainElements("Abbey Road", "IV"))
				Expect(rockAlbumNames).ToNot(ContainElements("Kind of Blue", "Giant Steps"))

				// Verify that jazz library only contains jazz content
				jazzAlbums, err := ds.Album(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"library_id": lib2.ID},
				})
				Expect(err).ToNot(HaveOccurred())
				jazzAlbumNames := slice.Map(jazzAlbums, func(a model.Album) string { return a.Name })
				Expect(jazzAlbumNames).To(ContainElements("Kind of Blue", "Giant Steps"))
				Expect(jazzAlbumNames).ToNot(ContainElements("Abbey Road", "IV"))
			})
		})

		When("same artist appears in different libraries", func() {
			It("should associate artist with both libraries correctly", func() {
				// Create libraries with Jeff Beck albums in both
				jeffRock := template(_t{"albumartist": "Jeff Beck", "album": "Truth", "year": 1968, "genre": "Rock"})
				jeffJazz := template(_t{"albumartist": "Jeff Beck", "album": "Blow by Blow", "year": 1975, "genre": "Jazz"})
				beatles := template(_t{"albumartist": "The Beatles", "album": "Abbey Road", "year": 1969, "genre": "Rock"})
				miles := template(_t{"albumartist": "Miles Davis", "album": "Kind of Blue", "year": 1959, "genre": "Jazz"})

				// Create rock library with Jeff Beck's Truth album
				_ = createFS("rock", fstest.MapFS{
					"The Beatles/Abbey Road/01 - Come Together.mp3": beatles(track(1, "Come Together")),
					"The Beatles/Abbey Road/02 - Something.mp3":     beatles(track(2, "Something")),
					"Jeff Beck/Truth/01 - Beck's Bolero.mp3":        jeffRock(track(1, "Beck's Bolero")),
					"Jeff Beck/Truth/02 - Ol' Man River.mp3":        jeffRock(track(2, "Ol' Man River")),
				})

				// Create jazz library with Jeff Beck's Blow by Blow album
				_ = createFS("jazz", fstest.MapFS{
					"Miles Davis/Kind of Blue/01 - So What.mp3":            miles(track(1, "So What")),
					"Miles Davis/Kind of Blue/02 - Freddie Freeloader.mp3": miles(track(2, "Freddie Freeloader")),
					"Jeff Beck/Blow by Blow/01 - You Know What I Mean.mp3": jeffJazz(track(1, "You Know What I Mean")),
					"Jeff Beck/Blow by Blow/02 - She's a Woman.mp3":        jeffJazz(track(2, "She's a Woman")),
				})

				Expect(runScanner(ctx, true)).To(Succeed())

				// Jeff Beck should be associated with both libraries
				var rockCount, jazzCount int64

				// Get Jeff Beck artist ID
				jeffArtists, err := ds.Artist(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"name": "Jeff Beck"},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(jeffArtists).To(HaveLen(1))
				jeffID := jeffArtists[0].ID

				// Check rock library association
				err = db.Db().QueryRow(
					"SELECT COUNT(*) FROM library_artist WHERE library_id = ? AND artist_id = ?",
					lib1.ID, jeffID,
				).Scan(&rockCount)
				Expect(err).ToNot(HaveOccurred())
				Expect(rockCount).To(Equal(int64(1)))

				// Check jazz library association
				err = db.Db().QueryRow(
					"SELECT COUNT(*) FROM library_artist WHERE library_id = ? AND artist_id = ?",
					lib2.ID, jeffID,
				).Scan(&jazzCount)
				Expect(err).ToNot(HaveOccurred())
				Expect(jazzCount).To(Equal(int64(1)))

				// Verify Jeff Beck albums are in correct libraries
				rockAlbums, err := ds.Album(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"library_id": lib1.ID, "album_artist": "Jeff Beck"},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(rockAlbums).To(HaveLen(1))
				Expect(rockAlbums[0].Name).To(Equal("Truth"))

				jazzAlbums, err := ds.Album(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"library_id": lib2.ID, "album_artist": "Jeff Beck"},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(jazzAlbums).To(HaveLen(1))
				Expect(jazzAlbums[0].Name).To(Equal("Blow by Blow"))
			})
		})
	})

	Context("Incremental Scan Behavior", func() {
		BeforeEach(func() {
			// Start with minimal content in both libraries
			rock := template(_t{"albumartist": "Queen", "album": "News of the World", "year": 1977, "genre": "Rock"})
			jazz := template(_t{"albumartist": "Bill Evans", "album": "Waltz for Debby", "year": 1961, "genre": "Jazz"})

			createFS("rock", fstest.MapFS{
				"Queen/News of the World/01 - We Will Rock You.mp3": rock(track(1, "We Will Rock You")),
			})

			createFS("jazz", fstest.MapFS{
				"Bill Evans/Waltz for Debby/01 - My Foolish Heart.mp3": jazz(track(1, "My Foolish Heart")),
			})
		})

		It("should handle incremental scans per library correctly", func() {
			// Initial full scan
			Expect(runScanner(ctx, true)).To(Succeed())

			// Verify initial state
			rockFiles, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"library_id": lib1.ID},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(rockFiles).To(HaveLen(1))

			jazzFiles, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"library_id": lib2.ID},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(jazzFiles).To(HaveLen(1))

			// Incremental scan should not duplicate existing files
			Expect(runScanner(ctx, false)).To(Succeed())

			// Verify counts remain the same
			rockFiles, err = ds.MediaFile(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"library_id": lib1.ID},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(rockFiles).To(HaveLen(1))

			jazzFiles, err = ds.MediaFile(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"library_id": lib2.ID},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(jazzFiles).To(HaveLen(1))
		})
	})

	Context("Missing Files Handling", func() {
		var rockFS storagetest.FakeFS

		BeforeEach(func() {
			rock := template(_t{"albumartist": "AC/DC", "album": "Back in Black", "year": 1980, "genre": "Rock"})

			rockFS = createFS("rock", fstest.MapFS{
				"AC-DC/Back in Black/01 - Hells Bells.mp3":     rock(track(1, "Hells Bells")),
				"AC-DC/Back in Black/02 - Shoot to Thrill.mp3": rock(track(2, "Shoot to Thrill")),
			})

			createFS("jazz", fstest.MapFS{
				"Herbie Hancock/Head Hunters/01 - Chameleon.mp3": template(_t{
					"albumartist": "Herbie Hancock", "album": "Head Hunters", "year": 1973, "genre": "Jazz",
				})(track(1, "Chameleon")),
			})
		})

		It("should mark missing files correctly per library", func() {
			// Initial scan
			Expect(runScanner(ctx, true)).To(Succeed())

			// Remove one file from rock library only
			rockFS.Remove("AC-DC/Back in Black/02 - Shoot to Thrill.mp3")

			// Rescan
			Expect(runScanner(ctx, false)).To(Succeed())

			// Check that only the rock library file is marked as missing
			missingRockFiles, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.And{
					squirrel.Eq{"library_id": lib1.ID},
					squirrel.Eq{"missing": true},
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(missingRockFiles).To(HaveLen(1))
			Expect(missingRockFiles[0].Title).To(Equal("Shoot to Thrill"))

			// Check that jazz library files are not affected
			missingJazzFiles, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.And{
					squirrel.Eq{"library_id": lib2.ID},
					squirrel.Eq{"missing": true},
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(missingJazzFiles).To(HaveLen(0))

			// Verify non-missing files
			presentRockFiles, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.And{
					squirrel.Eq{"library_id": lib1.ID},
					squirrel.Eq{"missing": false},
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(presentRockFiles).To(HaveLen(1))
			Expect(presentRockFiles[0].Title).To(Equal("Hells Bells"))
		})
	})

	Context("Error Handling - Multi-Library", func() {
		Context("Filesystem errors affecting one library", func() {
			var rockFS storagetest.FakeFS

			BeforeEach(func() {
				// Set up content for both libraries
				rock := template(_t{"albumartist": "AC/DC", "album": "Back in Black", "year": 1980, "genre": "Rock"})
				jazz := template(_t{"albumartist": "Miles Davis", "album": "Kind of Blue", "year": 1959, "genre": "Jazz"})

				rockFS = createFS("rock", fstest.MapFS{
					"AC-DC/Back in Black/01 - Hells Bells.mp3":     rock(track(1, "Hells Bells")),
					"AC-DC/Back in Black/02 - Shoot to Thrill.mp3": rock(track(2, "Shoot to Thrill")),
				})

				createFS("jazz", fstest.MapFS{
					"Miles Davis/Kind of Blue/01 - So What.mp3":            jazz(track(1, "So What")),
					"Miles Davis/Kind of Blue/02 - Freddie Freeloader.mp3": jazz(track(2, "Freddie Freeloader")),
				})
			})

			It("should not affect scanning of other libraries", func() {
				// Inject filesystem read error in rock library only
				rockFS.SetError("AC-DC/Back in Black/01 - Hells Bells.mp3", errors.New("filesystem read error"))

				// Scan should succeed overall and return warnings
				warnings, err := s.ScanAll(ctx, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(warnings).ToNot(BeEmpty(), "Should have warnings for filesystem errors")

				// Jazz library should have been scanned successfully
				jazzFiles, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"library_id": lib2.ID},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(jazzFiles).To(HaveLen(2))
				Expect(jazzFiles[0].Title).To(BeElementOf("So What", "Freddie Freeloader"))
				Expect(jazzFiles[1].Title).To(BeElementOf("So What", "Freddie Freeloader"))

				// Rock library may have partial content (depending on scanner implementation)
				rockFiles, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"library_id": lib1.ID},
				})
				Expect(err).ToNot(HaveOccurred())
				// No specific expectation - some files may have been imported despite errors
				_ = rockFiles

				// Verify jazz library stats are correct
				jazzLib, err := ds.Library(ctx).Get(lib2.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(jazzLib.TotalSongs).To(Equal(2))

				// Error should be empty (warnings don't count as scan errors)
				lastError, err := ds.Property(ctx).DefaultGet(consts.LastScanErrorKey, "unset")
				Expect(err).ToNot(HaveOccurred())
				Expect(lastError).To(BeEmpty())
			})

			It("should continue with warnings for affected library", func() {
				// Inject read errors on multiple files in rock library
				rockFS.SetError("AC-DC/Back in Black/01 - Hells Bells.mp3", errors.New("read error 1"))
				rockFS.SetError("AC-DC/Back in Black/02 - Shoot to Thrill.mp3", errors.New("read error 2"))

				// Scan should complete with warnings for multiple filesystem errors
				warnings, err := s.ScanAll(ctx, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(warnings).ToNot(BeEmpty(), "Should have warnings for multiple filesystem errors")

				// Jazz library should be completely unaffected
				jazzFiles, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"library_id": lib2.ID},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(jazzFiles).To(HaveLen(2))

				// Jazz library statistics should be accurate
				jazzLib, err := ds.Library(ctx).Get(lib2.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(jazzLib.TotalSongs).To(Equal(2))
				Expect(jazzLib.TotalAlbums).To(Equal(1))

				// Error should be empty (warnings don't count as scan errors)
				lastError, err := ds.Property(ctx).DefaultGet(consts.LastScanErrorKey, "unset")
				Expect(err).ToNot(HaveOccurred())
				Expect(lastError).To(BeEmpty())
			})
		})

		Context("Database errors during multi-library scanning", func() {
			BeforeEach(func() {
				// Set up content for both libraries
				rock := template(_t{"albumartist": "Queen", "album": "News of the World", "year": 1977, "genre": "Rock"})
				jazz := template(_t{"albumartist": "Bill Evans", "album": "Waltz for Debby", "year": 1961, "genre": "Jazz"})

				createFS("rock", fstest.MapFS{
					"Queen/News of the World/01 - We Will Rock You.mp3": rock(track(1, "We Will Rock You")),
				})

				createFS("jazz", fstest.MapFS{
					"Bill Evans/Waltz for Debby/01 - My Foolish Heart.mp3": jazz(track(1, "My Foolish Heart")),
				})
			})

			It("should propagate database errors and stop scanning", func() {
				// Install mock repo that injects DB error
				mfRepo := &mockMediaFileRepo{
					MediaFileRepository:        ds.RealDS.MediaFile(ctx),
					GetMissingAndMatchingError: errors.New("database connection failed"),
				}
				ds.MockedMediaFile = mfRepo

				// Scan should return the database error
				Expect(runScanner(ctx, false)).To(MatchError(ContainSubstring("database connection failed")))

				// Error should be recorded in scanner properties
				lastError, err := ds.Property(ctx).DefaultGet(consts.LastScanErrorKey, "")
				Expect(err).ToNot(HaveOccurred())
				Expect(lastError).To(ContainSubstring("database connection failed"))
			})

			It("should preserve error information in scanner properties", func() {
				// Install mock repo that injects DB error
				mfRepo := &mockMediaFileRepo{
					MediaFileRepository:        ds.RealDS.MediaFile(ctx),
					GetMissingAndMatchingError: errors.New("critical database error"),
				}
				ds.MockedMediaFile = mfRepo

				// Attempt scan (should fail)
				Expect(runScanner(ctx, false)).To(HaveOccurred())

				// Check that error is recorded in scanner properties
				lastError, err := ds.Property(ctx).DefaultGet(consts.LastScanErrorKey, "")
				Expect(err).ToNot(HaveOccurred())
				Expect(lastError).To(ContainSubstring("critical database error"))

				// Scan type should still be recorded
				scanType, _ := ds.Property(ctx).DefaultGet(consts.LastScanTypeKey, "")
				Expect(scanType).To(BeElementOf("incremental", "quick"))
			})
		})

		Context("Mixed error scenarios", func() {
			var rockFS storagetest.FakeFS

			BeforeEach(func() {
				// Set up rock library with filesystem that can error
				rock := template(_t{"albumartist": "Metallica", "album": "Master of Puppets", "year": 1986, "genre": "Metal"})
				rockFS = createFS("rock", fstest.MapFS{
					"Metallica/Master of Puppets/01 - Battery.mp3":           rock(track(1, "Battery")),
					"Metallica/Master of Puppets/02 - Master of Puppets.mp3": rock(track(2, "Master of Puppets")),
				})

				// Set up jazz library normally
				jazz := template(_t{"albumartist": "Herbie Hancock", "album": "Head Hunters", "year": 1973, "genre": "Jazz"})
				createFS("jazz", fstest.MapFS{
					"Herbie Hancock/Head Hunters/01 - Chameleon.mp3": jazz(track(1, "Chameleon")),
				})
			})

			It("should handle filesystem errors in one library while other succeeds", func() {
				// Inject filesystem error in rock library
				rockFS.SetError("Metallica/Master of Puppets/01 - Battery.mp3", errors.New("disk read error"))

				// Scan should complete with warnings (not hard error)
				warnings, err := s.ScanAll(ctx, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(warnings).ToNot(BeEmpty(), "Should have warnings for filesystem error")

				// Jazz library should scan completely successfully
				jazzFiles, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"library_id": lib2.ID},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(jazzFiles).To(HaveLen(1))
				Expect(jazzFiles[0].Title).To(Equal("Chameleon"))

				// Jazz library statistics should be accurate
				jazzLib, err := ds.Library(ctx).Get(lib2.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(jazzLib.TotalSongs).To(Equal(1))
				Expect(jazzLib.TotalAlbums).To(Equal(1))

				// Rock library may have partial content (depending on scanner implementation)
				rockFiles, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"library_id": lib1.ID},
				})
				Expect(err).ToNot(HaveOccurred())
				// No specific expectation - some files may have been imported despite errors
				_ = rockFiles

				// Error should be empty (warnings don't count as scan errors)
				lastError, err := ds.Property(ctx).DefaultGet(consts.LastScanErrorKey, "unset")
				Expect(err).ToNot(HaveOccurred())
				Expect(lastError).To(BeEmpty())
			})

			It("should handle partial failures gracefully", func() {
				// Create a scenario where rock has filesystem issues and jazz has normal content
				rockFS.SetError("Metallica/Master of Puppets/01 - Battery.mp3", errors.New("file corruption"))

				// Do an initial scan with filesystem error
				warnings, err := s.ScanAll(ctx, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(warnings).ToNot(BeEmpty(), "Should have warnings for file corruption")

				// Verify that the working parts completed successfully
				jazzFiles, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"library_id": lib2.ID},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(jazzFiles).To(HaveLen(1))

				// Scanner properties should reflect successful completion despite warnings
				scanType, _ := ds.Property(ctx).DefaultGet(consts.LastScanTypeKey, "")
				Expect(scanType).To(Equal("full"))

				// Start time should be recorded
				startTimeStr, _ := ds.Property(ctx).DefaultGet(consts.LastScanStartTimeKey, "")
				Expect(startTimeStr).ToNot(BeEmpty())

				// Error should be empty (warnings don't count as scan errors)
				lastError, err := ds.Property(ctx).DefaultGet(consts.LastScanErrorKey, "unset")
				Expect(err).ToNot(HaveOccurred())
				Expect(lastError).To(BeEmpty())
			})
		})

		Context("Error recovery in multi-library context", func() {
			It("should recover from previous library-specific errors", func() {
				// Set up initial content
				rock := template(_t{"albumartist": "Iron Maiden", "album": "The Number of the Beast", "year": 1982, "genre": "Metal"})
				jazz := template(_t{"albumartist": "John Coltrane", "album": "Giant Steps", "year": 1960, "genre": "Jazz"})

				rockFS := createFS("rock", fstest.MapFS{
					"Iron Maiden/The Number of the Beast/01 - Invaders.mp3": rock(track(1, "Invaders")),
				})

				createFS("jazz", fstest.MapFS{
					"John Coltrane/Giant Steps/01 - Giant Steps.mp3": jazz(track(1, "Giant Steps")),
				})

				// First scan with filesystem error in rock
				rockFS.SetError("Iron Maiden/The Number of the Beast/01 - Invaders.mp3", errors.New("temporary disk error"))
				warnings, err := s.ScanAll(ctx, true)
				Expect(err).ToNot(HaveOccurred()) // Should succeed with warnings
				Expect(warnings).ToNot(BeEmpty(), "Should have warnings for temporary disk error")

				// Clear the error and add more content - recreate the filesystem completely
				rockFS.ClearError("Iron Maiden/The Number of the Beast/01 - Invaders.mp3")

				// Create a new filesystem with both files
				createFS("rock", fstest.MapFS{
					"Iron Maiden/The Number of the Beast/01 - Invaders.mp3":               rock(track(1, "Invaders")),
					"Iron Maiden/The Number of the Beast/02 - Children of the Damned.mp3": rock(track(2, "Children of the Damned")),
				})

				// Second scan should recover and import all rock content
				warnings, err = s.ScanAll(ctx, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(warnings).ToNot(BeEmpty(), "Should have warnings for temporary disk error")

				// Verify both libraries now have content (at least jazz should work)
				rockFiles, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"library_id": lib1.ID},
				})
				Expect(err).ToNot(HaveOccurred())
				// The scanner should recover and import both rock files
				Expect(len(rockFiles)).To(Equal(2))

				jazzFiles, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"library_id": lib2.ID},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(jazzFiles).To(HaveLen(1))

				// Both libraries should have correct content counts
				rockLib, err := ds.Library(ctx).Get(lib1.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(rockLib.TotalSongs).To(Equal(2))

				jazzLib, err := ds.Library(ctx).Get(lib2.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(jazzLib.TotalSongs).To(Equal(1))

				// Error should be empty (successful recovery)
				lastError, err := ds.Property(ctx).DefaultGet(consts.LastScanErrorKey, "unset")
				Expect(err).ToNot(HaveOccurred())
				Expect(lastError).To(BeEmpty())
			})
		})
	})

	Context("Scanner Properties", func() {
		It("should persist last scan type, start time and error properties", func() {
			// trivial FS setup
			rock := template(_t{"albumartist": "AC/DC", "album": "Back in Black", "year": 1980, "genre": "Rock"})
			_ = createFS("rock", fstest.MapFS{
				"AC-DC/Back in Black/01 - Hells Bells.mp3": rock(track(1, "Hells Bells")),
			})

			// Run a full scan
			Expect(runScanner(ctx, true)).To(Succeed())

			// Validate properties
			scanType, _ := ds.Property(ctx).DefaultGet(consts.LastScanTypeKey, "")
			Expect(scanType).To(Equal("full"))

			startTimeStr, _ := ds.Property(ctx).DefaultGet(consts.LastScanStartTimeKey, "")
			Expect(startTimeStr).ToNot(BeEmpty())
			_, err := time.Parse(time.RFC3339, startTimeStr)
			Expect(err).ToNot(HaveOccurred())

			lastError, err := ds.Property(ctx).DefaultGet(consts.LastScanErrorKey, "unset")
			Expect(err).ToNot(HaveOccurred())
			Expect(lastError).To(BeEmpty())
		})
	})
})
