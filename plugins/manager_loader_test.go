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

var _ = Describe("parsePluginConfig", func() {
	It("returns nil for empty string", func() {
		result, err := parsePluginConfig("")
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeNil())
	})

	It("serializes object values as JSON strings", func() {
		result, err := parsePluginConfig(`{"settings": {"enabled": true, "count": 5}}`)
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(HaveLen(1))
		Expect(result["settings"]).To(Equal(`{"count":5,"enabled":true}`))
	})

	It("handles mixed value types", func() {
		result, err := parsePluginConfig(`{"api_key": "secret", "timeout": 30, "rate": 1.5, "enabled": true, "tags": ["a", "b"]}`)
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(HaveLen(5))
		Expect(result["api_key"]).To(Equal("secret"))
		Expect(result["timeout"]).To(Equal("30"))
		Expect(result["rate"]).To(Equal("1.5"))
		Expect(result["enabled"]).To(Equal("true"))
		Expect(result["tags"]).To(Equal(`["a","b"]`))
	})

	It("returns error for invalid JSON", func() {
		_, err := parsePluginConfig(`{invalid json}`)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("parsing plugin config"))
	})

	It("returns error for non-object JSON", func() {
		_, err := parsePluginConfig(`["array", "not", "object"]`)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("parsing plugin config"))
	})

	It("handles null values", func() {
		result, err := parsePluginConfig(`{"key": null}`)
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(HaveLen(1))
		Expect(result["key"]).To(Equal("null"))
	})

	It("handles empty object", func() {
		result, err := parsePluginConfig(`{}`)
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(HaveLen(0))
		Expect(result).ToNot(BeNil())
	})
})

var _ = Describe("buildAllowedPaths", func() {
	var libraries model.Libraries

	BeforeEach(func() {
		libraries = model.Libraries{
			{ID: 1, Path: "/music/library1"},
			{ID: 2, Path: "/music/library2"},
			{ID: 3, Path: "/music/library3"},
		}
	})

	Context("read-only (default)", func() {
		It("mounts all libraries with ro: prefix when allLibraries is true", func() {
			result := buildAllowedPaths(nil, libraries, nil, true, false)
			Expect(result).To(HaveLen(3))
			Expect(result).To(HaveKeyWithValue("ro:/music/library1", "/libraries/1"))
			Expect(result).To(HaveKeyWithValue("ro:/music/library2", "/libraries/2"))
			Expect(result).To(HaveKeyWithValue("ro:/music/library3", "/libraries/3"))
		})

		It("mounts only selected libraries with ro: prefix", func() {
			result := buildAllowedPaths(nil, libraries, []int{1, 3}, false, false)
			Expect(result).To(HaveLen(2))
			Expect(result).To(HaveKeyWithValue("ro:/music/library1", "/libraries/1"))
			Expect(result).To(HaveKeyWithValue("ro:/music/library3", "/libraries/3"))
			Expect(result).ToNot(HaveKey("ro:/music/library2"))
		})
	})

	Context("read-write (allowWriteAccess=true)", func() {
		It("mounts all libraries without ro: prefix when allLibraries is true", func() {
			result := buildAllowedPaths(nil, libraries, nil, true, true)
			Expect(result).To(HaveLen(3))
			Expect(result).To(HaveKeyWithValue("/music/library1", "/libraries/1"))
			Expect(result).To(HaveKeyWithValue("/music/library2", "/libraries/2"))
			Expect(result).To(HaveKeyWithValue("/music/library3", "/libraries/3"))
		})

		It("mounts only selected libraries without ro: prefix", func() {
			result := buildAllowedPaths(nil, libraries, []int{2}, false, true)
			Expect(result).To(HaveLen(1))
			Expect(result).To(HaveKeyWithValue("/music/library2", "/libraries/2"))
		})
	})

	Context("edge cases", func() {
		It("returns empty map when no libraries match", func() {
			result := buildAllowedPaths(nil, libraries, []int{99}, false, false)
			Expect(result).To(BeEmpty())
		})

		It("returns empty map when libraries list is empty", func() {
			result := buildAllowedPaths(nil, nil, []int{1}, false, false)
			Expect(result).To(BeEmpty())
		})

		It("returns empty map when allLibraries is false and no IDs provided", func() {
			result := buildAllowedPaths(nil, libraries, nil, false, false)
			Expect(result).To(BeEmpty())
		})
	})
})

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
			ID:      "test-taskqueue",
			Path:    dest,
			SHA256:  hex.EncodeToString(hash[:]),
			Enabled: false,
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
})
