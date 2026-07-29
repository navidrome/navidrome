//go:build !windows

package plugins

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("loadPluginWithConfig", func() {
	var manager *Manager
	var dataDir string

	BeforeEach(func() {
		pluginsDir := GinkgoT().TempDir()
		dataDir = GinkgoT().TempDir()

		src := filepath.Join(testdataDir, "test-taskqueue"+PackageExtension)
		data, err := os.ReadFile(src)
		Expect(err).ToNot(HaveOccurred())
		dest := filepath.Join(pluginsDir, "test-taskqueue"+PackageExtension)
		Expect(os.WriteFile(dest, data, 0600)).To(Succeed())
		hash := sha256.Sum256(data)

		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = conf.NewDir(pluginsDir)
		conf.Server.Plugins.AutoReload = false
		conf.Server.DataFolder = conf.NewDir(dataDir)

		repo := tests.CreateMockPluginRepo()
		repo.Permitted = true
		repo.SetData(model.Plugins{{
			ID:                    "test-taskqueue",
			Path:                  dest,
			SHA256:                hex.EncodeToString(hash[:]),
			ManifestSchemaVersion: CurrentManifestSchemaVersion,
			Enabled:               false,
		}})
		manager = &Manager{
			plugins:        make(map[string]*plugin),
			ds:             &tests.MockDataStore{MockedPlugin: repo},
			metrics:        noopMetricsRecorder{},
			subsonicRouter: http.NotFoundHandler(),
		}
	})

	Describe("host service creation failures", func() {
		It("reports the Task service creation error instead of a missing host function", func() {
			Expect(manager.Start(GinkgoT().Context())).To(Succeed())
			DeferCleanup(func() { _ = manager.Stop() })

			// Block the taskqueue data dir by creating a file where the directory should be
			Expect(os.WriteFile(filepath.Join(dataDir, "plugins"), nil, 0600)).To(Succeed())

			err := manager.EnablePlugin(GinkgoT().Context(), "test-taskqueue")
			Expect(err).To(MatchError(ContainSubstring("creating Task service")))
			Expect(err).ToNot(MatchError(ContainSubstring("not exported")))
		})
	})

	Describe("unstarted manager", func() {
		It("enables a taskqueue plugin on a manager that was never started", func() {
			// CLI commands (navidrome plugin enable) use the manager without calling Start
			Expect(manager.EnablePlugin(GinkgoT().Context(), "test-taskqueue")).To(Succeed())
			DeferCleanup(func() { _ = manager.unloadPlugin("test-taskqueue") })
		})
	})

	Describe("schema-only manifest re-extraction", func() {
		It("transparently reloads and stays enabled when the plugin was already running", func() {
			// Regression test for the bug this was written to fix: a manifest
			// re-extraction triggered purely by CurrentManifestSchemaVersion
			// advancing (not by the .ndp file itself changing) must not leave
			// an already-working plugin quietly disabled with no user-visible
			// signal - it should come back up on its own.
			Expect(manager.Start(GinkgoT().Context())).To(Succeed())
			DeferCleanup(func() { _ = manager.Stop() })

			Expect(manager.EnablePlugin(GinkgoT().Context(), "test-taskqueue")).To(Succeed())
			DeferCleanup(func() { _ = manager.unloadPlugin("test-taskqueue") })

			repo := manager.ds.Plugin(GinkgoT().Context())
			existing, err := repo.Get("test-taskqueue")
			Expect(err).ToNot(HaveOccurred())
			Expect(existing.Enabled).To(BeTrue())

			metadata := &PluginMetadata{
				Manifest: rereadManifest(existing),
				SHA256:   existing.SHA256, // file itself is unchanged
			}
			Expect(manager.updatePluginInDB(GinkgoT().Context(), repo, existing, existing.Path, metadata)).To(Succeed())

			stored, err := repo.Get("test-taskqueue")
			Expect(err).ToNot(HaveOccurred())
			Expect(stored.ManifestSchemaVersion).To(Equal(CurrentManifestSchemaVersion))
			Expect(stored.Enabled).To(BeTrue(), "schema-only re-extraction of an already-working plugin should reload it, not leave it disabled")
			Expect(stored.LastError).To(BeEmpty())
		})
	})
})

// rereadManifest re-parses a loaded plugin's own manifest.json, so the
// schema-only-re-extraction test above can simulate "this build now
// understands the manifest differently" without needing a second .ndp fixture.
func rereadManifest(p *model.Plugin) *Manifest {
	pkg, err := openPackage(p.Path)
	Expect(err).ToNot(HaveOccurred())
	return pkg.Manifest
}
