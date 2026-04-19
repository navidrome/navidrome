package artworke2e_test

import (
	"context"
	"path/filepath"
	"testing"

	_ "github.com/navidrome/navidrome/adapters/gotaglib"
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
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestArtworkE2E(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Artwork E2E Suite")
}

const fakeLibScheme = "artworkfake"
const fakeLibPath = fakeLibScheme + ":///music"

var (
	ctx    context.Context
	ds     *tests.MockDataStore
	aw     artwork.Artwork
	fakeFS *storagetest.FakeFS
)

// The DB file lives in a suite-level tempdir: the go-sqlite3 singleton keeps
// the file open for the whole suite, and Ginkgo's per-spec TempDir cleanup
// can't unlink a file with a live handle on Windows. A suite-level tempdir
// combined with an AfterSuite close avoids the lock conflict.
var suiteDBTempDir string

var _ = BeforeSuite(func() {
	suiteDBTempDir = GinkgoT().TempDir()
})

var _ = AfterSuite(func() {
	db.Close(GinkgoT().Context())
})

func setupHarness() {
	DeferCleanup(configtest.SetupConfig())

	tempDir := GinkgoT().TempDir()
	// Reuse the suite-level DB path so the singleton connection keeps working
	// across specs (see suiteDBTempDir comment).
	conf.Server.DbPath = filepath.Join(suiteDBTempDir, "artwork-e2e.db") + "?_journal_mode=WAL"
	conf.Server.DataFolder = tempDir
	conf.Server.MusicFolder = fakeLibPath
	conf.Server.DevExternalScanner = false
	conf.Server.ImageCacheSize = "0" // disabled cache → reader runs on every call
	conf.Server.EnableExternalServices = false

	db.Db().SetMaxOpenConns(1)
	ctx = request.WithUser(GinkgoT().Context(), model.User{ID: "admin-1", UserName: "admin", IsAdmin: true})
	db.Init(ctx)
	DeferCleanup(func() { Expect(tests.ClearDB()).To(Succeed()) })

	ds = &tests.MockDataStore{RealDS: persistence.New(db.Db())}

	adminUser := model.User{ID: "admin-1", UserName: "admin", Name: "Admin", IsAdmin: true, NewPassword: "password"}
	Expect(ds.User(ctx).Put(&adminUser)).To(Succeed())

	lib := model.Library{ID: 1, Name: "Music", Path: fakeLibPath}
	Expect(ds.Library(ctx).Put(&lib)).To(Succeed())
	Expect(ds.User(ctx).SetUserLibraries(adminUser.ID, []int{lib.ID})).To(Succeed())

	fakeFS = &storagetest.FakeFS{}
	storagetest.Register(fakeLibScheme, fakeFS)

	aw = artwork.NewArtwork(ds, artwork.GetImageCache(), newNoopFFmpeg(), &noopProvider{})
}

func scan() {
	GinkgoHelper()
	s := scanner.New(ctx, ds, artwork.NoopCacheWarmer(), events.NoopBroker(),
		playlists.NewPlaylists(ds, core.NewImageUploadService()), metrics.NewNoopInstance())
	_, err := s.ScanAll(ctx, true)
	Expect(err).ToNot(HaveOccurred())
}

func firstAlbum() model.Album {
	GinkgoHelper()
	albums, err := ds.Album(ctx).GetAll(model.QueryOptions{})
	Expect(err).ToNot(HaveOccurred())
	Expect(albums).To(HaveLen(1), "expected exactly one album, got %d", len(albums))
	return albums[0]
}
