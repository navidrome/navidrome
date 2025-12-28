//go:build !windows

package plugins

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const testDataDir = "plugins/testdata"

// Shared test state initialized in BeforeSuite
var (
	testdataDir   string // Path to testdata folder with test-metadata-agent.wasm
	tmpPluginsDir string // Temp directory for plugin tests that modify files
	testManager   *Manager
)

func TestPlugins(t *testing.T) {
	tests.Init(t, false)
	buildTestPlugins(t, testDataDir)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugins Suite")
}

func buildTestPlugins(t *testing.T, path string) {
	t.Helper()
	t.Logf("[BeforeSuite] Current working directory: %s", path)
	cmd := exec.Command("make", "-C", path)
	out, err := cmd.CombinedOutput()
	t.Logf("[BeforeSuite] Make output: %s", string(out))
	if err != nil {
		t.Fatalf("Failed to build test plugins: %v", err)
	}
}

// createTestManager creates a new plugin Manager with the given plugin config.
// It creates a temp directory, copies the test-metadata-agent plugin, and starts the manager.
// Returns the manager, temp directory path, and a cleanup function.
func createTestManager(pluginConfig map[string]map[string]string) (*Manager, string) {
	return createTestManagerWithPlugins(pluginConfig, "test-metadata-agent.wasm")
}

// createTestManagerWithPlugins creates a new plugin Manager with the given plugin config
// and specified plugins. It creates a temp directory, copies the specified plugins, and starts the manager.
// Returns the manager and temp directory path.
func createTestManagerWithPlugins(pluginConfig map[string]map[string]string, plugins ...string) (*Manager, string) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "plugins-test-*")
	Expect(err).ToNot(HaveOccurred())

	// Copy test plugins to temp dir and build plugin list with SHA256
	var enabledPlugins model.Plugins
	for _, plugin := range plugins {
		srcPath := filepath.Join(testdataDir, plugin)
		destPath := filepath.Join(tmpDir, plugin)
		data, err := os.ReadFile(srcPath)
		Expect(err).ToNot(HaveOccurred())
		err = os.WriteFile(destPath, data, 0600)
		Expect(err).ToNot(HaveOccurred())

		// Compute SHA256 for the plugin
		hash := sha256.Sum256(data)
		hashHex := hex.EncodeToString(hash[:])
		pluginName := plugin[:len(plugin)-5] // Remove .wasm extension

		// Build config JSON if provided
		configJSON := ""
		if pluginConfig != nil && pluginConfig[pluginName] != nil {
			// Encode config to JSON
			configBytes, err := json.Marshal(pluginConfig[pluginName])
			Expect(err).ToNot(HaveOccurred())
			configJSON = string(configBytes)
		}

		enabledPlugins = append(enabledPlugins, model.Plugin{
			ID:      pluginName,
			Path:    destPath,
			SHA256:  hashHex,
			Enabled: true,
			Config:  configJSON,
		})
	}

	// Setup config
	DeferCleanup(configtest.SetupConfig())
	conf.Server.Plugins.Enabled = true
	conf.Server.Plugins.Folder = tmpDir
	conf.Server.Plugins.AutoReload = false
	conf.Server.CacheFolder = filepath.Join(tmpDir, "cache")

	// Setup mock DataStore with pre-enabled plugins
	mockPluginRepo := tests.CreateMockPluginRepo()
	mockPluginRepo.Permitted = true
	mockPluginRepo.SetData(enabledPlugins)
	dataStore := &tests.MockDataStore{MockedPlugin: mockPluginRepo}

	// Create and start manager
	manager := &Manager{
		plugins:        make(map[string]*plugin),
		ds:             dataStore,
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

var _ = BeforeSuite(func() {
	// Get testdata directory (where test-metadata-agent.wasm lives)
	_, currentFile, _, ok := runtime.Caller(0)
	Expect(ok).To(BeTrue())
	testdataDir = filepath.Join(filepath.Dir(currentFile), "testdata")

	// Create shared manager for most tests
	testManager, tmpPluginsDir = createTestManager(nil)
})

var _ = AfterSuite(func() {
	if testManager != nil {
		_ = testManager.Stop()
	}
	if tmpPluginsDir != "" {
		_ = os.RemoveAll(tmpPluginsDir)
	}
})
