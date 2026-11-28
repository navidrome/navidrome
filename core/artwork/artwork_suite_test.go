package artwork

import (
	"cmp"
	"context"
	"errors"
	"io/fs"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/httpclient"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/goleak"
)

func TestArtwork(t *testing.T) {
	// Runs unconditionally: the two leaks below are pre-existing and out of this
	// package's control, so they're ignored by exact top-function instead.
	// A cleanup rather than a defer, so it runs after persistencetest closes the database.
	t.Cleanup(func() {
		goleak.VerifyNone(t,
			goleak.IgnoreTopFunction("github.com/onsi/ginkgo/v2/internal/interrupt_handler.(*InterruptHandler).registerForInterrupts.func2"),
			// notify's own init() starts a singleton tree the moment it's imported (via
			// core/storage/local or plugins); recursive on darwin, nonrecursive on linux.
			goleak.IgnoreTopFunction("github.com/rjeczalik/notify.(*recursiveTree).dispatch"),
			goleak.IgnoreTopFunction("github.com/rjeczalik/notify.(*nonrecursiveTree).dispatch"),
			goleak.IgnoreTopFunction("github.com/rjeczalik/notify.(*nonrecursiveTree).internal"),
		)
	})

	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	persistencetest.Init(t)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Artwork Suite")
}

var _ = configtest.RestoreAfterEachSpec()
var _ = persistencetest.ResetBeforeEachSpec()

// productionImageClient keeps the guarded client for the specs that assert it refuses loopback.
var productionImageClient = remoteImageClient

// httptest servers listen on loopback, which the production client refuses.
var _ = BeforeSuite(func() {
	remoteImageClient = httpclient.NewExternal(5*time.Second, netip.MustParsePrefix("127.0.0.0/8"), netip.MustParsePrefix("::1/128"))
})

// osDirFS wraps os.DirFS as a storage.MusicFS for integration tests.
type osDirFS struct{ fs.FS }

func (o osDirFS) ReadTags(...string) (map[string]metadata.Info, error) { return nil, nil }

// testFileScheme is the URL scheme registered to expose a tempdir as a
// storage.MusicFS for artwork integration tests.
const testFileScheme = "testfile"

// testFileLibPath builds a `testfile://` library URL for the given absolute
// filesystem path. On Windows, the native path (e.g. `C:\foo`) has no leading
// slash after ToSlash, which makes url.Parse treat the drive letter as a
// host. We prepend a `/` so parsing yields `u.Path == /C:/foo`, and the
// registered constructor below strips that leading slash back off.
func testFileLibPath(absPath string) string {
	p := filepath.ToSlash(absPath)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return testFileScheme + "://" + p
}

func init() {
	// Register the testfile storage scheme (os.DirFS-backed MusicFS). Used by
	// integration tests that need real files but not the taglib extractor.
	storage.Register(testFileScheme, func(u url.URL) storage.Storage {
		root := u.Path
		// Undo the leading slash added by testFileLibPath on Windows so that
		// os.Stat / os.DirFS receive a native path like `C:\foo`.
		if runtime.GOOS == "windows" && len(root) >= 3 && root[0] == '/' && root[2] == ':' {
			root = root[1:]
		}
		return &osDirStorage{root: filepath.FromSlash(root)}
	})
}

type osDirStorage struct{ root string }

func (s *osDirStorage) FS() (storage.MusicFS, error) {
	if _, err := os.Stat(s.root); err != nil {
		return nil, err
	}
	return osDirFS{os.DirFS(s.root)}, nil
}

// fixtureAlbumDir is relative to the repo root, which tests.Init makes the working directory.
const fixtureAlbumDir = "tests/fixtures/artist/an-album"

var testAdmin = model.User{ID: "admin-1", UserName: "admin", IsAdmin: true}

func asAdmin(ctx context.Context) context.Context {
	return request.WithUser(ctx, testAdmin)
}

// newTestDS returns a store over the suite database, with testAdmin stored.
func newTestDS(ctx context.Context) model.DataStore {
	GinkgoHelper()
	ds := persistence.New(db.Db())
	admin := testAdmin
	Expect(ds.User().Put(ctx, &admin)).To(Succeed())
	return ds
}

// useLibrary moves library 1 to root, read through the testfile scheme.
func useLibrary(ctx context.Context, ds model.DataStore, root string) {
	GinkgoHelper()
	persistencetest.SetLibraries(ctx, ds, testFileLibPath(root))
}

// putFolder stores library 1's folder at rel holding images, and returns its ID.
func putFolder(ctx context.Context, ds model.DataStore, rel string, images ...string) string {
	GinkgoHelper()
	lib, err := ds.Library().Get(ctx, model.DefaultLibraryID)
	Expect(err).ToNot(HaveOccurred())
	f := model.NewFolder(*lib, rel)
	f.ImageFiles = images
	f.ImagesUpdatedAt = time.Now().Truncate(time.Second)
	Expect(ds.Folder().Put(ctx, f)).To(Succeed())
	return f.ID
}

// putArtistAlbums stores the artist and makes it the sole album artist of albums.
func putArtistAlbums(ctx context.Context, ds model.DataStore, ar model.Artist, albums ...model.Album) {
	GinkgoHelper()
	persistencetest.PutArtists(ctx, ds, model.DefaultLibraryID, ar)
	for i := range albums {
		albums[i].Participants = model.Participants{model.RoleAlbumArtist: {{Artist: ar}}}
	}
	persistencetest.PutAlbums(ctx, ds, albums...)
}

// putRadio stores the radio without the artwork request its Put enqueues.
func putRadio(ctx context.Context, ds model.DataStore, rd model.Radio) {
	GinkgoHelper()
	Expect(ds.Radio().Put(asAdmin(ctx), &rd)).To(Succeed())
	unqueue(ctx, model.KindRadioArtwork, rd.ID)
}

// putPlaylist stores a testAdmin playlist with one track per album, without the cover rebuild its
// Put enqueues.
func putPlaylist(ctx context.Context, ds model.DataStore, pls model.Playlist, albumIDs ...string) {
	GinkgoHelper()
	pls.OwnerID = cmp.Or(pls.OwnerID, testAdmin.ID)
	for _, albumID := range albumIDs {
		mf := model.MediaFile{ID: pls.ID + "-" + albumID, AlbumID: albumID, Path: pls.ID + "/" + albumID + ".mp3"}
		persistencetest.PutMediaFiles(ctx, ds, mf)
		pls.Tracks = append(pls.Tracks, model.PlaylistTrack{MediaFileID: mf.ID, MediaFile: mf})
	}
	Expect(ds.Playlist().Put(asAdmin(ctx), &pls)).To(Succeed())
	unqueue(ctx, model.KindPlaylistArtwork, pls.ID)
}

func unqueue(ctx context.Context, kind model.Kind, id string) {
	GinkgoHelper()
	_, err := db.Db().ExecContext(ctx, "DELETE FROM artwork_queue WHERE item_kind = ? AND item_id = ?", kind.Prefix(), id)
	Expect(err).ToNot(HaveOccurred())
}

// queued returns the primary image's queue row, or nil when there is none.
func queued(ctx context.Context, ds model.DataStore, kind, id string) *model.ArtworkQueueItem {
	GinkgoHelper()
	k, ok := model.ParseKind(kind)
	Expect(ok).To(BeTrue(), kind)
	it, err := ds.ArtworkQueue().Get(ctx, k, id, model.ImageTypePrimary)
	if errors.Is(err, model.ErrNotFound) {
		return nil
	}
	Expect(err).ToNot(HaveOccurred())
	return it
}

func queueCount(ctx context.Context, ds model.DataStore) int64 {
	GinkgoHelper()
	n, err := ds.ArtworkQueue().Count(ctx)
	Expect(err).ToNot(HaveOccurred())
	return n
}

// expireQueued ages a row past the retry budget, so the next drain settles it instead of retrying.
func expireQueued(kind, id string) {
	GinkgoHelper()
	_, err := db.Db().ExecContext(GinkgoT().Context(), "UPDATE artwork_queue SET enqueued_at = ? WHERE item_kind = ? AND item_id = ?",
		time.Now().Add(-(giveUpAfter + time.Hour)), kind, id)
	Expect(err).ToNot(HaveOccurred())
}
