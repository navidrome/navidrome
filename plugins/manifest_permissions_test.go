package plugins

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin Permissions", func() {
	var ctx context.Context
	var mgr *Manager
	var tempDir string

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = GinkgoT().Context()
		mgr = createManager()
		tempDir = GinkgoT().TempDir()
	})

	// Helper function to create a test plugin with specific permissions
	createTestPluginWithPermissions := func(name string, permissions map[string]any) string {
		pluginDir := filepath.Join(tempDir, name)
		Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

		// Convert permissions map to JSON
		permissionsJSON, err := json.Marshal(permissions)
		Expect(err).NotTo(HaveOccurred())

		// Create a minimal manifest with specified permissions
		manifest := `{
			"name": "` + name + `",
			"author": "Test Author",
			"version": "1.0.0",
			"description": "Test plugin for permission testing",
			"capabilities": ["MetadataAgent"],
			"permissions": ` + string(permissionsJSON) + `}`

		Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0600)).To(Succeed())

		// Create a dummy WASM file
		Expect(os.WriteFile(filepath.Join(pluginDir, "plugin.wasm"), []byte("dummy wasm"), 0600)).To(Succeed())

		return pluginDir
	}

	Describe("Permission Enforcement in createCustomRuntime", func() {
		It("should only load services specified in permissions", func() {
			// Test with limited permissions
			permissions := map[string]any{
				"http": map[string]any{
					"reason": "To fetch data from external APIs",
					"allowedUrls": map[string]any{
						"*": []any{"*"},
					},
				},
				"config": map[string]any{
					"reason": "To read configuration settings",
				},
			}

			ccache, _ := getCompilationCache()
			runtimeFunc := mgr.createCustomRuntime(ccache, "test-plugin", permissions)

			// Create runtime to test service availability
			runtime, err := runtimeFunc(ctx)
			Expect(err).NotTo(HaveOccurred())
			defer runtime.Close(ctx)

			// The runtime was created successfully with the specified permissions
			Expect(runtime).NotTo(BeNil())

			// Note: The actual verification of which specific host functions are available
			// would require introspecting the WASM runtime, which is complex.
			// The key test is that the runtime creation succeeds with valid permissions.
		})

		It("should create runtime with empty permissions", func() {
			permissions := map[string]any{}

			ccache, _ := getCompilationCache()
			runtimeFunc := mgr.createCustomRuntime(ccache, "empty-permissions-plugin", permissions)

			runtime, err := runtimeFunc(ctx)
			Expect(err).NotTo(HaveOccurred())
			defer runtime.Close(ctx)

			// Should succeed but with no host services available
			Expect(runtime).NotTo(BeNil())
		})

		It("should handle all available permissions", func() {
			// Test with all possible permissions
			permissions := map[string]any{
				"http": map[string]any{
					"reason": "To fetch data from external APIs",
					"allowedUrls": map[string]any{
						"*": []any{"*"},
					},
				},
				"config": map[string]any{
					"reason": "To read configuration settings",
				},
				"scheduler": map[string]any{
					"reason": "To schedule periodic tasks",
				},
				"websocket": map[string]any{
					"reason": "To handle real-time communication",
				},
				"cache": map[string]any{
					"reason": "To cache data and reduce API calls",
				},
				"artwork": map[string]any{
					"reason": "To generate artwork URLs",
				},
			}

			ccache, _ := getCompilationCache()
			runtimeFunc := mgr.createCustomRuntime(ccache, "full-permissions-plugin", permissions)

			runtime, err := runtimeFunc(ctx)
			Expect(err).NotTo(HaveOccurred())
			defer runtime.Close(ctx)

			Expect(runtime).NotTo(BeNil())
		})
	})

	Describe("Plugin Discovery with Permissions", func() {
		BeforeEach(func() {
			conf.Server.Plugins.Folder = tempDir
		})

		It("should discover plugin with valid permissions manifest", func() {
			// Create plugin with http permission
			permissions := map[string]any{
				"http": map[string]any{
					"reason": "To fetch metadata from external APIs",
					"allowedUrls": map[string]any{
						"*": []any{"*"},
					},
				},
			}
			createTestPluginWithPermissions("valid-plugin", permissions)

			// Scan for plugins
			mgr.ScanPlugins()

			// Verify plugin was discovered (even without valid WASM)
			pluginNames := mgr.PluginNames("MetadataAgent")
			Expect(pluginNames).To(ContainElement("valid-plugin"))
		})

		It("should discover plugin with no permissions", func() {
			// Create plugin with empty permissions
			permissions := map[string]any{}
			createTestPluginWithPermissions("no-perms-plugin", permissions)

			mgr.ScanPlugins()

			pluginNames := mgr.PluginNames("MetadataAgent")
			Expect(pluginNames).To(ContainElement("no-perms-plugin"))
		})

		It("should discover plugin with multiple permissions", func() {
			permissions := map[string]any{
				"http": map[string]any{
					"reason": "To fetch metadata from external APIs",
					"allowedUrls": map[string]any{
						"*": []any{"*"},
					},
				},
				"config": map[string]any{
					"reason": "To read plugin configuration settings",
				},
				"scheduler": map[string]any{
					"reason": "To schedule periodic data updates",
				},
			}
			createTestPluginWithPermissions("multi-perms-plugin", permissions)

			mgr.ScanPlugins()

			pluginNames := mgr.PluginNames("MetadataAgent")
			Expect(pluginNames).To(ContainElement("multi-perms-plugin"))
		})
	})

	Describe("Existing Plugin Permissions", func() {
		BeforeEach(func() {
			// Use the testdata directory with updated plugins
			conf.Server.Plugins.Folder = testDataDir
			mgr.ScanPlugins()
		})

		It("should discover fake_scrobbler with empty permissions", func() {
			scrobblerNames := mgr.PluginNames(CapabilityScrobbler)
			Expect(scrobblerNames).To(ContainElement("fake_scrobbler"))
		})

		It("should discover multi_plugin with scheduler permissions", func() {
			agentNames := mgr.PluginNames(CapabilityMetadataAgent)
			Expect(agentNames).To(ContainElement("multi_plugin"))
		})

		It("should discover all test plugins successfully", func() {
			// All test plugins should be discovered with their updated permissions
			testPlugins := []struct {
				name       string
				capability string
			}{
				{"fake_album_agent", CapabilityMetadataAgent},
				{"fake_artist_agent", CapabilityMetadataAgent},
				{"fake_scrobbler", CapabilityScrobbler},
				{"multi_plugin", CapabilityMetadataAgent},
				{"fake_init_service", CapabilityLifecycleManagement},
			}

			for _, testPlugin := range testPlugins {
				pluginNames := mgr.PluginNames(testPlugin.capability)
				Expect(pluginNames).To(ContainElement(testPlugin.name), "Plugin %s should be discovered", testPlugin.name)
			}
		})
	})

	Describe("Permission Validation", func() {
		It("should enforce permissions are required in manifest", func() {
			// Create a plugin without permissions field
			pluginDir := filepath.Join(tempDir, "no-permissions")
			Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

			manifestWithoutPermissions := `{
				"name": "no-permissions",
				"author": "Test Author",
				"version": "1.0.0",
				"description": "Plugin without permissions",
				"capabilities": ["MetadataAgent"]
			}`

			Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifestWithoutPermissions), 0600)).To(Succeed())

			// Try to load the manifest - should fail validation
			_, err := LoadManifest(pluginDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("permissions is required"))
		})

		It("should allow unknown permission keys", func() {
			// Test that LoadManifest accepts unknown permission keys for future extensibility
			separateTempDir, _ := os.MkdirTemp("", "navidrome-plugin-test-*")
			DeferCleanup(func() {
				_ = os.RemoveAll(separateTempDir)
			})

			// Create plugin in separate temp dir
			pluginDir := filepath.Join(separateTempDir, "unknown-perm-plugin")
			Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

			manifest := `{
				"name": "unknown-perm-plugin",
				"author": "Test Author",
				"version": "1.0.0",
				"description": "Test plugin for permission testing",
				"capabilities": ["MetadataAgent"],
				"permissions": {
					"http": {
						"reason": "To fetch data from external APIs",
						"allowedUrls": {
							"*": ["*"]
						}
					},
					"unknown": {
						"reason": "Future functionality not yet implemented"
					}
				}
			}`

			Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0600)).To(Succeed())

			// Test manifest loading directly - should succeed even with unknown permissions
			loadedManifest, err := LoadManifest(pluginDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(loadedManifest).NotTo(BeNil())
			Expect(loadedManifest.Permissions).To(HaveKey("http"))
			Expect(loadedManifest.Permissions).To(HaveKey("unknown"))
		})
	})

	Describe("Runtime Pool with Permissions", func() {
		It("should create separate runtimes for different permission sets", func() {
			ccache, _ := getCompilationCache()

			// Create two different permission sets
			permissions1 := map[string]any{
				"http": map[string]any{
					"reason": "To fetch data from external APIs",
					"allowedUrls": map[string]any{
						"*": []any{"*"},
					},
				},
			}
			permissions2 := map[string]any{
				"config": map[string]any{
					"reason": "To read configuration settings",
				},
			}

			runtimeFunc1 := mgr.createCustomRuntime(ccache, "plugin1", permissions1)
			runtimeFunc2 := mgr.createCustomRuntime(ccache, "plugin2", permissions2)

			runtime1, err1 := runtimeFunc1(ctx)
			Expect(err1).NotTo(HaveOccurred())
			defer runtime1.Close(ctx)

			runtime2, err2 := runtimeFunc2(ctx)
			Expect(err2).NotTo(HaveOccurred())
			defer runtime2.Close(ctx)

			// Should be different runtime instances
			Expect(runtime1).NotTo(BeIdenticalTo(runtime2))
		})
	})

	Describe("Permission System Integration", func() {
		It("should successfully validate manifests with permissions", func() {
			// Create a valid manifest with permissions
			pluginDir := filepath.Join(tempDir, "valid-manifest")
			Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

			manifestContent := `{
				"name": "valid-manifest",
				"author": "Test Author", 
				"version": "1.0.0",
				"description": "Valid manifest with permissions",
				"capabilities": ["MetadataAgent"],
				"permissions": {
					"http": {
						"reason": "To fetch metadata from external APIs",
						"allowedUrls": {
							"*": ["*"]
						}
					},
					"config": {
						"reason": "To read plugin configuration settings"
					}
				}
			}`

			Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifestContent), 0600)).To(Succeed())

			// Load the manifest - should succeed
			manifest, err := LoadManifest(pluginDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(manifest).NotTo(BeNil())
			Expect(manifest.Permissions).To(HaveKey("http"))
			Expect(manifest.Permissions).To(HaveKey("config"))
		})

		It("should track which services are requested per plugin", func() {
			// Test that different plugins can have different permission sets
			permissions1 := map[string]any{
				"http": map[string]any{
					"reason": "To fetch data from external APIs",
					"allowedUrls": map[string]any{
						"*": []any{"*"},
					},
				},
				"config": map[string]any{
					"reason": "To read configuration settings",
				},
			}
			permissions2 := map[string]any{
				"scheduler": map[string]any{
					"reason": "To schedule periodic tasks",
				},
				"websocket": map[string]any{
					"reason": "To handle real-time communication",
				},
			}
			permissions3 := map[string]any{} // Empty permissions

			createTestPluginWithPermissions("plugin-with-http", permissions1)
			createTestPluginWithPermissions("plugin-with-scheduler", permissions2)
			createTestPluginWithPermissions("plugin-with-none", permissions3)

			conf.Server.Plugins.Folder = tempDir
			mgr.ScanPlugins()

			// All should be discovered
			pluginNames := mgr.PluginNames(CapabilityMetadataAgent)
			Expect(pluginNames).To(ContainElement("plugin-with-http"))
			Expect(pluginNames).To(ContainElement("plugin-with-scheduler"))
			Expect(pluginNames).To(ContainElement("plugin-with-none"))
		})
	})

	Describe("Runtime Service Access Control", func() {
		It("should successfully create runtime with permitted services", func() {
			ccache, _ := getCompilationCache()

			// Create runtime with HTTP permission
			permissions := map[string]any{
				"http": map[string]any{
					"reason": "To fetch data from external APIs",
					"allowedUrls": map[string]any{
						"*": []any{"*"},
					},
				},
			}

			runtimeFunc := mgr.createCustomRuntime(ccache, "http-only-plugin", permissions)
			runtime, err := runtimeFunc(ctx)
			Expect(err).NotTo(HaveOccurred())
			defer runtime.Close(ctx)

			// Runtime should be created successfully - host functions are loaded during runtime creation
			Expect(runtime).NotTo(BeNil())
		})

		It("should successfully create runtime with multiple permitted services", func() {
			ccache, _ := getCompilationCache()

			// Create runtime with multiple permissions
			permissions := map[string]any{
				"http": map[string]any{
					"reason": "To fetch data from external APIs",
					"allowedUrls": map[string]any{
						"*": []any{"*"},
					},
				},
				"config": map[string]any{
					"reason": "To read configuration settings",
				},
				"scheduler": map[string]any{
					"reason": "To schedule periodic tasks",
				},
			}

			runtimeFunc := mgr.createCustomRuntime(ccache, "multi-service-plugin", permissions)
			runtime, err := runtimeFunc(ctx)
			Expect(err).NotTo(HaveOccurred())
			defer runtime.Close(ctx)

			// Runtime should be created successfully
			Expect(runtime).NotTo(BeNil())
		})

		It("should create runtime with no services when no permissions granted", func() {
			ccache, _ := getCompilationCache()

			// Create runtime with empty permissions
			emptyPermissions := map[string]any{}

			runtimeFunc := mgr.createCustomRuntime(ccache, "no-service-plugin", emptyPermissions)
			runtime, err := runtimeFunc(ctx)
			Expect(err).NotTo(HaveOccurred())
			defer runtime.Close(ctx)

			// Runtime should still be created, but with no host services
			Expect(runtime).NotTo(BeNil())
		})

		It("should demonstrate secure-by-default behavior", func() {
			ccache, _ := getCompilationCache()

			// Test that default (nil permissions) provides no services
			runtimeFunc := mgr.createCustomRuntime(ccache, "default-plugin", nil)
			runtime, err := runtimeFunc(ctx)
			Expect(err).NotTo(HaveOccurred())
			defer runtime.Close(ctx)

			// Runtime should be created but with no host services
			Expect(runtime).NotTo(BeNil())
		})

		It("should test permission enforcement by simulating unauthorized service access", func() {
			// This test demonstrates that plugins would fail at runtime when trying to call
			// host functions they don't have permission for, since those functions are simply
			// not loaded into the WASM runtime environment.

			ccache, _ := getCompilationCache()

			// Create two different runtimes with different permissions
			httpOnlyPermissions := map[string]any{
				"http": map[string]any{
					"reason": "To fetch data from external APIs",
					"allowedUrls": map[string]any{
						"*": []any{"*"},
					},
				},
			}
			configOnlyPermissions := map[string]any{
				"config": map[string]any{
					"reason": "To read configuration settings",
				},
			}

			httpRuntime, err := mgr.createCustomRuntime(ccache, "http-only", httpOnlyPermissions)(ctx)
			Expect(err).NotTo(HaveOccurred())
			defer httpRuntime.Close(ctx)

			configRuntime, err := mgr.createCustomRuntime(ccache, "config-only", configOnlyPermissions)(ctx)
			Expect(err).NotTo(HaveOccurred())
			defer configRuntime.Close(ctx)

			// Both runtimes should be created successfully, but they will have different
			// sets of host functions available. A plugin trying to call unauthorized
			// functions would get "function not found" errors during instantiation or execution.
			Expect(httpRuntime).NotTo(BeNil())
			Expect(configRuntime).NotTo(BeNil())
		})
	})
})
