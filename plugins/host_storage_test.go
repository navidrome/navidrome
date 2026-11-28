package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	extism "github.com/extism/go-sdk"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("utility functions", Ordered, func() {
	var tmpDir string

	BeforeAll(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "storage-test-*")
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(conf.SnapshotConfig())
		conf.Update(func(c *conf.Options) { c.DataFolder = conf.NewDir(tmpDir) })

		DeferCleanup(func() {
			_ = os.RemoveAll(tmpDir)
		})
	})

	Describe("GetHostStoragePath", func() {
		It("should join data folder, plugins, plugin name, and storage", func() {
			actual := getHostStoragePath("plugin-name")
			expected := filepath.Join(tmpDir, "plugins", "plugin-name", "storage")
			Expect(actual).To(Equal(expected))
		})
	})

	Describe("GetStoragePath", func() {
		It("should return the fixed path", func() {
			impl := storageServiceImpl{}
			Expect(impl.GetStoragePath(GinkgoT().Context())).To(Equal("/storage"))
		})
	})

	Describe("netStorageService", func() {
		It("should create the directory on init", func() {
			svc, err := newStorageService("plugin-name")
			Expect(err).ToNot(HaveOccurred())
			Expect(svc).ToNot(BeNil())
			DeferCleanup(svc.Close)

			dataDir := filepath.Join(tmpDir, "plugins", "plugin-name", "storage")
			Expect(dataDir).To(BeADirectory())
		})
	})
})

var _ = Describe("storageServiceImpl", func() {
	var (
		ctx  context.Context
		svc  *storageServiceImpl
		root string
	)

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		DeferCleanup(conf.SnapshotConfig())
		conf.Update(func(c *conf.Options) { c.DataFolder = conf.NewDir(GinkgoT().TempDir()) })
		var err error
		svc, err = newStorageService("plugin")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(svc.Close)
		root = getHostStoragePath("plugin")
	})

	readFile := func(rc io.ReadCloser) string {
		GinkgoHelper()
		defer rc.Close()
		data, err := io.ReadAll(rc)
		Expect(err).ToNot(HaveOccurred())
		return string(data)
	}

	DescribeTable("rejects paths outside the storage folder",
		func(path string) {
			_, err := svc.Open(ctx, path)
			Expect(err).To(MatchError(ContainSubstring("invalid storage path")))
			_, err = svc.Create(ctx, path)
			Expect(err).To(MatchError(ContainSubstring("invalid storage path")))
		},
		Entry("empty", ""),
		Entry("parent", "../x"),
		Entry("parent after a folder", "a/../../x"),
		Entry("absolute", "/etc/passwd"),
	)

	It("refuses a symlink that leads outside the storage folder", func() {
		outside := filepath.Join(filepath.Dir(root), "secret")
		Expect(os.WriteFile(outside, []byte("secret"), 0600)).To(Succeed())
		Expect(os.Symlink(outside, filepath.Join(root, "link"))).To(Succeed())

		_, err := svc.Open(ctx, "link")
		Expect(err).To(HaveOccurred())
		_, err = svc.Create(ctx, "link")
		Expect(err).To(HaveOccurred())
		Expect(os.ReadFile(outside)).To(Equal([]byte("secret")))
	})

	It("opens a file for reading as a seekable *os.File", func() {
		Expect(os.WriteFile(filepath.Join(root, "f.txt"), []byte("hello"), 0600)).To(Succeed())
		rc, err := svc.Open(ctx, "f.txt")
		Expect(err).ToNot(HaveOccurred())
		Expect(rc).To(BeAssignableToTypeOf(&os.File{}))
		Expect(readFile(rc)).To(Equal("hello"))
	})

	It("creates missing parent folders and truncates an existing file", func() {
		for _, text := range []string{"a longer first version", "second"} {
			wc, err := svc.Create(ctx, "a/b/c.txt")
			Expect(err).ToNot(HaveOccurred())
			_, err = io.WriteString(wc, text)
			Expect(err).ToNot(HaveOccurred())
			Expect(wc.Close()).To(Succeed())
		}
		Expect(os.ReadFile(filepath.Join(root, "a", "b", "c.txt"))).To(Equal([]byte("second")))
	})

	It("keeps files opened before Close usable and opens nothing after", func() {
		Expect(os.WriteFile(filepath.Join(root, "f.txt"), []byte("still here"), 0600)).To(Succeed())
		rc, err := svc.Open(ctx, "f.txt")
		Expect(err).ToNot(HaveOccurred())

		Expect(svc.Close()).To(Succeed())
		_, err = svc.Open(ctx, "f.txt")
		Expect(err).To(MatchError(os.ErrClosed))
		Expect(readFile(rc)).To(Equal("still here"))
	})
})

var _ = Describe("Storage Host Function", Ordered, func() {
	var ctx context.Context

	const ID = "test-storage-plugin"

	var (
		manager *Manager
		tmpDir  string
	)

	BeforeAll(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "storage-test-*")
		Expect(err).ToNot(HaveOccurred())

		// Setup config
		DeferCleanup(conf.SnapshotConfig())
		conf.Update(func(c *conf.Options) {
			c.DataFolder = conf.NewDir(tmpDir)
			c.Plugins.Enabled = true
			c.Plugins.Folder = conf.NewDir(tmpDir)
			c.Plugins.AutoReload = false
		})

		pluginPaths := []string{ID, ID + "-2"}
		plugins := []model.Plugin{}

		for idx := range pluginPaths {
			path := pluginPaths[idx] + PackageExtension
			// Copy test plugin to temp dir
			srcPath := filepath.Join(testdataDir, path)
			destPath := filepath.Join(tmpDir, path)
			data, err := os.ReadFile(srcPath)
			Expect(err).ToNot(HaveOccurred())
			err = os.WriteFile(destPath, data, 0600)
			Expect(err).ToNot(HaveOccurred())

			// Pre-enable the plugin in the DB so it loads on startup
			// Compute SHA256 of the plugin file to match what syncPlugins will compute
			pluginPath := filepath.Join(tmpDir, path)
			wasmData, err := os.ReadFile(pluginPath)
			Expect(err).ToNot(HaveOccurred())
			hash := sha256.Sum256(wasmData)
			hashHex := hex.EncodeToString(hash[:])

			plugins = append(plugins, model.Plugin{
				ID:       pluginPaths[idx],
				Path:     pluginPath,
				SHA256:   hashHex,
				Enabled:  true,
				AllUsers: true, // Allow all users for test plugin
			})
		}

		// Create and configure manager
		manager = &Manager{
			plugins: make(map[string]*plugin),
			ds:      newTestDataStore(plugins...),
		}
		manager.SetSubsonicRouter(&fakeSubsonicRouter{})

		// Start the manager
		err = manager.Start(GinkgoT().Context())
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(func(ctx SpecContext) {
			_ = manager.Stop(ctx)
			_ = os.RemoveAll(tmpDir)
		})
	})

	var instance *extism.Plugin
	BeforeEach(func() {
		ctx = GinkgoT().Context()
		var err error
		manager.mu.RLock()
		plugin := manager.plugins[ID]
		manager.mu.RUnlock()
		Expect(plugin).ToNot(BeNil())

		ctx := GinkgoT().Context()
		instance, err = plugin.instance(ctx)
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(func() {
			instance.Close(ctx)
		})
	})

	Describe("Read", func() {
		BeforeAll(func() {
			path := filepath.Join(getHostStoragePath(ID), "real")
			err := os.WriteFile(path, []byte("1234"), 0600)
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func() {
				_ = os.Remove(path)
			})
		})

		It("should fail to read missing file", func() {
			exit, _, err := instance.CallWithContext(ctx, "call_read", []byte("missing"))
			Expect(exit).To(Equal(uint32(1)))
			Expect(err).To(HaveOccurred())
		})

		It("should read an existing file", func() {
			exit, output, err := instance.CallWithContext(ctx, "call_read", []byte("real"))
			Expect(exit).To(Equal(uint32(0)))
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal([]byte("1234")))
		})

		It("should not escape read", func() {
			path := filepath.Join(getHostStoragePath(ID), "..", "outside")
			err := os.WriteFile(path, []byte("outside"), 0600)
			Expect(err).ToNot(HaveOccurred())

			exit, _, err := instance.CallWithContext(ctx, "call_read", []byte("../outside"))
			Expect(exit).To(Equal(uint32(1)))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Write", func() {
		BeforeAll(func() {
			path := filepath.Join(getHostStoragePath(ID), "real")
			err := os.WriteFile(path, []byte("1234"), 0600)
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func() {
				_ = os.Remove(path)
			})
		})

		It("should fail to write to nested file", func() {
			exit, _, err := instance.CallWithContext(ctx, "call_write", []byte(`{"path":"nested/file","contents":"1234"}`))
			Expect(exit).To(Equal(uint32(1)))
			Expect(err).To(HaveOccurred())
		})

		It("should write to a file", func() {
			exit, _, err := instance.CallWithContext(ctx, "call_write", []byte(`{"path":"new","contents":"contents"}`))
			Expect(exit).To(Equal(uint32(0)))
			Expect(err).ToNot(HaveOccurred())

			data, err := os.ReadFile(filepath.Join(getHostStoragePath(ID), "new"))
			Expect(err).ToNot(HaveOccurred())
			Expect(data).To(Equal([]byte("contents")))

			exit, output, err := instance.CallWithContext(ctx, "call_read", []byte("new"))
			Expect(exit).To(Equal(uint32(0)))
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal([]byte("contents")))
		})

		It("should not escape writing a file", func() {
			path := filepath.Join(getHostStoragePath(ID), "..", "outside")
			err := os.WriteFile(path, []byte("outside"), 0600)
			Expect(err).ToNot(HaveOccurred())

			exit, _, err := instance.CallWithContext(ctx, "call_write", []byte(`{"path":"../new","contents":"contents"}`))
			Expect(exit).To(Equal(uint32(1)))
			Expect(err).To(HaveOccurred())
		})

		It("should have independent storage for multiple plugins", func() {
			manager.mu.RLock()
			plugin2 := manager.plugins[ID+"-2"]
			manager.mu.RUnlock()

			Expect(plugin2).ToNot(BeNil())

			secondInstance, err := plugin2.instance(GinkgoT().Context())
			Expect(err).ToNot(HaveOccurred())
			defer secondInstance.Close(GinkgoT().Context())

			instances := []*extism.Plugin{instance, secondInstance}
			names := []string{ID, ID + "-2"}

			for idx := range instances {
				exit, _, err := instances[idx].CallWithContext(ctx, "call_write", fmt.Appendf(nil, `{"path":"new","contents":"%s"}`, names[idx]))
				Expect(exit).To(Equal(uint32(0)))
				Expect(err).ToNot(HaveOccurred())
			}

			for idx := range names {
				data, err := os.ReadFile(filepath.Join(getHostStoragePath(names[idx]), "new"))
				Expect(err).ToNot(HaveOccurred())
				Expect(data).To(Equal([]byte(names[idx])))
			}

			for idx := range instances {
				exit, output, err := instances[idx].CallWithContext(ctx, "call_read", []byte("new"))
				Expect(exit).To(Equal(uint32(0)))
				Expect(err).ToNot(HaveOccurred())
				Expect(output).To(Equal([]byte(names[idx])))
			}
		})
	})
})
