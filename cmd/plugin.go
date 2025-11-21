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
	"github.com/navidrome/navidrome/plugins/schema"
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/spf13/cobra"
)

const (
	pluginPackageExtension = ".ndp"
	pluginDirPermissions   = 0700
	pluginFilePermissions  = 0600
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

// Validation helpers

func validatePluginPackageFile(path string) error {
	if !utils.FileExists(path) {
		return fmt.Errorf("plugin package not found: %s", path)
	}
	if filepath.Ext(path) != pluginPackageExtension {
		return fmt.Errorf("not a valid plugin package: %s (expected %s extension)", path, pluginPackageExtension)
	}
	return nil
}

func validatePluginDirectory(pluginsDir, pluginName string) (string, error) {
	pluginDir := filepath.Join(pluginsDir, pluginName)
	if !utils.FileExists(pluginDir) {
		return "", fmt.Errorf("plugin not found: %s (path: %s)", pluginName, pluginDir)
	}
	return pluginDir, nil
}

func resolvePluginPath(pluginDir string) (resolvedPath string, isSymlink bool, err error) {
	// Check if it's a directory or a symlink
	lstat, err := os.Lstat(pluginDir)
	if err != nil {
		return "", false, fmt.Errorf("failed to stat plugin: %w", err)
	}

	isSymlink = lstat.Mode()&os.ModeSymlink != 0

	if isSymlink {
		// Resolve the symlink target
		targetDir, err := os.Readlink(pluginDir)
		if err != nil {
			return "", true, fmt.Errorf("failed to resolve symlink: %w", err)
		}

		// If target is a relative path, make it absolute
		if !filepath.IsAbs(targetDir) {
			targetDir = filepath.Join(filepath.Dir(pluginDir), targetDir)
		}

		// Verify the target exists and is a directory
		targetInfo, err := os.Stat(targetDir)
		if err != nil {
			return "", true, fmt.Errorf("failed to access symlink target %s: %w", targetDir, err)
		}

		if !targetInfo.IsDir() {
			return "", true, fmt.Errorf("symlink target is not a directory: %s", targetDir)
		}

		return targetDir, true, nil
	} else if !lstat.IsDir() {
		return "", false, fmt.Errorf("not a valid plugin directory: %s", pluginDir)
	}

	return pluginDir, false, nil
}

// Package handling helpers

func loadAndValidatePackage(ndpPath string) (*plugins.PluginPackage, error) {
	if err := validatePluginPackageFile(ndpPath); err != nil {
		return nil, err
	}

	pkg, err := plugins.LoadPackage(ndpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin package: %w", err)
	}

	return pkg, nil
}

func extractAndSetupPlugin(ndpPath, targetDir string) error {
	if err := plugins.ExtractPackage(ndpPath, targetDir); err != nil {
		return fmt.Errorf("failed to extract plugin package: %w", err)
	}

	ensurePluginDirPermissions(targetDir)
	return nil
}

// Display helpers

func displayPluginTableRow(w *tabwriter.Writer, discovery plugins.PluginDiscoveryEntry) {
	if discovery.Error != nil {
		// Handle global errors (like directory read failure)
		if discovery.ID == "" {
			log.Error("Failed to read plugins directory", "folder", conf.Server.Plugins.Folder, discovery.Error)
			return
		}
		// Handle individual plugin errors - show them in the table
		fmt.Fprintf(w, "%s\tERROR\tERROR\tERROR\tERROR\t%v\n", discovery.ID, discovery.Error)
		return
	}

	// Mark symlinks with an indicator
	nameDisplay := discovery.Manifest.Name
	if discovery.IsSymlink {
		nameDisplay = nameDisplay + " (dev)"
	}

	// Convert capabilities to strings
	capabilities := slice.Map(discovery.Manifest.Capabilities, func(cap schema.PluginManifestCapabilitiesElem) string {
		return string(cap)
	})

	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
		discovery.ID,
		nameDisplay,
		cmp.Or(discovery.Manifest.Author, "-"),
		cmp.Or(discovery.Manifest.Version, "-"),
		strings.Join(capabilities, ", "),
		cmp.Or(discovery.Manifest.Description, "-"))
}

func displayTypedPermissions(permissions schema.PluginManifestPermissions, indent string) {
	if permissions.Http != nil {
		fmt.Printf("%shttp:\n", indent)
		fmt.Printf("%s  Reason: %s\n", indent, permissions.Http.Reason)
		fmt.Printf("%s  Allow Local Network: %t\n", indent, permissions.Http.AllowLocalNetwork)
		fmt.Printf("%s  Allowed URLs:\n", indent)
		for urlPattern, methodEnums := range permissions.Http.AllowedUrls {
			methods := make([]string, len(methodEnums))
			for i, methodEnum := range methodEnums {
				methods[i] = string(methodEnum)
			}
			fmt.Printf("%s    %s: [%s]\n", indent, urlPattern, strings.Join(methods, ", "))
		}
		fmt.Println()
	}

	if permissions.Config != nil {
		fmt.Printf("%sconfig:\n", indent)
		fmt.Printf("%s  Reason: %s\n", indent, permissions.Config.Reason)
		fmt.Println()
	}

	if permissions.Scheduler != nil {
		fmt.Printf("%sscheduler:\n", indent)
		fmt.Printf("%s  Reason: %s\n", indent, permissions.Scheduler.Reason)
		fmt.Println()
	}

	if permissions.Websocket != nil {
		fmt.Printf("%swebsocket:\n", indent)
		fmt.Printf("%s  Reason: %s\n", indent, permissions.Websocket.Reason)
		fmt.Printf("%s  Allow Local Network: %t\n", indent, permissions.Websocket.AllowLocalNetwork)
		fmt.Printf("%s  Allowed URLs: [%s]\n", indent, strings.Join(permissions.Websocket.AllowedUrls, ", "))
		fmt.Println()
	}

	if permissions.Cache != nil {
		fmt.Printf("%scache:\n", indent)
		fmt.Printf("%s  Reason: %s\n", indent, permissions.Cache.Reason)
		fmt.Println()
	}

	if permissions.Artwork != nil {
		fmt.Printf("%sartwork:\n", indent)
		fmt.Printf("%s  Reason: %s\n", indent, permissions.Artwork.Reason)
		fmt.Println()
	}

	if permissions.Subsonicapi != nil {
		allowedUsers := "All Users"
		if len(permissions.Subsonicapi.AllowedUsernames) > 0 {
			allowedUsers = strings.Join(permissions.Subsonicapi.AllowedUsernames, ", ")
		}
		fmt.Printf("%ssubsonicapi:\n", indent)
		fmt.Printf("%s  Reason: %s\n", indent, permissions.Subsonicapi.Reason)
		fmt.Printf("%s  Allow Admins: %t\n", indent, permissions.Subsonicapi.AllowAdmins)
		fmt.Printf("%s  Allowed Usernames: [%s]\n", indent, allowedUsers)
		fmt.Println()
	}
}

func displayPluginDetails(manifest *schema.PluginManifest, fileInfo *pluginFileInfo, permInfo *pluginPermissionInfo) {
	fmt.Println("\nPlugin Information:")
	fmt.Printf("  Name:        %s\n", manifest.Name)
	fmt.Printf("  Author:      %s\n", manifest.Author)
	fmt.Printf("  Version:     %s\n", manifest.Version)
	fmt.Printf("  Description: %s\n", manifest.Description)

	fmt.Print("  Capabilities:    ")
	capabilities := make([]string, len(manifest.Capabilities))
	for i, cap := range manifest.Capabilities {
		capabilities[i] = string(cap)
	}
	fmt.Print(strings.Join(capabilities, ", "))
	fmt.Println()

	// Display manifest permissions using the typed permissions
	fmt.Println("  Required Permissions:")
	displayTypedPermissions(manifest.Permissions, "    ")

	// Print file information if available
	if fileInfo != nil {
		fmt.Println("Package Information:")
		fmt.Printf("  File:        %s\n", fileInfo.path)
		fmt.Printf("  Size:        %d bytes (%.2f KB)\n", fileInfo.size, float64(fileInfo.size)/1024)
		fmt.Printf("  SHA-256:     %s\n", fileInfo.hash)
		fmt.Printf("  Modified:    %s\n", fileInfo.modTime.Format(time.RFC3339))
	}

	// Print file permissions information if available
	if permInfo != nil {
		fmt.Println("File Permissions:")
		fmt.Printf("  Plugin Directory: %s (%s)\n", permInfo.dirPath, permInfo.dirMode)
		if permInfo.isSymlink {
			fmt.Printf("  Symlink Target:   %s (%s)\n", permInfo.targetPath, permInfo.targetMode)
		}
		fmt.Printf("  Manifest File:    %s\n", permInfo.manifestMode)
		if permInfo.wasmMode != "" {
			fmt.Printf("  WASM File:        %s\n", permInfo.wasmMode)
		}
	}
}

type pluginFileInfo struct {
	path    string
	size    int64
	hash    string
	modTime time.Time
}

type pluginPermissionInfo struct {
	dirPath      string
	dirMode      string
	isSymlink    bool
	targetPath   string
	targetMode   string
	manifestMode string
	wasmMode     string
}

func getFileInfo(path string) *pluginFileInfo {
	fileInfo, err := os.Stat(path)
	if err != nil {
		log.Error("Failed to get file information", err)
		return nil
	}

	return &pluginFileInfo{
		path:    path,
		size:    fileInfo.Size(),
		hash:    calculateSHA256(path),
		modTime: fileInfo.ModTime(),
	}
}

func getPermissionInfo(pluginDir string) *pluginPermissionInfo {
	// Get plugin directory permissions
	dirInfo, err := os.Lstat(pluginDir)
	if err != nil {
		log.Error("Failed to get plugin directory permissions", err)
		return nil
	}

	permInfo := &pluginPermissionInfo{
		dirPath: pluginDir,
		dirMode: dirInfo.Mode().String(),
	}

	// Check if it's a symlink
	if dirInfo.Mode()&os.ModeSymlink != 0 {
		permInfo.isSymlink = true

		// Get target path and permissions
		targetPath, err := os.Readlink(pluginDir)
		if err == nil {
			if !filepath.IsAbs(targetPath) {
				targetPath = filepath.Join(filepath.Dir(pluginDir), targetPath)
			}
			permInfo.targetPath = targetPath

			if targetInfo, err := os.Stat(targetPath); err == nil {
				permInfo.targetMode = targetInfo.Mode().String()
			}
		}
	}

	// Get manifest file permissions
	manifestPath := filepath.Join(pluginDir, "manifest.json")
	if manifestInfo, err := os.Stat(manifestPath); err == nil {
		permInfo.manifestMode = manifestInfo.Mode().String()
	}

	// Get WASM file permissions (look for .wasm files)
	entries, err := os.ReadDir(pluginDir)
	if err == nil {
		for _, entry := range entries {
			if filepath.Ext(entry.Name()) == ".wasm" {
				wasmPath := filepath.Join(pluginDir, entry.Name())
				if wasmInfo, err := os.Stat(wasmPath); err == nil {
					permInfo.wasmMode = wasmInfo.Mode().String()
					break // Just show the first WASM file found
				}
			}
		}
	}

	return permInfo
}

// Command implementations

func pluginList(cmd *cobra.Command, args []string) {
	discoveries := plugins.DiscoverPlugins(conf.Server.Plugins.Folder)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tAUTHOR\tVERSION\tCAPABILITIES\tDESCRIPTION")

	for _, discovery := range discoveries {
		displayPluginTableRow(w, discovery)
	}
	w.Flush()
}

func pluginInfo(cmd *cobra.Command, args []string) {
	path := args[0]
	pluginsDir := conf.Server.Plugins.Folder

	var manifest *schema.PluginManifest
	var fileInfo *pluginFileInfo
	var permInfo *pluginPermissionInfo

	if filepath.Ext(path) == pluginPackageExtension {
		// It's a package file
		pkg, err := loadAndValidatePackage(path)
		if err != nil {
			log.Fatal("Failed to load plugin package", err)
		}
		manifest = pkg.Manifest
		fileInfo = getFileInfo(path)
		// No permission info for package files
	} else {
		// It's a plugin name
		pluginDir, err := validatePluginDirectory(pluginsDir, path)
		if err != nil {
			log.Fatal("Plugin validation failed", err)
		}

		manifest, err = plugins.LoadManifest(pluginDir)
		if err != nil {
			log.Fatal("Failed to load plugin manifest", err)
		}

		// Get permission info for installed plugins
		permInfo = getPermissionInfo(pluginDir)
	}

	displayPluginDetails(manifest, fileInfo, permInfo)
}

func pluginInstall(cmd *cobra.Command, args []string) {
	ndpPath := args[0]
	pluginsDir := conf.Server.Plugins.Folder

	pkg, err := loadAndValidatePackage(ndpPath)
	if err != nil {
		log.Fatal("Package validation failed", err)
	}

	// Create target directory based on plugin name
	targetDir := filepath.Join(pluginsDir, pkg.Manifest.Name)

	// Check if plugin already exists
	if utils.FileExists(targetDir) {
		log.Fatal("Plugin already installed", "name", pkg.Manifest.Name, "path", targetDir,
			"use", "navidrome plugin update")
	}

	if err := extractAndSetupPlugin(ndpPath, targetDir); err != nil {
		log.Fatal("Plugin installation failed", err)
	}

	fmt.Printf("Plugin '%s' v%s installed successfully\n", pkg.Manifest.Name, pkg.Manifest.Version)
}

func pluginRemove(cmd *cobra.Command, args []string) {
	pluginName := args[0]
	pluginsDir := conf.Server.Plugins.Folder

	pluginDir, err := validatePluginDirectory(pluginsDir, pluginName)
	if err != nil {
		log.Fatal("Plugin validation failed", err)
	}

	_, isSymlink, err := resolvePluginPath(pluginDir)
	if err != nil {
		log.Fatal("Failed to resolve plugin path", err)
	}

	if isSymlink {
		// For symlinked plugins (dev mode), just remove the symlink
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

	pkg, err := loadAndValidatePackage(ndpPath)
	if err != nil {
		log.Fatal("Package validation failed", err)
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
	if err := extractAndSetupPlugin(ndpPath, targetDir); err != nil {
		// Restore backup if extraction failed
		os.RemoveAll(targetDir)
		_ = os.Rename(backupDir, targetDir) // Ignore error as we're already in a fatal path
		log.Fatal("Plugin update failed", err)
	}

	// Remove the backup
	os.RemoveAll(backupDir)

	fmt.Printf("Plugin '%s' updated to v%s successfully\n", pkg.Manifest.Name, pkg.Manifest.Version)
}

func pluginRefresh(cmd *cobra.Command, args []string) {
	pluginName := args[0]
	pluginsDir := conf.Server.Plugins.Folder

	pluginDir, err := validatePluginDirectory(pluginsDir, pluginName)
	if err != nil {
		log.Fatal("Plugin validation failed", err)
	}

	resolvedPath, isSymlink, err := resolvePluginPath(pluginDir)
	if err != nil {
		log.Fatal("Failed to resolve plugin path", err)
	}

	if isSymlink {
		log.Debug("Processing symlinked plugin", "name", pluginName, "link", pluginDir, "target", resolvedPath)
	}

	fmt.Printf("Refreshing plugin '%s'...\n", pluginName)

	// Get the plugin manager and refresh
	mgr := GetPluginManager(cmd.Context())
	log.Debug("Scanning plugins directory", "path", pluginsDir)
	mgr.ScanPlugins()

	log.Info("Waiting for plugin compilation to complete", "name", pluginName)

	// Wait for compilation to complete
	if err := mgr.EnsureCompiled(pluginName); err != nil {
		log.Fatal("Failed to compile refreshed plugin", "name", pluginName, err)
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

	// Validate source directory and manifest
	if err := validateDevSource(sourcePath); err != nil {
		log.Fatal("Source validation failed", err)
	}

	// Load manifest to get plugin name
	manifest, err := plugins.LoadManifest(sourcePath)
	if err != nil {
		log.Fatal("Failed to load plugin manifest", "path", filepath.Join(sourcePath, "manifest.json"), err)
	}

	pluginName := cmp.Or(manifest.Name, filepath.Base(sourcePath))
	targetPath := filepath.Join(pluginsDir, pluginName)

	// Handle existing target
	if err := handleExistingTarget(targetPath, sourcePath); err != nil {
		log.Fatal("Failed to handle existing target", err)
	}

	// Create target directory if needed
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

// Utility functions

func validateDevSource(sourcePath string) error {
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("source folder not found: %s (%w)", sourcePath, err)
	}
	if !sourceInfo.IsDir() {
		return fmt.Errorf("source path is not a directory: %s", sourcePath)
	}

	manifestPath := filepath.Join(sourcePath, "manifest.json")
	if !utils.FileExists(manifestPath) {
		return fmt.Errorf("source folder missing manifest.json: %s", sourcePath)
	}

	return nil
}

func handleExistingTarget(targetPath, sourcePath string) error {
	if !utils.FileExists(targetPath) {
		return nil // Nothing to handle
	}

	// Check if it's already a symlink to our source
	existingLink, err := os.Readlink(targetPath)
	if err == nil && existingLink == sourcePath {
		fmt.Printf("Symlink already exists and points to the correct source\n")
		return fmt.Errorf("symlink already exists") // This will cause early return in caller
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
		return fmt.Errorf("operation canceled")
	}

	// Remove existing target
	if err := os.RemoveAll(targetPath); err != nil {
		return fmt.Errorf("failed to remove existing target %s: %w", targetPath, err)
	}

	return nil
}

func ensurePluginDirPermissions(dir string) {
	if err := os.Chmod(dir, pluginDirPermissions); err != nil {
		log.Error("Failed to set plugin directory permissions", "dir", dir, err)
	}

	// Apply permissions to all files in the directory
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

		mode := os.FileMode(pluginFilePermissions) // Files
		if info.IsDir() {
			mode = os.FileMode(pluginDirPermissions) // Directories
			ensurePluginDirPermissions(path)         // Recursive
		}

		if err := os.Chmod(path, mode); err != nil {
			log.Error("Failed to set file permissions", "path", path, err)
		}
	}
}

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
