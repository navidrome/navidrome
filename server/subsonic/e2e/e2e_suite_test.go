package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/lyrics"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/core/playback"
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
	"github.com/navidrome/navidrome/server/subsonic"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/tests/harness"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSubsonicE2E(t *testing.T) {
	tests.Init(t, false)
	defer db.Close(t.Context())
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Subsonic API E2E Suite")
}

// Easy aliases for the storagetest package
type _t = map[string]any

var template = storagetest.Template
var track = storagetest.Track
var file = storagetest.File

// MusicBrainz ID constants for test data (valid UUID v4 values)
const (
	mbidBeatlesArtist     = "b10bbbfc-cf9e-42e0-be17-e2c3e1d2600d"
	mbidAbbeyRoadAlbum    = "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d"
	mbidAbbeyRoadRelGroup = "d4c3b2a1-f6e5-4b7a-9d8c-1f0e3a2b5c4d"
	mbidComeTogether      = "11111111-1111-4111-a111-111111111111" // mbz_release_track_id
	mbidComeTogetherRec   = "22222222-2222-4222-a222-222222222222" // mbz_recording_id
	mbidSomething         = "33333333-3333-4333-a333-333333333333" // mbz_release_track_id
	mbidSomethingRec      = "44444444-4444-4444-a444-444444444444" // mbz_recording_id
)

// lyricFixture reads a public-domain lyric fixture (the same files the parser
// benchmarks use) so the e2e fixtures stay in sync with real-world content,
// including the word-level timing carried by the .elrc and .yaml variants.
func lyricFixture(name string) string {
	// tests.Init chdirs to the project root, so reference fixtures from there.
	data, err := os.ReadFile(filepath.Join("tests", "fixtures", "lyrics", name))
	Expect(err).ToNot(HaveOccurred(), "reading lyric fixture %q", name)
	return string(data)
}

// firstFixtureLine is the opening lyric line shared by every auld-lang-syne
// fixture; tests assert against it regardless of source format.
const firstFixtureLine = "Should auld acquaintance be forgot,"

// Shared test state
var (
	ctx         context.Context
	ds          *tests.MockDataStore
	router      *subsonic.Router
	streamerSpy *harness.SpyStreamer
	goldenDB    *harness.DB
	lib         model.Library

	// Admin user used for most tests
	adminUser = model.User{
		ID:       "admin-1",
		UserName: "admin",
		Name:     "Admin User",
		IsAdmin:  true,
	}

	// Regular (non-admin) user for permission tests
	regularUser = model.User{
		ID:       "regular-1",
		UserName: "regular",
		Name:     "Regular User",
		IsAdmin:  false,
	}
)

// buildTestFS creates the full test filesystem matching the plan
func buildTestFS() storagetest.FakeFS {
	abbeyRoad := template(_t{
		"albumartist":                "The Beatles",
		"artist":                     "The Beatles",
		"album":                      "Abbey Road",
		"year":                       1969,
		"genre":                      "Rock",
		"musicbrainz_artistid":       mbidBeatlesArtist,
		"musicbrainz_albumartistid":  mbidBeatlesArtist,
		"musicbrainz_albumid":        mbidAbbeyRoadAlbum,
		"musicbrainz_releasegroupid": mbidAbbeyRoadRelGroup,
	})
	help := template(_t{"albumartist": "The Beatles", "artist": "The Beatles", "album": "Help!", "year": 1965, "genre": "Rock"})
	ledZepIV := template(_t{"albumartist": "Led Zeppelin", "artist": "Led Zeppelin", "album": "IV", "year": 1971, "genre": "Rock"})
	kindOfBlue := template(_t{"albumartist": "Miles Davis", "artist": "Miles Davis", "album": "Kind of Blue", "year": 1959, "genre": "Jazz"})
	popTrack := template(_t{"albumartist": "Various", "artist": "Various", "album": "Pop", "year": 2020, "genre": "Pop"})
	cowboyBebop := template(_t{"albumartist": "シートベルツ", "artist": "シートベルツ", "album": "COWBOY BEBOP", "year": 1998, "genre": "Jazz"})

	// Template for diverse-format transcode test tracks
	tcBase := _t{"albumartist": "Test Artist", "artist": "Test Artist", "album": "Transcode Formats", "year": 2024, "genre": "Test"}

	// Template for lyrics e2e fixture tracks — isolated under Lyrics/ to keep other suite counts stable
	lyricsAlbum := template(_t{"albumartist": "Lyric Tester", "artist": "Lyric Tester", "album": "Lyrics", "year": 2024, "genre": "Test"})

	return harness.CreateFS(fstest.MapFS{
		// Rock / The Beatles / Abbey Road (with MBIDs)
		// Note: "musicbrainz_trackid" is an alias for the musicbrainz_recordingid tag (populates MbzRecordingID),
		//       "musicbrainz_releasetrackid" is an alias for the musicbrainz_trackid tag (populates MbzReleaseTrackID).
		"Rock/The Beatles/Abbey Road/01 - Come Together.mp3": abbeyRoad(track(1, "Come Together",
			_t{"musicbrainz_releasetrackid": mbidComeTogether, "musicbrainz_trackid": mbidComeTogetherRec, "bpm": 120})),
		"Rock/The Beatles/Abbey Road/02 - Something.mp3": abbeyRoad(track(2, "Something",
			_t{"musicbrainz_releasetrackid": mbidSomething, "musicbrainz_trackid": mbidSomethingRec})),
		// Rock / The Beatles / Help! (no MBIDs)
		"Rock/The Beatles/Help!/01 - Help.mp3": help(track(1, "Help!")),
		// Rock / Led Zeppelin / IV (no MBIDs)
		"Rock/Led Zeppelin/IV/01 - Stairway To Heaven.mp3": ledZepIV(track(1, "Stairway To Heaven")),
		// Jazz / Miles Davis / Kind of Blue (no MBIDs)
		"Jazz/Miles Davis/Kind of Blue/01 - So What.mp3": kindOfBlue(track(1, "So What")),
		// Pop (standalone track, no MBIDs)
		"Pop/01 - Standalone Track.mp3": popTrack(track(1, "Standalone Track")),
		// CJK / シートベルツ / COWBOY BEBOP (Japanese artist, for CJK search tests)
		"CJK/シートベルツ/COWBOY BEBOP/01 - プラチナ・ジェット.mp3": cowboyBebop(track(1, "プラチナ・ジェット")),

		// Diverse audio format tracks for transcode e2e tests
		"Test/Transcode Formats/01 - TC FLAC Standard.flac": file(tcBase, _t{
			"title": "TC FLAC Standard", "track": 1, "suffix": "flac",
			"bitrate": 900, "samplerate": 44100, "bitdepth": 16, "channels": 2, "duration": int64(240),
		}),
		"Test/Transcode Formats/02 - TC FLAC HiRes.flac": file(tcBase, _t{
			"title": "TC FLAC HiRes", "track": 2, "suffix": "flac",
			"bitrate": 3000, "samplerate": 96000, "bitdepth": 24, "channels": 2, "duration": int64(180),
		}),
		"Test/Transcode Formats/03 - TC ALAC Track.m4a": file(tcBase, _t{
			"title": "TC ALAC Track", "track": 3, "suffix": "m4a",
			"bitrate": 900, "samplerate": 44100, "bitdepth": 16, "channels": 2, "duration": int64(200),
		}),
		"Test/Transcode Formats/04 - TC DSD Track.dsf": file(tcBase, _t{
			"title": "TC DSD Track", "track": 4, "suffix": "dsf",
			"bitrate": 5645, "samplerate": 2822400, "bitdepth": 1, "channels": 2, "duration": int64(300),
		}),
		"Test/Transcode Formats/05 - TC Opus Track.opus": file(tcBase, _t{
			"title": "TC Opus Track", "track": 5, "suffix": "opus",
			"bitrate": 128, "samplerate": 48000, "bitdepth": 0, "channels": 2, "duration": int64(210),
		}),
		"Test/Transcode Formats/06 - TC MKA Opus.mka": file(tcBase, _t{
			"title": "TC MKA Opus", "track": 6, "suffix": "mka", "codec": "opus",
			"bitrate": 128, "samplerate": 48000, "bitdepth": 0, "channels": 2, "duration": int64(220),
		}),
		"Test/Transcode Formats/07 - TC FLAC Multichannel.flac": file(tcBase, _t{
			"title": "TC FLAC Multichannel", "track": 7, "suffix": "flac",
			"bitrate": 4500, "samplerate": 48000, "bitdepth": 24, "channels": 6, "duration": int64(180),
		}),

		// Lyrics fixtures (isolated under Lyrics/ to keep other suite counts stable).
		// Content comes from tests/fixtures/lyrics (the same public-domain files the
		// parser benchmarks use); the .elrc and .yaml variants carry word-level
		// timing, which drives the v1 (line-level) vs v2 (enhanced/word-level) tests.
		//
		// Embedded — lyrics delivered via the "lyrics" tag, parsed at scan time.
		// "Enhanced LRC" embeds ELRC (word-level) content; the title is kept generic
		// since ELRC is still valid LRC.
		"Lyrics/Embedded/01 - Embedded Enhanced LRC.mp3": lyricsAlbum(track(1, "Embedded Enhanced LRC",
			_t{"lyrics": lyricFixture("auld-lang-syne.elrc")})),
		"Lyrics/Embedded/02 - Embedded Plain.mp3": lyricsAlbum(track(2, "Embedded Plain",
			_t{"lyrics": lyricFixture("auld-lang-syne.txt")})),
		"Lyrics/Embedded/03 - Embedded TTML.mp3": lyricsAlbum(track(3, "Embedded TTML",
			_t{"lyrics": lyricFixture("auld-lang-syne.ttml")})),

		// Sidecar — raw lyric text files read from the library FS at request time via fromExternalFile.
		// The scanner skips non-audio extensions (.lrc, .srt, .yaml), so placing them as raw MapFile
		// entries is safe: they are visible to the fake FS but invisible to the scanner.
		"Lyrics/Sidecar/01 - Sidecar LRC.mp3":   lyricsAlbum(track(1, "Sidecar LRC")),
		"Lyrics/Sidecar/01 - Sidecar LRC.lrc":   &fstest.MapFile{Data: []byte(lyricFixture("auld-lang-syne.lrc")), ModTime: time.Now()},
		"Lyrics/Sidecar/02 - Sidecar SRT.mp3":   lyricsAlbum(track(2, "Sidecar SRT")),
		"Lyrics/Sidecar/02 - Sidecar SRT.srt":   &fstest.MapFile{Data: []byte(lyricFixture("auld-lang-syne.srt")), ModTime: time.Now()},
		"Lyrics/Sidecar/03 - Sidecar YAML.mp3":  lyricsAlbum(track(3, "Sidecar YAML")),
		"Lyrics/Sidecar/03 - Sidecar YAML.yaml": &fstest.MapFile{Data: []byte(lyricFixture("auld-lang-syne.yaml")), ModTime: time.Now()},

		// _empty folder (directory with no audio)
		"_empty/.keep": &fstest.MapFile{Data: []byte{}, ModTime: time.Now()},
	})
}

// createUser creates a user in the database with the given properties, assigns them to the test
// library, and returns the fully-loaded user (with Libraries populated).
func createUser(id, username, name string, isAdmin bool) model.User {
	user := model.User{
		ID:          id,
		UserName:    username,
		Name:        name,
		IsAdmin:     isAdmin,
		NewPassword: "password",
	}
	Expect(ds.User(ctx).Put(&user)).To(Succeed())
	Expect(ds.User(ctx).SetUserLibraries(user.ID, []int{lib.ID})).To(Succeed())

	loadedUser, err := ds.User(ctx).FindByUsername(user.UserName)
	Expect(err).ToNot(HaveOccurred())
	user.Libraries = loadedUser.Libraries
	return user
}

// doReq makes a full HTTP round-trip through the router and returns the parsed Subsonic response.
func doReq(endpoint string, params ...string) *responses.Subsonic {
	return doReqWithUser(adminUser, endpoint, params...)
}

// doReqWithUser makes a full HTTP round-trip for the given user and returns the parsed Subsonic response.
func doReqWithUser(user model.User, endpoint string, params ...string) *responses.Subsonic {
	w := httptest.NewRecorder()
	r := buildReq(user, endpoint, params...)
	router.ServeHTTP(w, r)
	return parseJSONResponse(w)
}

// doRawReq returns the raw ResponseRecorder for endpoints that write binary data (stream, download, getCoverArt).
func doRawReq(endpoint string, params ...string) *httptest.ResponseRecorder {
	return doRawReqWithUser(adminUser, endpoint, params...)
}

// doRawReqWithUser returns the raw ResponseRecorder for the given user.
func doRawReqWithUser(user model.User, endpoint string, params ...string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := buildReq(user, endpoint, params...)
	router.ServeHTTP(w, r)
	return w
}

// buildReq creates a GET request with Subsonic auth params (u, p, v, c, f=json).
func buildReq(user model.User, endpoint string, params ...string) *http.Request {
	if len(params)%2 != 0 {
		panic("buildReq: odd number of parameters")
	}
	q := url.Values{}
	q.Add("u", user.UserName)
	q.Add("p", "password")
	q.Add("v", "1.16.1")
	q.Add("c", "test-client")
	q.Add("f", "json")
	for i := 0; i < len(params); i += 2 {
		q.Add(params[i], params[i+1])
	}
	return httptest.NewRequest("GET", "/"+endpoint+"?"+q.Encode(), nil)
}

// buildPostReq creates a POST request with a JSON body and Subsonic auth params in the query string.
func buildPostReq(user model.User, endpoint string, body string, params ...string) *http.Request {
	getReq := buildReq(user, endpoint, params...)
	r := httptest.NewRequest("POST", getReq.URL.RequestURI(), bytes.NewReader([]byte(body)))
	r.Header.Set("Content-Type", "application/json")
	return r
}

// doPostReq makes a POST round-trip as admin and returns the parsed Subsonic response.
func doPostReq(endpoint string, body string, params ...string) *responses.Subsonic {
	w := httptest.NewRecorder()
	r := buildPostReq(adminUser, endpoint, body, params...)
	router.ServeHTTP(w, r)
	return parseJSONResponse(w)
}

// doRawPostReq makes a POST round-trip as admin and returns the raw recorder.
func doRawPostReq(endpoint string, body string, params ...string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := buildPostReq(adminUser, endpoint, body, params...)
	router.ServeHTTP(w, r)
	return w
}

// parseJSONResponse parses the JSON response body into a Subsonic response struct.
func parseJSONResponse(w *httptest.ResponseRecorder) *responses.Subsonic {
	Expect(w.Code).To(Equal(http.StatusOK))
	var wrapper responses.JsonWrapper
	Expect(json.Unmarshal(w.Body.Bytes(), &wrapper)).To(Succeed())
	return &wrapper.Subsonic
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

// Compile-time interface checks
var (
	_ artwork.Artwork   = noopArtwork{}
	_ core.Archiver     = noopArchiver{}
	_ external.Provider = noopProvider{}
)

var _ = BeforeSuite(func() {
	ctx = request.WithUser(GinkgoT().Context(), adminUser)

	conf.Server.MusicFolder = "fake:///music"
	conf.Server.LyricsPriority = "embedded,.lrc,.srt,.yaml"
	conf.Server.DevExternalScanner = false

	buildTestFS()
	goldenDB = harness.SetupDB(ctx, &adminUser, &regularUser)
	lib = goldenDB.Library
	ctx = request.WithUser(GinkgoT().Context(), adminUser)
})

// Close the database before the suite's TempDir cleanup runs. Required on
// Windows where open SQLite handles hold file locks that block temp-dir
// removal; harmless on other OSes.
var _ = AfterSuite(func() {
	db.Close(ctx)
})

// setupTestDB restores the database from the golden snapshot and creates the
// Subsonic Router. Call this from BeforeEach/BeforeAll in each test container.
func setupTestDB() {
	ctx = request.WithUser(GinkgoT().Context(), adminUser)

	DeferCleanup(configtest.SetupConfig())
	DeferCleanup(func() {
		// Wait for any background scan (e.g. from startScan endpoint) to finish
		// before config cleanup runs, to avoid a data race on conf.Server.
		Eventually(scanner.IsScanning).Should(BeFalse())
	})
	conf.Server.MusicFolder = "fake:///music"
	conf.Server.DevExternalScanner = false
	conf.Server.DevEnableMediaFileProbe = false

	// Restore DB to golden state (no scan needed)
	goldenDB.Restore()

	ds = &tests.MockDataStore{RealDS: persistence.New(db.Db())}
	auth.Init(ds)

	// Create the Subsonic Router with real DS, streamer spy, and real Decider
	streamerSpy = &harness.SpyStreamer{}
	decider := stream.NewTranscodeDecider(ds, harness.NoopFFmpeg{})
	s := scanner.New(ctx, ds, artwork.NoopCacheWarmer(), events.NoopBroker(),
		playlists.NewPlaylists(ds, core.NewImageUploadService()), metrics.NewNoopInstance())
	router = subsonic.New(
		ds,
		noopArtwork{},
		streamerSpy,
		noopArchiver{},
		core.NewPlayers(ds),
		noopProvider{},
		s,
		events.NoopBroker(),
		playlists.NewPlaylists(ds, core.NewImageUploadService()),
		scrobbler.NewPlayTracker(ds, events.NoopBroker(), nil),
		core.NewShare(ds),
		playback.PlaybackServer(nil),
		metrics.NewNoopInstance(),
		lyrics.NewLyrics(ds, nil),
		decider,
		nil,
	)
}
