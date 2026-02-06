package e2e

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/core/playback"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/core/storage/storagetest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/server/subsonic"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSubsonicE2E(t *testing.T) {
	tests.Init(t, true)
	defer db.Close(context.Background())
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Subsonic API E2E Suite")
}

// Easy aliases for the storagetest package
type _t = map[string]any

var template = storagetest.Template
var track = storagetest.Track

// Shared test state — populated in BeforeEach
var (
	ctx    context.Context
	ds     *tests.MockDataStore
	router *subsonic.Router
	lib    model.Library

	// Admin user used for most tests
	adminUser = model.User{
		ID:       "admin-1",
		UserName: "admin",
		Name:     "Admin User",
		IsAdmin:  true,
	}
)

func createFS(files fstest.MapFS) storagetest.FakeFS {
	fs := storagetest.FakeFS{}
	fs.SetFiles(files)
	storagetest.Register("fake", &fs)
	return fs
}

// buildTestFS creates the full test filesystem matching the plan
func buildTestFS() storagetest.FakeFS {
	abbeyRoad := template(_t{"albumartist": "The Beatles", "artist": "The Beatles", "album": "Abbey Road", "year": 1969, "genre": "Rock"})
	help := template(_t{"albumartist": "The Beatles", "artist": "The Beatles", "album": "Help!", "year": 1965, "genre": "Rock"})
	ledZepIV := template(_t{"albumartist": "Led Zeppelin", "artist": "Led Zeppelin", "album": "IV", "year": 1971, "genre": "Rock"})
	kindOfBlue := template(_t{"albumartist": "Miles Davis", "artist": "Miles Davis", "album": "Kind of Blue", "year": 1959, "genre": "Jazz"})
	popTrack := template(_t{"albumartist": "Various", "artist": "Various", "album": "Pop", "year": 2020, "genre": "Pop"})

	return createFS(fstest.MapFS{
		// Rock / The Beatles / Abbey Road
		"Rock/The Beatles/Abbey Road/01 - Come Together.mp3": abbeyRoad(track(1, "Come Together")),
		"Rock/The Beatles/Abbey Road/02 - Something.mp3":     abbeyRoad(track(2, "Something")),
		// Rock / The Beatles / Help!
		"Rock/The Beatles/Help!/01 - Help.mp3": help(track(1, "Help!")),
		// Rock / Led Zeppelin / IV
		"Rock/Led Zeppelin/IV/01 - Stairway To Heaven.mp3": ledZepIV(track(1, "Stairway To Heaven")),
		// Jazz / Miles Davis / Kind of Blue
		"Jazz/Miles Davis/Kind of Blue/01 - So What.mp3": kindOfBlue(track(1, "So What")),
		// Pop (standalone track)
		"Pop/01 - Standalone Track.mp3": popTrack(track(1, "Standalone Track")),
		// _empty folder (directory with no audio)
		"_empty/.keep": &fstest.MapFile{Data: []byte{}, ModTime: time.Now()},
	})
}

// newReq creates an authenticated GET request for the given endpoint with optional query parameters.
// Parameters are provided as key-value pairs: newReq("getAlbum", "id", "123")
func newReq(endpoint string, params ...string) *http.Request {
	return newReqWithUser(adminUser, endpoint, params...)
}

// newReqWithUser creates an authenticated GET request for the given user.
func newReqWithUser(user model.User, endpoint string, params ...string) *http.Request {
	u := "/rest/" + endpoint
	if len(params) > 0 {
		q := url.Values{}
		for i := 0; i < len(params)-1; i += 2 {
			q.Add(params[i], params[i+1])
		}
		u += "?" + q.Encode()
	}
	r := httptest.NewRequest("GET", u, nil)
	userCtx := request.WithUser(r.Context(), user)
	userCtx = request.WithUsername(userCtx, user.UserName)
	userCtx = request.WithClient(userCtx, "test-client")
	userCtx = request.WithPlayer(userCtx, model.Player{ID: "player-1", Name: "Test Player", Client: "test-client"})
	return r.WithContext(userCtx)
}

// newRawReq creates a ResponseRecorder + authenticated request for raw handlers (stream, download, getCoverArt).
func newRawReq(endpoint string, params ...string) (*httptest.ResponseRecorder, *http.Request) {
	return httptest.NewRecorder(), newReq(endpoint, params...)
}

// --- Noop stub implementations for Router dependencies ---

// noopArtwork implements artwork.Artwork
type noopArtwork struct{}

func (n noopArtwork) Get(context.Context, model.ArtworkID, int, bool) (io.ReadCloser, time.Time, error) {
	return nil, time.Time{}, model.ErrNotFound
}

func (n noopArtwork) GetOrPlaceholder(_ context.Context, _ string, _ int, _ bool) (io.ReadCloser, time.Time, error) {
	return io.NopCloser(io.LimitReader(nil, 0)), time.Time{}, nil
}

// noopStreamer implements core.MediaStreamer
type noopStreamer struct{}

func (n noopStreamer) NewStream(context.Context, string, string, int, int) (*core.Stream, error) {
	return nil, model.ErrNotFound
}

func (n noopStreamer) DoStream(context.Context, *model.MediaFile, string, int, int) (*core.Stream, error) {
	return nil, model.ErrNotFound
}

// noopArchiver implements core.Archiver
type noopArchiver struct{}

func (n noopArchiver) ZipAlbum(context.Context, string, string, int, io.Writer) error {
	return model.ErrNotFound
}

func (n noopArchiver) ZipArtist(context.Context, string, string, int, io.Writer) error {
	return model.ErrNotFound
}

func (n noopArchiver) ZipShare(context.Context, string, io.Writer) error {
	return model.ErrNotFound
}

func (n noopArchiver) ZipPlaylist(context.Context, string, string, int, io.Writer) error {
	return model.ErrNotFound
}

// noopProvider implements external.Provider
type noopProvider struct{}

func (n noopProvider) UpdateAlbumInfo(_ context.Context, _ string) (*model.Album, error) {
	return &model.Album{}, nil
}

func (n noopProvider) UpdateArtistInfo(_ context.Context, _ string, _ int, _ bool) (*model.Artist, error) {
	return &model.Artist{}, nil
}

func (n noopProvider) SimilarSongs(context.Context, string, int) (model.MediaFiles, error) {
	return nil, nil
}

func (n noopProvider) TopSongs(context.Context, string, int) (model.MediaFiles, error) {
	return nil, nil
}

func (n noopProvider) ArtistImage(context.Context, string) (*url.URL, error) {
	return nil, model.ErrNotFound
}

func (n noopProvider) AlbumImage(context.Context, string) (*url.URL, error) {
	return nil, model.ErrNotFound
}

// noopPlayTracker implements scrobbler.PlayTracker
type noopPlayTracker struct{}

func (n noopPlayTracker) NowPlaying(context.Context, string, string, string, int) error {
	return nil
}

func (n noopPlayTracker) GetNowPlaying(context.Context) ([]scrobbler.NowPlayingInfo, error) {
	return nil, nil
}

func (n noopPlayTracker) Submit(context.Context, []scrobbler.Submission) error {
	return nil
}

// noopShare implements core.Share
type noopShare struct{}

func (n noopShare) Load(context.Context, string) (*model.Share, error) {
	return nil, model.ErrNotFound
}

func (n noopShare) NewRepository(context.Context) rest.Repository {
	return nil
}

// Compile-time interface checks
var (
	_ artwork.Artwork       = noopArtwork{}
	_ core.MediaStreamer    = noopStreamer{}
	_ core.Archiver         = noopArchiver{}
	_ external.Provider     = noopProvider{}
	_ scrobbler.PlayTracker = noopPlayTracker{}
	_ core.Share            = noopShare{}
)

var _ = BeforeSuite(func() {
	ctx = request.WithUser(context.Background(), adminUser)
	tmpDir := GinkgoT().TempDir()
	conf.Server.DbPath = filepath.Join(tmpDir, "test-e2e.db?_journal_mode=WAL")
	db.Db().SetMaxOpenConns(1)
})

// setupTestDB initializes the database, creates the admin user, library, scans the
// test filesystem, and creates the Subsonic Router. Call this from BeforeEach in each
// test container that needs the full E2E environment.
func setupTestDB() {
	// Refresh context with the current spec's context to avoid using a canceled context
	ctx = request.WithUser(GinkgoT().Context(), adminUser)

	DeferCleanup(configtest.SetupConfig())
	conf.Server.MusicFolder = "fake:///music"
	conf.Server.DevExternalScanner = false

	db.Init(ctx)
	DeferCleanup(func() {
		Expect(tests.ClearDB()).To(Succeed())
	})

	ds = &tests.MockDataStore{RealDS: persistence.New(db.Db())}

	// Initialize JWT auth (needed for public URL generation in search responses)
	auth.Init(ds)

	// Create admin user in DB
	adminUserWithPass := adminUser
	adminUserWithPass.NewPassword = "password"
	Expect(ds.User(ctx).Put(&adminUserWithPass)).To(Succeed())

	// Create library
	lib = model.Library{ID: 1, Name: "Music Library", Path: "fake:///music"}
	Expect(ds.Library(ctx).Put(&lib)).To(Succeed())

	// Set user libraries for access control
	Expect(ds.User(ctx).SetUserLibraries(adminUser.ID, []int{lib.ID})).To(Succeed())

	// Reload user with libraries for context
	loadedUser, err := ds.User(ctx).FindByUsername(adminUser.UserName)
	Expect(err).ToNot(HaveOccurred())
	adminUser.Libraries = loadedUser.Libraries
	ctx = request.WithUser(GinkgoT().Context(), adminUser)

	// Build the fake filesystem and run the scanner
	buildTestFS()
	s := scanner.New(ctx, ds, artwork.NoopCacheWarmer(), events.NoopBroker(),
		core.NewPlaylists(ds), metrics.NewNoopInstance())
	_, err = s.ScanAll(ctx, true)
	Expect(err).ToNot(HaveOccurred())

	// Create the Subsonic Router with real DS + noop stubs
	router = subsonic.New(
		ds,                           // DataStore (real)
		noopArtwork{},                // Artwork
		noopStreamer{},               // MediaStreamer
		noopArchiver{},               // Archiver
		core.NewPlayers(ds),          // Players (real)
		noopProvider{},               // Provider
		s,                            // Scanner (real)
		events.NoopBroker(),          // Broker
		core.NewPlaylists(ds),        // Playlists (real)
		noopPlayTracker{},            // PlayTracker
		noopShare{},                  // Share
		playback.PlaybackServer(nil), // PlaybackServer (nil, jukebox disabled)
		metrics.NewNoopInstance(),    // Metrics
	)
}
