package plugins

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/agents"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin Manager", func() {
	var mgr *Manager
	var ctx context.Context

	BeforeEach(func() {
		// We change the plugins folder to random location to avoid conflicts with other tests,
		// but, as this is an integration test, we can't use configtest.SetupConfig() as it causes
		// data races.
		originalPluginsFolder := conf.Server.Plugins.Folder
		DeferCleanup(func() {
			conf.Server.Plugins.Folder = originalPluginsFolder
		})
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = testDataDir

		ctx = GinkgoT().Context()
		mgr = createManager()
		mgr.ScanPlugins()
	})

	It("should scan and discover plugins from the testdata folder", func() {
		Expect(mgr).NotTo(BeNil())

		mediaAgentNames := mgr.PluginNames("MetadataAgent")
		Expect(mediaAgentNames).To(HaveLen(4))
		Expect(mediaAgentNames).To(ContainElement("fake_artist_agent"))
		Expect(mediaAgentNames).To(ContainElement("fake_album_agent"))
		Expect(mediaAgentNames).To(ContainElement("multi_plugin"))
		Expect(mediaAgentNames).To(ContainElement("unauthorized_plugin"))

		scrobblerNames := mgr.PluginNames("Scrobbler")
		Expect(scrobblerNames).To(ContainElement("fake_scrobbler"))

		initServiceNames := mgr.PluginNames("LifecycleManagement")
		Expect(initServiceNames).To(ContainElement("multi_plugin"))
		Expect(initServiceNames).To(ContainElement("fake_init_service"))
	})

	It("should load a MetadataAgent plugin and invoke artist-related methods", func() {
		plugin := mgr.LoadPlugin("fake_artist_agent", CapabilityMetadataAgent)
		Expect(plugin).NotTo(BeNil())

		agent, ok := plugin.(agents.Interface)
		Expect(ok).To(BeTrue(), "plugin should implement agents.Interface")
		Expect(agent.AgentName()).To(Equal("fake_artist_agent"))

		mbidRetriever, ok := agent.(agents.ArtistMBIDRetriever)
		Expect(ok).To(BeTrue())
		mbid, err := mbidRetriever.GetArtistMBID(ctx, "123", "The Beatles")
		Expect(err).NotTo(HaveOccurred())
		Expect(mbid).To(Equal("1234567890"))
	})

	It("should load all MetadataAgent plugins", func() {
		agents := mgr.LoadAllMediaAgents()
		Expect(agents).To(HaveLen(4))
		var names []string
		for _, a := range agents {
			names = append(names, a.AgentName())
		}
		Expect(names).To(ContainElements("fake_artist_agent", "fake_album_agent", "multi_plugin", "unauthorized_plugin"))
	})

	It("should use DevPluginCompilationTimeout config for plugin compilation timeout", func() {
		conf.Server.DevPluginCompilationTimeout = 123 * time.Second
		Expect(pluginCompilationTimeout()).To(Equal(123 * time.Second))

		conf.Server.DevPluginCompilationTimeout = 0
		Expect(pluginCompilationTimeout()).To(Equal(time.Minute))
	})

	Describe("ScanPlugins", func() {
		var tempPluginsDir string
		var m *Manager

		BeforeEach(func() {
			tempPluginsDir, _ = os.MkdirTemp("", "navidrome-plugins-test-*")
			DeferCleanup(func() {
				_ = os.RemoveAll(tempPluginsDir)
			})

			conf.Server.Plugins.Folder = tempPluginsDir
			m = createManager()
		})

		// Helper to create a complete valid plugin for manager testing
		createValidPlugin := func(folderName, manifestName string) {
			pluginDir := filepath.Join(tempPluginsDir, folderName)
			Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

			// Copy real WASM file from testdata
			sourceWasmPath := filepath.Join(testDataDir, "fake_artist_agent", "plugin.wasm")
			targetWasmPath := filepath.Join(pluginDir, "plugin.wasm")
			sourceWasm, err := os.ReadFile(sourceWasmPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(os.WriteFile(targetWasmPath, sourceWasm, 0600)).To(Succeed())

			manifest := `{
				"name": "` + manifestName + `",
				"version": "1.0.0",
				"capabilities": ["MetadataAgent"],
				"author": "Test Author",
				"description": "Test Plugin",
				"website": "https://test.navidrome.org/` + manifestName + `",
				"permissions": {}
			}`
			Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0600)).To(Succeed())
		}

		It("should register and compile discovered plugins", func() {
			createValidPlugin("test-plugin", "test-plugin")

			m.ScanPlugins()

			// Focus on manager behavior: registration and compilation
			Expect(m.plugins).To(HaveLen(1))
			Expect(m.plugins).To(HaveKey("test-plugin"))

			plugin := m.plugins["test-plugin"]
			Expect(plugin.ID).To(Equal("test-plugin"))
			Expect(plugin.Manifest.Name).To(Equal("test-plugin"))

			// Verify plugin can be loaded (compilation successful)
			loadedPlugin := m.LoadPlugin("test-plugin", CapabilityMetadataAgent)
			Expect(loadedPlugin).NotTo(BeNil())
		})

		It("should handle multiple plugins with different IDs but same manifest names", func() {
			// This tests manager-specific behavior: how it handles ID conflicts
			createValidPlugin("lastfm-official", "lastfm")
			createValidPlugin("lastfm-custom", "lastfm")

			m.ScanPlugins()

			// Both should be registered with their folder names as IDs
			Expect(m.plugins).To(HaveLen(2))
			Expect(m.plugins).To(HaveKey("lastfm-official"))
			Expect(m.plugins).To(HaveKey("lastfm-custom"))

			// Both should be loadable independently
			official := m.LoadPlugin("lastfm-official", CapabilityMetadataAgent)
			custom := m.LoadPlugin("lastfm-custom", CapabilityMetadataAgent)
			Expect(official).NotTo(BeNil())
			Expect(custom).NotTo(BeNil())
			Expect(official.PluginID()).To(Equal("lastfm-official"))
			Expect(custom.PluginID()).To(Equal("lastfm-custom"))
		})
	})

	Describe("LoadPlugin", func() {
		It("should load a MetadataAgent plugin and invoke artist-related methods", func() {
			plugin := mgr.LoadPlugin("fake_artist_agent", CapabilityMetadataAgent)
			Expect(plugin).NotTo(BeNil())

			agent, ok := plugin.(agents.Interface)
			Expect(ok).To(BeTrue(), "plugin should implement agents.Interface")
			Expect(agent.AgentName()).To(Equal("fake_artist_agent"))

			mbidRetriever, ok := agent.(agents.ArtistMBIDRetriever)
			Expect(ok).To(BeTrue())
			mbid, err := mbidRetriever.GetArtistMBID(ctx, "id", "Test Artist")
			Expect(err).NotTo(HaveOccurred())
			Expect(mbid).To(Equal("1234567890"))
		})
	})

	Describe("Invoke Methods", func() {
		It("should load all MetadataAgent plugins and invoke methods", func() {
			mediaAgentNames := mgr.PluginNames("MetadataAgent")
			Expect(mediaAgentNames).NotTo(BeEmpty())

			plugins := mgr.LoadAllPlugins("MetadataAgent")
			Expect(plugins).To(HaveLen(len(mediaAgentNames)))

			var fakeAlbumPlugin agents.Interface
			for _, p := range plugins {
				if agent, ok := p.(agents.Interface); ok {
					if agent.AgentName() == "fake_album_agent" {
						fakeAlbumPlugin = agent
						break
					}
				}
			}

			Expect(fakeAlbumPlugin).NotTo(BeNil(), "fake_album_agent should be loaded")

			albumInfo, err := fakeAlbumPlugin.(agents.AlbumInfoRetriever).GetAlbumInfo(ctx, "Test Album", "Test Artist", "mbid")
			Expect(err).NotTo(HaveOccurred())
			Expect(albumInfo.Name).To(Equal("Test Album"))
		})
	})

	Describe("Permission Enforcement Integration", func() {
		It("should fail when plugin tries to access unauthorized services", func() {
			// Load the unauthorized_plugin which has no permissions but tries to call HTTP service
			plugin := mgr.LoadPlugin("unauthorized_plugin", CapabilityMetadataAgent)
			Expect(plugin).NotTo(BeNil(), "unauthorized_plugin should be loaded")

			agent, ok := plugin.(agents.Interface)
			Expect(ok).To(BeTrue(), "plugin should implement agents.Interface")
			Expect(agent.AgentName()).To(Equal("unauthorized_plugin"))

			// Try to call GetAlbumInfo which attempts to make an HTTP call without permission
			// This should fail because the plugin has no "http" permission in its manifest
			albumRetriever, ok := agent.(agents.AlbumInfoRetriever)
			Expect(ok).To(BeTrue(), "plugin should implement AlbumInfoRetriever")

			_, err := albumRetriever.GetAlbumInfo(ctx, "Test Album", "Test Artist", "test-mbid")
			Expect(err).To(HaveOccurred(), "should fail when trying to access unauthorized HTTP service")

			// The error should indicate that the HTTP function is not available
			// This happens because the HTTP host functions were not loaded into the WASM runtime
			// since the plugin doesn't have "http" permission
			Expect(err.Error()).To(ContainSubstring("is not exported"), "error should indicate missing HTTP functions")
		})
	})

	Describe("DiscoverPlugins", func() {
		var tempDir string

		// Helper to create a complete valid plugin
		createValidPlugin := func(name, manifestName, author, version string, capabilities []string) {
			pluginDir := filepath.Join(tempDir, name)
			Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

			// Create manifest.json
			manifest := `{
				"name": "` + manifestName + `",
				"author": "` + author + `",
				"version": "` + version + `",
				"description": "Test plugin",
				"website": "https://test.navidrome.org/` + manifestName + `",
				"capabilities": ["` + capabilities[0] + `"],
				"permissions": {}
			}`
			Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0600)).To(Succeed())

			// Create dummy WASM file
			wasmContent := []byte("dummy wasm content")
			Expect(os.WriteFile(filepath.Join(pluginDir, "plugin.wasm"), wasmContent, 0600)).To(Succeed())
		}

		// Helper to create plugin directory with only manifest (missing WASM)
		createManifestOnlyPlugin := func(name string) {
			pluginDir := filepath.Join(tempDir, name)
			Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

			manifest := `{
				"name": "` + name + `",
				"author": "Test Author",
				"version": "1.0.0",
				"description": "Test plugin",
				"website": "https://test.navidrome.org/` + name + `",
				"capabilities": ["MetadataAgent"],
				"permissions": {}
			}`
			Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0600)).To(Succeed())
		}

		// Helper to create plugin directory with only WASM (missing manifest)
		createWasmOnlyPlugin := func(name string) {
			pluginDir := filepath.Join(tempDir, name)
			Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

			wasmContent := []byte("dummy wasm content")
			Expect(os.WriteFile(filepath.Join(pluginDir, "plugin.wasm"), wasmContent, 0600)).To(Succeed())
		}

		// Helper to create plugin with invalid manifest
		createInvalidManifestPlugin := func(name string) {
			pluginDir := filepath.Join(tempDir, name)
			Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

			invalidManifest := `{ "invalid": json content }`
			Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(invalidManifest), 0600)).To(Succeed())

			wasmContent := []byte("dummy wasm content")
			Expect(os.WriteFile(filepath.Join(pluginDir, "plugin.wasm"), wasmContent, 0600)).To(Succeed())
		}

		// Helper to create plugin with empty capabilities
		createEmptyCapabilitiesPlugin := func(name string) {
			pluginDir := filepath.Join(tempDir, name)
			Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

			manifest := `{
				"name": "` + name + `",
				"author": "Test Author", 
				"version": "1.0.0",
				"description": "Test plugin",
				"website": "https://test.navidrome.org/` + name + `",
				"capabilities": [],
				"permissions": {}
			}`
			Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0600)).To(Succeed())

			wasmContent := []byte("dummy wasm content")
			Expect(os.WriteFile(filepath.Join(pluginDir, "plugin.wasm"), wasmContent, 0600)).To(Succeed())
		}

		BeforeEach(func() {
			tempDir = GinkgoT().TempDir()
		})

		Context("Valid plugins", func() {
			It("should discover valid plugins with all required files", func() {
				createValidPlugin("plugin1", "Plugin One", "Author 1", "1.0.0", []string{"MetadataAgent"})
				createValidPlugin("plugin2", "Plugin Two", "Author 2", "2.0.0", []string{"Scrobbler"})

				discoveries := DiscoverPlugins(tempDir)

				Expect(discoveries).To(HaveLen(2))

				// Check plugin1
				plugin1 := discoveries[0]
				if plugin1.ID == "plugin2" {
					plugin1 = discoveries[1]
				}
				Expect(plugin1.ID).To(Equal("plugin1"))
				Expect(plugin1.Path).To(Equal(filepath.Join(tempDir, "plugin1")))
				Expect(plugin1.WasmPath).To(Equal(filepath.Join(tempDir, "plugin1", "plugin.wasm")))
				Expect(plugin1.Manifest.Name).To(Equal("Plugin One"))
				Expect(plugin1.Manifest.Author).To(Equal("Author 1"))
				Expect(plugin1.IsSymlink).To(BeFalse())
				Expect(plugin1.Error).To(BeNil())
			})

			It("should handle plugins with same manifest name in different directories", func() {
				createValidPlugin("lastfm-official", "lastfm", "Official", "1.0", []string{"MetadataAgent"})
				createValidPlugin("lastfm-custom", "lastfm", "Custom", "2.0", []string{"MetadataAgent"})

				discoveries := DiscoverPlugins(tempDir)

				Expect(discoveries).To(HaveLen(2))

				var official, custom PluginDiscoveryEntry
				for _, d := range discoveries {
					if d.ID == "lastfm-official" {
						official = d
					} else if d.ID == "lastfm-custom" {
						custom = d
					}
				}

				Expect(official.ID).To(Equal("lastfm-official"))
				Expect(official.Manifest.Name).To(Equal("lastfm"))
				Expect(official.Manifest.Author).To(Equal("Official"))
				Expect(official.Error).To(BeNil())

				Expect(custom.ID).To(Equal("lastfm-custom"))
				Expect(custom.Manifest.Name).To(Equal("lastfm"))
				Expect(custom.Manifest.Author).To(Equal("Custom"))
				Expect(custom.Error).To(BeNil())
			})
		})

		Context("Missing files", func() {
			It("should report error for plugins missing WASM files", func() {
				createManifestOnlyPlugin("no-wasm-plugin")

				discoveries := DiscoverPlugins(tempDir)

				Expect(discoveries).To(HaveLen(1))
				discovery := discoveries[0]
				Expect(discovery.ID).To(Equal("no-wasm-plugin"))
				Expect(discovery.Error).To(HaveOccurred())
				Expect(discovery.Error.Error()).To(ContainSubstring("no plugin.wasm found"))
			})

			It("should skip directories missing manifest files", func() {
				createWasmOnlyPlugin("no-manifest-plugin")

				discoveries := DiscoverPlugins(tempDir)

				Expect(discoveries).To(HaveLen(1))
				discovery := discoveries[0]
				Expect(discovery.ID).To(Equal("no-manifest-plugin"))
				Expect(discovery.Error).To(HaveOccurred())
				Expect(discovery.Error.Error()).To(ContainSubstring("failed to load manifest"))
			})
		})

		Context("Invalid content", func() {
			It("should report error for invalid manifest JSON", func() {
				createInvalidManifestPlugin("invalid-manifest")

				discoveries := DiscoverPlugins(tempDir)

				Expect(discoveries).To(HaveLen(1))
				discovery := discoveries[0]
				Expect(discovery.ID).To(Equal("invalid-manifest"))
				Expect(discovery.Error).To(HaveOccurred())
				Expect(discovery.Error.Error()).To(ContainSubstring("failed to load manifest"))
			})

			It("should report error for plugins with empty capabilities", func() {
				createEmptyCapabilitiesPlugin("empty-caps")

				discoveries := DiscoverPlugins(tempDir)

				Expect(discoveries).To(HaveLen(1))
				discovery := discoveries[0]
				Expect(discovery.ID).To(Equal("empty-caps"))
				Expect(discovery.Error).To(HaveOccurred())
				// The manifest validation now catches empty capabilities during LoadManifest
				Expect(discovery.Error.Error()).To(ContainSubstring("capabilities"))
			})
		})

		Context("Symlinks", func() {
			It("should discover symlinked plugins correctly", func() {
				// Create real plugin
				createValidPlugin("real-plugin", "Real Plugin", "Author", "1.0", []string{"MetadataAgent"})

				// Create symlink
				realPath := filepath.Join(tempDir, "real-plugin")
				symlinkPath := filepath.Join(tempDir, "symlinked-plugin")
				Expect(os.Symlink(realPath, symlinkPath)).To(Succeed())

				discoveries := DiscoverPlugins(tempDir)

				Expect(discoveries).To(HaveLen(2))

				var realPlugin, symlinkPlugin PluginDiscoveryEntry
				for _, d := range discoveries {
					if d.ID == "real-plugin" {
						realPlugin = d
					} else if d.ID == "symlinked-plugin" {
						symlinkPlugin = d
					}
				}

				// Real plugin
				Expect(realPlugin.ID).To(Equal("real-plugin"))
				Expect(realPlugin.IsSymlink).To(BeFalse())
				Expect(realPlugin.Error).To(BeNil())

				// Symlinked plugin
				Expect(symlinkPlugin.ID).To(Equal("symlinked-plugin"))
				Expect(symlinkPlugin.IsSymlink).To(BeTrue())
				Expect(symlinkPlugin.Path).To(Equal(realPath)) // Should resolve to real path
				Expect(symlinkPlugin.Error).To(BeNil())
			})

			It("should handle relative symlinks", func() {
				// Create plugin in subdirectory outside the tempDir to avoid discovery conflicts
				externalDir := GinkgoT().TempDir()
				pluginDir := filepath.Join(externalDir, "real-plugin")
				Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

				manifest := `{
					"name": "real-plugin",
					"author": "Author",
					"version": "1.0",
					"description": "Test plugin",
					"website": "https://test.navidrome.org/real-plugin",
					"capabilities": ["MetadataAgent"],
					"permissions": {}
				}`
				Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0600)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(pluginDir, "plugin.wasm"), []byte("wasm"), 0600)).To(Succeed())

				// Create relative symlink
				symlinkPath := filepath.Join(tempDir, "relative-link")
				Expect(os.Symlink(pluginDir, symlinkPath)).To(Succeed())

				discoveries := DiscoverPlugins(tempDir)

				Expect(discoveries).To(HaveLen(1))
				discovery := discoveries[0]
				Expect(discovery.ID).To(Equal("relative-link"))
				Expect(discovery.IsSymlink).To(BeTrue())
				Expect(discovery.Path).To(Equal(pluginDir)) // Should resolve to absolute path
				Expect(discovery.Error).To(BeNil())
			})

			It("should report error for broken symlinks", func() {
				// Create symlink to non-existent target
				symlinkPath := filepath.Join(tempDir, "broken-link")
				Expect(os.Symlink("/non/existent/path", symlinkPath)).To(Succeed())

				discoveries := DiscoverPlugins(tempDir)

				Expect(discoveries).To(HaveLen(1))
				discovery := discoveries[0]
				Expect(discovery.ID).To(Equal("broken-link"))
				Expect(discovery.IsSymlink).To(BeTrue())
				Expect(discovery.Error).To(HaveOccurred())
				Expect(discovery.Error.Error()).To(ContainSubstring("failed to stat symlink target"))
			})

			It("should report error for symlinks pointing to files", func() {
				// Create a regular file
				filePath := filepath.Join(tempDir, "regular-file.txt")
				Expect(os.WriteFile(filePath, []byte("content"), 0600)).To(Succeed())

				// Create symlink to file
				symlinkPath := filepath.Join(tempDir, "file-link")
				Expect(os.Symlink(filePath, symlinkPath)).To(Succeed())

				discoveries := DiscoverPlugins(tempDir)

				Expect(discoveries).To(HaveLen(1))
				discovery := discoveries[0]
				Expect(discovery.ID).To(Equal("file-link"))
				Expect(discovery.IsSymlink).To(BeTrue())
				Expect(discovery.Error).To(HaveOccurred())
				Expect(discovery.Error.Error()).To(ContainSubstring("symlink target is not a directory"))
			})
		})

		Context("Directory filtering", func() {
			It("should ignore hidden directories", func() {
				createValidPlugin("visible-plugin", "Visible", "Author", "1.0", []string{"MetadataAgent"})

				// Create hidden directory
				hiddenDir := filepath.Join(tempDir, ".hidden-plugin")
				Expect(os.MkdirAll(hiddenDir, 0755)).To(Succeed())

				discoveries := DiscoverPlugins(tempDir)

				Expect(discoveries).To(HaveLen(1))
				Expect(discoveries[0].ID).To(Equal("visible-plugin"))
			})

			It("should ignore regular files", func() {
				createValidPlugin("valid-plugin", "Valid", "Author", "1.0", []string{"MetadataAgent"})

				// Create regular file in plugins directory
				Expect(os.WriteFile(filepath.Join(tempDir, "regular-file.txt"), []byte("content"), 0600)).To(Succeed())

				discoveries := DiscoverPlugins(tempDir)

				Expect(discoveries).To(HaveLen(1))
				Expect(discoveries[0].ID).To(Equal("valid-plugin"))
			})

			It("should handle mixed valid and invalid plugins", func() {
				createValidPlugin("valid1", "Valid One", "Author", "1.0", []string{"MetadataAgent"})
				createManifestOnlyPlugin("invalid1")
				createValidPlugin("valid2", "Valid Two", "Author", "2.0", []string{"Scrobbler"})
				createInvalidManifestPlugin("invalid2")

				discoveries := DiscoverPlugins(tempDir)

				Expect(discoveries).To(HaveLen(4))

				var validCount, errorCount int
				for _, d := range discoveries {
					if d.Error == nil {
						validCount++
					} else {
						errorCount++
					}
				}

				Expect(validCount).To(Equal(2))
				Expect(errorCount).To(Equal(2))
			})
		})

		Context("Error handling", func() {
			It("should handle non-existent plugins directory", func() {
				discoveries := DiscoverPlugins("/non/existent/directory")

				Expect(discoveries).To(HaveLen(1))
				discovery := discoveries[0]
				Expect(discovery.ID).To(BeEmpty())
				Expect(discovery.Error).To(HaveOccurred())
				Expect(discovery.Error.Error()).To(ContainSubstring("failed to read plugins directory"))
			})
		})
	})
})
