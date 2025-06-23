package plugins

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/plugins/schema"
)

// PluginDiscoveryEntry represents the result of plugin discovery
type PluginDiscoveryEntry struct {
	ID        string                 // Plugin ID (directory name)
	Path      string                 // Resolved plugin directory path
	WasmPath  string                 // Path to the WASM file
	Manifest  *schema.PluginManifest // Loaded manifest (nil if failed)
	IsSymlink bool                   // Whether the plugin is a development symlink
	Error     error                  // Error encountered during discovery
}

// DiscoverPlugins scans the plugins directory and returns information about all discoverable plugins
// This shared function eliminates duplication between ScanPlugins and plugin list commands
func DiscoverPlugins(pluginsDir string) []PluginDiscoveryEntry {
	var discoveries []PluginDiscoveryEntry

	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		// Return a single entry with the error
		return []PluginDiscoveryEntry{{
			Error: fmt.Errorf("failed to read plugins directory %s: %w", pluginsDir, err),
		}}
	}

	for _, entry := range entries {
		name := entry.Name()
		pluginPath := filepath.Join(pluginsDir, name)

		// Skip hidden files
		if name[0] == '.' {
			continue
		}

		// Check if it's a directory or symlink
		info, err := os.Lstat(pluginPath)
		if err != nil {
			discoveries = append(discoveries, PluginDiscoveryEntry{
				ID:    name,
				Error: fmt.Errorf("failed to stat entry %s: %w", pluginPath, err),
			})
			continue
		}

		isSymlink := info.Mode()&os.ModeSymlink != 0
		isDir := info.IsDir()

		// Skip if not a directory or symlink
		if !isDir && !isSymlink {
			continue
		}

		// Resolve symlinks
		pluginDir := pluginPath
		if isSymlink {
			targetDir, err := os.Readlink(pluginPath)
			if err != nil {
				discoveries = append(discoveries, PluginDiscoveryEntry{
					ID:        name,
					IsSymlink: true,
					Error:     fmt.Errorf("failed to resolve symlink %s: %w", pluginPath, err),
				})
				continue
			}

			// If target is a relative path, make it absolute
			if !filepath.IsAbs(targetDir) {
				targetDir = filepath.Join(filepath.Dir(pluginPath), targetDir)
			}

			// Verify that the target is a directory
			targetInfo, err := os.Stat(targetDir)
			if err != nil {
				discoveries = append(discoveries, PluginDiscoveryEntry{
					ID:        name,
					IsSymlink: true,
					Error:     fmt.Errorf("failed to stat symlink target %s: %w", targetDir, err),
				})
				continue
			}

			if !targetInfo.IsDir() {
				discoveries = append(discoveries, PluginDiscoveryEntry{
					ID:        name,
					IsSymlink: true,
					Error:     fmt.Errorf("symlink target is not a directory: %s", targetDir),
				})
				continue
			}

			pluginDir = targetDir
		}

		// Check for WASM file
		wasmPath := filepath.Join(pluginDir, "plugin.wasm")
		if _, err := os.Stat(wasmPath); err != nil {
			discoveries = append(discoveries, PluginDiscoveryEntry{
				ID:    name,
				Path:  pluginDir,
				Error: fmt.Errorf("no plugin.wasm found: %w", err),
			})
			continue
		}

		// Load manifest
		manifest, err := LoadManifest(pluginDir)
		if err != nil {
			discoveries = append(discoveries, PluginDiscoveryEntry{
				ID:    name,
				Path:  pluginDir,
				Error: fmt.Errorf("failed to load manifest: %w", err),
			})
			continue
		}

		// Check for capabilities
		if len(manifest.Capabilities) == 0 {
			discoveries = append(discoveries, PluginDiscoveryEntry{
				ID:    name,
				Path:  pluginDir,
				Error: fmt.Errorf("no capabilities found in manifest"),
			})
			continue
		}

		// Success!
		discoveries = append(discoveries, PluginDiscoveryEntry{
			ID:        name,
			Path:      pluginDir,
			WasmPath:  wasmPath,
			Manifest:  manifest,
			IsSymlink: isSymlink,
		})
	}

	return discoveries
}
