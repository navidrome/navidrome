package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	extism "github.com/extism/go-sdk"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("utility functions", Ordered, func() {
	var tmpDir string

	BeforeAll(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "storage-test-*")
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(configtest.SetupConfig())
		conf.Server.DataFolder = conf.NewDir(tmpDir)

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
			Expect(impl.GetStoragePath(context.TODO())).To(Equal("/storage"))
		})
	})

	Describe("netStorageService", func() {
		It("should create the directory on init", func() {
			svc, err := newStorageService("plugin-name")
			Expect(err).ToNot(HaveOccurred())
			Expect(svc).ToNot(BeNil())

			dataDir := filepath.Join(tmpDir, "plugins", "plugin-name", "storage")
			Expect(dataDir).To(BeADirectory())
		})
	})
})

var _ = Describe("Storage Host Function", Ordered, func() {
	const ID = "test-storage-plugin"

	var (
		manager *Manager
		tmpDir  string
	)

	BeforeAll(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "storage-test-*")
		Expect(err).ToNot(HaveOccurred())

		// Setup mock router and data store
		router := &fakeSubsonicRouter{}
		userRepo := tests.CreateMockUserRepo()
		dataStore := &tests.MockDataStore{MockedUser: userRepo}

		// Create and configure manager
		manager = &Manager{
			plugins: make(map[string]*plugin),
			ds:      dataStore,
		}
		manager.SetSubsonicRouter(router)

		mockPluginRepo := dataStore.Plugin(GinkgoT().Context()).(*tests.MockPluginRepo)
		mockPluginRepo.Permitted = true

		// Setup config
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DataFolder = conf.NewDir(tmpDir)
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = conf.NewDir(tmpDir)
		conf.Server.Plugins.AutoReload = false

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

			// Pre-enable the plugin in the mock repo so it loads on startup
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

		mockPluginRepo.SetData(plugins)

		// Start the manager
		err = manager.Start(GinkgoT().Context())
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(func() {
			_ = manager.Stop()
			_ = os.RemoveAll(tmpDir)
		})
	})

	var instance *extism.Plugin
	BeforeEach(func() {
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
			exit, _, err := instance.Call("call_read", []byte("missing"))
			Expect(exit).To(Equal(uint32(1)))
			Expect(err).To(HaveOccurred())
		})

		It("should read an existing file", func() {
			exit, output, err := instance.Call("call_read", []byte("real"))
			Expect(exit).To(Equal(uint32(0)))
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal([]byte("1234")))
		})

		It("should not escape read", func() {
			path := filepath.Join(getHostStoragePath(ID), "..", "outside")
			err := os.WriteFile(path, []byte("outside"), 0600)
			Expect(err).ToNot(HaveOccurred())

			exit, _, err := instance.Call("call_read", []byte("../outside"))
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
			exit, _, err := instance.Call("call_write", []byte(`{"path":"nested/file","contents":"1234"}`))
			Expect(exit).To(Equal(uint32(1)))
			Expect(err).To(HaveOccurred())
		})

		It("should write to a file", func() {
			exit, _, err := instance.Call("call_write", []byte(`{"path":"new","contents":"contents"}`))
			Expect(exit).To(Equal(uint32(0)))
			Expect(err).ToNot(HaveOccurred())

			data, err := os.ReadFile(filepath.Join(getHostStoragePath(ID), "new"))
			Expect(err).ToNot(HaveOccurred())
			Expect(data).To(Equal([]byte("contents")))

			exit, output, err := instance.Call("call_read", []byte("new"))
			Expect(exit).To(Equal(uint32(0)))
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal([]byte("contents")))
		})

		It("should not escape writing a file", func() {
			path := filepath.Join(getHostStoragePath(ID), "..", "outside")
			err := os.WriteFile(path, []byte("outside"), 0600)
			Expect(err).ToNot(HaveOccurred())

			exit, _, err := instance.Call("call_write", []byte(`{"path":"../new","contents":"contents"}`))
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
				exit, _, err := instances[idx].Call("call_write", fmt.Appendf(nil, `{"path":"new","contents":"%s"}`, names[idx]))
				Expect(exit).To(Equal(uint32(0)))
				Expect(err).ToNot(HaveOccurred())
			}

			for idx := range names {
				data, err := os.ReadFile(filepath.Join(getHostStoragePath(names[idx]), "new"))
				Expect(err).ToNot(HaveOccurred())
				Expect(data).To(Equal([]byte(names[idx])))
			}

			for idx := range instances {
				exit, output, err := instances[idx].Call("call_read", []byte("new"))
				Expect(exit).To(Equal(uint32(0)))
				Expect(err).ToNot(HaveOccurred())
				Expect(output).To(Equal([]byte(names[idx])))
			}
		})
	})
})
