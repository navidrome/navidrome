// Package harness holds the pieces shared by the API e2e suites (server/subsonic/e2e and
// server/jellyfin/e2e): golden-database lifecycle, snapshot restore, fixture-FS registration,
// and service doubles. Like core/storage/storagetest, it must only be imported from test code.
package harness

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing/fstest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/core/storage/storagetest"
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/tests"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega" //nolint:staticcheck
)

// DB is a golden e2e database: scanned once in BeforeSuite, restored per test via Restore.
type DB struct {
	FilePath     string
	SnapshotPath string
	Library      model.Library
}

// CreateFS registers files under the "fake:" storage scheme the suites use as MusicFolder.
func CreateFS(files fstest.MapFS) storagetest.FakeFS {
	fs := storagetest.FakeFS{}
	fs.SetFiles(files)
	storagetest.Register("fake", &fs)
	return fs
}

// SetupDB boots the golden database: a temp SQLite file, the given users (password "password",
// all with access to the seeded "Music Library"), a full scan of the registered fake FS, and a
// snapshot for per-test restore. Callers must set conf.Server.MusicFolder and register the FS
// first; each user's Libraries field is populated in place.
func SetupDB(ctx context.Context, users ...*model.User) *DB {
	tmpDir := ginkgo.GinkgoT().TempDir()
	h := &DB{FilePath: filepath.Join(tmpDir, "test-e2e.db")}
	h.SnapshotPath = h.FilePath + ".snapshot"
	conf.Server.DbPath = h.FilePath + "?_journal_mode=WAL"
	db.Db().SetMaxOpenConns(1)
	db.Init(ctx)

	ds := &tests.MockDataStore{RealDS: persistence.New(db.Db())}
	auth.Init(ds)

	h.Library = model.Library{ID: 1, Name: "Music Library", Path: "fake:///music"}
	Expect(ds.Library(ctx).Put(&h.Library)).To(Succeed())

	for _, u := range users {
		seeded := *u
		seeded.NewPassword = "password"
		Expect(ds.User(ctx).Put(&seeded)).To(Succeed())
		Expect(ds.User(ctx).SetUserLibraries(u.ID, []int{h.Library.ID})).To(Succeed())
		loaded, err := ds.User(ctx).FindByUsername(u.UserName)
		Expect(err).ToNot(HaveOccurred())
		u.Libraries = loaded.Libraries
	}

	s := scanner.New(ctx, ds, artwork.NoopCacheWarmer(), events.NoopBroker(),
		playlists.NewPlaylists(ds, core.NewImageUploadService()), metrics.NewNoopInstance())
	_, err := s.ScanAll(ctx, true)
	Expect(err).ToNot(HaveOccurred())

	_, err = db.Db().Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	Expect(err).ToNot(HaveOccurred())
	data, err := os.ReadFile(h.FilePath)
	Expect(err).ToNot(HaveOccurred())
	Expect(os.WriteFile(h.SnapshotPath, data, 0o600)).To(Succeed()) //nolint:gosec // path derives from TempDir
	return h
}

// Restore reloads every table from the golden snapshot via ATTACH DATABASE — much faster than a
// rescan. FTS shadow tables are skipped; they are kept in sync by their content tables' triggers.
func (h *DB) Restore() {
	sqlDB := db.Db()
	_, err := sqlDB.Exec("PRAGMA foreign_keys = OFF")
	Expect(err).ToNot(HaveOccurred())
	_, err = sqlDB.Exec("ATTACH DATABASE ? AS snapshot", h.SnapshotPath)
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
		// Table names come from sqlite_master, not user input.
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

// SpyStreamer captures the Request passed to NewStream and returns a minimal fake stream.
type SpyStreamer struct {
	LastRequest         stream.Request
	LastMediaFile       *model.MediaFile
	SimulateError       error // when set, NewStream returns this error
	SimulateEmptyStream bool  // when true, returns a 0-byte stream (ffmpeg produced no output)
}

func (s *SpyStreamer) NewStream(_ context.Context, mf *model.MediaFile, req stream.Request) (*stream.Stream, error) {
	s.LastRequest = req
	s.LastMediaFile = mf
	if s.SimulateError != nil {
		return nil, s.SimulateError
	}
	format := req.Format
	if format == "" || format == "raw" {
		format = mf.Suffix
	}
	content := "fake audio data"
	if s.SimulateEmptyStream {
		content = ""
	}
	return stream.NewStream(mf, format, req.BitRate, io.NopCloser(strings.NewReader(content))), nil
}

// NoopFFmpeg implements ffmpeg.FFmpeg; transcoding never actually runs in e2e.
type NoopFFmpeg struct{}

func (NoopFFmpeg) Transcode(context.Context, ffmpeg.TranscodeOptions) (io.ReadCloser, error) {
	return nil, errors.New("noop ffmpeg: transcode not supported")
}

func (NoopFFmpeg) ExtractImage(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("noop ffmpeg: extract image not supported")
}

func (NoopFFmpeg) Probe(context.Context, []string) (string, error) { return "", nil }

func (NoopFFmpeg) ProbeAudioStream(context.Context, string) (*ffmpeg.AudioProbeResult, error) {
	return nil, errors.New("noop ffmpeg: probe not supported")
}

func (NoopFFmpeg) ConvertAnimatedImage(context.Context, io.Reader, int, int) (io.ReadCloser, error) {
	return nil, errors.New("noop ffmpeg: convert animated image not supported")
}

func (NoopFFmpeg) CmdPath() (string, error) { return "", nil }
func (NoopFFmpeg) IsAvailable() bool        { return false }
func (NoopFFmpeg) IsProbeAvailable() bool   { return true }
func (NoopFFmpeg) Version() string          { return "noop" }

var (
	_ stream.MediaStreamer = &SpyStreamer{}
	_ ffmpeg.FFmpeg        = NoopFFmpeg{}
)
