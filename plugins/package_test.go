package plugins

import (
	"archive/zip"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/plugins/schema"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Plugin Package", func() {
	var tempDir string
	var ndpPath string

	BeforeEach(func() {
		tempDir = GinkgoT().TempDir()

		// Create a test .ndp file
		ndpPath = filepath.Join(tempDir, "test-plugin.ndp")

		// Create the required plugin files
		manifestContent := []byte(`{
			"name": "test-plugin",
			"author": "Test Author",
			"version": "1.0.0",
			"description": "A test plugin",
			"website": "https://test.navidrome.org/test-plugin",
			"capabilities": ["MetadataAgent"],
			"permissions": {}
		}`)

		wasmContent := []byte("dummy wasm content")
		readmeContent := []byte("# Test Plugin\nThis is a test plugin")

		// Create the zip file
		zipFile, err := os.Create(ndpPath)
		Expect(err).NotTo(HaveOccurred())
		defer zipFile.Close()

		zipWriter := zip.NewWriter(zipFile)
		defer zipWriter.Close()

		// Add manifest.json
		manifestWriter, err := zipWriter.Create("manifest.json")
		Expect(err).NotTo(HaveOccurred())
		_, err = manifestWriter.Write(manifestContent)
		Expect(err).NotTo(HaveOccurred())

		// Add plugin.wasm
		wasmWriter, err := zipWriter.Create("plugin.wasm")
		Expect(err).NotTo(HaveOccurred())
		_, err = wasmWriter.Write(wasmContent)
		Expect(err).NotTo(HaveOccurred())

		// Add README.md
		readmeWriter, err := zipWriter.Create("README.md")
		Expect(err).NotTo(HaveOccurred())
		_, err = readmeWriter.Write(readmeContent)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should load and validate a plugin package", func() {
		pkg, err := LoadPackage(ndpPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(pkg).NotTo(BeNil())

		// Check manifest was parsed
		Expect(pkg.Manifest).NotTo(BeNil())
		Expect(pkg.Manifest.Name).To(Equal("test-plugin"))
		Expect(pkg.Manifest.Author).To(Equal("Test Author"))
		Expect(pkg.Manifest.Version).To(Equal("1.0.0"))
		Expect(pkg.Manifest.Description).To(Equal("A test plugin"))
		Expect(pkg.Manifest.Capabilities).To(HaveLen(1))
		Expect(pkg.Manifest.Capabilities[0]).To(Equal(schema.PluginManifestCapabilitiesElemMetadataAgent))

		// Check WASM file was loaded
		Expect(pkg.WasmBytes).NotTo(BeEmpty())

		// Check docs were loaded
		Expect(pkg.Docs).To(HaveKey("README.md"))
	})

	It("should extract a plugin package to a directory", func() {
		targetDir := filepath.Join(tempDir, "extracted")

		err := ExtractPackage(ndpPath, targetDir)
		Expect(err).NotTo(HaveOccurred())

		// Check files were extracted
		Expect(filepath.Join(targetDir, "manifest.json")).To(BeARegularFile())
		Expect(filepath.Join(targetDir, "plugin.wasm")).To(BeARegularFile())
		Expect(filepath.Join(targetDir, "README.md")).To(BeARegularFile())
	})

	It("should fail to load an invalid package", func() {
		// Create an invalid package (missing required files)
		invalidPath := filepath.Join(tempDir, "invalid.ndp")
		zipFile, err := os.Create(invalidPath)
		Expect(err).NotTo(HaveOccurred())

		zipWriter := zip.NewWriter(zipFile)
		// Only add a README, missing manifest and wasm
		readmeWriter, err := zipWriter.Create("README.md")
		Expect(err).NotTo(HaveOccurred())
		_, err = readmeWriter.Write([]byte("Invalid package"))
		Expect(err).NotTo(HaveOccurred())
		zipWriter.Close()
		zipFile.Close()

		// Test loading fails
		_, err = LoadPackage(invalidPath)
		Expect(err).To(HaveOccurred())
	})
})
