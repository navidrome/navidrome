package e2e

import (
	"context"
	"encoding/json"
	"sort"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/storage/storagetest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/tests/harness"
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

var _ = configtest.RestoreAfterEachSpec()

type _t = map[string]any

var template = storagetest.Template
var track = storagetest.Track

var (
	ctx      context.Context
	ds       model.DataStore
	goldenDB *harness.DB

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
		"albumartist":           "The Beatles",
		"artist":                "The Beatles",
		"album":                 "Abbey Road",
		"year":                  1969,
		"genre":                 "Rock;Blues",
		"replaygain_album_gain": "-6.5 dB",
		"replaygain_album_peak": "0.98",
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

	harness.CreateFS(fstest.MapFS{
		"Rock/The Beatles/Abbey Road/01 - Come Together.mp3": abbeyRoad(track(1, "Come Together",
			_t{"genre": "Rock;Blues", "composer": "Lennon/McCartney", "bpm": 120, "grouping": "Beatles Tracks",
				"replaygain_track_gain": "-7.1 dB", "replaygain_track_peak": "0.95"})),
		"Rock/The Beatles/Abbey Road/02 - Something.mp3": abbeyRoad(track(2, "Something",
			_t{"genre": "Rock", "composer": "Harrison", "bpm": 100, "grouping": "Beatles Tracks",
				"replaygain_track_gain": "-6.0 dB", "replaygain_track_peak": "0.92"})),
		// Stairway To Heaven has track gain but no album gain, to distinguish the two fields
		"Rock/Led Zeppelin/IV/01 - Stairway To Heaven.flac": ledZepIV(track(1, "Stairway To Heaven",
			_t{"genre": "Rock;Folk", "composer": "Page/Plant", "bpm": 82, "suffix": "flac",
				"bitrate": 900, "samplerate": 44100, "bitdepth": 16,
				"replaygain_track_gain": "-8.25 dB", "replaygain_track_peak": "0.99"})),
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
}

func findMediaFileByTitle(title string) string {
	mfs, err := ds.MediaFile().GetAll(ctx, model.QueryOptions{
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
	err = ds.Playlist().Put(userCtx, pls)
	Expect(err).ToNot(HaveOccurred())

	loaded, err := ds.Playlist().GetWithTracks(userCtx, pls.ID, true, false)
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
	Expect(ds.Playlist().Put(ctx, pls)).To(Succeed())
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
	Expect(ds.Playlist().Put(ctx, pls)).To(Succeed())
	return pls.ID
}

var _ = BeforeSuite(func() {
	ctx = request.WithUser(GinkgoT().Context(), adminUser)
	conf.Update(func(c *conf.Options) {
		c.MusicFolder = "fake:///music"
		c.DevExternalScanner = false
		c.SmartPlaylistRefreshDelay = 0
	})

	buildTestFS()
	goldenDB = harness.SetupDB(ctx, &adminUser, &regularUser)
	// SetupDB filled in adminUser.Libraries.
	ctx = request.WithUser(GinkgoT().Context(), adminUser)
	ds = persistence.New(db.Db())

	comeTogetherID := findMediaFileByTitle("Come Together")
	Expect(ds.MediaFile().SetStar(ctx, true, comeTogetherID)).To(Succeed())
	Expect(ds.MediaFile().SetStar(ctx, true, findMediaFileByTitle("So What"))).To(Succeed())
	Expect(ds.MediaFile().SetRating(ctx, 3, findMediaFileByTitle("Stairway To Heaven"))).To(Succeed())
	Expect(ds.MediaFile().SetRating(ctx, 5, findMediaFileByTitle("Bohemian Rhapsody"))).To(Succeed())
	for range 10 {
		Expect(ds.MediaFile().IncPlayCount(ctx, comeTogetherID, time.Now())).To(Succeed())
	}
	Expect(ds.MediaFile().IncPlayCount(ctx, findMediaFileByTitle("Black Dog"), time.Now())).To(Succeed())

	goldenDB.Resnapshot()
})

var _ = AfterSuite(func() {
	db.Close(ctx)
})

func setupTestDB() {
	ctx = request.WithUser(GinkgoT().Context(), adminUser)
	DeferCleanup(conf.SnapshotConfig())
	conf.Update(func(c *conf.Options) {
		c.MusicFolder = "fake:///music"
		c.DevExternalScanner = false
		c.SmartPlaylistRefreshDelay = 0
	})

	goldenDB.Restore()
	ds = persistence.New(db.Db())
}
