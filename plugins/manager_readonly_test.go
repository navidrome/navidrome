package plugins

import (
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager.LoadPlugins", func() {
	var (
		mgr    *Manager
		repo   *tests.MockPluginRepo
		tmpDir string
	)

	// newManager builds a manager over rows the caller can corrupt, with no Subsonic router: a CLI
	// has none, and Start would log.Fatal on that.
	newManager := func(rows model.Plugins) *Manager {
		DeferCleanup(configtest.SetupConfig())
		var err error
		tmpDir, err = os.MkdirTemp("", "plugins-readonly-*")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { _ = os.RemoveAll(tmpDir) })

		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = conf.NewDir(tmpDir)
		conf.Server.Plugins.AutoReload = false
		conf.Server.CacheFolder = conf.NewDir(tmpDir)

		if rows == nil {
			rows = installTestPlugins(tmpDir, "test-metadata-agent"+PackageExtension)
			for i := range rows {
				rows[i].AllUsers = true
			}
		}
		repo = tests.CreateMockPluginRepo()
		repo.Permitted = true
		repo.SetData(rows)
		m := &Manager{
			plugins: make(map[string]*plugin),
			ds:      &tests.MockDataStore{MockedPlugin: repo},
			metrics: noopMetricsRecorder{},
		}
		DeferCleanup(func() { _ = m.Stop() })
		return m
	}

	It("detects capabilities without a Subsonic router configured", func() {
		mgr = newManager(nil)

		Expect(mgr.LoadPlugins(GinkgoT().Context(), []string{"test-metadata-agent", "broken"}, false)).To(Succeed())

		Expect(mgr.PluginNames(string(CapabilityMetadataAgent))).To(ContainElement("test-metadata-agent"))
	})

	Context("when a plugin cannot be loaded", func() {
		brokenRows := func() model.Plugins {
			return model.Plugins{{
				ID: "broken", Path: filepath.Join(GinkgoT().TempDir(), "does-not-exist.ndp"),
				Enabled: true, AllUsers: true,
			}}
		}

		It("leaves the stored row untouched", func() {
			mgr = newManager(brokenRows())

			Expect(mgr.LoadPlugins(GinkgoT().Context(), []string{"test-metadata-agent", "broken"}, false)).To(Succeed())

			stored, err := repo.Get("broken")
			Expect(err).ToNot(HaveOccurred())
			Expect(stored.Enabled).To(BeTrue(), "inspecting a plugin must never disable it")
			Expect(stored.LastError).To(BeEmpty())
		})

		// Without this the test above would pass for the wrong reason. Start cannot be used: it
		// syncs the folder first, dropping a row whose file is missing before any load.
		It("still disables it when not read-only", func() {
			mgr = newManager(brokenRows())

			Expect(mgr.loadEnabledPlugins(GinkgoT().Context())).To(Succeed())

			stored, err := repo.Get("broken")
			Expect(err).ToNot(HaveOccurred())
			Expect(stored.Enabled).To(BeFalse())
			Expect(stored.LastError).ToNot(BeEmpty())
		})
	})

	// Loading a plugin creates its host services — a KVStore or task queue database on disk — so a
	// plugin that could never supply an image must not be instantiated just to be ignored.
	It("does not load a plugin that is not in the agent list", func() {
		mgr = newManager(nil)

		Expect(mgr.LoadPlugins(GinkgoT().Context(), []string{"some-other-agent"}, false)).To(Succeed())

		Expect(mgr.PluginNames(string(CapabilityMetadataAgent))).To(BeEmpty())
	})

	It("does nothing when no agents are configured", func() {
		mgr = newManager(nil)

		Expect(mgr.LoadPlugins(GinkgoT().Context(), nil, false)).To(Succeed())

		Expect(mgr.PluginNames(string(CapabilityMetadataAgent))).To(BeEmpty())
		// Not even the wazero cache: with nothing to load there is nothing to compile.
		Expect(filepath.Join(tmpDir, "plugins")).ToNot(BeADirectory())
	})

	It("does nothing when the plugin system is disabled", func() {
		mgr = newManager(nil)
		conf.Server.Plugins.Enabled = false

		Expect(mgr.LoadPlugins(GinkgoT().Context(), []string{"test-metadata-agent", "broken"}, false)).To(Succeed())

		Expect(mgr.PluginNames(string(CapabilityMetadataAgent))).To(BeEmpty())
	})
})
