package plugins

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/plugins/schema"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tetratelabs/wazero"
)

var _ = Describe("Runtime", func() {
	Describe("pluginCompilationTimeout", func() {
		It("should use DevPluginCompilationTimeout config for plugin compilation timeout", func() {
			originalTimeout := conf.Server.DevPluginCompilationTimeout
			DeferCleanup(func() {
				conf.Server.DevPluginCompilationTimeout = originalTimeout
			})

			conf.Server.DevPluginCompilationTimeout = 123 * time.Second
			Expect(pluginCompilationTimeout()).To(Equal(123 * time.Second))

			conf.Server.DevPluginCompilationTimeout = 0
			Expect(pluginCompilationTimeout()).To(Equal(time.Minute))
		})
	})
})

var _ = Describe("PooledRuntime", func() {
	var (
		ctx    context.Context
		mgr    *Manager
		plugin *wasmScrobblerPlugin
		prt    *pooledRuntime
	)

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		mgr = createManager()
		// Add permissions for the test plugin using typed struct
		permissions := schema.PluginManifestPermissions{
			Http: &schema.PluginManifestPermissionsHttp{
				Reason: "For testing HTTP functionality",
				AllowedUrls: map[string][]schema.PluginManifestPermissionsHttpAllowedUrlsValueElem{
					"*": {schema.PluginManifestPermissionsHttpAllowedUrlsValueElemWildcard},
				},
				AllowLocalNetwork: false,
			},
			Config: &schema.PluginManifestPermissionsConfig{
				Reason: "For testing config functionality",
			},
		}
		rtFunc := mgr.createCustomRuntime("fake_scrobbler", permissions)
		plugin = newWasmScrobblerPlugin(
			filepath.Join(testDataDir, "fake_scrobbler", "plugin.wasm"),
			"fake_scrobbler",
			rtFunc,
			wazero.NewModuleConfig().WithStartFunctions("_initialize"),
		).(*wasmScrobblerPlugin)
		// runtime will be created on first plugin load
	})

	It("reuses module instances across calls", func() {
		_, done, err := plugin.getInstance(ctx, "first")
		Expect(err).ToNot(HaveOccurred())
		done()

		val, ok := runtimePool.Load("fake_scrobbler")
		Expect(ok).To(BeTrue())
		prt = val.(*pooledRuntime)
		prt.mu.Lock()
		Expect(len(prt.pool.items)).To(Equal(1))
		ptr1 := reflect.ValueOf(prt.pool.items[0].value).Pointer()
		prt.mu.Unlock()

		_, done, err = plugin.getInstance(ctx, "second")
		Expect(err).ToNot(HaveOccurred())
		done()

		prt.mu.Lock()
		Expect(len(prt.pool.items)).To(Equal(1))
		ptr2 := reflect.ValueOf(prt.pool.items[0].value).Pointer()
		active := len(prt.active)
		prt.mu.Unlock()

		Expect(ptr2).To(Equal(ptr1))
		Expect(active).To(Equal(0))
	})
})

var _ = Describe("purgeCacheBySize", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "cache_test")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(os.RemoveAll, tmpDir)
	})

	It("removes oldest entries when above the size limit", func() {
		oldDir := filepath.Join(tmpDir, "d1")
		newDir := filepath.Join(tmpDir, "d2")
		Expect(os.Mkdir(oldDir, 0700)).To(Succeed())
		Expect(os.Mkdir(newDir, 0700)).To(Succeed())

		oldFile := filepath.Join(oldDir, "old")
		newFile := filepath.Join(newDir, "new")
		Expect(os.WriteFile(oldFile, []byte("xx"), 0600)).To(Succeed())
		Expect(os.WriteFile(newFile, []byte("xx"), 0600)).To(Succeed())

		oldTime := time.Now().Add(-2 * time.Hour)
		Expect(os.Chtimes(oldFile, oldTime, oldTime)).To(Succeed())

		purgeCacheBySize(tmpDir, "3")

		_, err := os.Stat(oldFile)
		Expect(os.IsNotExist(err)).To(BeTrue())
		_, err = os.Stat(oldDir)
		Expect(os.IsNotExist(err)).To(BeTrue())

		_, err = os.Stat(newFile)
		Expect(err).ToNot(HaveOccurred())
	})

	It("does nothing when below the size limit", func() {
		dir1 := filepath.Join(tmpDir, "a")
		dir2 := filepath.Join(tmpDir, "b")
		Expect(os.Mkdir(dir1, 0700)).To(Succeed())
		Expect(os.Mkdir(dir2, 0700)).To(Succeed())

		file1 := filepath.Join(dir1, "f1")
		file2 := filepath.Join(dir2, "f2")
		Expect(os.WriteFile(file1, []byte("x"), 0600)).To(Succeed())
		Expect(os.WriteFile(file2, []byte("x"), 0600)).To(Succeed())

		purgeCacheBySize(tmpDir, "10MB")

		_, err := os.Stat(file1)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Stat(file2)
		Expect(err).ToNot(HaveOccurred())
	})
})
