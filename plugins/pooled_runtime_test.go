package plugins

import (
	"context"
	"path/filepath"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tetratelabs/wazero"
)

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
		ccache, _ := getCompilationCache()
		// Add permissions for the test plugin
		permissions := map[string]any{
			"http": map[string]any{
				"reason": "For testing HTTP functionality",
				"allowedUrls": map[string]any{
					"*": []any{"*"},
				},
			},
			"config": struct{}{},
		}
		rtFunc := mgr.createCustomRuntime(ccache, "fake_scrobbler", permissions)
		plugin = NewWasmScrobblerPlugin(
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
