package scanner_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing/fstest"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
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

// Easy aliases for the storagetest package
type _t = map[string]any

var template = storagetest.Template
var track = storagetest.Track

func createFS(files fstest.MapFS) storagetest.FakeFS {
	fs := storagetest.FakeFS{}
	fs.SetFiles(files)
	storagetest.Register("fake", &fs)
	return fs
}

var _ = Describe("Scanner", Ordered, func() {
	var ctx context.Context
	var lib model.Library
	var ds *tests.MockDataStore
	var mfRepo *mockMediaFileRepo
	var s model.Scanner

	BeforeAll(func() {
		ctx = request.WithUser(GinkgoT().Context(), model.User{ID: "123", IsAdmin: true})
		tmpDir := GinkgoT().TempDir()
		conf.Server.DbPath = filepath.Join(tmpDir, "test-scanner.db?_journal_mode=WAL")
		log.Warn("Using DB at " + conf.Server.DbPath)
		//conf.Server.DbPath = ":memory:"
		db.Db().SetMaxOpenConns(1)
	})

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.MusicFolder = "fake:///music" // Set to match test library path
		conf.Server.DevExternalScanner = false

		db.Init(ctx)
		DeferCleanup(func() {
			Expect(tests.ClearDB()).To(Succeed())
		})

		ds = &tests.MockDataStore{RealDS: persistence.New(db.Db())}
		mfRepo = &mockMediaFileRepo{
			MediaFileRepository: ds.RealDS.MediaFile(ctx),
		}
		ds.MockedMediaFile = mfRepo

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
	})

	runScanner := func(ctx context.Context, fullScan bool) error {
		_, err := s.ScanAll(ctx, fullScan)
		return err
	}

	Context("Simple library, 'artis/album/track - title.mp3'", func() {
		var help, revolver func(...map[string]any) *fstest.MapFile
		var fsys storagetest.FakeFS
		BeforeEach(func() {
			revolver = template(_t{"albumartist": "The Beatles", "album": "Revolver", "year": 1966})
			help = template(_t{"albumartist": "The Beatles", "album": "Help!", "year": 1965})
			fsys = createFS(fstest.MapFS{
				"The Beatles/Revolver/01 - Taxman.mp3":                         revolver(track(1, "Taxman")),
				"The Beatles/Revolver/02 - Eleanor Rigby.mp3":                  revolver(track(2, "Eleanor Rigby")),
				"The Beatles/Revolver/03 - I'm Only Sleeping.mp3":              revolver(track(3, "I'm Only Sleeping")),
				"The Beatles/Revolver/04 - Love You To.mp3":                    revolver(track(4, "Love You To")),
				"The Beatles/Help!/01 - Help!.mp3":                             help(track(1, "Help!")),
				"The Beatles/Help!/02 - The Night Before.mp3":                  help(track(2, "The Night Before")),
				"The Beatles/Help!/03 - You've Got to Hide Your Love Away.mp3": help(track(3, "You've Got to Hide Your Love Away")),
			})
		})
		When("it is the first scan", func() {
			It("should import all folders", func() {
				Expect(runScanner(ctx, true)).To(Succeed())

				folders, _ := ds.Folder(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"library_id": lib.ID}})
				paths := slice.Map(folders, func(f model.Folder) string { return f.Name })
				Expect(paths).To(SatisfyAll(
					HaveLen(4),
					ContainElements(".", "The Beatles", "Revolver", "Help!"),
				))
			})
			It("should import all mediafiles", func() {
				Expect(runScanner(ctx, true)).To(Succeed())

				mfs, _ := ds.MediaFile(ctx).GetAll()
				paths := slice.Map(mfs, func(f model.MediaFile) string { return f.Title })
				Expect(paths).To(SatisfyAll(
					HaveLen(7),
					ContainElements(
						"Taxman", "Eleanor Rigby", "I'm Only Sleeping", "Love You To",
						"Help!", "The Night Before", "You've Got to Hide Your Love Away",
					),
				))
			})
			It("should import all albums", func() {
				Expect(runScanner(ctx, true)).To(Succeed())

				albums, _ := ds.Album(ctx).GetAll(model.QueryOptions{Sort: "name"})
				Expect(albums).To(HaveLen(2))
				Expect(albums[0]).To(SatisfyAll(
					HaveField("Name", Equal("Help!")),
					HaveField("SongCount", Equal(3)),
				))
				Expect(albums[1]).To(SatisfyAll(
					HaveField("Name", Equal("Revolver")),
					HaveField("SongCount", Equal(4)),
				))
			})
		})
		When("a file was changed", func() {
			It("should update the media_file", func() {
				Expect(runScanner(ctx, true)).To(Succeed())

				mf, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"title": "Help!"}})
				Expect(err).ToNot(HaveOccurred())
				Expect(mf[0].Tags).ToNot(HaveKey("barcode"))

				fsys.UpdateTags("The Beatles/Help!/01 - Help!.mp3", _t{"barcode": "123"})
				Expect(runScanner(ctx, true)).To(Succeed())

				mf, err = ds.MediaFile(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"title": "Help!"}})
				Expect(err).ToNot(HaveOccurred())
				Expect(mf[0].Tags).To(HaveKeyWithValue(model.TagName("barcode"), []string{"123"}))
			})

			It("should update the album", func() {
				Expect(runScanner(ctx, true)).To(Succeed())

				albums, err := ds.Album(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album.name": "Help!"}})
				Expect(err).ToNot(HaveOccurred())
				Expect(albums).ToNot(BeEmpty())
				Expect(albums[0].Participants.First(model.RoleProducer).Name).To(BeEmpty())
				Expect(albums[0].SongCount).To(Equal(3))

				fsys.UpdateTags("The Beatles/Help!/01 - Help!.mp3", _t{"producer": "George Martin"})
				Expect(runScanner(ctx, false)).To(Succeed())

				albums, err = ds.Album(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album.name": "Help!"}})
				Expect(err).ToNot(HaveOccurred())
				Expect(albums[0].Participants.First(model.RoleProducer).Name).To(Equal("George Martin"))
				Expect(albums[0].SongCount).To(Equal(3))
			})
		})
	})

	Context("Ignored entries", func() {
		BeforeEach(func() {
			revolver := template(_t{"albumartist": "The Beatles", "album": "Revolver", "year": 1966})
			createFS(fstest.MapFS{
				"The Beatles/Revolver/01 - Taxman.mp3":   revolver(track(1, "Taxman")),
				"The Beatles/Revolver/._01 - Taxman.mp3": &fstest.MapFile{Data: []byte("garbage data")},
			})
		})

		It("should not import the ignored file", func() {
			Expect(runScanner(ctx, true)).To(Succeed())

			mfs, err := ds.MediaFile(ctx).GetAll()
			Expect(err).ToNot(HaveOccurred())
			Expect(mfs).To(HaveLen(1))
			for _, mf := range mfs {
				Expect(mf.Title).To(Equal("Taxman"))
				Expect(mf.Path).To(Equal("The Beatles/Revolver/01 - Taxman.mp3"))
			}
		})
	})

	Context("Same album in two different folders", func() {
		BeforeEach(func() {
			revolver := template(_t{"albumartist": "The Beatles", "album": "Revolver", "year": 1966})
			createFS(fstest.MapFS{
				"The Beatles/Revolver/01 - Taxman.mp3":         revolver(track(1, "Taxman")),
				"The Beatles/Revolver2/02 - Eleanor Rigby.mp3": revolver(track(2, "Eleanor Rigby")),
			})
		})

		It("should import as one album", func() {
			Expect(runScanner(ctx, true)).To(Succeed())

			albums, err := ds.Album(ctx).GetAll()
			Expect(err).ToNot(HaveOccurred())
			Expect(albums).To(HaveLen(1))

			mfs, err := ds.MediaFile(ctx).GetAll()
			Expect(err).ToNot(HaveOccurred())
			Expect(mfs).To(HaveLen(2))
			for _, mf := range mfs {
				Expect(mf.AlbumID).To(Equal(albums[0].ID))
			}
		})
	})

	Context("Same album, different release dates", func() {
		BeforeEach(func() {
			help := template(_t{"albumartist": "The Beatles", "album": "Help!", "releasedate": 1965})
			help2 := template(_t{"albumartist": "The Beatles", "album": "Help!", "releasedate": 2000})
			createFS(fstest.MapFS{
				"The Beatles/Help!/01 - Help!.mp3":            help(track(1, "Help!")),
				"The Beatles/Help! (remaster)/01 - Help!.mp3": help2(track(1, "Help!")),
			})
		})

		It("should import as two distinct albums", func() {
			Expect(runScanner(ctx, true)).To(Succeed())

			albums, err := ds.Album(ctx).GetAll(model.QueryOptions{Sort: "release_date"})
			Expect(err).ToNot(HaveOccurred())
			Expect(albums).To(HaveLen(2))
			Expect(albums[0]).To(SatisfyAll(
				HaveField("Name", Equal("Help!")),
				HaveField("ReleaseDate", Equal("1965")),
			))
			Expect(albums[1]).To(SatisfyAll(
				HaveField("Name", Equal("Help!")),
				HaveField("ReleaseDate", Equal("2000")),
			))
		})
	})

	Describe("Library changes'", func() {
		var help, revolver func(...map[string]any) *fstest.MapFile
		var fsys storagetest.FakeFS
		var findByPath func(string) (*model.MediaFile, error)
		var beatlesMBID = uuid.NewString()

		BeforeEach(func() {
			By("Having two MP3 albums")
			beatles := _t{
				"artist":               "The Beatles",
				"artistsort":           "Beatles, The",
				"musicbrainz_artistid": beatlesMBID,
			}
			help = template(beatles, _t{"album": "Help!", "year": 1965})
			revolver = template(beatles, _t{"album": "Revolver", "year": 1966})
			fsys = createFS(fstest.MapFS{
				"The Beatles/Help!/01 - Help!.mp3":            help(track(1, "Help!")),
				"The Beatles/Help!/02 - The Night Before.mp3": help(track(2, "The Night Before")),
				"The Beatles/Revolver/01 - Taxman.mp3":        revolver(track(1, "Taxman")),
				"The Beatles/Revolver/02 - Eleanor Rigby.mp3": revolver(track(2, "Eleanor Rigby")),
			})

			By("Doing a full scan")
			Expect(runScanner(ctx, true)).To(Succeed())
			Expect(ds.MediaFile(ctx).CountAll()).To(Equal(int64(4)))
			findByPath = createFindByPath(ctx, ds)
		})

		It("adds new files to the library", func() {
			fsys.Add("The Beatles/Revolver/03 - I'm Only Sleeping.mp3", revolver(track(3, "I'm Only Sleeping")))

			Expect(runScanner(ctx, false)).To(Succeed())
			Expect(ds.MediaFile(ctx).CountAll()).To(Equal(int64(5)))
			mf, err := findByPath("The Beatles/Revolver/03 - I'm Only Sleeping.mp3")
			Expect(err).ToNot(HaveOccurred())
			Expect(mf.Title).To(Equal("I'm Only Sleeping"))
		})

		It("updates tags of a file in the library", func() {
			fsys.UpdateTags("The Beatles/Revolver/02 - Eleanor Rigby.mp3", _t{"title": "Eleanor Rigby (remix)"})

			Expect(runScanner(ctx, false)).To(Succeed())
			Expect(ds.MediaFile(ctx).CountAll()).To(Equal(int64(4)))
			mf, _ := findByPath("The Beatles/Revolver/02 - Eleanor Rigby.mp3")
			Expect(mf.Title).To(Equal("Eleanor Rigby (remix)"))
		})

		It("upgrades file with same format in the library", func() {
			fsys.Add("The Beatles/Revolver/01 - Taxman.mp3", revolver(track(1, "Taxman", _t{"bitrate": 640})))

			Expect(runScanner(ctx, false)).To(Succeed())
			Expect(ds.MediaFile(ctx).CountAll()).To(Equal(int64(4)))
			mf, _ := findByPath("The Beatles/Revolver/01 - Taxman.mp3")
			Expect(mf.BitRate).To(Equal(640))
		})

		It("detects a file was removed from the library", func() {
			By("Removing a file")
			fsys.Remove("The Beatles/Revolver/02 - Eleanor Rigby.mp3")

			By("Rescanning the library")
			Expect(runScanner(ctx, false)).To(Succeed())

			By("Checking the file is marked as missing")
			Expect(ds.MediaFile(ctx).CountAll(model.QueryOptions{
				Filters: squirrel.Eq{"missing": false},
			})).To(Equal(int64(3)))
			mf, err := findByPath("The Beatles/Revolver/02 - Eleanor Rigby.mp3")
			Expect(err).ToNot(HaveOccurred())
			Expect(mf.Missing).To(BeTrue())
		})

		It("detects a file was moved to a different folder", func() {
			By("Storing the original ID")
			original, err := findByPath("The Beatles/Revolver/02 - Eleanor Rigby.mp3")
			Expect(err).ToNot(HaveOccurred())
			originalId := original.ID

			By("Moving the file to a different folder")
			fsys.Move("The Beatles/Revolver/02 - Eleanor Rigby.mp3", "The Beatles/Help!/02 - Eleanor Rigby.mp3")

			By("Rescanning the library")
			Expect(runScanner(ctx, false)).To(Succeed())

			By("Checking the old file is not in the library")
			Expect(ds.MediaFile(ctx).CountAll(model.QueryOptions{
				Filters: squirrel.Eq{"missing": false},
			})).To(Equal(int64(4)))
			_, err = findByPath("The Beatles/Revolver/02 - Eleanor Rigby.mp3")
			Expect(err).To(MatchError(model.ErrNotFound))

			By("Checking the new file is in the library")
			Expect(ds.MediaFile(ctx).CountAll(model.QueryOptions{
				Filters: squirrel.Eq{"missing": true},
			})).To(BeZero())
			mf, err := findByPath("The Beatles/Help!/02 - Eleanor Rigby.mp3")
			Expect(err).ToNot(HaveOccurred())
			Expect(mf.Title).To(Equal("Eleanor Rigby"))
			Expect(mf.Missing).To(BeFalse())

			By("Checking the new file has the same ID as the original")
			Expect(mf.ID).To(Equal(originalId))
		})

		It("detects a move after a scan is interrupted by an error", func() {
			By("Storing the original ID")
			By("Moving the file to a different folder")
			fsys.Move("The Beatles/Revolver/01 - Taxman.mp3", "The Beatles/Help!/01 - Taxman.mp3")

			By("Interrupting the scan with an error before the move is processed")
			mfRepo.GetMissingAndMatchingError = errors.New("I/O read error")
			Expect(runScanner(ctx, false)).To(MatchError(ContainSubstring("I/O read error")))

			By("Checking the both instances of the file are in the lib")
			Expect(ds.MediaFile(ctx).CountAll(model.QueryOptions{
				Filters: squirrel.Eq{"title": "Taxman"},
			})).To(Equal(int64(2)))

			By("Rescanning the library without error")
			mfRepo.GetMissingAndMatchingError = nil
			Expect(runScanner(ctx, false)).To(Succeed())

			By("Checking the old file is not in the library")
			mfs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"title": "Taxman"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(mfs).To(HaveLen(1))
			Expect(mfs[0].Path).To(Equal("The Beatles/Help!/01 - Taxman.mp3"))
		})

		It("detects file format upgrades", func() {
			By("Storing the original ID")
			original, err := findByPath("The Beatles/Revolver/02 - Eleanor Rigby.mp3")
			Expect(err).ToNot(HaveOccurred())
			originalId := original.ID

			By("Replacing the file with a different format")
			fsys.Move("The Beatles/Revolver/02 - Eleanor Rigby.mp3", "The Beatles/Revolver/02 - Eleanor Rigby.flac")

			By("Rescanning the library")
			Expect(runScanner(ctx, false)).To(Succeed())

			By("Checking the old file is not in the library")
			Expect(ds.MediaFile(ctx).CountAll(model.QueryOptions{
				Filters: squirrel.Eq{"missing": true},
			})).To(BeZero())
			_, err = findByPath("The Beatles/Revolver/02 - Eleanor Rigby.mp3")
			Expect(err).To(MatchError(model.ErrNotFound))

			By("Checking the new file is in the library")
			Expect(ds.MediaFile(ctx).CountAll(model.QueryOptions{
				Filters: squirrel.Eq{"missing": false},
			})).To(Equal(int64(4)))
			mf, err := findByPath("The Beatles/Revolver/02 - Eleanor Rigby.flac")
			Expect(err).ToNot(HaveOccurred())
			Expect(mf.Title).To(Equal("Eleanor Rigby"))
			Expect(mf.Missing).To(BeFalse())

			By("Checking the new file has the same ID as the original")
			Expect(mf.ID).To(Equal(originalId))
		})

		It("detects old missing tracks being added back", func() {
			By("Removing a file")
			origFile := fsys.Remove("The Beatles/Revolver/02 - Eleanor Rigby.mp3")

			By("Rescanning the library")
			Expect(runScanner(ctx, false)).To(Succeed())

			By("Checking the file is marked as missing")
			Expect(ds.MediaFile(ctx).CountAll(model.QueryOptions{
				Filters: squirrel.Eq{"missing": false},
			})).To(Equal(int64(3)))
			mf, err := findByPath("The Beatles/Revolver/02 - Eleanor Rigby.mp3")
			Expect(err).ToNot(HaveOccurred())
			Expect(mf.Missing).To(BeTrue())

			By("Adding the file back")
			fsys.Add("The Beatles/Revolver/02 - Eleanor Rigby.mp3", origFile)

			By("Rescanning the library again")
			Expect(runScanner(ctx, false)).To(Succeed())

			By("Checking the file is not marked as missing")
			Expect(ds.MediaFile(ctx).CountAll(model.QueryOptions{
				Filters: squirrel.Eq{"missing": false},
			})).To(Equal(int64(4)))
			mf, err = findByPath("The Beatles/Revolver/02 - Eleanor Rigby.mp3")
			Expect(err).ToNot(HaveOccurred())
			Expect(mf.Missing).To(BeFalse())

			By("Removing it again")
			fsys.Remove("The Beatles/Revolver/02 - Eleanor Rigby.mp3")

			By("Rescanning the library again")
			Expect(runScanner(ctx, false)).To(Succeed())

			By("Checking the file is marked as missing")
			mf, err = findByPath("The Beatles/Revolver/02 - Eleanor Rigby.mp3")
			Expect(err).ToNot(HaveOccurred())
			Expect(mf.Missing).To(BeTrue())

			By("Adding the file back in a different folder")
			fsys.Add("The Beatles/Help!/02 - Eleanor Rigby.mp3", origFile)

			By("Rescanning the library once more")
			Expect(runScanner(ctx, false)).To(Succeed())

			By("Checking the file was found in the new folder")
			Expect(ds.MediaFile(ctx).CountAll(model.QueryOptions{
				Filters: squirrel.Eq{"missing": false},
			})).To(Equal(int64(4)))
			mf, err = findByPath("The Beatles/Help!/02 - Eleanor Rigby.mp3")
			Expect(err).ToNot(HaveOccurred())
			Expect(mf.Missing).To(BeFalse())
		})

		It("marks tracks as missing when scanning a deleted folder with ScanFolders", func() {
			By("Adding a third track to Revolver to have more test data")
			fsys.Add("The Beatles/Revolver/03 - I'm Only Sleeping.mp3", revolver(track(3, "I'm Only Sleeping")))
			Expect(runScanner(ctx, false)).To(Succeed())

			By("Verifying initial state has 5 tracks")
			Expect(ds.MediaFile(ctx).CountAll(model.QueryOptions{
				Filters: squirrel.Eq{"missing": false},
			})).To(Equal(int64(5)))

			By("Removing the entire Revolver folder from filesystem")
			fsys.Remove("The Beatles/Revolver/01 - Taxman.mp3")
			fsys.Remove("The Beatles/Revolver/02 - Eleanor Rigby.mp3")
			fsys.Remove("The Beatles/Revolver/03 - I'm Only Sleeping.mp3")

			By("Scanning the parent folder (simulating watcher behavior)")
			targets := []model.ScanTarget{
				{LibraryID: lib.ID, FolderPath: "The Beatles"},
			}
			_, err := s.ScanFolders(ctx, false, targets)
			Expect(err).To(Succeed())

			By("Checking all Revolver tracks are marked as missing")
			mf, err := findByPath("The Beatles/Revolver/01 - Taxman.mp3")
			Expect(err).ToNot(HaveOccurred())
			Expect(mf.Missing).To(BeTrue())

			mf, err = findByPath("The Beatles/Revolver/02 - Eleanor Rigby.mp3")
			Expect(err).ToNot(HaveOccurred())
			Expect(mf.Missing).To(BeTrue())

			mf, err = findByPath("The Beatles/Revolver/03 - I'm Only Sleeping.mp3")
			Expect(err).ToNot(HaveOccurred())
			Expect(mf.Missing).To(BeTrue())

			By("Checking the Help! tracks are not affected")
			mf, err = findByPath("The Beatles/Help!/01 - Help!.mp3")
			Expect(err).ToNot(HaveOccurred())
			Expect(mf.Missing).To(BeFalse())

			mf, err = findByPath("The Beatles/Help!/02 - The Night Before.mp3")
			Expect(err).ToNot(HaveOccurred())
			Expect(mf.Missing).To(BeFalse())

			By("Verifying only 2 non-missing tracks remain (Help! tracks)")
			Expect(ds.MediaFile(ctx).CountAll(model.QueryOptions{
				Filters: squirrel.Eq{"missing": false},
			})).To(Equal(int64(2)))
		})

		It("does not override artist fields when importing an undertagged file", func() {
			By("Making sure artist in the DB contains MBID and sort name")
			aa, err := ds.Artist(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"name": "The Beatles"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(aa).To(HaveLen(1))
			Expect(aa[0].Name).To(Equal("The Beatles"))
			Expect(aa[0].MbzArtistID).To(Equal(beatlesMBID))
			Expect(aa[0].SortArtistName).To(Equal("Beatles, The"))

			By("Adding a new undertagged file (no MBID or sort name)")
			newTrack := revolver(track(4, "Love You Too",
				_t{"artist": "The Beatles", "musicbrainz_artistid": "", "artistsort": ""}),
			)
			fsys.Add("The Beatles/Revolver/04 - Love You Too.mp3", newTrack)

			By("Doing a partial scan")
			Expect(runScanner(ctx, false)).To(Succeed())

			By("Asserting MediaFile have the artist name, but not the MBID or sort name")
			mf, err := findByPath("The Beatles/Revolver/04 - Love You Too.mp3")
			Expect(err).ToNot(HaveOccurred())
			Expect(mf.Title).To(Equal("Love You Too"))
			Expect(mf.AlbumArtist).To(Equal("The Beatles"))
			Expect(mf.MbzAlbumArtistID).To(BeEmpty())
			Expect(mf.SortArtistName).To(BeEmpty())

			By("Makingsure the artist in the DB has not changed")
			aa, err = ds.Artist(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"name": "The Beatles"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(aa).To(HaveLen(1))
			Expect(aa[0].Name).To(Equal("The Beatles"))
			Expect(aa[0].MbzArtistID).To(Equal(beatlesMBID))
			Expect(aa[0].SortArtistName).To(Equal("Beatles, The"))
		})

		Context("When PurgeMissing is configured", func() {
			When("PurgeMissing is set to 'never'", func() {
				BeforeEach(func() {
					DeferCleanup(configtest.SetupConfig())
					conf.Server.Scanner.PurgeMissing = consts.PurgeMissingNever
				})

				It("should mark files as missing but not delete them", func() {
					By("Running initial scan")
					Expect(runScanner(ctx, true)).To(Succeed())

					By("Removing a file")
					fsys.Remove("The Beatles/Revolver/02 - Eleanor Rigby.mp3")

					By("Running another scan")
					Expect(runScanner(ctx, true)).To(Succeed())

					By("Checking files are marked as missing but not deleted")
					count, err := ds.MediaFile(ctx).CountAll(model.QueryOptions{
						Filters: squirrel.Eq{"missing": true},
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(count).To(Equal(int64(1)))

					mf, err := findByPath("The Beatles/Revolver/02 - Eleanor Rigby.mp3")
					Expect(err).ToNot(HaveOccurred())
					Expect(mf.Missing).To(BeTrue())
				})
			})

			When("PurgeMissing is set to 'always'", func() {
				BeforeEach(func() {
					conf.Server.Scanner.PurgeMissing = consts.PurgeMissingAlways
				})

				It("should purge missing files on any scan", func() {
					By("Running initial scan")
					Expect(runScanner(ctx, false)).To(Succeed())

					By("Removing a file")
					fsys.Remove("The Beatles/Revolver/02 - Eleanor Rigby.mp3")

					By("Running an incremental scan")
					Expect(runScanner(ctx, false)).To(Succeed())

					By("Checking missing files are deleted")
					count, err := ds.MediaFile(ctx).CountAll(model.QueryOptions{
						Filters: squirrel.Eq{"missing": true},
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(count).To(BeZero())

					_, err = findByPath("The Beatles/Revolver/02 - Eleanor Rigby.mp3")
					Expect(err).To(MatchError(model.ErrNotFound))
				})
			})

			When("PurgeMissing is set to 'full'", func() {
				BeforeEach(func() {
					conf.Server.Scanner.PurgeMissing = consts.PurgeMissingFull
				})

				It("should not purge missing files on incremental scans", func() {
					By("Running initial scan")
					Expect(runScanner(ctx, true)).To(Succeed())

					By("Removing a file")
					fsys.Remove("The Beatles/Revolver/02 - Eleanor Rigby.mp3")

					By("Running an incremental scan")
					Expect(runScanner(ctx, false)).To(Succeed())

					By("Checking files are marked as missing but not deleted")
					count, err := ds.MediaFile(ctx).CountAll(model.QueryOptions{
						Filters: squirrel.Eq{"missing": true},
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(count).To(Equal(int64(1)))

					mf, err := findByPath("The Beatles/Revolver/02 - Eleanor Rigby.mp3")
					Expect(err).ToNot(HaveOccurred())
					Expect(mf.Missing).To(BeTrue())
				})

				It("should purge missing files only on full scans", func() {
					By("Running initial scan")
					Expect(runScanner(ctx, true)).To(Succeed())

					By("Removing a file")
					fsys.Remove("The Beatles/Revolver/02 - Eleanor Rigby.mp3")

					By("Running a full scan")
					Expect(runScanner(ctx, true)).To(Succeed())

					By("Checking missing files are deleted")
					count, err := ds.MediaFile(ctx).CountAll(model.QueryOptions{
						Filters: squirrel.Eq{"missing": true},
					})
					Expect(err).ToNot(HaveOccurred())
					Expect(count).To(BeZero())

					_, err = findByPath("The Beatles/Revolver/02 - Eleanor Rigby.mp3")
					Expect(err).To(MatchError(model.ErrNotFound))
				})
			})
		})
	})

	Describe("Interrupted scan resumption", func() {
		var fsys storagetest.FakeFS
		var help func(...map[string]any) *fstest.MapFile

		BeforeEach(func() {
			help = template(_t{"albumartist": "The Beatles", "album": "Help!", "year": 1965})
			fsys = createFS(fstest.MapFS{
				"The Beatles/Help!/01 - Help!.mp3":            help(track(1, "Help!")),
				"The Beatles/Help!/02 - The Night Before.mp3": help(track(2, "The Night Before")),
			})
		})

		simulateInterruptedScan := func(fullScan bool) {
			// Call ScanBegin to properly set LastScanStartedAt and FullScanInProgress
			// This simulates what would happen if a scan was interrupted (ScanBegin called but ScanEnd not)
			Expect(ds.Library(ctx).ScanBegin(lib.ID, fullScan)).To(Succeed())

			// Verify the update was persisted
			reloaded, err := ds.Library(ctx).Get(lib.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(reloaded.LastScanStartedAt).ToNot(BeZero())
			Expect(reloaded.FullScanInProgress).To(Equal(fullScan))
		}

		Context("when a quick scan is interrupted and resumed with a full scan request", func() {
			BeforeEach(func() {
				// First, complete a full scan to populate the database
				Expect(runScanner(ctx, true)).To(Succeed())

				// Verify files were imported
				mfs, err := ds.MediaFile(ctx).GetAll()
				Expect(err).ToNot(HaveOccurred())
				Expect(mfs).To(HaveLen(2))

				// Now simulate an interrupted quick scan
				// (LastScanStartedAt is set, FullScanInProgress is false)
				simulateInterruptedScan(false)
			})

			It("should rescan all folders when resumed as full scan", func() {
				// Update a tag without changing the folder hash by preserving the original modtime.
				// In a quick scan, this wouldn't be detected because the folder hash hasn't changed.
				// But in a full scan, all files should be re-read regardless of hash.
				origModTime := fsys.MapFS["The Beatles/Help!/01 - Help!.mp3"].ModTime
				fsys.UpdateTags("The Beatles/Help!/01 - Help!.mp3", _t{"comment": "updated comment"}, origModTime)

				// Resume with a full scan - this should process all folders
				// even though folder hashes haven't changed
				Expect(runScanner(ctx, true)).To(Succeed())

				// Verify the comment was updated (which means the folder was processed and file re-imported)
				mfs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"title": "Help!"},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(mfs).To(HaveLen(1))
				Expect(mfs[0].Comment).To(Equal("updated comment"))
			})
		})

		Context("when a full scan is interrupted and resumed with a quick scan request", func() {
			BeforeEach(func() {
				// First, complete a full scan to populate the database
				Expect(runScanner(ctx, true)).To(Succeed())

				// Verify files were imported
				mfs, err := ds.MediaFile(ctx).GetAll()
				Expect(err).ToNot(HaveOccurred())
				Expect(mfs).To(HaveLen(2))

				// Now simulate an interrupted full scan
				// (LastScanStartedAt is set, FullScanInProgress is true)
				simulateInterruptedScan(true)
			})

			It("should continue as full scan even when quick scan is requested", func() {
				// Update a tag without changing the folder hash by preserving the original modtime.
				origModTime := fsys.MapFS["The Beatles/Help!/01 - Help!.mp3"].ModTime
				fsys.UpdateTags("The Beatles/Help!/01 - Help!.mp3", _t{"comment": "full scan comment"}, origModTime)

				// Request a quick scan - but because a full scan was in progress,
				// it should continue as a full scan
				Expect(runScanner(ctx, false)).To(Succeed())

				// Verify the comment was updated (folder was processed despite unchanged hash)
				mfs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"title": "Help!"},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(mfs).To(HaveLen(1))
				Expect(mfs[0].Comment).To(Equal("full scan comment"))
			})
		})

		Context("when no scan was in progress", func() {
			BeforeEach(func() {
				// First, complete a full scan to populate the database
				Expect(runScanner(ctx, true)).To(Succeed())

				// Verify files were imported
				mfs, err := ds.MediaFile(ctx).GetAll()
				Expect(err).ToNot(HaveOccurred())
				Expect(mfs).To(HaveLen(2))

				// Library should have LastScanStartedAt cleared after successful scan
				updatedLib, err := ds.Library(ctx).Get(lib.ID)
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedLib.LastScanStartedAt).To(BeZero())
				Expect(updatedLib.FullScanInProgress).To(BeFalse())
			})

			It("should respect the full scan flag for new scans", func() {
				// Update a tag without changing the folder hash by preserving the original modtime.
				origModTime := fsys.MapFS["The Beatles/Help!/01 - Help!.mp3"].ModTime
				fsys.UpdateTags("The Beatles/Help!/01 - Help!.mp3", _t{"comment": "new full scan"}, origModTime)

				// Start a new full scan
				Expect(runScanner(ctx, true)).To(Succeed())

				// Verify the comment was updated
				mfs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"title": "Help!"},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(mfs).To(HaveLen(1))
				Expect(mfs[0].Comment).To(Equal("new full scan"))
			})

			It("should not rescan unchanged folders during quick scan", func() {
				// Update a tag without changing the folder hash by preserving the original modtime.
				// This simulates editing tags in a file (e.g., with a tag editor) without modifying its timestamp.
				// In a quick scan, this should NOT be detected because the folder hash remains unchanged.
				origModTime := fsys.MapFS["The Beatles/Help!/01 - Help!.mp3"].ModTime
				fsys.UpdateTags("The Beatles/Help!/01 - Help!.mp3", _t{"comment": "should not appear"}, origModTime)

				// Do a quick scan - unchanged folders should be skipped
				Expect(runScanner(ctx, false)).To(Succeed())

				// Verify the comment was NOT updated (folder was skipped)
				mfs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
					Filters: squirrel.Eq{"title": "Help!"},
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(mfs).To(HaveLen(1))
				Expect(mfs[0].Comment).To(BeEmpty())
			})
		})
	})

	Describe("RefreshStats", func() {
		var refreshStatsCalls []bool
		var fsys storagetest.FakeFS
		var help func(...map[string]any) *fstest.MapFile

		BeforeEach(func() {
			refreshStatsCalls = nil

			// Create a mock artist repository that tracks RefreshStats calls
			originalArtistRepo := ds.RealDS.Artist(ctx)
			ds.MockedArtist = &testArtistRepo{
				ArtistRepository: originalArtistRepo,
				callTracker:      &refreshStatsCalls,
			}

			// Create a simple filesystem for testing
			help = template(_t{"albumartist": "The Beatles", "album": "Help!", "year": 1965})
			fsys = createFS(fstest.MapFS{
				"The Beatles/Help!/01 - Help!.mp3": help(track(1, "Help!")),
			})
		})

		It("should call RefreshStats with allArtists=true for full scans", func() {
			Expect(runScanner(ctx, true)).To(Succeed())

			Expect(refreshStatsCalls).To(HaveLen(1))
			Expect(refreshStatsCalls[0]).To(BeTrue(), "RefreshStats should be called with allArtists=true for full scans")
		})

		It("should call RefreshStats with allArtists=false for incremental scans", func() {
			// First do a full scan to set up the data
			Expect(runScanner(ctx, true)).To(Succeed())

			// Reset the tracker to only track the incremental scan
			refreshStatsCalls = nil

			// Add a new file to trigger changes detection
			fsys.Add("The Beatles/Help!/02 - The Night Before.mp3", help(track(2, "The Night Before")))

			// Do an incremental scan
			Expect(runScanner(ctx, false)).To(Succeed())

			Expect(refreshStatsCalls).To(HaveLen(1))
			Expect(refreshStatsCalls[0]).To(BeFalse(), "RefreshStats should be called with allArtists=false for incremental scans")
		})

		It("should update artist stats during quick scans when new albums are added", func() {
			// Don't use the mocked artist repo for this test - we need the real one
			ds.MockedArtist = nil

			By("Initial scan with one album")
			Expect(runScanner(ctx, true)).To(Succeed())

			// Verify initial artist stats - should have 1 album, 1 song
			artists, err := ds.Artist(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"name": "The Beatles"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(artists).To(HaveLen(1))
			artist := artists[0]
			Expect(artist.AlbumCount).To(Equal(1)) // 1 album
			Expect(artist.SongCount).To(Equal(1))  // 1 song

			By("Adding files to an existing directory during incremental scan")
			// Add more files to the existing Help! album - this should trigger artist stats update during incremental scan
			fsys.Add("The Beatles/Help!/02 - The Night Before.mp3", help(track(2, "The Night Before")))
			fsys.Add("The Beatles/Help!/03 - You've Got to Hide Your Love Away.mp3", help(track(3, "You've Got to Hide Your Love Away")))

			// Do a quick scan (incremental)
			Expect(runScanner(ctx, false)).To(Succeed())

			By("Verifying artist stats were updated correctly")
			// Fetch the artist again to check updated stats
			artists, err = ds.Artist(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"name": "The Beatles"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(artists).To(HaveLen(1))
			updatedArtist := artists[0]

			// Should now have 1 album and 3 songs total
			// This is the key test - that artist stats are updated during quick scans
			Expect(updatedArtist.AlbumCount).To(Equal(1)) // 1 album
			Expect(updatedArtist.SongCount).To(Equal(3))  // 3 songs

			// Also verify that role-specific stats are updated (albumartist role)
			Expect(updatedArtist.Stats).To(HaveKey(model.RoleAlbumArtist))
			albumArtistStats := updatedArtist.Stats[model.RoleAlbumArtist]
			Expect(albumArtistStats.AlbumCount).To(Equal(1)) // 1 album
			Expect(albumArtistStats.SongCount).To(Equal(3))  // 3 songs
		})
	})
})

func createFindByPath(ctx context.Context, ds model.DataStore) func(string) (*model.MediaFile, error) {
	return func(path string) (*model.MediaFile, error) {
		list, err := ds.MediaFile(ctx).FindByPaths([]string{path})
		if err != nil {
			return nil, err
		}
		if len(list) == 0 {
			return nil, model.ErrNotFound
		}
		return &list[0], nil
	}
}

type mockMediaFileRepo struct {
	model.MediaFileRepository
	GetMissingAndMatchingError error
}

func (m *mockMediaFileRepo) GetMissingAndMatching(libId int) (model.MediaFileCursor, error) {
	if m.GetMissingAndMatchingError != nil {
		return nil, m.GetMissingAndMatchingError
	}
	return m.MediaFileRepository.GetMissingAndMatching(libId)
}

type testArtistRepo struct {
	model.ArtistRepository
	callTracker *[]bool
}

func (m *testArtistRepo) RefreshStats(allArtists bool) (int64, error) {
	*m.callTracker = append(*m.callTracker, allArtists)
	return m.ArtistRepository.RefreshStats(allArtists)
}
