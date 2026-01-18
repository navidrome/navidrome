//go:build !windows

package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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

var _ = Describe("ConfigService", func() {
	var service *configServiceImpl
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("newConfigService", func() {
		It("creates service with provided config", func() {
			config := map[string]string{"key1": "value1", "key2": "value2"}
			service = newConfigService("test_plugin", config)
			Expect(service.pluginName).To(Equal("test_plugin"))
			Expect(service.config).To(Equal(config))
		})

		It("creates service with empty config when nil", func() {
			service = newConfigService("test_plugin", nil)
			Expect(service.config).ToNot(BeNil())
			Expect(service.config).To(BeEmpty())
		})
	})

	Describe("Get", func() {
		BeforeEach(func() {
			service = newConfigService("test_plugin", map[string]string{
				"api_key":    "secret123",
				"debug_mode": "true",
				"max_items":  "100",
			})
		})

		It("returns value for existing key", func() {
			value, exists := service.Get(ctx, "api_key")
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal("secret123"))
		})

		It("returns not exists for missing key", func() {
			value, exists := service.Get(ctx, "missing_key")
			Expect(exists).To(BeFalse())
			Expect(value).To(Equal(""))
		})
	})

	Describe("GetInt", func() {
		BeforeEach(func() {
			service = newConfigService("test_plugin", map[string]string{
				"max_items":    "100",
				"timeout":      "30",
				"negative":     "-50",
				"not_a_number": "abc",
				"float":        "3.14",
			})
		})

		It("returns integer for valid numeric value", func() {
			value, exists := service.GetInt(ctx, "max_items")
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal(int64(100)))
		})

		It("returns negative integer", func() {
			value, exists := service.GetInt(ctx, "negative")
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal(int64(-50)))
		})

		It("returns not exists for non-numeric value", func() {
			value, exists := service.GetInt(ctx, "not_a_number")
			Expect(exists).To(BeFalse())
			Expect(value).To(Equal(int64(0)))
		})

		It("returns not exists for float value", func() {
			value, exists := service.GetInt(ctx, "float")
			Expect(exists).To(BeFalse())
			Expect(value).To(Equal(int64(0)))
		})

		It("returns not exists for missing key", func() {
			value, exists := service.GetInt(ctx, "missing_key")
			Expect(exists).To(BeFalse())
			Expect(value).To(Equal(int64(0)))
		})
	})

	Describe("Keys", func() {
		BeforeEach(func() {
			service = newConfigService("test_plugin", map[string]string{
				"zebra":        "z",
				"apple":        "a",
				"banana":       "b",
				"user_alice":   "token1",
				"user_bob":     "token2",
				"user_charlie": "token3",
			})
		})

		It("returns all keys in sorted order when prefix is empty", func() {
			keys := service.Keys(ctx, "")
			Expect(keys).To(Equal([]string{"apple", "banana", "user_alice", "user_bob", "user_charlie", "zebra"}))
		})

		It("returns only keys matching prefix", func() {
			keys := service.Keys(ctx, "user_")
			Expect(keys).To(Equal([]string{"user_alice", "user_bob", "user_charlie"}))
		})

		It("returns empty slice when no keys match prefix", func() {
			keys := service.Keys(ctx, "nonexistent_")
			Expect(keys).To(BeEmpty())
		})

		It("returns empty slice for empty config", func() {
			service = newConfigService("test_plugin", nil)
			keys := service.Keys(ctx, "")
			Expect(keys).To(BeEmpty())
		})
	})
})

var _ = Describe("ConfigService Integration", Ordered, func() {
	var (
		manager *Manager
		tmpDir  string
	)

	BeforeAll(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "config-test-*")
		Expect(err).ToNot(HaveOccurred())

		// Copy the test-config plugin
		srcPath := filepath.Join(testdataDir, "test-config"+PackageExtension)
		destPath := filepath.Join(tmpDir, "test-config"+PackageExtension)
		data, err := os.ReadFile(srcPath)
		Expect(err).ToNot(HaveOccurred())
		err = os.WriteFile(destPath, data, 0600)
		Expect(err).ToNot(HaveOccurred())

		// Compute SHA256 for the plugin
		hash := sha256.Sum256(data)
		hashHex := hex.EncodeToString(hash[:])

		// Setup config
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = tmpDir
		conf.Server.Plugins.AutoReload = false
		conf.Server.CacheFolder = filepath.Join(tmpDir, "cache")

		// Setup mock DataStore with pre-enabled plugin and config
		mockPluginRepo := tests.CreateMockPluginRepo()
		mockPluginRepo.Permitted = true
		mockPluginRepo.SetData(model.Plugins{{
			ID:      "test-config",
			Path:    destPath,
			SHA256:  hashHex,
			Enabled: true,
			Config:  `{"api_key":"test_secret","max_retries":"5","timeout":"30"}`,
		}})
		dataStore := &tests.MockDataStore{MockedPlugin: mockPluginRepo}

		// Create and start manager
		manager = &Manager{
			plugins:        make(map[string]*plugin),
			ds:             dataStore,
			subsonicRouter: http.NotFoundHandler(),
		}
		err = manager.Start(GinkgoT().Context())
		Expect(err).ToNot(HaveOccurred())

		DeferCleanup(func() {
			_ = manager.Stop()
			_ = os.RemoveAll(tmpDir)
		})
	})

	Describe("Plugin Loading", func() {
		It("should load plugin without config permission", func() {
			manager.mu.RLock()
			p, ok := manager.plugins["test-config"]
			manager.mu.RUnlock()
			Expect(ok).To(BeTrue())
			// Config service doesn't require permission, so Permissions can be nil
			// Just verify the plugin loaded
			Expect(p.manifest.Name).To(Equal("Test Config Plugin"))
		})
	})

	Describe("Config Operations via Plugin", func() {
		type testConfigInput struct {
			Operation string `json:"operation"`
			Key       string `json:"key,omitempty"`
			Prefix    string `json:"prefix,omitempty"`
		}
		type testConfigOutput struct {
			StringVal string   `json:"string_val,omitempty"`
			IntVal    int64    `json:"int_val,omitempty"`
			Keys      []string `json:"keys,omitempty"`
			Exists    bool     `json:"exists,omitempty"`
			Error     *string  `json:"error,omitempty"`
		}

		// Helper to call test plugin's exported function
		callTestConfig := func(ctx context.Context, input testConfigInput) (*testConfigOutput, error) {
			manager.mu.RLock()
			p := manager.plugins["test-config"]
			manager.mu.RUnlock()

			instance, err := p.instance(ctx)
			if err != nil {
				return nil, err
			}
			defer instance.Close(ctx)

			inputBytes, _ := json.Marshal(input)
			_, outputBytes, err := instance.Call("nd_test_config", inputBytes)
			if err != nil {
				return nil, err
			}

			var output testConfigOutput
			if err := json.Unmarshal(outputBytes, &output); err != nil {
				return nil, err
			}
			if output.Error != nil {
				return nil, errors.New(*output.Error)
			}
			return &output, nil
		}

		It("should get string value", func() {
			output, err := callTestConfig(GinkgoT().Context(), testConfigInput{
				Operation: "get",
				Key:       "api_key",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.StringVal).To(Equal("test_secret"))
			Expect(output.Exists).To(BeTrue())
		})

		It("should return not exists for missing key", func() {
			output, err := callTestConfig(GinkgoT().Context(), testConfigInput{
				Operation: "get",
				Key:       "nonexistent",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Exists).To(BeFalse())
		})

		It("should get integer value", func() {
			output, err := callTestConfig(GinkgoT().Context(), testConfigInput{
				Operation: "get_int",
				Key:       "max_retries",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.IntVal).To(Equal(int64(5)))
			Expect(output.Exists).To(BeTrue())
		})

		It("should return not exists for non-integer value", func() {
			output, err := callTestConfig(GinkgoT().Context(), testConfigInput{
				Operation: "get_int",
				Key:       "api_key", // This is a string, not an integer
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Exists).To(BeFalse())
		})

		It("should list all config keys with empty prefix", func() {
			output, err := callTestConfig(GinkgoT().Context(), testConfigInput{
				Operation: "list",
				Prefix:    "",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Keys).To(ConsistOf("api_key", "max_retries", "timeout"))
		})

		It("should list config keys with prefix filter", func() {
			output, err := callTestConfig(GinkgoT().Context(), testConfigInput{
				Operation: "list",
				Prefix:    "max",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Keys).To(ConsistOf("max_retries"))
		})
	})
})
