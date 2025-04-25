package plugins

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin Manifest", func() {
	var tempDir string

	BeforeEach(func() {
		tempDir = GinkgoT().TempDir()
	})

	It("should load and parse a valid manifest", func() {
		// Create test manifest
		manifestContent := `{
			"name": "test-plugin",
			"author": "Test Author",
			"version": "1.0.0",
			"description": "A test plugin",
			"services": ["MediaMetadataService", "ScrobblerService"]
		}`

		pluginDir := filepath.Join(tempDir, "test-plugin")
		Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifestContent), 0600)).To(Succeed())

		// Test loading the manifest
		manifest, err := LoadManifest(pluginDir)
		Expect(err).NotTo(HaveOccurred())
		Expect(manifest).NotTo(BeNil())
		Expect(manifest.Name).To(Equal("test-plugin"))
		Expect(manifest.Author).To(Equal("Test Author"))
		Expect(manifest.Version).To(Equal("1.0.0"))
		Expect(manifest.Description).To(Equal("A test plugin"))
		Expect(manifest.Services).To(ConsistOf("MediaMetadataService", "ScrobblerService"))
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
			"services": ["MediaMetadataService"]
		}`

		pluginDir := filepath.Join(tempDir, "invalid-json")
		Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(invalidJSON), 0600)).To(Succeed())

		// Test validation fails
		_, err := LoadManifest(pluginDir)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to parse manifest JSON"))
	})

	It("should validate manifest against schema with detailed error for missing required field", func() {
		// Create invalid manifest (missing required 'name' field)
		invalidManifest := `{
			"author": "Test Author",
			"version": "1.0.0",
			"description": "A test plugin",
			"services": ["MediaMetadataService"]
		}`

		pluginDir := filepath.Join(tempDir, "missing-name")
		Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(invalidManifest), 0600)).To(Succeed())

		// Test validation fails
		_, err := LoadManifest(pluginDir)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("(root): name is required"))
	})

	It("should validate manifest with wrong service type", func() {
		// Create invalid manifest with invalid service type
		invalidManifest := `{
			"name": "test-plugin",
			"author": "Test Author",
			"version": "1.0.0",
			"description": "A test plugin",
			"services": ["UnsupportedService"]
		}`

		pluginDir := filepath.Join(tempDir, "invalid-service")
		Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(invalidManifest), 0600)).To(Succeed())

		// Test validation fails
		_, err := LoadManifest(pluginDir)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("services.0: services.0 must be one of the following"))
		Expect(err.Error()).To(ContainSubstring("UnsupportedService"))
	})

	It("should validate manifest with empty services array", func() {
		// Create invalid manifest with empty services array
		invalidManifest := `{
			"name": "test-plugin",
			"author": "Test Author",
			"version": "1.0.0",
			"description": "A test plugin",
			"services": []
		}`

		pluginDir := filepath.Join(tempDir, "empty-services")
		Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(invalidManifest), 0600)).To(Succeed())

		// Test validation fails
		_, err := LoadManifest(pluginDir)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("services: Array must have at least 1 items"))
	})
})
