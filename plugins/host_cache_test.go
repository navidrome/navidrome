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
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CacheService", func() {
	var service *cacheServiceImpl
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		service = newCacheService("test_plugin")
	})

	AfterEach(func() {
		if service != nil {
			service.Close()
		}
	})

	Describe("getTTL", func() {
		It("returns default TTL when seconds is 0", func() {
			ttl := service.getTTL(0)
			Expect(ttl).To(Equal(defaultCacheTTL))
		})

		It("returns default TTL when seconds is negative", func() {
			ttl := service.getTTL(-10)
			Expect(ttl).To(Equal(defaultCacheTTL))
		})

		It("returns correct duration when seconds is positive", func() {
			ttl := service.getTTL(60)
			Expect(ttl).To(Equal(time.Minute))
		})
	})

	Describe("Plugin Isolation", func() {
		It("isolates keys between plugins", func() {
			service1 := newCacheService("plugin1")
			defer service1.Close()
			service2 := newCacheService("plugin2")
			defer service2.Close()

			// Both plugins set same key
			err := service1.SetString(ctx, "shared", "value1", 0)
			Expect(err).ToNot(HaveOccurred())
			err = service2.SetString(ctx, "shared", "value2", 0)
			Expect(err).ToNot(HaveOccurred())

			// Each plugin should get their own value
			val1, exists1, err := service1.GetString(ctx, "shared")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists1).To(BeTrue())
			Expect(val1).To(Equal("value1"))

			val2, exists2, err := service2.GetString(ctx, "shared")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists2).To(BeTrue())
			Expect(val2).To(Equal("value2"))
		})
	})

	Describe("String Operations", func() {
		It("sets and gets a string value", func() {
			err := service.SetString(ctx, "string_key", "test_value", 300)
			Expect(err).ToNot(HaveOccurred())

			value, exists, err := service.GetString(ctx, "string_key")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal("test_value"))
		})

		It("returns not exists for missing key", func() {
			value, exists, err := service.GetString(ctx, "missing_key")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
			Expect(value).To(Equal(""))
		})
	})

	Describe("Integer Operations", func() {
		It("sets and gets an integer value", func() {
			err := service.SetInt(ctx, "int_key", 42, 300)
			Expect(err).ToNot(HaveOccurred())

			value, exists, err := service.GetInt(ctx, "int_key")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal(int64(42)))
		})

		It("returns not exists for missing key", func() {
			value, exists, err := service.GetInt(ctx, "missing_int_key")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
			Expect(value).To(Equal(int64(0)))
		})
	})

	Describe("Float Operations", func() {
		It("sets and gets a float value", func() {
			err := service.SetFloat(ctx, "float_key", 3.14, 300)
			Expect(err).ToNot(HaveOccurred())

			value, exists, err := service.GetFloat(ctx, "float_key")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal(3.14))
		})

		It("returns not exists for missing key", func() {
			value, exists, err := service.GetFloat(ctx, "missing_float_key")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
			Expect(value).To(Equal(float64(0)))
		})
	})

	Describe("Bytes Operations", func() {
		It("sets and gets a bytes value", func() {
			byteData := []byte("hello world")
			err := service.SetBytes(ctx, "bytes_key", byteData, 300)
			Expect(err).ToNot(HaveOccurred())

			value, exists, err := service.GetBytes(ctx, "bytes_key")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
			Expect(value).To(Equal(byteData))
		})

		It("returns not exists for missing key", func() {
			value, exists, err := service.GetBytes(ctx, "missing_bytes_key")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
			Expect(value).To(BeNil())
		})
	})

	Describe("Type mismatch handling", func() {
		It("returns not exists when type doesn't match the getter", func() {
			// Set string
			err := service.SetString(ctx, "mixed_key", "string value", 0)
			Expect(err).ToNot(HaveOccurred())

			// Try to get as int
			value, exists, err := service.GetInt(ctx, "mixed_key")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
			Expect(value).To(Equal(int64(0)))
		})

		It("returns not exists when getting string as float", func() {
			err := service.SetString(ctx, "str_as_float", "not a float", 0)
			Expect(err).ToNot(HaveOccurred())

			value, exists, err := service.GetFloat(ctx, "str_as_float")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
			Expect(value).To(Equal(float64(0)))
		})

		It("returns not exists when getting int as bytes", func() {
			err := service.SetInt(ctx, "int_as_bytes", 123, 0)
			Expect(err).ToNot(HaveOccurred())

			value, exists, err := service.GetBytes(ctx, "int_as_bytes")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
			Expect(value).To(BeNil())
		})
	})

	Describe("Has Operation", func() {
		It("returns true for existing key", func() {
			err := service.SetString(ctx, "existing_key", "exists", 0)
			Expect(err).ToNot(HaveOccurred())

			exists, err := service.Has(ctx, "existing_key")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("returns false for non-existing key", func() {
			exists, err := service.Has(ctx, "non_existing_key")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
	})

	Describe("Remove Operation", func() {
		It("removes a value from the cache", func() {
			// Set a value
			err := service.SetString(ctx, "remove_key", "to be removed", 0)
			Expect(err).ToNot(HaveOccurred())

			// Verify it exists
			exists, err := service.Has(ctx, "remove_key")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())

			// Remove it
			err = service.Remove(ctx, "remove_key")
			Expect(err).ToNot(HaveOccurred())

			// Verify it's gone
			exists, err = service.Has(ctx, "remove_key")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("does not error when removing non-existing key", func() {
			err := service.Remove(ctx, "never_existed")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("TTL Behavior", func() {
		It("uses default TTL when 0 is provided", func() {
			err := service.SetString(ctx, "default_ttl", "value", 0)
			Expect(err).ToNot(HaveOccurred())

			// Value should exist immediately
			exists, err := service.Has(ctx, "default_ttl")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("uses custom TTL when provided", func() {
			err := service.SetString(ctx, "custom_ttl", "value", 300)
			Expect(err).ToNot(HaveOccurred())

			// Value should exist immediately
			exists, err := service.Has(ctx, "custom_ttl")
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})
	})

	Describe("Close", func() {
		It("removes all cache entries for the plugin", func() {
			// Use a dedicated service for this test
			closeService := newCacheService("close_test_plugin")

			// Set multiple values
			err := closeService.SetString(ctx, "key1", "value1", 0)
			Expect(err).ToNot(HaveOccurred())
			err = closeService.SetInt(ctx, "key2", 42, 0)
			Expect(err).ToNot(HaveOccurred())
			err = closeService.SetFloat(ctx, "key3", 3.14, 0)
			Expect(err).ToNot(HaveOccurred())

			// Verify they exist
			exists, _ := closeService.Has(ctx, "key1")
			Expect(exists).To(BeTrue())
			exists, _ = closeService.Has(ctx, "key2")
			Expect(exists).To(BeTrue())
			exists, _ = closeService.Has(ctx, "key3")
			Expect(exists).To(BeTrue())

			// Close the service
			err = closeService.Close()
			Expect(err).ToNot(HaveOccurred())

			// All entries should be gone
			exists, _ = closeService.Has(ctx, "key1")
			Expect(exists).To(BeFalse())
			exists, _ = closeService.Has(ctx, "key2")
			Expect(exists).To(BeFalse())
			exists, _ = closeService.Has(ctx, "key3")
			Expect(exists).To(BeFalse())
		})

		It("does not affect other plugins' cache entries", func() {
			// Create two services for different plugins
			service1 := newCacheService("plugin_close_test1")
			service2 := newCacheService("plugin_close_test2")
			defer service2.Close()

			// Set values for both plugins
			err := service1.SetString(ctx, "key", "value1", 0)
			Expect(err).ToNot(HaveOccurred())
			err = service2.SetString(ctx, "key", "value2", 0)
			Expect(err).ToNot(HaveOccurred())

			// Close only service1
			err = service1.Close()
			Expect(err).ToNot(HaveOccurred())

			// service1's key should be gone
			exists, _ := service1.Has(ctx, "key")
			Expect(exists).To(BeFalse())

			// service2's key should still exist
			exists, _ = service2.Has(ctx, "key")
			Expect(exists).To(BeTrue())
		})
	})
})

var _ = Describe("CacheService Integration", Ordered, func() {
	var (
		manager *Manager
		tmpDir  string
	)

	BeforeAll(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "cache-test-*")
		Expect(err).ToNot(HaveOccurred())

		// Copy the test-cache-plugin
		srcPath := filepath.Join(testdataDir, "test-cache-plugin"+PackageExtension)
		destPath := filepath.Join(tmpDir, "test-cache-plugin"+PackageExtension)
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

		// Setup mock DataStore with pre-enabled plugin
		mockPluginRepo := tests.CreateMockPluginRepo()
		mockPluginRepo.Permitted = true
		mockPluginRepo.SetData(model.Plugins{{
			ID:      "test-cache-plugin",
			Path:    destPath,
			SHA256:  hashHex,
			Enabled: true,
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
		It("should load plugin with cache permission", func() {
			manager.mu.RLock()
			p, ok := manager.plugins["test-cache-plugin"]
			manager.mu.RUnlock()
			Expect(ok).To(BeTrue())
			Expect(p.manifest.Permissions).ToNot(BeNil())
			Expect(p.manifest.Permissions.Cache).ToNot(BeNil())
		})
	})

	Describe("Cache Operations via Plugin", func() {
		type testCacheInput struct {
			Operation  string  `json:"operation"`
			Key        string  `json:"key"`
			StringVal  string  `json:"string_val,omitempty"`
			IntVal     int64   `json:"int_val,omitempty"`
			FloatVal   float64 `json:"float_val,omitempty"`
			BytesVal   []byte  `json:"bytes_val,omitempty"`
			TTLSeconds int64   `json:"ttl_seconds,omitempty"`
		}
		type testCacheOutput struct {
			StringVal string  `json:"string_val,omitempty"`
			IntVal    int64   `json:"int_val,omitempty"`
			FloatVal  float64 `json:"float_val,omitempty"`
			BytesVal  []byte  `json:"bytes_val,omitempty"`
			Exists    bool    `json:"exists,omitempty"`
			Error     *string `json:"error,omitempty"`
		}

		callTestCache := func(ctx context.Context, input testCacheInput) (*testCacheOutput, error) {
			manager.mu.RLock()
			p := manager.plugins["test-cache-plugin"]
			manager.mu.RUnlock()

			instance, err := p.instance(ctx)
			if err != nil {
				return nil, err
			}
			defer instance.Close(ctx)

			inputBytes, _ := json.Marshal(input)
			_, outputBytes, err := instance.Call("nd_test_cache", inputBytes)
			if err != nil {
				return nil, err
			}

			var output testCacheOutput
			if err := json.Unmarshal(outputBytes, &output); err != nil {
				return nil, err
			}
			if output.Error != nil {
				return nil, errors.New(*output.Error)
			}
			return &output, nil
		}

		It("should set and get string value", func() {
			ctx := GinkgoT().Context()

			// Set string
			_, err := callTestCache(ctx, testCacheInput{
				Operation:  "set_string",
				Key:        "test_string",
				StringVal:  "hello world",
				TTLSeconds: 300,
			})
			Expect(err).ToNot(HaveOccurred())

			// Get string
			output, err := callTestCache(ctx, testCacheInput{
				Operation: "get_string",
				Key:       "test_string",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Exists).To(BeTrue())
			Expect(output.StringVal).To(Equal("hello world"))
		})

		It("should set and get integer value", func() {
			ctx := GinkgoT().Context()

			// Set int
			_, err := callTestCache(ctx, testCacheInput{
				Operation:  "set_int",
				Key:        "test_int",
				IntVal:     42,
				TTLSeconds: 300,
			})
			Expect(err).ToNot(HaveOccurred())

			// Get int
			output, err := callTestCache(ctx, testCacheInput{
				Operation: "get_int",
				Key:       "test_int",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Exists).To(BeTrue())
			Expect(output.IntVal).To(Equal(int64(42)))
		})

		It("should set and get float value", func() {
			ctx := GinkgoT().Context()

			// Set float
			_, err := callTestCache(ctx, testCacheInput{
				Operation:  "set_float",
				Key:        "test_float",
				FloatVal:   3.14159,
				TTLSeconds: 300,
			})
			Expect(err).ToNot(HaveOccurred())

			// Get float
			output, err := callTestCache(ctx, testCacheInput{
				Operation: "get_float",
				Key:       "test_float",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Exists).To(BeTrue())
			Expect(output.FloatVal).To(Equal(3.14159))
		})

		It("should set and get bytes value", func() {
			ctx := GinkgoT().Context()
			testBytes := []byte{0x01, 0x02, 0x03, 0x04}

			// Set bytes
			_, err := callTestCache(ctx, testCacheInput{
				Operation:  "set_bytes",
				Key:        "test_bytes",
				BytesVal:   testBytes,
				TTLSeconds: 300,
			})
			Expect(err).ToNot(HaveOccurred())

			// Get bytes
			output, err := callTestCache(ctx, testCacheInput{
				Operation: "get_bytes",
				Key:       "test_bytes",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Exists).To(BeTrue())
			Expect(output.BytesVal).To(Equal(testBytes))
		})

		It("should handle binary data with null bytes through WASM", func() {
			ctx := GinkgoT().Context()

			// Binary data with null bytes, high bytes, and other edge cases
			binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0x00, 0x80, 0x7F}

			// Set binary bytes
			_, err := callTestCache(ctx, testCacheInput{
				Operation:  "set_bytes",
				Key:        "binary_test",
				BytesVal:   binaryData,
				TTLSeconds: 300,
			})
			Expect(err).ToNot(HaveOccurred())

			// Get binary bytes and verify exact match
			output, err := callTestCache(ctx, testCacheInput{
				Operation: "get_bytes",
				Key:       "binary_test",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Exists).To(BeTrue())
			Expect(output.BytesVal).To(Equal(binaryData))
		})

		It("should check if key exists", func() {
			ctx := GinkgoT().Context()

			// Set a value
			_, err := callTestCache(ctx, testCacheInput{
				Operation: "set_string",
				Key:       "exists_test",
				StringVal: "value",
			})
			Expect(err).ToNot(HaveOccurred())

			// Check has
			output, err := callTestCache(ctx, testCacheInput{
				Operation: "has",
				Key:       "exists_test",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Exists).To(BeTrue())

			// Check non-existent
			output, err = callTestCache(ctx, testCacheInput{
				Operation: "has",
				Key:       "nonexistent",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Exists).To(BeFalse())
		})

		It("should remove a key", func() {
			ctx := GinkgoT().Context()

			// Set a value
			_, err := callTestCache(ctx, testCacheInput{
				Operation: "set_string",
				Key:       "remove_test",
				StringVal: "value",
			})
			Expect(err).ToNot(HaveOccurred())

			// Remove it
			_, err = callTestCache(ctx, testCacheInput{
				Operation: "remove",
				Key:       "remove_test",
			})
			Expect(err).ToNot(HaveOccurred())

			// Verify it's gone
			output, err := callTestCache(ctx, testCacheInput{
				Operation: "has",
				Key:       "remove_test",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(output.Exists).To(BeFalse())
		})
	})
})
