package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/events"
)

// PluginMetadata holds the extracted information from a plugin file
// without fully initializing the plugin.
type PluginMetadata struct {
	Manifest *Manifest
	SHA256   string
}

// adminContext returns a context with admin privileges for DB operations.
func adminContext(ctx context.Context) context.Context {
	return request.WithUser(ctx, model.User{IsAdmin: true})
}

// marshalManifest marshals a manifest to JSON string, returning empty string on error.
func marshalManifest(m *Manifest) string {
	b, _ := json.Marshal(m)
	return string(b)
}

// computeFileSHA256 computes the SHA-256 hash of a file without loading it into memory.
// This is used for quick change detection before full plugin compilation.
func computeFileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// addPluginToDB adds a new plugin to the database as disabled.
func (m *Manager) addPluginToDB(ctx context.Context, repo model.PluginRepository, name, path string, metadata *PluginMetadata) error {
	now := time.Now()
	newPlugin := &model.Plugin{
		ID:        name,
		Path:      path,
		Manifest:  marshalManifest(metadata.Manifest),
		SHA256:    metadata.SHA256,
		Enabled:   false,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Put(newPlugin); err != nil {
		return fmt.Errorf("adding plugin to DB: %w", err)
	}
	log.Info(ctx, "Discovered new plugin", "plugin", name)
	m.sendPluginRefreshEvent(ctx, events.Any)
	return nil
}

// updatePluginInDB updates an existing plugin in the database after a file change.
// If the plugin was enabled, it will be unloaded and disabled.
func (m *Manager) updatePluginInDB(ctx context.Context, repo model.PluginRepository, dbPlugin *model.Plugin, path string, metadata *PluginMetadata) error {
	wasEnabled := dbPlugin.Enabled
	if wasEnabled {
		if err := m.unloadPlugin(dbPlugin.ID); err != nil {
			log.Debug(ctx, "Plugin not loaded during change", "plugin", dbPlugin.ID, err)
		}
	}
	dbPlugin.Path = path
	dbPlugin.Manifest = marshalManifest(metadata.Manifest)
	dbPlugin.SHA256 = metadata.SHA256
	dbPlugin.Enabled = false
	dbPlugin.LastError = ""
	dbPlugin.UpdatedAt = time.Now()
	if err := repo.Put(dbPlugin); err != nil {
		return fmt.Errorf("updating plugin in DB: %w", err)
	}
	log.Info(ctx, "Plugin file changed", "plugin", dbPlugin.ID, "wasEnabled", wasEnabled)
	m.sendPluginRefreshEvent(ctx, dbPlugin.ID)
	return nil
}

// removePluginFromDB removes a plugin from the database.
// If the plugin was enabled, it will be unloaded first.
func (m *Manager) removePluginFromDB(ctx context.Context, repo model.PluginRepository, dbPlugin *model.Plugin) error {
	pluginID := dbPlugin.ID
	if dbPlugin.Enabled {
		if err := m.unloadPlugin(pluginID); err != nil {
			log.Debug(ctx, "Plugin not loaded during removal", "plugin", pluginID, err)
		}
	}
	if err := repo.Delete(pluginID); err != nil {
		return fmt.Errorf("deleting plugin from DB: %w", err)
	}
	log.Info(ctx, "Plugin removed", "plugin", pluginID)
	m.sendPluginRefreshEvent(ctx, events.Any)
	return nil
}

// syncPlugins scans the plugins folder and synchronizes with the database.
// It handles new, changed, and removed plugins by comparing SHA-256 hashes.
// - New plugins are added to DB as disabled
// - Changed plugins are updated in DB and disabled if they were enabled
// - Removed plugins are deleted from DB (after unloading if enabled)
func (m *Manager) syncPlugins(ctx context.Context, folder string) error {
	if m.ds == nil {
		return fmt.Errorf("datastore not configured")
	}

	adminCtx := adminContext(ctx)

	// Read current plugins from folder
	entries, err := os.ReadDir(folder)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debug(ctx, "Plugins folder does not exist", "folder", folder)
			return nil
		}
		return fmt.Errorf("reading plugins folder: %w", err)
	}

	// Build map of files in folder
	filesOnDisk := make(map[string]string) // name -> path
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), PackageExtension) {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), PackageExtension)
		filesOnDisk[name] = filepath.Join(folder, entry.Name())
	}

	// Get all plugins from DB
	repo := m.ds.Plugin(adminCtx)
	dbPlugins, err := repo.GetAll()
	if err != nil {
		return fmt.Errorf("reading plugins from DB: %w", err)
	}
	pluginsInDB := make(map[string]*model.Plugin)
	for i := range dbPlugins {
		pluginsInDB[dbPlugins[i].ID] = &dbPlugins[i]
	}

	now := time.Now()

	// Process files on disk
	for name, path := range filesOnDisk {
		dbPlugin, exists := pluginsInDB[name]

		// Compute SHA256 first (lightweight operation) to check if plugin changed
		sha256Hash, err := computeFileSHA256(path)
		if err != nil {
			log.Error(ctx, "Failed to compute SHA256 for plugin", "plugin", name, "path", path, err)
			continue
		}

		// If plugin exists in DB with same hash, skip full manifest extraction
		if exists && dbPlugin.SHA256 == sha256Hash {
			// Plugin unchanged - just update path in case folder moved
			if dbPlugin.Path != path {
				dbPlugin.Path = path
				dbPlugin.UpdatedAt = now
				if err := repo.Put(dbPlugin); err != nil {
					log.Error(ctx, "Failed to update plugin path in DB", "plugin", name, err)
				}
			}
			delete(pluginsInDB, name)
			continue
		}

		// Plugin is new or changed - need full manifest extraction
		metadata, err := m.extractManifest(path)
		if err != nil {
			log.Error(ctx, "Failed to extract manifest from plugin", "plugin", name, "path", path, err)
			// Store error in DB if plugin exists
			if exists {
				dbPlugin.LastError = err.Error()
				dbPlugin.UpdatedAt = now
				if dbPlugin.Enabled {
					// Unload broken plugin
					if unloadErr := m.unloadPlugin(name); unloadErr != nil {
						log.Debug(ctx, "Plugin not loaded", "plugin", name)
					}
					dbPlugin.Enabled = false
				}
				if putErr := repo.Put(dbPlugin); putErr != nil {
					log.Error(ctx, "Failed to update plugin in DB", "plugin", name, err)
				}
			}
			delete(pluginsInDB, name)
			continue
		}

		if !exists {
			// New plugin - add to DB as disabled
			if err := m.addPluginToDB(ctx, repo, name, path, metadata); err != nil {
				log.Error(ctx, "Failed to add plugin to DB", "plugin", name, err)
			}
		} else {
			// Plugin changed - update DB
			if err := m.updatePluginInDB(ctx, repo, dbPlugin, path, metadata); err != nil {
				log.Error(ctx, "Failed to update plugin in DB", "plugin", name, err)
			}
		}
		// Mark as processed
		delete(pluginsInDB, name)
	}

	// Remove plugins no longer on disk
	for _, dbPlugin := range pluginsInDB {
		if err := m.removePluginFromDB(ctx, repo, dbPlugin); err != nil {
			log.Error(ctx, "Failed to delete plugin from DB", "plugin", dbPlugin.ID, err)
		}
	}

	return nil
}
