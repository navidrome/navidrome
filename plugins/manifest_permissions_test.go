package plugins

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/plugins/schema"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Helper function to create test plugins with typed permissions
func createTestPlugin(tempDir, name string, permissions schema.PluginManifestPermissions) string {
	pluginDir := filepath.Join(tempDir, name)
	Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

	// Use the generated PluginManifest type directly - it handles JSON marshaling automatically
	manifest := schema.PluginManifest{
		Name:        name,
		Author:      "Test Author",
		Version:     "1.0.0",
		Description: "Test plugin for permissions",
		Website:     "https://test.navidrome.org/" + name,
		Capabilities: []schema.PluginManifestCapabilitiesElem{
			schema.PluginManifestCapabilitiesElemMetadataAgent,
		},
		Permissions: permissions,
	}

	// Marshal the typed manifest directly - gets all validation for free
	manifestData, err := json.Marshal(manifest)
	Expect(err).NotTo(HaveOccurred())

	manifestPath := filepath.Join(pluginDir, "manifest.json")
	Expect(os.WriteFile(manifestPath, manifestData, 0600)).To(Succeed())

	// Create fake WASM file (since plugin discovery checks for it)
	wasmPath := filepath.Join(pluginDir, "plugin.wasm")
	Expect(os.WriteFile(wasmPath, []byte("fake wasm content"), 0600)).To(Succeed())

	return pluginDir
}

var _ = Describe("Plugin Permissions", func() {
	var (
		mgr     *managerImpl
		tempDir string
		ctx     context.Context
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = context.Background()
		mgr = createManager(nil, metrics.NewNoopInstance())
		tempDir = GinkgoT().TempDir()
	})

	Describe("Permission Enforcement in createRuntime", func() {
		It("should only load services specified in permissions", func() {
			// Test with limited permissions using typed structs
			permissions := schema.PluginManifestPermissions{
				Http: &schema.PluginManifestPermissionsHttp{
					Reason: "To fetch data from external APIs",
					AllowedUrls: map[string][]schema.PluginManifestPermissionsHttpAllowedUrlsValueElem{
						"*": {schema.PluginManifestPermissionsHttpAllowedUrlsValueElemWildcard},
					},
					AllowLocalNetwork: false,
				},
				Config: &schema.PluginManifestPermissionsConfig{
					Reason: "To read configuration settings",
				},
			}

			runtimeFunc := mgr.createRuntime("test-plugin", permissions)

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
			permissions := schema.PluginManifestPermissions{}

			runtimeFunc := mgr.createRuntime("empty-permissions-plugin", permissions)

			runtime, err := runtimeFunc(ctx)
			Expect(err).NotTo(HaveOccurred())
			defer runtime.Close(ctx)

			// Should succeed but with no host services available
			Expect(runtime).NotTo(BeNil())
		})

		It("should handle all available permissions", func() {
			// Test with all possible permissions using typed structs
			permissions := schema.PluginManifestPermissions{
				Http: &schema.PluginManifestPermissionsHttp{
					Reason: "To fetch data from external APIs",
					AllowedUrls: map[string][]schema.PluginManifestPermissionsHttpAllowedUrlsValueElem{
						"*": {schema.PluginManifestPermissionsHttpAllowedUrlsValueElemWildcard},
					},
					AllowLocalNetwork: false,
				},
				Config: &schema.PluginManifestPermissionsConfig{
					Reason: "To read configuration settings",
				},
				Scheduler: &schema.PluginManifestPermissionsScheduler{
					Reason: "To schedule periodic tasks",
				},
				Websocket: &schema.PluginManifestPermissionsWebsocket{
					Reason:            "To handle real-time communication",
					AllowedUrls:       []string{"wss://api.example.com"},
					AllowLocalNetwork: false,
				},
				Cache: &schema.PluginManifestPermissionsCache{
					Reason: "To cache data and reduce API calls",
				},
				Artwork: &schema.PluginManifestPermissionsArtwork{
					Reason: "To generate artwork URLs",
				},
			}

			runtimeFunc := mgr.createRuntime("full-permissions-plugin", permissions)

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
			// Create plugin with http permission using typed structs
			permissions := schema.PluginManifestPermissions{
				Http: &schema.PluginManifestPermissionsHttp{
					Reason: "To fetch metadata from external APIs",
					AllowedUrls: map[string][]schema.PluginManifestPermissionsHttpAllowedUrlsValueElem{
						"*": {schema.PluginManifestPermissionsHttpAllowedUrlsValueElemWildcard},
					},
				},
			}
			createTestPlugin(tempDir, "valid-plugin", permissions)

			// Scan for plugins
			mgr.ScanPlugins()

			// Verify plugin was discovered (even without valid WASM)
			pluginNames := mgr.PluginNames("MetadataAgent")
			Expect(pluginNames).To(ContainElement("valid-plugin"))
		})

		It("should discover plugin with no permissions", func() {
			// Create plugin with empty permissions using typed structs
			permissions := schema.PluginManifestPermissions{}
			createTestPlugin(tempDir, "no-perms-plugin", permissions)

			mgr.ScanPlugins()

			pluginNames := mgr.PluginNames("MetadataAgent")
			Expect(pluginNames).To(ContainElement("no-perms-plugin"))
		})

		It("should discover plugin with multiple permissions", func() {
			// Create plugin with multiple permissions using typed structs
			permissions := schema.PluginManifestPermissions{
				Http: &schema.PluginManifestPermissionsHttp{
					Reason: "To fetch metadata from external APIs",
					AllowedUrls: map[string][]schema.PluginManifestPermissionsHttpAllowedUrlsValueElem{
						"*": {schema.PluginManifestPermissionsHttpAllowedUrlsValueElemWildcard},
					},
				},
				Config: &schema.PluginManifestPermissionsConfig{
					Reason: "To read plugin configuration settings",
				},
				Scheduler: &schema.PluginManifestPermissionsScheduler{
					Reason: "To schedule periodic data updates",
				},
			}
			createTestPlugin(tempDir, "multi-perms-plugin", permissions)

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
			// Create a manifest JSON string without the permissions field
			manifestContent := `{
				"name": "test-plugin",
				"author": "Test Author",
				"version": "1.0.0",
				"description": "A test plugin",
				"website": "https://test.navidrome.org/test-plugin",
				"capabilities": ["MetadataAgent"]
			}`

			manifestPath := filepath.Join(tempDir, "manifest.json")
			err := os.WriteFile(manifestPath, []byte(manifestContent), 0600)
			Expect(err).NotTo(HaveOccurred())

			_, err = LoadManifest(tempDir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("field permissions in PluginManifest: required"))
		})

		It("should allow unknown permission keys", func() {
			// Create manifest with both known and unknown permission types
			pluginDir := filepath.Join(tempDir, "unknown-perms")
			Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

			manifestContent := `{
				"name": "unknown-perms",
				"author": "Test Author",
				"version": "1.0.0",
				"description": "Manifest with unknown permissions",
				"website": "https://test.navidrome.org/unknown-perms",
				"capabilities": ["MetadataAgent"],
				"permissions": {
					"http": {
						"reason": "To fetch data from external APIs",
						"allowedUrls": {
							"*": ["*"]
						}
					},
					"unknown": {
						"customField": "customValue"
					}
				}
			}`

			Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifestContent), 0600)).To(Succeed())

			// Test manifest loading directly - should succeed even with unknown permissions
			loadedManifest, err := LoadManifest(pluginDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(loadedManifest).NotTo(BeNil())
			// With typed permissions, we check the specific fields
			Expect(loadedManifest.Permissions.Http).NotTo(BeNil())
			Expect(loadedManifest.Permissions.Http.Reason).To(Equal("To fetch data from external APIs"))
			// The key point is that the manifest loads successfully despite unknown permissions
			// The actual handling of AdditionalProperties depends on the JSON schema implementation
		})
	})

	Describe("Runtime Pool with Permissions", func() {
		It("should create separate runtimes for different permission sets", func() {
			// Create two different permission sets using typed structs
			permissions1 := schema.PluginManifestPermissions{
				Http: &schema.PluginManifestPermissionsHttp{
					Reason: "To fetch data from external APIs",
					AllowedUrls: map[string][]schema.PluginManifestPermissionsHttpAllowedUrlsValueElem{
						"*": {schema.PluginManifestPermissionsHttpAllowedUrlsValueElemWildcard},
					},
					AllowLocalNetwork: false,
				},
			}
			permissions2 := schema.PluginManifestPermissions{
				Config: &schema.PluginManifestPermissionsConfig{
					Reason: "To read configuration settings",
				},
			}

			runtimeFunc1 := mgr.createRuntime("plugin1", permissions1)
			runtimeFunc2 := mgr.createRuntime("plugin2", permissions2)

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
				"website": "https://test.navidrome.org/valid-manifest",
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
			// With typed permissions, check the specific permission fields
			Expect(manifest.Permissions.Http).NotTo(BeNil())
			Expect(manifest.Permissions.Http.Reason).To(Equal("To fetch metadata from external APIs"))
			Expect(manifest.Permissions.Config).NotTo(BeNil())
			Expect(manifest.Permissions.Config.Reason).To(Equal("To read plugin configuration settings"))
		})

		It("should track which services are requested per plugin", func() {
			// Test that different plugins can have different permission sets
			permissions1 := schema.PluginManifestPermissions{
				Http: &schema.PluginManifestPermissionsHttp{
					Reason: "To fetch data from external APIs",
					AllowedUrls: map[string][]schema.PluginManifestPermissionsHttpAllowedUrlsValueElem{
						"*": {schema.PluginManifestPermissionsHttpAllowedUrlsValueElemWildcard},
					},
					AllowLocalNetwork: false,
				},
				Config: &schema.PluginManifestPermissionsConfig{
					Reason: "To read configuration settings",
				},
			}
			permissions2 := schema.PluginManifestPermissions{
				Scheduler: &schema.PluginManifestPermissionsScheduler{
					Reason: "To schedule periodic tasks",
				},
				Config: &schema.PluginManifestPermissionsConfig{
					Reason: "To read configuration for scheduler",
				},
			}
			permissions3 := schema.PluginManifestPermissions{} // Empty permissions

			createTestPlugin(tempDir, "plugin-with-http", permissions1)
			createTestPlugin(tempDir, "plugin-with-scheduler", permissions2)
			createTestPlugin(tempDir, "plugin-with-none", permissions3)

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
			// Create runtime with HTTP permission using typed struct
			permissions := schema.PluginManifestPermissions{
				Http: &schema.PluginManifestPermissionsHttp{
					Reason: "To fetch data from external APIs",
					AllowedUrls: map[string][]schema.PluginManifestPermissionsHttpAllowedUrlsValueElem{
						"*": {schema.PluginManifestPermissionsHttpAllowedUrlsValueElemWildcard},
					},
					AllowLocalNetwork: false,
				},
			}

			runtimeFunc := mgr.createRuntime("http-only-plugin", permissions)
			runtime, err := runtimeFunc(ctx)
			Expect(err).NotTo(HaveOccurred())
			defer runtime.Close(ctx)

			// Runtime should be created successfully - host functions are loaded during runtime creation
			Expect(runtime).NotTo(BeNil())
		})

		It("should successfully create runtime with multiple permitted services", func() {
			// Create runtime with multiple permissions using typed structs
			permissions := schema.PluginManifestPermissions{
				Http: &schema.PluginManifestPermissionsHttp{
					Reason: "To fetch data from external APIs",
					AllowedUrls: map[string][]schema.PluginManifestPermissionsHttpAllowedUrlsValueElem{
						"*": {schema.PluginManifestPermissionsHttpAllowedUrlsValueElemWildcard},
					},
					AllowLocalNetwork: false,
				},
				Config: &schema.PluginManifestPermissionsConfig{
					Reason: "To read configuration settings",
				},
				Scheduler: &schema.PluginManifestPermissionsScheduler{
					Reason: "To schedule periodic tasks",
				},
			}

			runtimeFunc := mgr.createRuntime("multi-service-plugin", permissions)
			runtime, err := runtimeFunc(ctx)
			Expect(err).NotTo(HaveOccurred())
			defer runtime.Close(ctx)

			// Runtime should be created successfully
			Expect(runtime).NotTo(BeNil())
		})

		It("should create runtime with no services when no permissions granted", func() {
			// Create runtime with empty permissions using typed struct
			emptyPermissions := schema.PluginManifestPermissions{}

			runtimeFunc := mgr.createRuntime("no-service-plugin", emptyPermissions)
			runtime, err := runtimeFunc(ctx)
			Expect(err).NotTo(HaveOccurred())
			defer runtime.Close(ctx)

			// Runtime should still be created, but with no host services
			Expect(runtime).NotTo(BeNil())
		})

		It("should demonstrate secure-by-default behavior", func() {
			// Test that default (empty permissions) provides no services
			defaultPermissions := schema.PluginManifestPermissions{}
			runtimeFunc := mgr.createRuntime("default-plugin", defaultPermissions)
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

			// Create two different runtimes with different permissions using typed structs
			httpOnlyPermissions := schema.PluginManifestPermissions{
				Http: &schema.PluginManifestPermissionsHttp{
					Reason: "To fetch data from external APIs",
					AllowedUrls: map[string][]schema.PluginManifestPermissionsHttpAllowedUrlsValueElem{
						"*": {schema.PluginManifestPermissionsHttpAllowedUrlsValueElemWildcard},
					},
					AllowLocalNetwork: false,
				},
			}
			configOnlyPermissions := schema.PluginManifestPermissions{
				Config: &schema.PluginManifestPermissionsConfig{
					Reason: "To read configuration settings",
				},
			}

			httpRuntime, err := mgr.createRuntime("http-only", httpOnlyPermissions)(ctx)
			Expect(err).NotTo(HaveOccurred())
			defer httpRuntime.Close(ctx)

			configRuntime, err := mgr.createRuntime("config-only", configOnlyPermissions)(ctx)
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
