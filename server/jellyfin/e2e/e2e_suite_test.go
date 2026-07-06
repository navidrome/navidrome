// Package e2e provides end-to-end integration tests for the Navidrome Jellyfin API.
//
// These tests exercise the full HTTP request/response cycle through the Jellyfin API router,
// using a real SQLite database and real repository implementations while stubbing out external
// services (artwork, streaming, transcoding) with spy/noop implementations.
//
// The harness mirrors server/e2e (the Subsonic suite): BeforeSuite creates a temporary SQLite
// database, seeds two users (admin + regular) and one library backed by a fake in-memory
// filesystem, runs the scanner, and snapshots the golden DB. Each top-level Describe restores
// that snapshot and builds a fresh jellyfin.Router.
//
// # Seeded library (see buildTestFS)
//
//	Rock/The Beatles/Abbey Road/  01 Something (1969), 02 Come Together (1969)
//	Rock/The Beatles/Help!/       01 Help! (1965)
//	Rock/Led Zeppelin/IV/         01 Stairway To Heaven (1971)
//	Jazz/Miles Davis/Kind of Blue/01 So What (1959)
//	Pop/Solo Artist/Singles/      01 Standalone Track (2020), 02 Duet (artist "Featured Guest")
//
// Totals: 7 songs, 5 albums, 4 album artists (+ 1 performer-only "Featured Guest" = 5 artists),
// 3 genres (Rock=4, Jazz=1, Pop=2).
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/core/storage/storagetest"
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/server/jellyfin"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestJellyfinE2E(t *testing.T) {
	tests.Init(t, false)
	defer db.Close(t.Context())
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Jellyfin API E2E Suite")
}

// Easy aliases for the storagetest package
type _t = map[string]any

var (
	template = storagetest.Template
	track    = storagetest.Track
)

// Shared test state
var (
	ctx         context.Context
	ds          *tests.MockDataStore
	router      http.Handler
	streamerSpy *spyStreamer
	artworkSpy  *spyArtwork
	lib         model.Library

	// Snapshot paths for fast DB restore
	dbFilePath   string
	snapshotPath string
	dataFolder   string

	adminUser = model.User{
		ID:       "admin-1",
		UserName: "admin",
		Name:     "Admin User",
		IsAdmin:  true,
	}

	regularUser = model.User{
		ID:       "regular-1",
		UserName: "regular",
		Name:     "Regular User",
		IsAdmin:  false,
	}
)

func createFS(files fstest.MapFS) storagetest.FakeFS {
	fs := storagetest.FakeFS{}
	fs.SetFiles(files)
	storagetest.Register("fake", &fs)
	return fs
}

// buildTestFS creates the seeded test filesystem (see package doc for totals).
func buildTestFS() storagetest.FakeFS {
	abbeyRoad := template(_t{"albumartist": "The Beatles", "artist": "The Beatles", "album": "Abbey Road", "year": 1969, "genre": "Rock"})
	help := template(_t{"albumartist": "The Beatles", "artist": "The Beatles", "album": "Help!", "year": 1965, "genre": "Rock"})
	ledZepIV := template(_t{"albumartist": "Led Zeppelin", "artist": "Led Zeppelin", "album": "IV", "year": 1971, "genre": "Rock"})
	kindOfBlue := template(_t{"albumartist": "Miles Davis", "artist": "Miles Davis", "album": "Kind of Blue", "year": 1959, "genre": "Jazz"})
	singles := template(_t{"albumartist": "Solo Artist", "artist": "Solo Artist", "album": "Singles", "year": 2020, "genre": "Pop"})

	return createFS(fstest.MapFS{
		// Track numbers are deliberately reversed vs. alphabetical title order (Something=1,
		// Come Together=2) so tests can tell track-order sorting apart from title sorting.
		"Rock/The Beatles/Abbey Road/01 - Something.mp3":     abbeyRoad(track(1, "Something")),
		"Rock/The Beatles/Abbey Road/02 - Come Together.mp3": abbeyRoad(track(2, "Come Together")),
		"Rock/The Beatles/Help!/01 - Help.mp3":               help(track(1, "Help!")),
		"Rock/Led Zeppelin/IV/01 - Stairway To Heaven.mp3":   ledZepIV(track(1, "Stairway To Heaven")),
		"Jazz/Miles Davis/Kind of Blue/01 - So What.mp3":     kindOfBlue(track(1, "So What")),
		"Pop/Solo Artist/Singles/01 - Standalone Track.mp3":  singles(track(1, "Standalone Track")),
		// "Featured Guest" is the track artist here (album artist stays "Solo Artist"), so it's a
		// performer but not an album artist — lets tests tell /Artists from /Artists/AlbumArtists.
		"Pop/Solo Artist/Singles/02 - Duet.mp3": singles(track(2, "Duet", _t{"artist": "Featured Guest"})),
	})
}

// --- Request helpers ---

// jReq performs a full HTTP round-trip as the given user (token auth) and returns the recorder.
func jReq(user model.User, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	r := httptest.NewRequest(method, path, reader)
	token, err := auth.CreateToken(&user)
	Expect(err).ToNot(HaveOccurred())
	r.Header.Set("X-Emby-Token", token)
	r.Header.Set("X-Emby-Authorization", `MediaBrowser Client="e2e", Device="test", DeviceId="e2e-device", Version="1.0"`)
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	router.ServeHTTP(w, r)
	return w
}

// getWithBareToken performs a GET authenticated only by a bare Authorization header carrying the
// raw token, as react-native-nitro-player (Jellify's native audio player) does — no X-Emby-Token,
// no MediaBrowser scheme.
func getWithBareToken(path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", path, nil)
	token, err := auth.CreateToken(&adminUser)
	Expect(err).ToNot(HaveOccurred())
	r.Header.Set("Authorization", token)
	router.ServeHTTP(w, r)
	return w
}

// rawReq performs a request with no authentication (for public routes).
func rawReq(method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	r := httptest.NewRequest(method, path, reader)
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	router.ServeHTTP(w, r)
	return w
}

func get(path string) *httptest.ResponseRecorder                 { return jReq(adminUser, "GET", path, "") }
func getAs(u model.User, path string) *httptest.ResponseRecorder { return jReq(u, "GET", path, "") }
func post(path, body string) *httptest.ResponseRecorder          { return jReq(adminUser, "POST", path, body) }
func postAs(u model.User, path, body string) *httptest.ResponseRecorder {
	return jReq(u, "POST", path, body)
}
func del(path string) *httptest.ResponseRecorder                 { return jReq(adminUser, "DELETE", path, "") }
func delAs(u model.User, path string) *httptest.ResponseRecorder { return jReq(u, "DELETE", path, "") }

// upload performs an authenticated POST with a custom Content-Type and raw body (image upload).
func upload(user model.User, path, contentType string, body []byte) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", path, bytes.NewReader(body))
	token, err := auth.CreateToken(&user)
	Expect(err).ToNot(HaveOccurred())
	r.Header.Set("X-Emby-Token", token)
	r.Header.Set("X-Emby-Authorization", `MediaBrowser Client="e2e", Device="test", DeviceId="e2e-device", Version="1.0"`)
	r.Header.Set("Content-Type", contentType)
	router.ServeHTTP(w, r)
	return w
}

// parseInto asserts a 200 and unmarshals the JSON body into target.
func parseInto(w *httptest.ResponseRecorder, target any) {
	Expect(w.Code).To(Equal(http.StatusOK), "body: %s", w.Body.String())
	Expect(json.Unmarshal(w.Body.Bytes(), target)).To(Succeed())
}

// queryResult asserts a 200 and returns the parsed QueryResult.
func queryResult(w *httptest.ResponseRecorder) dto.QueryResult {
	var q dto.QueryResult
	parseInto(w, &q)
	return q
}

// createPlaylist creates a playlist as admin (encodedIds are the Jellyfin-encoded item ids a
// client would send) and returns its decoded Navidrome id.
func createPlaylist(name string, encodedIds []string) string {
	return createPlaylistAs(adminUser, name, encodedIds...)
}

// createPlaylistAs creates a playlist owned by the given user and returns its decoded id.
func createPlaylistAs(user model.User, name string, encodedIds ...string) string {
	if encodedIds == nil {
		encodedIds = []string{}
	}
	body, err := json.Marshal(map[string]any{"Name": name, "Ids": encodedIds})
	Expect(err).ToNot(HaveOccurred())
	var res map[string]string
	parseInto(postAs(user, "/Playlists", string(body)), &res)
	Expect(res["Id"]).ToNot(BeEmpty())
	return dto.DecodeID(res["Id"])
}

// --- Seeded-id lookup helpers (return Navidrome ids; wrap with enc() for URLs) ---

func enc(id string) string { return dto.EncodeID(id) }

// The seeded library is tiny, so the id lookups fetch-all and match by name in Go rather than
// guessing repository filter column names.

func albumID(name string) string {
	albums, err := ds.Album(ctx).GetAll()
	Expect(err).ToNot(HaveOccurred())
	for _, a := range albums {
		if a.Name == name {
			return a.ID
		}
	}
	Fail("album not found: " + name)
	return ""
}

func songID(title string) string {
	mfs, err := ds.MediaFile(ctx).GetAll()
	Expect(err).ToNot(HaveOccurred())
	for _, mf := range mfs {
		if mf.Title == title {
			return mf.ID
		}
	}
	Fail("song not found: " + title)
	return ""
}

func artistID(name string) string {
	artists, err := ds.Artist(ctx).GetAll()
	Expect(err).ToNot(HaveOccurred())
	for _, a := range artists {
		if a.Name == name {
			return a.ID
		}
	}
	Fail("artist not found: " + name)
	return ""
}

// --- Suite lifecycle ---

var _ = BeforeSuite(func() {
	ctx = request.WithUser(GinkgoT().Context(), adminUser)
	tmpDir := GinkgoT().TempDir()
	dbFilePath = filepath.Join(tmpDir, "test-jf-e2e.db")
	snapshotPath = filepath.Join(tmpDir, "test-jf-e2e.db.snapshot")
	dataFolder = filepath.Join(tmpDir, "data")
	Expect(os.MkdirAll(dataFolder, 0o755)).To(Succeed())
	conf.Server.DbPath = dbFilePath + "?_journal_mode=WAL"
	db.Db().SetMaxOpenConns(1)

	conf.Server.MusicFolder = "fake:///music"
	conf.Server.DataFolder = conf.NewDir(dataFolder)
	conf.Server.DevExternalScanner = false

	db.Init(ctx)

	initDS := &tests.MockDataStore{RealDS: persistence.New(db.Db())}
	auth.Init(initDS)

	adminWithPass := adminUser
	adminWithPass.NewPassword = "password"
	Expect(initDS.User(ctx).Put(&adminWithPass)).To(Succeed())
	regularWithPass := regularUser
	regularWithPass.NewPassword = "password"
	Expect(initDS.User(ctx).Put(&regularWithPass)).To(Succeed())

	lib = model.Library{ID: 1, Name: "Music Library", Path: "fake:///music"}
	Expect(initDS.Library(ctx).Put(&lib)).To(Succeed())
	Expect(initDS.User(ctx).SetUserLibraries(adminUser.ID, []int{lib.ID})).To(Succeed())
	Expect(initDS.User(ctx).SetUserLibraries(regularUser.ID, []int{lib.ID})).To(Succeed())

	loadedAdmin, err := initDS.User(ctx).FindByUsername(adminUser.UserName)
	Expect(err).ToNot(HaveOccurred())
	adminUser.Libraries = loadedAdmin.Libraries
	loadedRegular, err := initDS.User(ctx).FindByUsername(regularUser.UserName)
	Expect(err).ToNot(HaveOccurred())
	regularUser.Libraries = loadedRegular.Libraries

	ctx = request.WithUser(GinkgoT().Context(), adminUser)

	buildTestFS()
	s := scanner.New(ctx, initDS, artwork.NoopCacheWarmer(), events.NoopBroker(),
		playlists.NewPlaylists(initDS, core.NewImageUploadService()), metrics.NewNoopInstance())
	_, err = s.ScanAll(ctx, true)
	Expect(err).ToNot(HaveOccurred())

	_, err = db.Db().Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	Expect(err).ToNot(HaveOccurred())
	data, err := os.ReadFile(dbFilePath)
	Expect(err).ToNot(HaveOccurred())
	Expect(os.WriteFile(snapshotPath, data, 0o600)).To(Succeed())
})

var _ = AfterSuite(func() {
	db.Close(ctx)
})

// setupTestDB restores the golden snapshot and builds a fresh jellyfin.Router. Call from
// BeforeEach in each test container.
func setupTestDB() {
	ctx = request.WithUser(GinkgoT().Context(), adminUser)

	DeferCleanup(configtest.SetupConfig())
	conf.Server.MusicFolder = "fake:///music"
	conf.Server.DataFolder = conf.NewDir(dataFolder)
	conf.Server.DevExternalScanner = false
	conf.Server.DevEnableMediaFileProbe = false

	restoreDB()

	ds = &tests.MockDataStore{RealDS: persistence.New(db.Db())}
	auth.Init(ds)

	streamerSpy = &spyStreamer{}
	artworkSpy = &spyArtwork{}
	decider := stream.NewTranscodeDecider(ds, noopFFmpeg{})
	router = jellyfin.New(
		ds,
		artworkSpy,
		streamerSpy,
		decider,
		core.NewPlayers(ds),
		scrobbler.NewPlayTracker(ds, events.NoopBroker(), nil),
		playlists.NewPlaylists(ds, core.NewImageUploadService()),
	)
}

// restoreDB restores all table data from the snapshot using ATTACH DATABASE (faster than rescan).
func restoreDB() {
	sqlDB := db.Db()
	_, err := sqlDB.Exec("PRAGMA foreign_keys = OFF")
	Expect(err).ToNot(HaveOccurred())
	_, err = sqlDB.Exec("ATTACH DATABASE ? AS snapshot", snapshotPath)
	Expect(err).ToNot(HaveOccurred())

	rows, err := sqlDB.Query("SELECT name FROM main.sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name NOT LIKE '%_fts' AND name NOT LIKE '%_fts_%'")
	Expect(err).ToNot(HaveOccurred())
	var tables []string
	for rows.Next() {
		var name string
		Expect(rows.Scan(&name)).To(Succeed())
		tables = append(tables, name)
	}
	Expect(rows.Err()).ToNot(HaveOccurred())
	rows.Close()

	for _, table := range tables {
		// Table names come from sqlite_master, not user input, so concatenation is safe here.
		_, err = sqlDB.Exec(`DELETE FROM main."` + table + `"`) //nolint:gosec
		Expect(err).ToNot(HaveOccurred())
		_, err = sqlDB.Exec(`INSERT INTO main."` + table + `" SELECT * FROM snapshot."` + table + `"`) //nolint:gosec
		Expect(err).ToNot(HaveOccurred())
	}

	_, err = sqlDB.Exec("DETACH DATABASE snapshot")
	Expect(err).ToNot(HaveOccurred())
	_, err = sqlDB.Exec("PRAGMA foreign_keys = ON")
	Expect(err).ToNot(HaveOccurred())
}

// --- Spy/noop dependencies ---

// spyArtwork captures the id and context passed to GetOrPlaceholder so image tests can assert the
// resolved ArtworkID and that resolution runs under an elevated (admin) context.
type spyArtwork struct {
	lastID  string
	lastCtx context.Context
	data    []byte
}

func (s *spyArtwork) Get(context.Context, model.ArtworkID, int, bool) (io.ReadCloser, time.Time, error) {
	return nil, time.Time{}, model.ErrNotFound
}

func (s *spyArtwork) GetOrPlaceholder(c context.Context, id string, _ int, _ bool) (io.ReadCloser, time.Time, error) {
	s.lastID = id
	s.lastCtx = c
	d := s.data
	if d == nil {
		d = []byte("IMG")
	}
	return io.NopCloser(bytes.NewReader(d)), time.Time{}, nil
}

// spyStreamer captures the stream.Request and returns a minimal fake stream.
type spyStreamer struct {
	LastRequest   stream.Request
	LastMediaFile *model.MediaFile
	SimulateError error
}

func (s *spyStreamer) NewStream(_ context.Context, mf *model.MediaFile, req stream.Request) (*stream.Stream, error) {
	s.LastRequest = req
	s.LastMediaFile = mf
	if s.SimulateError != nil {
		return nil, s.SimulateError
	}
	format := req.Format
	if format == "" || format == "raw" {
		format = mf.Suffix
	}
	r := io.NopCloser(strings.NewReader("fake audio data"))
	return stream.NewStream(mf, format, req.BitRate, r), nil
}

// noopFFmpeg implements ffmpeg.FFmpeg with no-op methods (transcoding never actually runs in e2e).
type noopFFmpeg struct{}

func (noopFFmpeg) Transcode(context.Context, ffmpeg.TranscodeOptions) (io.ReadCloser, error) {
	return nil, model.ErrNotFound
}
func (noopFFmpeg) ExtractImage(context.Context, string) (io.ReadCloser, error) {
	return nil, model.ErrNotFound
}
func (noopFFmpeg) Probe(context.Context, []string) (string, error) { return "", nil }
func (noopFFmpeg) ProbeAudioStream(context.Context, string) (*ffmpeg.AudioProbeResult, error) {
	return nil, model.ErrNotFound
}
func (noopFFmpeg) ConvertAnimatedImage(context.Context, io.Reader, int, int) (io.ReadCloser, error) {
	return nil, model.ErrNotFound
}
func (noopFFmpeg) CmdPath() (string, error) { return "", nil }
func (noopFFmpeg) IsAvailable() bool        { return false }
func (noopFFmpeg) IsProbeAvailable() bool   { return true }
func (noopFFmpeg) Version() string          { return "noop" }

var (
	_ artwork.Artwork      = &spyArtwork{}
	_ stream.MediaStreamer = &spyStreamer{}
	_ ffmpeg.FFmpeg        = noopFFmpeg{}
)
