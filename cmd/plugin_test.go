package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("Plugin CLI Commands", func() {
	var tempDir string
	var cmd *cobra.Command
	var stdOut *os.File
	var origStdout *os.File
	var outReader *os.File

	// Helper to create a test plugin with the given name and details
	createTestPlugin := func(name, author, version string, capabilities []string) string {
		pluginDir := filepath.Join(tempDir, name)
		Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())

		// Create a properly formatted capabilities JSON array
		capabilitiesJSON := `"` + strings.Join(capabilities, `", "`) + `"`

		manifest := `{
			"name": "` + name + `",
			"author": "` + author + `",
			"version": "` + version + `",
			"description": "Plugin for testing",
			"website": "https://test.navidrome.org/` + name + `",
			"capabilities": [` + capabilitiesJSON + `],
			"permissions": {}
		}`

		Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0600)).To(Succeed())

		// Create a dummy WASM file
		wasmContent := []byte("dummy wasm content for testing")
		Expect(os.WriteFile(filepath.Join(pluginDir, "plugin.wasm"), wasmContent, 0600)).To(Succeed())

		return pluginDir
	}

	// Helper to execute a command and return captured output
	captureOutput := func(reader io.Reader) string {
		stdOut.Close()
		outputBytes, err := io.ReadAll(reader)
		Expect(err).NotTo(HaveOccurred())
		return string(outputBytes)
	}

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		tempDir = GinkgoT().TempDir()

		// Setup config
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = tempDir

		// Create a command for testing
		cmd = &cobra.Command{Use: "test"}

		// Setup stdout capture
		origStdout = os.Stdout
		var err error
		outReader, stdOut, err = os.Pipe()
		Expect(err).NotTo(HaveOccurred())
		os.Stdout = stdOut

		DeferCleanup(func() {
			os.Stdout = origStdout
		})
	})

	AfterEach(func() {
		os.Stdout = origStdout
		if stdOut != nil {
			stdOut.Close()
		}
		if outReader != nil {
			outReader.Close()
		}
	})

	Describe("Plugin list command", func() {
		It("should list installed plugins", func() {
			// Create test plugins
			createTestPlugin("plugin1", "Test Author", "1.0.0", []string{"MetadataAgent"})
			createTestPlugin("plugin2", "Another Author", "2.1.0", []string{"Scrobbler"})

			// Execute command
			pluginList(cmd, []string{})

			// Verify output
			output := captureOutput(outReader)

			Expect(output).To(ContainSubstring("plugin1"))
			Expect(output).To(ContainSubstring("Test Author"))
			Expect(output).To(ContainSubstring("1.0.0"))
			Expect(output).To(ContainSubstring("MetadataAgent"))

			Expect(output).To(ContainSubstring("plugin2"))
			Expect(output).To(ContainSubstring("Another Author"))
			Expect(output).To(ContainSubstring("2.1.0"))
			Expect(output).To(ContainSubstring("Scrobbler"))
		})
	})

	Describe("Plugin info command", func() {
		It("should display information about an installed plugin", func() {
			// Create test plugin with multiple capabilities
			createTestPlugin("test-plugin", "Test Author", "1.0.0",
				[]string{"MetadataAgent", "Scrobbler"})

			// Execute command
			pluginInfo(cmd, []string{"test-plugin"})

			// Verify output
			output := captureOutput(outReader)

			Expect(output).To(ContainSubstring("Name:        test-plugin"))
			Expect(output).To(ContainSubstring("Author:      Test Author"))
			Expect(output).To(ContainSubstring("Version:     1.0.0"))
			Expect(output).To(ContainSubstring("Description: Plugin for testing"))
			Expect(output).To(ContainSubstring("Capabilities:    MetadataAgent, Scrobbler"))
		})
	})

	Describe("Plugin remove command", func() {
		It("should remove a regular plugin directory", func() {
			// Create test plugin
			pluginDir := createTestPlugin("regular-plugin", "Test Author", "1.0.0",
				[]string{"MetadataAgent"})

			// Execute command
			pluginRemove(cmd, []string{"regular-plugin"})

			// Verify output
			output := captureOutput(outReader)
			Expect(output).To(ContainSubstring("Plugin 'regular-plugin' removed successfully"))

			// Verify directory is actually removed
			_, err := os.Stat(pluginDir)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		It("should remove only the symlink for a development plugin", func() {
			// Create a real source directory
			sourceDir := filepath.Join(GinkgoT().TempDir(), "dev-plugin-source")
			Expect(os.MkdirAll(sourceDir, 0755)).To(Succeed())

			manifest := `{
				"name": "dev-plugin",
				"author": "Dev Author",
				"version": "0.1.0",
				"description": "Development plugin for testing",
				"website": "https://test.navidrome.org/dev-plugin",
				"capabilities": ["Scrobbler"],
				"permissions": {}
			}`
			Expect(os.WriteFile(filepath.Join(sourceDir, "manifest.json"), []byte(manifest), 0600)).To(Succeed())

			// Create a dummy WASM file
			wasmContent := []byte("dummy wasm content for testing")
			Expect(os.WriteFile(filepath.Join(sourceDir, "plugin.wasm"), wasmContent, 0600)).To(Succeed())

			// Create a symlink in the plugins directory
			symlinkPath := filepath.Join(tempDir, "dev-plugin")
			Expect(os.Symlink(sourceDir, symlinkPath)).To(Succeed())

			// Execute command
			pluginRemove(cmd, []string{"dev-plugin"})

			// Verify output
			output := captureOutput(outReader)
			Expect(output).To(ContainSubstring("Development plugin symlink 'dev-plugin' removed successfully"))
			Expect(output).To(ContainSubstring("target directory preserved"))

			// Verify the symlink is removed but source directory exists
			_, err := os.Lstat(symlinkPath)
			Expect(os.IsNotExist(err)).To(BeTrue())

			_, err = os.Stat(sourceDir)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
