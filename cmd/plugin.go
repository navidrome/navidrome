package cmd

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins"
	"github.com/navidrome/navidrome/utils"
	"github.com/spf13/cobra"
)

func init() {
	pluginCmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage Navidrome plugins",
		Long:  "Commands for managing Navidrome plugins",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List installed plugins",
		Long:  "List all installed plugins with their metadata",
		Run:   pluginList,
	}

	infoCmd := &cobra.Command{
		Use:   "info [pluginPackage|pluginName]",
		Short: "Show details of a plugin",
		Long:  "Show detailed information about a plugin package (.ndp file) or an installed plugin",
		Args:  cobra.ExactArgs(1),
		Run:   pluginInfo,
	}

	installCmd := &cobra.Command{
		Use:   "install [pluginPackage]",
		Short: "Install a plugin from a .ndp file",
		Long:  "Install a Navidrome Plugin Package (.ndp) file",
		Args:  cobra.ExactArgs(1),
		Run:   pluginInstall,
	}

	removeCmd := &cobra.Command{
		Use:   "remove [pluginName]",
		Short: "Remove an installed plugin",
		Long:  "Remove a plugin by name",
		Args:  cobra.ExactArgs(1),
		Run:   pluginRemove,
	}

	updateCmd := &cobra.Command{
		Use:   "update [pluginPackage]",
		Short: "Update an existing plugin",
		Long:  "Update an installed plugin with a new version from a .ndp file",
		Args:  cobra.ExactArgs(1),
		Run:   pluginUpdate,
	}

	refreshCmd := &cobra.Command{
		Use:   "refresh [pluginName]",
		Short: "Reload a plugin without restarting Navidrome",
		Long:  "Reload and recompile a plugin without needing to restart Navidrome",
		Args:  cobra.ExactArgs(1),
		Run:   pluginRefresh,
	}

	devCmd := &cobra.Command{
		Use:   "dev [folder_path]",
		Short: "Create symlink to development folder",
		Long:  "Create a symlink from a plugin development folder to the plugins directory for easier development",
		Args:  cobra.ExactArgs(1),
		Run:   pluginDev,
	}

	pluginCmd.AddCommand(listCmd, infoCmd, installCmd, removeCmd, updateCmd, refreshCmd, devCmd)
	rootCmd.AddCommand(pluginCmd)
}

func pluginList(cmd *cobra.Command, args []string) {
	// Get plugins directory
	pluginsDir := conf.Server.Plugins.Folder

	// Create a tab writer for aligned output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tAUTHOR\tVERSION\tSERVICES\tDESCRIPTION")

	// Scan plugin directories
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		log.Error("Failed to read plugins directory", "folder", pluginsDir, err)
		return
	}

	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files
		if name[0] == '.' {
			continue
		}

		pluginPath := filepath.Join(pluginsDir, name)

		// Get file info to check if it's a directory or symlink
		info, err := os.Lstat(pluginPath)
		if err != nil {
			log.Error("Failed to stat entry", "path", pluginPath, err)
			continue
		}

		isSymlink := info.Mode()&os.ModeSymlink != 0
		isDir := info.IsDir()

		// Skip if not a directory or symlink
		if !isDir && !isSymlink {
			continue
		}

		// If it's a symlink, resolve it
		pluginDir := pluginPath
		if isSymlink {
			targetDir, err := os.Readlink(pluginPath)
			if err != nil {
				log.Error("Failed to resolve symlink", "path", pluginPath, err)
				continue
			}

			// If target is a relative path, make it absolute
			if !filepath.IsAbs(targetDir) {
				targetDir = filepath.Join(filepath.Dir(pluginPath), targetDir)
			}

			// Verify that the target is a directory
			targetInfo, err := os.Stat(targetDir)
			if err != nil {
				log.Error("Failed to stat symlink target", "path", targetDir, err)
				continue
			}

			if !targetInfo.IsDir() {
				log.Debug("Symlink target is not a directory, skipping", "name", name, "target", targetDir)
				continue
			}
		}

		manifest, err := plugins.LoadManifest(pluginDir)
		if err != nil {
			fmt.Fprintf(w, "%s\tERROR\tERROR\tERROR\t%v\n", name, err)
			continue
		}

		// Format services as comma-separated list
		services := manifest.Services[0]
		for i := 1; i < len(manifest.Services); i++ {
			services += ", " + manifest.Services[i]
		}

		// Mark symlinks with an indicator
		nameDisplay := manifest.Name
		if isSymlink {
			nameDisplay = nameDisplay + " (dev)"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			nameDisplay,
			cmp.Or(manifest.Author, "-"),
			cmp.Or(manifest.Version, "-"),
			services,
			cmp.Or(manifest.Description, "-"))
	}
	w.Flush()
}

func pluginInfo(cmd *cobra.Command, args []string) {
	path := args[0]
	pluginsDir := conf.Server.Plugins.Folder

	// Check if it's a file or installed plugin name
	var manifest *plugins.PluginManifest
	var fileInfo os.FileInfo
	var err error
	var fileSize int64
	var fileHash string

	if filepath.Ext(path) == ".ndp" {
		// It's a file path
		if !utils.FileExists(path) {
			log.Fatal("Plugin package not found", "path", path)
		}

		pkg, err := plugins.LoadPackage(path)
		if err != nil {
			log.Fatal("Failed to load plugin package", err)
		}
		manifest = pkg.Manifest

		// Get file information
		fileInfo, err = os.Stat(path)
		if err != nil {
			log.Error("Failed to get file information", err)
		} else {
			fileSize = fileInfo.Size()
			fileHash = calculateSHA256(path)
		}
	} else {
		// Assumed to be a plugin name
		pluginDir := filepath.Join(pluginsDir, path)
		if !utils.FileExists(pluginDir) {
			log.Fatal("Plugin not found", "name", path)
		}

		manifest, err = plugins.LoadManifest(pluginDir)
		if err != nil {
			log.Fatal("Failed to load plugin manifest", err)
		}
	}

	// Print plugin information
	fmt.Println("Plugin Information:")
	fmt.Printf("  Name:        %s\n", manifest.Name)
	fmt.Printf("  Author:      %s\n", manifest.Author)
	fmt.Printf("  Version:     %s\n", manifest.Version)
	fmt.Printf("  Description: %s\n", manifest.Description)

	fmt.Print("  Services:    ")
	for i, service := range manifest.Services {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(service)
	}
	fmt.Println()

	// Print file information if available
	if fileInfo != nil {
		fmt.Println("\nPackage Information:")
		fmt.Printf("  File:        %s\n", path)
		fmt.Printf("  Size:        %d bytes (%.2f KB)\n", fileSize, float64(fileSize)/1024)
		fmt.Printf("  SHA-256:     %s\n", fileHash)
		fmt.Printf("  Modified:    %s\n", fileInfo.ModTime().Format(time.RFC3339))
	}
}

func pluginInstall(cmd *cobra.Command, args []string) {
	ndpPath := args[0]
	pluginsDir := conf.Server.Plugins.Folder

	// Check if file exists and is a .ndp file
	if !utils.FileExists(ndpPath) {
		log.Fatal("Plugin package not found", "path", ndpPath)
	}
	if filepath.Ext(ndpPath) != ".ndp" {
		log.Fatal("Not a valid plugin package", "path", ndpPath, "expected extension", ".ndp")
	}

	// Load and validate the package
	pkg, err := plugins.LoadPackage(ndpPath)
	if err != nil {
		log.Fatal("Failed to load plugin package", err)
	}

	// Create target directory based on plugin name
	targetDir := filepath.Join(pluginsDir, pkg.Manifest.Name)

	// Check if plugin already exists
	if utils.FileExists(targetDir) {
		log.Fatal("Plugin already installed", "name", pkg.Manifest.Name, "path", targetDir,
			"use", "navidrome plugin update")
	}

	// Extract the package
	if err := plugins.ExtractPackage(ndpPath, targetDir); err != nil {
		log.Fatal("Failed to extract plugin package", err)
	}

	// Set correct permissions
	ensurePluginDirPermissions(targetDir)

	fmt.Printf("Plugin '%s' v%s installed successfully\n", pkg.Manifest.Name, pkg.Manifest.Version)
}

func pluginRemove(cmd *cobra.Command, args []string) {
	pluginName := args[0]
	pluginsDir := conf.Server.Plugins.Folder
	pluginDir := filepath.Join(pluginsDir, pluginName)

	// Check if plugin exists
	if !utils.FileExists(pluginDir) {
		log.Fatal("Plugin not found", "name", pluginName, "path", pluginDir)
	}

	// Check if it's a directory or a symlink
	info, err := os.Lstat(pluginDir)
	if err != nil {
		log.Fatal("Failed to stat plugin", "name", pluginName, "path", pluginDir, err)
	}

	isSymlink := info.Mode()&os.ModeSymlink != 0
	isDir := info.IsDir()

	if !isDir && !isSymlink {
		log.Fatal("Not a valid plugin directory or symlink", "path", pluginDir)
	}

	if isSymlink {
		// For symlinked plugins (dev mode), just remove the symlink
		targetDir, err := os.Readlink(pluginDir)
		if err != nil {
			log.Error("Failed to resolve symlink", "path", pluginDir, err)
		} else {
			log.Debug("Removing symlink only, keeping target directory", "symlink", pluginDir, "target", targetDir)
		}

		if err := os.Remove(pluginDir); err != nil {
			log.Fatal("Failed to remove plugin symlink", "name", pluginName, err)
		}
		fmt.Printf("Development plugin symlink '%s' removed successfully (target directory preserved)\n", pluginName)
	} else {
		// For regular plugins, remove the entire directory
		if err := os.RemoveAll(pluginDir); err != nil {
			log.Fatal("Failed to remove plugin directory", "name", pluginName, err)
		}
		fmt.Printf("Plugin '%s' removed successfully\n", pluginName)
	}
}

func pluginUpdate(cmd *cobra.Command, args []string) {
	ndpPath := args[0]
	pluginsDir := conf.Server.Plugins.Folder

	// Check if file exists and is a .ndp file
	if !utils.FileExists(ndpPath) {
		log.Fatal("Plugin package not found", "path", ndpPath)
	}
	if filepath.Ext(ndpPath) != ".ndp" {
		log.Fatal("Not a valid plugin package", "path", ndpPath, "expected extension", ".ndp")
	}

	// Load and validate the package
	pkg, err := plugins.LoadPackage(ndpPath)
	if err != nil {
		log.Fatal("Failed to load plugin package", err)
	}

	// Check if plugin exists
	targetDir := filepath.Join(pluginsDir, pkg.Manifest.Name)
	if !utils.FileExists(targetDir) {
		log.Fatal("Plugin not found", "name", pkg.Manifest.Name, "path", targetDir,
			"use", "navidrome plugin install")
	}

	// Create a backup of the existing plugin
	backupDir := targetDir + ".bak." + time.Now().Format("20060102150405")
	if err := os.Rename(targetDir, backupDir); err != nil {
		log.Fatal("Failed to backup existing plugin", err)
	}

	// Extract the new package
	if err := plugins.ExtractPackage(ndpPath, targetDir); err != nil {
		// Restore backup if extraction failed
		os.RemoveAll(targetDir)
		// Explicitly ignore error here, as we are already in a fatal error path
		_ = os.Rename(backupDir, targetDir)
		log.Fatal("Failed to extract plugin package", err)
	}

	ensurePluginDirPermissions(targetDir)

	// Remove the backup
	os.RemoveAll(backupDir)

	fmt.Printf("Plugin '%s' updated to v%s successfully\n", pkg.Manifest.Name, pkg.Manifest.Version)
}

// ensurePluginDirPermissions ensures the plugin directory has the correct permissions
func ensurePluginDirPermissions(dir string) {
	if err := os.Chmod(dir, 0700); err != nil {
		log.Error("Failed to set plugin directory permissions", "dir", dir, err)
	}

	// Apply same permissions to all files in the directory
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Error("Failed to read plugin directory", "dir", dir, err)
		return
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		info, err := os.Stat(path)
		if err != nil {
			log.Error("Failed to stat file", "path", path, err)
			continue
		}

		// Different permissions for directories vs files
		mode := os.FileMode(0600) // read-write for files
		if info.IsDir() {
			mode = os.FileMode(0700) // read-write-execute for directories
			// Recursively set permissions for subdirectories
			ensurePluginDirPermissions(path)
		}

		if err := os.Chmod(path, mode); err != nil {
			log.Error("Failed to set file permissions", "path", path, err)
		}
	}
}

// calculateSHA256 computes the SHA-256 hash of a file
func calculateSHA256(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		log.Error("Failed to open file for hashing", err)
		return "N/A"
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		log.Error("Failed to calculate hash", err)
		return "N/A"
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

func pluginRefresh(cmd *cobra.Command, args []string) {
	pluginName := args[0]
	pluginsDir := conf.Server.Plugins.Folder
	pluginDir := filepath.Join(pluginsDir, pluginName)

	// Check if plugin exists
	if !utils.FileExists(pluginDir) {
		log.Fatal("Plugin not found", "name", pluginName, "path", pluginDir)
	}

	// Check if it's a directory or a symlink to a directory
	lstat, err := os.Lstat(pluginDir)
	if err != nil {
		log.Fatal("Failed to stat plugin", "name", pluginName, "path", pluginDir, err)
	}

	isSymlink := lstat.Mode()&os.ModeSymlink != 0

	// If it's a symlink, verify the target is a directory
	if isSymlink {
		// Resolve the symlink target
		targetDir, err := os.Readlink(pluginDir)
		if err != nil {
			log.Fatal("Failed to resolve symlink", "path", pluginDir, err)
		}

		// If target is a relative path, make it absolute
		if !filepath.IsAbs(targetDir) {
			targetDir = filepath.Join(filepath.Dir(pluginDir), targetDir)
		}

		// Verify the target exists and is a directory
		targetInfo, err := os.Stat(targetDir)
		if err != nil {
			log.Fatal("Failed to access symlink target", "target", targetDir, err)
		}

		if !targetInfo.IsDir() {
			log.Fatal("Symlink target is not a directory", "path", targetDir)
		}

		log.Debug("Processing symlinked plugin", "name", pluginName, "link", pluginDir, "target", targetDir)
	} else if !lstat.IsDir() {
		// If it's not a symlink and not a directory, it's invalid
		log.Fatal("Not a valid plugin directory", "path", pluginDir)
	}

	fmt.Printf("Refreshing plugin '%s'...\n", pluginName)

	// Get the plugin manager
	mgr := plugins.GetManager()

	// Scan all plugins (this will clear and recompile all plugins including the target one)
	log.Debug("Scanning plugins directory", "path", pluginsDir)
	mgr.ScanPlugins()

	log.Info("Waiting for plugin compilation to complete", "name", pluginName)

	// Load the plugin to wait for compilation to complete
	plugin := mgr.LoadPlugin(pluginName, "")
	if plugin == nil {
		log.Fatal("Failed to load refreshed plugin - compilation may have failed", "name", pluginName)
	}

	log.Info("Plugin compilation completed successfully", "name", pluginName)
	fmt.Printf("Plugin '%s' refreshed successfully\n", pluginName)
}

func pluginDev(cmd *cobra.Command, args []string) {
	sourcePath, err := filepath.Abs(args[0])
	if err != nil {
		log.Fatal("Invalid path", "path", args[0], err)
	}
	pluginsDir := conf.Server.Plugins.Folder

	// Check if source folder exists
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		log.Fatal("Source folder not found", "path", sourcePath, err)
	}
	if !sourceInfo.IsDir() {
		log.Fatal("Source path is not a directory", "path", sourcePath)
	}

	// Validate that the source folder has the required plugin files
	manifestPath := filepath.Join(sourcePath, "manifest.json")
	if !utils.FileExists(manifestPath) {
		log.Fatal("Source folder missing manifest.json", "path", sourcePath)
	}

	// Load and validate manifest to extract plugin name
	manifest, err := plugins.LoadManifest(sourcePath)
	if err != nil {
		log.Fatal("Failed to load plugin manifest", "path", manifestPath, err)
	}

	// Use the plugin name from the manifest if available
	pluginName := manifest.Name
	if pluginName == "" {
		// Fall back to directory name if manifest doesn't have a name
		pluginName = filepath.Base(sourcePath)
	}

	targetPath := filepath.Join(pluginsDir, pluginName)

	// Check if target already exists
	if utils.FileExists(targetPath) {
		// Check if it's already a symlink to our source
		existingLink, err := os.Readlink(targetPath)
		if err == nil && existingLink == sourcePath {
			fmt.Printf("Symlink already exists and points to the correct source\n")
			return
		}

		// Handle case where target exists but is not a symlink to our source
		fmt.Printf("Target path '%s' already exists.\n", targetPath)
		fmt.Print("Do you want to replace it? (y/N): ")
		var response string
		_, err = fmt.Scanln(&response)
		if err != nil || strings.ToLower(response) != "y" {
			if err != nil {
				log.Debug("Error reading input, assuming 'no'", err)
			}
			fmt.Println("Operation canceled")
			return
		}

		// Remove existing target
		if err := os.RemoveAll(targetPath); err != nil {
			log.Fatal("Failed to remove existing target", "path", targetPath, err)
		}
	}

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		log.Fatal("Failed to create plugins directory", "path", filepath.Dir(targetPath), err)
	}

	// Create the symlink
	if err := os.Symlink(sourcePath, targetPath); err != nil {
		log.Fatal("Failed to create symlink", "source", sourcePath, "target", targetPath, err)
	}

	fmt.Printf("Development symlink created: '%s' -> '%s'\n", targetPath, sourcePath)
	fmt.Println("Plugin can be refreshed with: navidrome plugin refresh", pluginName)
}
