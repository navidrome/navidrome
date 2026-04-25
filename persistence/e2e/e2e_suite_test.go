package e2e

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/core/storage/storagetest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSmartPlaylistE2E(t *testing.T) {
	tests.Init(t, false)
	defer db.Close(t.Context())
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Smart Playlist E2E Suite")
}

type _t = map[string]any

var template = storagetest.Template
var track = storagetest.Track

var (
	ctx context.Context
	ds  *tests.MockDataStore
	lib model.Library

	dbFilePath     string
	snapshotPath   string
	snapshotTables []string

	adminUser = model.User{
		ID:       "sp-test-user-1",
		UserName: "sptestuser",
		Name:     "SP Test User",
		IsAdmin:  true,
	}

	regularUser = model.User{
		ID:       "sp-test-user-2",
		UserName: "spotheruser",
		Name:     "SP Other User",
		IsAdmin:  false,
	}
)

func buildTestFS() {
	abbeyRoad := template(_t{
		"albumartist": "The Beatles",
		"artist":      "The Beatles",
		"album":       "Abbey Road",
		"year":        1969,
		"genre":       "Rock;Blues",
	})
	ledZepIV := template(_t{
		"albumartist": "Led Zeppelin",
		"artist":      "Led Zeppelin",
		"album":       "IV",
		"year":        1971,
	})
	kindOfBlue := template(_t{
		"albumartist": "Miles Davis",
		"artist":      "Miles Davis",
		"album":       "Kind of Blue",
		"year":        1959,
		"genre":       "Jazz",
		"composer":    "Miles Davis",
	})
	nightAtOpera := template(_t{
		"albumartist": "Queen",
		"artist":      "Queen",
		"album":       "A Night at the Opera",
		"year":        1975,
		"genre":       "Rock",
	})
	electricLadyland := template(_t{
		"albumartist": "Jimi Hendrix",
		"artist":      "Jimi Hendrix",
		"album":       "Electric Ladyland",
		"year":        1968,
		"genre":       "Rock;Blues",
	})
	newsOfWorld := template(_t{
		"albumartist": "Queen",
		"artist":      "Queen",
		"album":       "News of the World",
		"year":        1977,
		"genre":       "Rock;Pop",
		"compilation": "1",
	})

	fs := storagetest.FakeFS{}
	fs.SetFiles(fstest.MapFS{
		"Rock/The Beatles/Abbey Road/01 - Come Together.mp3": abbeyRoad(track(1, "Come Together",
			_t{"genre": "Rock;Blues", "composer": "Lennon/McCartney", "bpm": 120})),
		"Rock/The Beatles/Abbey Road/02 - Something.mp3": abbeyRoad(track(2, "Something",
			_t{"genre": "Rock", "composer": "Harrison", "bpm": 100})),
		"Rock/Led Zeppelin/IV/01 - Stairway To Heaven.flac": ledZepIV(track(1, "Stairway To Heaven",
			_t{"genre": "Rock;Folk", "composer": "Page/Plant", "bpm": 82, "suffix": "flac",
				"bitrate": 900, "samplerate": 44100, "bitdepth": 16})),
		"Rock/Led Zeppelin/IV/02 - Black Dog.flac": ledZepIV(track(2, "Black Dog",
			_t{"genre": "Rock;Blues", "composer": "Page/Plant/Jones", "bpm": 150, "suffix": "flac",
				"bitrate": 900, "samplerate": 44100, "bitdepth": 16})),
		"Jazz/Miles Davis/Kind of Blue/01 - So What.mp3": kindOfBlue(track(1, "So What",
			_t{"bpm": 136})),
		"Rock/Queen/A Night at the Opera/01 - Bohemian Rhapsody.mp3": nightAtOpera(track(1, "Bohemian Rhapsody",
			_t{"composer": "Freddie Mercury", "bpm": 72})),
		"Rock/Jimi Hendrix/Electric Ladyland/01 - All Along the Watchtower.mp3": electricLadyland(track(1, "All Along the Watchtower",
			_t{"composer": "Bob Dylan", "bpm": 112})),
		"Rock/Queen/News of the World/01 - We Are the Champions.mp3": newsOfWorld(track(1, "We Are the Champions",
			_t{"composer": "Freddie Mercury", "bpm": 64})),
	})
	storagetest.Register("fake", &fs)
}

func findMediaFileByTitle(title string) string {
	mfs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.Eq{"media_file.title": title},
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(mfs).To(HaveLen(1), "expected exactly one media file with title %q", title)
	return mfs[0].ID
}

func evaluateRule(jsonRule string) []string {
	titles := evaluateRuleOrderedAs(adminUser, jsonRule)
	sort.Strings(titles)
	return titles
}

func evaluateRuleOrdered(jsonRule string) []string {
	return evaluateRuleOrderedAs(adminUser, jsonRule)
}

func evaluateRuleAs(owner model.User, jsonRule string) []string {
	titles := evaluateRuleOrderedAs(owner, jsonRule)
	sort.Strings(titles)
	return titles
}

func evaluateRuleOrderedAs(owner model.User, jsonRule string) []string {
	userCtx := request.WithUser(GinkgoT().Context(), owner)
	var rules criteria.Criteria
	err := json.Unmarshal([]byte(jsonRule), &rules)
	Expect(err).ToNot(HaveOccurred(), "invalid criteria JSON: %s", jsonRule)

	pls := &model.Playlist{
		Name:    "test-smart-playlist",
		OwnerID: owner.ID,
		Rules:   &rules,
	}
	err = ds.Playlist(userCtx).Put(pls)
	Expect(err).ToNot(HaveOccurred())

	loaded, err := ds.Playlist(userCtx).GetWithTracks(pls.ID, true, false)
	Expect(err).ToNot(HaveOccurred())

	titles := make([]string, len(loaded.Tracks))
	for i, t := range loaded.Tracks {
		titles[i] = t.Title
	}
	return titles
}

func createPlaylist(owner model.User, public bool, titles ...string) string {
	pls := &model.Playlist{
		Name:    "ref-playlist",
		OwnerID: owner.ID,
		Public:  public,
	}
	for _, title := range titles {
		mfID := findMediaFileByTitle(title)
		pls.AddMediaFilesByID([]string{mfID})
	}
	Expect(ds.Playlist(ctx).Put(pls)).To(Succeed())
	return pls.ID
}

func createPublicPlaylist(owner model.User, titles ...string) string {
	return createPlaylist(owner, true, titles...)
}

func createPrivatePlaylist(owner model.User, titles ...string) string {
	return createPlaylist(owner, false, titles...)
}

func createPublicSmartPlaylist(owner model.User, jsonRule string) string {
	return createSmartPlaylist(owner, true, jsonRule)
}

func createPrivateSmartPlaylist(owner model.User, jsonRule string) string {
	return createSmartPlaylist(owner, false, jsonRule)
}

func createSmartPlaylist(owner model.User, public bool, jsonRule string) string {
	var rules criteria.Criteria
	Expect(json.Unmarshal([]byte(jsonRule), &rules)).To(Succeed())
	pls := &model.Playlist{
		Name:    "ref-smart-playlist",
		OwnerID: owner.ID,
		Public:  public,
		Rules:   &rules,
	}
	Expect(ds.Playlist(ctx).Put(pls)).To(Succeed())
	return pls.ID
}

var _ = BeforeSuite(func() {
	ctx = request.WithUser(GinkgoT().Context(), adminUser)
	tmpDir := GinkgoT().TempDir()
	dbFilePath = filepath.Join(tmpDir, "smartplaylist-e2e.db")
	snapshotPath = filepath.Join(tmpDir, "smartplaylist-e2e.db.snapshot")
	conf.Server.DbPath = dbFilePath + "?_journal_mode=WAL"
	db.Db().SetMaxOpenConns(1)

	conf.Server.MusicFolder = "fake:///music"
	conf.Server.DevExternalScanner = false
	conf.Server.SmartPlaylistRefreshDelay = 0

	db.Init(ctx)

	initDS := &tests.MockDataStore{RealDS: persistence.New(db.Db())}

	userWithPass := adminUser
	userWithPass.NewPassword = "password"
	Expect(initDS.User(ctx).Put(&userWithPass)).To(Succeed())

	regularUserWithPass := regularUser
	regularUserWithPass.NewPassword = "password"
	Expect(initDS.User(ctx).Put(&regularUserWithPass)).To(Succeed())

	lib = model.Library{ID: 1, Name: "Music Library", Path: "fake:///music"}
	Expect(initDS.Library(ctx).Put(&lib)).To(Succeed())
	Expect(initDS.User(ctx).SetUserLibraries(adminUser.ID, []int{lib.ID})).To(Succeed())
	Expect(initDS.User(ctx).SetUserLibraries(regularUser.ID, []int{lib.ID})).To(Succeed())

	loadedUser, err := initDS.User(ctx).FindByUsername(adminUser.UserName)
	Expect(err).ToNot(HaveOccurred())
	adminUser.Libraries = loadedUser.Libraries

	loadedOther, err := initDS.User(ctx).FindByUsername(regularUser.UserName)
	Expect(err).ToNot(HaveOccurred())
	regularUser.Libraries = loadedOther.Libraries

	ctx = request.WithUser(GinkgoT().Context(), adminUser)

	buildTestFS()
	s := scanner.New(ctx, initDS, artwork.NoopCacheWarmer(), events.NoopBroker(),
		playlists.NewPlaylists(initDS, core.NewImageUploadService()), metrics.NewNoopInstance())
	_, err = s.ScanAll(ctx, true)
	Expect(err).ToNot(HaveOccurred())

	ds = &tests.MockDataStore{RealDS: persistence.New(db.Db())}

	comeTogetherID := findMediaFileByTitle("Come Together")
	Expect(ds.MediaFile(ctx).SetStar(true, comeTogetherID)).To(Succeed())
	Expect(ds.MediaFile(ctx).SetStar(true, findMediaFileByTitle("So What"))).To(Succeed())
	Expect(ds.MediaFile(ctx).SetRating(3, findMediaFileByTitle("Stairway To Heaven"))).To(Succeed())
	Expect(ds.MediaFile(ctx).SetRating(5, findMediaFileByTitle("Bohemian Rhapsody"))).To(Succeed())
	for range 10 {
		Expect(ds.MediaFile(ctx).IncPlayCount(comeTogetherID, time.Now())).To(Succeed())
	}
	Expect(ds.MediaFile(ctx).IncPlayCount(findMediaFileByTitle("Black Dog"), time.Now())).To(Succeed())

	rows, err := db.Db().Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name NOT LIKE '%_fts' AND name NOT LIKE '%_fts_%'")
	Expect(err).ToNot(HaveOccurred())
	defer rows.Close()
	for rows.Next() {
		var name string
		Expect(rows.Scan(&name)).To(Succeed())
		snapshotTables = append(snapshotTables, name)
	}
	Expect(rows.Err()).ToNot(HaveOccurred())

	_, err = db.Db().Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	Expect(err).ToNot(HaveOccurred())
	data, err := os.ReadFile(dbFilePath)
	Expect(err).ToNot(HaveOccurred())
	Expect(os.WriteFile(snapshotPath, data, 0600)).To(Succeed())
})

var _ = AfterSuite(func() {
	db.Close(ctx)
})

func restoreDB() {
	sqlDB := db.Db()

	_, err := sqlDB.Exec("PRAGMA foreign_keys = OFF")
	Expect(err).ToNot(HaveOccurred())
	defer func() { _, _ = sqlDB.Exec("PRAGMA foreign_keys = ON") }()

	_, err = sqlDB.Exec("ATTACH DATABASE ? AS snapshot", snapshotPath)
	Expect(err).ToNot(HaveOccurred())
	defer func() { _, _ = sqlDB.Exec("DETACH DATABASE snapshot") }()

	_, err = sqlDB.Exec("BEGIN TRANSACTION")
	Expect(err).ToNot(HaveOccurred())
	defer func() { _, _ = sqlDB.Exec("ROLLBACK") }()

	for _, table := range snapshotTables {
		_, err = sqlDB.Exec(`DELETE FROM main."` + table + `"`) //nolint:gosec
		Expect(err).ToNot(HaveOccurred())
		_, err = sqlDB.Exec(`INSERT INTO main."` + table + `" SELECT * FROM snapshot."` + table + `"`) //nolint:gosec
		Expect(err).ToNot(HaveOccurred())
	}

	_, err = sqlDB.Exec("COMMIT")
	Expect(err).ToNot(HaveOccurred())
}

func setupTestDB() {
	ctx = request.WithUser(GinkgoT().Context(), adminUser)
	DeferCleanup(configtest.SetupConfig())
	conf.Server.MusicFolder = "fake:///music"
	conf.Server.DevExternalScanner = false
	conf.Server.SmartPlaylistRefreshDelay = 0

	restoreDB()
	ds = &tests.MockDataStore{RealDS: persistence.New(db.Db())}
}
