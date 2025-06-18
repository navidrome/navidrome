package plugins

import (
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/plugins/schema"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin Manifest", func() {
	var tempDir string

	BeforeEach(func() {
		tempDir = GinkgoT().TempDir()
	})

	It("should load and parse a valid manifest", func() {
		manifestPath := filepath.Join(tempDir, "manifest.json")
		manifestContent := []byte(`{
			"name": "test-plugin",
			"author": "Test Author",
			"version": "1.0.0",
			"description": "A test plugin",
			"website": "https://test.navidrome.org/test-plugin",
			"capabilities": ["MetadataAgent", "Scrobbler"],
			"permissions": {
				"http": {
					"reason": "To fetch metadata",
					"allowedUrls": {
						"https://api.example.com/*": ["GET"]
					}
				}
			}
		}`)

		err := os.WriteFile(manifestPath, manifestContent, 0600)
		Expect(err).NotTo(HaveOccurred())

		manifest, err := LoadManifest(tempDir)
		Expect(err).NotTo(HaveOccurred())
		Expect(manifest).NotTo(BeNil())
		Expect(manifest.Name).To(Equal("test-plugin"))
		Expect(manifest.Author).To(Equal("Test Author"))
		Expect(manifest.Version).To(Equal("1.0.0"))
		Expect(manifest.Description).To(Equal("A test plugin"))
		Expect(manifest.Capabilities).To(HaveLen(2))
		Expect(manifest.Capabilities[0]).To(Equal(schema.PluginManifestCapabilitiesElemMetadataAgent))
		Expect(manifest.Capabilities[1]).To(Equal(schema.PluginManifestCapabilitiesElemScrobbler))
		Expect(manifest.Permissions.Http).NotTo(BeNil())
		Expect(manifest.Permissions.Http.Reason).To(Equal("To fetch metadata"))
	})

	It("should fail with proper error for non-existent manifest", func() {
		_, err := LoadManifest(filepath.Join(tempDir, "non-existent"))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to read manifest file"))
	})

	It("should fail with JSON parse error for invalid JSON", func() {
		// Create invalid JSON
		invalidJSON := `{
			"name": "test-plugin",
			"author": "Test Author"
			"version": "1.0.0"
			"description": "A test plugin",
			"capabilities": ["MetadataAgent"],
			"permissions": {}
		}`

		pluginDir := filepath.Join(tempDir, "invalid-json")
		Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(invalidJSON), 0600)).To(Succeed())

		// Test validation fails
		_, err := LoadManifest(pluginDir)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid manifest"))
	})

	It("should validate manifest against schema with detailed error for missing required field", func() {
		// Create manifest missing required name field
		manifestContent := `{
			"author": "Test Author",
			"version": "1.0.0",
			"description": "A test plugin",
			"website": "https://test.navidrome.org/test-plugin",
			"capabilities": ["MetadataAgent"],
			"permissions": {}
		}`

		pluginDir := filepath.Join(tempDir, "test-plugin")
		Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifestContent), 0600)).To(Succeed())

		_, err := LoadManifest(pluginDir)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("field name in PluginManifest: required"))
	})

	It("should validate manifest with wrong capability type", func() {
		// Create manifest with invalid capability
		manifestContent := `{
			"name": "test-plugin",
			"author": "Test Author",
			"version": "1.0.0",
			"description": "A test plugin",
			"website": "https://test.navidrome.org/test-plugin",
			"capabilities": ["UnsupportedService"],
			"permissions": {}
		}`

		pluginDir := filepath.Join(tempDir, "test-plugin")
		Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifestContent), 0600)).To(Succeed())

		_, err := LoadManifest(pluginDir)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid value"))
		Expect(err.Error()).To(ContainSubstring("UnsupportedService"))
	})

	It("should validate manifest with empty capabilities array", func() {
		// Create manifest with empty capabilities array
		manifestContent := `{
			"name": "test-plugin",
			"author": "Test Author",
			"version": "1.0.0",
			"description": "A test plugin",
			"website": "https://test.navidrome.org/test-plugin",
			"capabilities": [],
			"permissions": {}
		}`

		pluginDir := filepath.Join(tempDir, "test-plugin")
		Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifestContent), 0600)).To(Succeed())

		_, err := LoadManifest(pluginDir)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("field capabilities length: must be >= 1"))
	})
})
