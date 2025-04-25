package cmd

import (
	"io"
	"os"
	"path/filepath"

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

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		tempDir = GinkgoT().TempDir()

		// Setup config
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = tempDir

		// Create a command for testing
		cmd = &cobra.Command{Use: "test"}

		// Save original stdout
		origStdout = os.Stdout

		// Create a pipe to capture stdout
		var err error
		outReader, stdOut, err = os.Pipe()
		Expect(err).NotTo(HaveOccurred())

		// Replace stdout with our pipe
		os.Stdout = stdOut

		DeferCleanup(func() {
			// Restore original stdout
			os.Stdout = origStdout
		})
	})

	AfterEach(func() {
		// Restore original stdout after each test
		os.Stdout = origStdout

		// Close pipe
		if stdOut != nil {
			stdOut.Close()
		}
		if outReader != nil {
			outReader.Close()
		}
	})

	Describe("Plugin list command", func() {
		It("should list installed plugins", func() {
			// Create test plugin directories with manifest files
			plugin1Dir := filepath.Join(tempDir, "plugin1")
			Expect(os.MkdirAll(plugin1Dir, 0755)).To(Succeed())
			manifest1 := `{
				"name": "plugin1",
				"author": "Test Author",
				"version": "1.0.0",
				"description": "Test Plugin 1",
				"services": ["MediaMetadataService"]
			}`
			Expect(os.WriteFile(filepath.Join(plugin1Dir, "manifest.json"), []byte(manifest1), 0600)).To(Succeed())

			plugin2Dir := filepath.Join(tempDir, "plugin2")
			Expect(os.MkdirAll(plugin2Dir, 0755)).To(Succeed())
			manifest2 := `{
				"name": "plugin2",
				"author": "Another Author",
				"version": "2.1.0",
				"description": "Test Plugin 2",
				"services": ["ScrobblerService"]
			}`
			Expect(os.WriteFile(filepath.Join(plugin2Dir, "manifest.json"), []byte(manifest2), 0600)).To(Succeed())

			// Execute the list command that prints to stdout
			pluginList(cmd, []string{})

			// Make sure output is flushed
			stdOut.Close()

			// Read the output
			outputBytes, err := io.ReadAll(outReader)
			Expect(err).NotTo(HaveOccurred())
			output := string(outputBytes)

			// Check output
			Expect(output).To(ContainSubstring("plugin1"))
			Expect(output).To(ContainSubstring("Test Author"))
			Expect(output).To(ContainSubstring("1.0.0"))
			Expect(output).To(ContainSubstring("MediaMetadataService"))

			Expect(output).To(ContainSubstring("plugin2"))
			Expect(output).To(ContainSubstring("Another Author"))
			Expect(output).To(ContainSubstring("2.1.0"))
			Expect(output).To(ContainSubstring("ScrobblerService"))
		})
	})

	Describe("Plugin info command", func() {
		It("should display information about an installed plugin", func() {
			// Create test plugin directory with manifest
			pluginDir := filepath.Join(tempDir, "test-plugin")
			Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())
			manifest := `{
				"name": "test-plugin",
				"author": "Test Author",
				"version": "1.0.0",
				"description": "Plugin for testing",
				"services": ["MediaMetadataService", "ScrobblerService"]
			}`
			Expect(os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0600)).To(Succeed())

			// Execute the info command
			pluginInfo(cmd, []string{"test-plugin"})

			// Make sure output is flushed
			stdOut.Close()

			// Read the output
			outputBytes, err := io.ReadAll(outReader)
			Expect(err).NotTo(HaveOccurred())
			output := string(outputBytes)

			// Check output
			Expect(output).To(ContainSubstring("Name:        test-plugin"))
			Expect(output).To(ContainSubstring("Author:      Test Author"))
			Expect(output).To(ContainSubstring("Version:     1.0.0"))
			Expect(output).To(ContainSubstring("Description: Plugin for testing"))
			Expect(output).To(ContainSubstring("Services:    MediaMetadataService, ScrobblerService"))
		})
	})
})
