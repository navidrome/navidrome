package plugins

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DiscoverPlugins", func() {
	var tempPluginsDir string

	// Helper to create a valid plugin for discovery testing
	createValidPlugin := func(name, manifestName, author, version string, capabilities []string) {
		pluginDir := filepath.Join(tempPluginsDir, name)
		Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

		// Copy real WASM file from testdata
		sourceWasmPath := filepath.Join(testDataDir, "fake_artist_agent", "plugin.wasm")
		targetWasmPath := filepath.Join(pluginDir, "plugin.wasm")
		sourceWasm, err := os.ReadFile(sourceWasmPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(os.WriteFile(targetWasmPath, sourceWasm, 0600)).To(Succeed())

		manifest := `{
			"name": "` + manifestName + `",
			"version": "` + version + `",
			"capabilities": [`
		for i, cap := range capabilities {
			if i > 0 {
				manifest += `, `
			}
			manifest += `"` + cap + `"`
		}
		manifest += `],
			"author": "` + author + `",
			"description": "Test Plugin",
			"website": "https://test.navidrome.org/` + manifestName + `",
			"permissions": {}
		}`
		Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0600)).To(Succeed())
	}

	createManifestOnlyPlugin := func(name string) {
		pluginDir := filepath.Join(tempPluginsDir, name)
		Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

		manifest := `{
			"name": "manifest-only",
			"version": "1.0.0",
			"capabilities": ["MetadataAgent"],
			"author": "Test Author",
			"description": "Test Plugin",
			"website": "https://test.navidrome.org/manifest-only",
			"permissions": {}
		}`
		Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0600)).To(Succeed())
	}

	createWasmOnlyPlugin := func(name string) {
		pluginDir := filepath.Join(tempPluginsDir, name)
		Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

		// Copy real WASM file from testdata
		sourceWasmPath := filepath.Join(testDataDir, "fake_artist_agent", "plugin.wasm")
		targetWasmPath := filepath.Join(pluginDir, "plugin.wasm")
		sourceWasm, err := os.ReadFile(sourceWasmPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(os.WriteFile(targetWasmPath, sourceWasm, 0600)).To(Succeed())
	}

	createInvalidManifestPlugin := func(name string) {
		pluginDir := filepath.Join(tempPluginsDir, name)
		Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

		// Copy real WASM file from testdata
		sourceWasmPath := filepath.Join(testDataDir, "fake_artist_agent", "plugin.wasm")
		targetWasmPath := filepath.Join(pluginDir, "plugin.wasm")
		sourceWasm, err := os.ReadFile(sourceWasmPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(os.WriteFile(targetWasmPath, sourceWasm, 0600)).To(Succeed())

		invalidManifest := `{ "invalid": "json" }`
		Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(invalidManifest), 0600)).To(Succeed())
	}

	createEmptyCapabilitiesPlugin := func(name string) {
		pluginDir := filepath.Join(tempPluginsDir, name)
		Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

		// Copy real WASM file from testdata
		sourceWasmPath := filepath.Join(testDataDir, "fake_artist_agent", "plugin.wasm")
		targetWasmPath := filepath.Join(pluginDir, "plugin.wasm")
		sourceWasm, err := os.ReadFile(sourceWasmPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(os.WriteFile(targetWasmPath, sourceWasm, 0600)).To(Succeed())

		manifest := `{
			"name": "empty-capabilities",
			"version": "1.0.0",
			"capabilities": [],
			"author": "Test Author",
			"description": "Test Plugin",
			"website": "https://test.navidrome.org/empty-capabilities",
			"permissions": {}
		}`
		Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0600)).To(Succeed())
	}

	BeforeEach(func() {
		tempPluginsDir, _ = os.MkdirTemp("", "navidrome-plugins-discovery-test-*")
		DeferCleanup(func() {
			_ = os.RemoveAll(tempPluginsDir)
		})
	})

	Context("Valid plugins", func() {
		It("should discover valid plugins with all required files", func() {
			createValidPlugin("test-plugin", "Test Plugin", "Test Author", "1.0.0", []string{"MetadataAgent"})
			createValidPlugin("another-plugin", "Another Plugin", "Another Author", "2.0.0", []string{"Scrobbler"})

			discoveries := DiscoverPlugins(tempPluginsDir)

			Expect(discoveries).To(HaveLen(2))

			// Find each plugin by ID
			var testPlugin, anotherPlugin *PluginDiscoveryEntry
			for i := range discoveries {
				switch discoveries[i].ID {
				case "test-plugin":
					testPlugin = &discoveries[i]
				case "another-plugin":
					anotherPlugin = &discoveries[i]
				}
			}

			Expect(testPlugin).NotTo(BeNil())
			Expect(testPlugin.Error).To(BeNil())
			Expect(testPlugin.Manifest.Name).To(Equal("Test Plugin"))
			Expect(string(testPlugin.Manifest.Capabilities[0])).To(Equal("MetadataAgent"))

			Expect(anotherPlugin).NotTo(BeNil())
			Expect(anotherPlugin.Error).To(BeNil())
			Expect(anotherPlugin.Manifest.Name).To(Equal("Another Plugin"))
			Expect(string(anotherPlugin.Manifest.Capabilities[0])).To(Equal("Scrobbler"))
		})

		It("should handle plugins with same manifest name in different directories", func() {
			createValidPlugin("lastfm-official", "lastfm", "Official Author", "1.0.0", []string{"MetadataAgent"})
			createValidPlugin("lastfm-custom", "lastfm", "Custom Author", "2.0.0", []string{"MetadataAgent"})

			discoveries := DiscoverPlugins(tempPluginsDir)

			Expect(discoveries).To(HaveLen(2))

			// Find each plugin by ID
			var officialPlugin, customPlugin *PluginDiscoveryEntry
			for i := range discoveries {
				switch discoveries[i].ID {
				case "lastfm-official":
					officialPlugin = &discoveries[i]
				case "lastfm-custom":
					customPlugin = &discoveries[i]
				}
			}

			Expect(officialPlugin).NotTo(BeNil())
			Expect(officialPlugin.Error).To(BeNil())
			Expect(officialPlugin.Manifest.Name).To(Equal("lastfm"))
			Expect(officialPlugin.Manifest.Author).To(Equal("Official Author"))

			Expect(customPlugin).NotTo(BeNil())
			Expect(customPlugin.Error).To(BeNil())
			Expect(customPlugin.Manifest.Name).To(Equal("lastfm"))
			Expect(customPlugin.Manifest.Author).To(Equal("Custom Author"))
		})
	})

	Context("Missing files", func() {
		It("should report error for plugins missing WASM files", func() {
			createManifestOnlyPlugin("manifest-only")

			discoveries := DiscoverPlugins(tempPluginsDir)

			Expect(discoveries).To(HaveLen(1))
			Expect(discoveries[0].ID).To(Equal("manifest-only"))
			Expect(discoveries[0].Error).To(HaveOccurred())
			Expect(discoveries[0].Error.Error()).To(ContainSubstring("no plugin.wasm found"))
		})

		It("should skip directories missing manifest files", func() {
			createWasmOnlyPlugin("wasm-only")

			discoveries := DiscoverPlugins(tempPluginsDir)

			Expect(discoveries).To(HaveLen(1))
			Expect(discoveries[0].ID).To(Equal("wasm-only"))
			Expect(discoveries[0].Error).To(HaveOccurred())
			Expect(discoveries[0].Error.Error()).To(ContainSubstring("failed to load manifest"))
		})
	})

	Context("Invalid content", func() {
		It("should report error for invalid manifest JSON", func() {
			createInvalidManifestPlugin("invalid-manifest")

			discoveries := DiscoverPlugins(tempPluginsDir)

			Expect(discoveries).To(HaveLen(1))
			Expect(discoveries[0].ID).To(Equal("invalid-manifest"))
			Expect(discoveries[0].Error).To(HaveOccurred())
			Expect(discoveries[0].Error.Error()).To(ContainSubstring("failed to load manifest"))
		})

		It("should report error for plugins with empty capabilities", func() {
			createEmptyCapabilitiesPlugin("empty-capabilities")

			discoveries := DiscoverPlugins(tempPluginsDir)

			Expect(discoveries).To(HaveLen(1))
			Expect(discoveries[0].ID).To(Equal("empty-capabilities"))
			Expect(discoveries[0].Error).To(HaveOccurred())
			Expect(discoveries[0].Error.Error()).To(ContainSubstring("field capabilities length: must be >= 1"))
		})
	})

	Context("Symlinks", func() {
		It("should discover symlinked plugins correctly", func() {
			// Create a real plugin directory outside tempPluginsDir
			realPluginDir, err := os.MkdirTemp("", "navidrome-real-plugin-*")
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(func() {
				_ = os.RemoveAll(realPluginDir)
			})

			// Create plugin files in the real directory
			sourceWasmPath := filepath.Join(testDataDir, "fake_artist_agent", "plugin.wasm")
			targetWasmPath := filepath.Join(realPluginDir, "plugin.wasm")
			sourceWasm, err := os.ReadFile(sourceWasmPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(os.WriteFile(targetWasmPath, sourceWasm, 0600)).To(Succeed())

			manifest := `{
				"name": "symlinked-plugin",
				"version": "1.0.0",
				"capabilities": ["MetadataAgent"],
				"author": "Test Author",
				"description": "Test Plugin",
				"website": "https://test.navidrome.org/symlinked-plugin",
				"permissions": {}
			}`
			Expect(os.WriteFile(filepath.Join(realPluginDir, "manifest.json"), []byte(manifest), 0600)).To(Succeed())

			// Create symlink
			symlinkPath := filepath.Join(tempPluginsDir, "symlinked-plugin")
			Expect(os.Symlink(realPluginDir, symlinkPath)).To(Succeed())

			discoveries := DiscoverPlugins(tempPluginsDir)

			Expect(discoveries).To(HaveLen(1))
			Expect(discoveries[0].ID).To(Equal("symlinked-plugin"))
			Expect(discoveries[0].Error).To(BeNil())
			Expect(discoveries[0].IsSymlink).To(BeTrue())
			Expect(discoveries[0].Path).To(Equal(realPluginDir))
			Expect(discoveries[0].Manifest.Name).To(Equal("symlinked-plugin"))
		})

		It("should handle relative symlinks", func() {
			// Create a real plugin directory in the same parent as tempPluginsDir
			parentDir := filepath.Dir(tempPluginsDir)
			realPluginDir := filepath.Join(parentDir, "real-plugin-dir")
			Expect(os.MkdirAll(realPluginDir, 0755)).To(Succeed())
			DeferCleanup(func() {
				_ = os.RemoveAll(realPluginDir)
			})

			// Create plugin files in the real directory
			sourceWasmPath := filepath.Join(testDataDir, "fake_artist_agent", "plugin.wasm")
			targetWasmPath := filepath.Join(realPluginDir, "plugin.wasm")
			sourceWasm, err := os.ReadFile(sourceWasmPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(os.WriteFile(targetWasmPath, sourceWasm, 0600)).To(Succeed())

			manifest := `{
				"name": "relative-symlinked-plugin",
				"version": "1.0.0",
				"capabilities": ["MetadataAgent"],
				"author": "Test Author",
				"description": "Test Plugin",
				"website": "https://test.navidrome.org/relative-symlinked-plugin",
				"permissions": {}
			}`
			Expect(os.WriteFile(filepath.Join(realPluginDir, "manifest.json"), []byte(manifest), 0600)).To(Succeed())

			// Create relative symlink
			symlinkPath := filepath.Join(tempPluginsDir, "relative-symlinked-plugin")
			relativeTarget := "../real-plugin-dir"
			Expect(os.Symlink(relativeTarget, symlinkPath)).To(Succeed())

			discoveries := DiscoverPlugins(tempPluginsDir)

			Expect(discoveries).To(HaveLen(1))
			Expect(discoveries[0].ID).To(Equal("relative-symlinked-plugin"))
			Expect(discoveries[0].Error).To(BeNil())
			Expect(discoveries[0].IsSymlink).To(BeTrue())
			Expect(discoveries[0].Path).To(Equal(realPluginDir))
			Expect(discoveries[0].Manifest.Name).To(Equal("relative-symlinked-plugin"))
		})

		It("should report error for broken symlinks", func() {
			symlinkPath := filepath.Join(tempPluginsDir, "broken-symlink")
			nonExistentTarget := "/non/existent/path"
			Expect(os.Symlink(nonExistentTarget, symlinkPath)).To(Succeed())

			discoveries := DiscoverPlugins(tempPluginsDir)

			Expect(discoveries).To(HaveLen(1))
			Expect(discoveries[0].ID).To(Equal("broken-symlink"))
			Expect(discoveries[0].Error).To(HaveOccurred())
			Expect(discoveries[0].Error.Error()).To(ContainSubstring("failed to stat symlink target"))
			Expect(discoveries[0].IsSymlink).To(BeTrue())
		})

		It("should report error for symlinks pointing to files", func() {
			// Create a regular file
			regularFile := filepath.Join(tempPluginsDir, "regular-file.txt")
			Expect(os.WriteFile(regularFile, []byte("content"), 0600)).To(Succeed())

			// Create symlink pointing to the file
			symlinkPath := filepath.Join(tempPluginsDir, "symlink-to-file")
			Expect(os.Symlink(regularFile, symlinkPath)).To(Succeed())

			discoveries := DiscoverPlugins(tempPluginsDir)

			Expect(discoveries).To(HaveLen(1))
			Expect(discoveries[0].ID).To(Equal("symlink-to-file"))
			Expect(discoveries[0].Error).To(HaveOccurred())
			Expect(discoveries[0].Error.Error()).To(ContainSubstring("symlink target is not a directory"))
			Expect(discoveries[0].IsSymlink).To(BeTrue())
		})
	})

	Context("Directory filtering", func() {
		It("should ignore hidden directories", func() {
			createValidPlugin(".hidden-plugin", "Hidden Plugin", "Test Author", "1.0.0", []string{"MetadataAgent"})
			createValidPlugin("visible-plugin", "Visible Plugin", "Test Author", "1.0.0", []string{"MetadataAgent"})

			discoveries := DiscoverPlugins(tempPluginsDir)

			Expect(discoveries).To(HaveLen(1))
			Expect(discoveries[0].ID).To(Equal("visible-plugin"))
		})

		It("should ignore regular files", func() {
			// Create a regular file
			Expect(os.WriteFile(filepath.Join(tempPluginsDir, "regular-file.txt"), []byte("content"), 0600)).To(Succeed())
			createValidPlugin("valid-plugin", "Valid Plugin", "Test Author", "1.0.0", []string{"MetadataAgent"})

			discoveries := DiscoverPlugins(tempPluginsDir)

			Expect(discoveries).To(HaveLen(1))
			Expect(discoveries[0].ID).To(Equal("valid-plugin"))
		})

		It("should handle mixed valid and invalid plugins", func() {
			createValidPlugin("valid-plugin", "Valid Plugin", "Test Author", "1.0.0", []string{"MetadataAgent"})
			createManifestOnlyPlugin("manifest-only")
			createInvalidManifestPlugin("invalid-manifest")
			createValidPlugin("another-valid", "Another Valid", "Test Author", "1.0.0", []string{"Scrobbler"})

			discoveries := DiscoverPlugins(tempPluginsDir)

			Expect(discoveries).To(HaveLen(4))

			var validCount int
			var errorCount int
			for _, discovery := range discoveries {
				if discovery.Error == nil {
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
			nonExistentDir := "/non/existent/plugins/dir"

			discoveries := DiscoverPlugins(nonExistentDir)

			Expect(discoveries).To(HaveLen(1))
			Expect(discoveries[0].Error).To(HaveOccurred())
			Expect(discoveries[0].Error.Error()).To(ContainSubstring("failed to read plugins directory"))
		})
	})
})
