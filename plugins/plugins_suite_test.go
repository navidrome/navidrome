package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	testDataDir    = "plugins/testdata"
	wazeroCacheDir = ".wazero-cache"
)

// Shared test state initialized in BeforeSuite
var (
	testdataDir   string // Path to testdata folder with test plugin .ndp packages
	tmpPluginsDir string // Temp directory for plugin tests that modify files
	testManager   *Manager
	// testManagerPlugins are testManager's DB rows, for Ordered specs that need them after a reset
	testManagerPlugins model.Plugins
)

func TestPlugins(t *testing.T) {
	tests.Init(t, false)

	// Set globally so every spec's config snapshot inherits it. The cache
	// persists between runs; entries are content-addressed, so a stale one only misses.
	conf.Update(func(c *conf.Options) {
		c.CacheFolder = conf.NewDir(filepath.Join(testDataDir, wazeroCacheDir))
		c.Plugins.CacheSize = "1GB"
	}) // the default evicts the cache mid-run

	log.SetLevel(log.LevelFatal)
	persistencetest.Init(t)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugins Suite")
}

var _ = configtest.RestoreAfterEachSpec()
var _ = persistencetest.ResetBeforeEachSpec()

// newTestDataStore stores the plugin rows in the real database and returns a DataStore over it.
func newTestDataStore(plugins ...model.Plugin) model.DataStore {
	GinkgoHelper()
	ds := persistence.New(db.Db())
	ctx := adminContext(GinkgoT().Context())
	for i := range plugins {
		Expect(ds.Plugin().Put(ctx, &plugins[i])).To(Succeed())
	}
	return ds
}

// putUsers stores users, giving non-admins access to the libraries marked for new users.
func putUsers(ds model.DataStore, users ...model.User) {
	GinkgoHelper()
	ctx := GinkgoT().Context()
	for i := range users {
		Expect(ds.User().Put(ctx, &users[i])).To(Succeed())
	}
}

// putLibraries stores libs, including library 1's path and the stats columns Put does not write.
func putLibraries(ds model.DataStore, libs ...model.Library) {
	GinkgoHelper()
	ctx := adminContext(GinkgoT().Context())
	for i := range libs {
		lib := libs[i]
		if lib.ID == model.DefaultLibraryID {
			// Library().Put refuses to change library 1's path; StoreMusicFolder is the only way.
			restore := conf.SnapshotConfig()
			conf.Update(func(c *conf.Options) { c.MusicFolder = lib.Path })
			Expect(ds.Library().StoreMusicFolder(ctx)).To(Succeed())
			restore()
		}
		Expect(ds.Library().Put(ctx, &lib)).To(Succeed())
		_, err := db.Db().Exec(`UPDATE library SET total_songs = ?, total_albums = ?, total_artists = ?,
			total_size = ?, total_duration = ? WHERE id = ?`,
			lib.TotalSongs, lib.TotalAlbums, lib.TotalArtists, lib.TotalSize, lib.TotalDuration, lib.ID)
		Expect(err).ToNot(HaveOccurred())
	}
}

// stubLocalhostDNS resolves "localhost." without real DNS: the trailing dot never matches /etc/hosts.
func stubLocalhostDNS() {
	dialResolver = tests.StubResolver(map[string]string{"localhost.": "127.0.0.1"})
	DeferCleanup(func() { dialResolver = nil })
}

// createTestManager creates a new plugin Manager with the given plugin config.
// It creates a temp directory, copies the test-metadata-agent plugin, and starts the manager.
// Returns the manager, temp directory path, and a cleanup function.
func createTestManager(pluginConfig map[string]map[string]string) (*Manager, string) {
	return createTestManagerWithPlugins(pluginConfig, "test-metadata-agent"+PackageExtension)
}

// createTestManagerWithPlugins creates a new plugin Manager with the given plugin config
// and specified plugins. It creates a temp directory, copies the specified plugins, and starts the manager.
// Returns the manager and temp directory path.
func createTestManagerWithPlugins(pluginConfig map[string]map[string]string, plugins ...string) (*Manager, string) {
	return createTestManagerWithPluginsAndMetrics(pluginConfig, noopMetricsRecorder{}, plugins...)
}

// installTestPlugins copies the given .ndp packages into dir and returns their
// enabled DB rows, so callers can grant whatever access the test needs.
func installTestPlugins(dir string, plugins ...string) model.Plugins {
	var rows model.Plugins
	for _, plugin := range plugins {
		data, err := os.ReadFile(filepath.Join(testdataDir, plugin))
		Expect(err).ToNot(HaveOccurred())
		destPath := filepath.Join(dir, plugin)
		Expect(os.WriteFile(destPath, data, 0600)).To(Succeed())

		hash := sha256.Sum256(data)
		rows = append(rows, model.Plugin{
			ID:      strings.TrimSuffix(plugin, PackageExtension),
			Path:    destPath,
			SHA256:  hex.EncodeToString(hash[:]),
			Enabled: true,
		})
	}
	return rows
}

// createTestManagerWithPluginsAndMetrics creates a new plugin Manager with the given plugin config,
// metrics recorder, and specified plugins. It creates a temp directory, copies the specified plugins,
// and starts the manager. Returns the manager and temp directory path.
func createTestManagerWithPluginsAndMetrics(pluginConfig map[string]map[string]string, metrics PluginMetricsRecorder, plugins ...string) (*Manager, string) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "plugins-test-*")
	Expect(err).ToNot(HaveOccurred())

	enabledPlugins := installTestPlugins(tmpDir, plugins...)
	for i, p := range enabledPlugins {
		enabledPlugins[i].AllUsers = true // Allow all users by default in tests
		if pluginConfig[p.ID] != nil {
			configBytes, err := json.Marshal(pluginConfig[p.ID])
			Expect(err).ToNot(HaveOccurred())
			enabledPlugins[i].Config = string(configBytes)
		}
	}

	// Setup config
	DeferCleanup(conf.SnapshotConfig())
	conf.Update(func(c *conf.Options) {
		c.Plugins.Enabled = true
		c.Plugins.Folder = conf.NewDir(tmpDir)
		c.Plugins.AutoReload = false
	})

	dataStore := newTestDataStore(enabledPlugins...)

	// Create and start manager
	manager := &Manager{
		plugins:        make(map[string]*plugin),
		ds:             dataStore,
		metrics:        metrics,
		subsonicRouter: http.NotFoundHandler(), // Stub router for tests
	}
	err = manager.Start(GinkgoT().Context())
	Expect(err).ToNot(HaveOccurred())

	DeferCleanup(func() {
		_ = manager.Stop()
		_ = os.RemoveAll(tmpDir)
	})

	return manager, tmpDir
}

var _ = SynchronizedBeforeSuite(func() {
	// Build once: the testdata Makefile is not safe to run concurrently.
	buildTestPlugins(testDataDir)
}, func() {
	// Get testdata directory (where test plugin .ndp packages live)
	_, currentFile, _, ok := runtime.Caller(0)
	Expect(ok).To(BeTrue())
	testdataDir = filepath.Join(filepath.Dir(currentFile), "testdata")

	// Create shared manager for most tests
	testManager, tmpPluginsDir = createTestManager(nil)
	var err error
	testManagerPlugins, err = testManager.ds.Plugin().GetAll(adminContext(GinkgoT().Context()))
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	if testManager != nil {
		_ = testManager.Stop()
	}
	if tmpPluginsDir != "" {
		_ = os.RemoveAll(tmpPluginsDir)
	}
})

// noopMetricsRecorder is a no-op implementation of PluginMetricsRecorder for tests
type noopMetricsRecorder struct{}

func (noopMetricsRecorder) RecordPluginRequest(context.Context, string, string, bool, int64) {}

// failingUsers makes the user lookups host services run fail with err.
func failingUsers(ds model.DataStore, err error) model.DataStore {
	return persistencetest.Stub(ds, persistencetest.Repos{User: func(r model.UserRepository) model.UserRepository {
		return failingUserRepo{UserRepository: r, err: err}
	}})
}

type failingUserRepo struct {
	model.UserRepository
	err error
}

func (r failingUserRepo) GetAll(context.Context, ...model.QueryOptions) (model.Users, error) {
	return nil, r.err
}

func (r failingUserRepo) FindByUsername(context.Context, string) (*model.User, error) {
	return nil, r.err
}
