package plugins

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/rjeczalik/notify"
)

// debounceDuration is the time to wait before acting on file events
// to handle multiple rapid events for the same file.
const debounceDuration = 2 * time.Second

// startWatcher starts the file watcher for the plugins folder.
// It watches for CREATE, WRITE, and REMOVE events on .wasm files.
func (m *Manager) startWatcher() error {
	folder := conf.Server.Plugins.Folder
	if folder == "" {
		return nil
	}

	m.watcherEvents = make(chan notify.EventInfo, 10)
	m.watcherDone = make(chan struct{})
	m.debounceTimers = make(map[string]*time.Timer)
	m.debounceMu = sync.Mutex{}

	// Watch the plugins folder (not recursive)
	// We filter for .wasm files in the event handler
	if err := notify.Watch(folder, m.watcherEvents, notify.Create, notify.Write, notify.Remove, notify.Rename); err != nil {
		close(m.watcherEvents)
		return err
	}

	log.Info(m.ctx, "Started plugin file watcher", "folder", folder)

	go m.watcherLoop()

	return nil
}

// stopWatcher stops the file watcher
func (m *Manager) stopWatcher() {
	if m.watcherEvents == nil {
		return
	}

	notify.Stop(m.watcherEvents)
	close(m.watcherDone)

	// Cancel any pending debounce timers
	m.debounceMu.Lock()
	for _, timer := range m.debounceTimers {
		timer.Stop()
	}
	m.debounceTimers = nil
	m.debounceMu.Unlock()

	log.Debug(m.ctx, "Stopped plugin file watcher")
}

// watcherLoop processes file watcher events
func (m *Manager) watcherLoop() {
	for {
		select {
		case event, ok := <-m.watcherEvents:
			if !ok {
				return
			}
			m.handleWatcherEvent(event)
		case <-m.ctx.Done():
			return
		case <-m.watcherDone:
			return
		}
	}
}

// handleWatcherEvent processes a single file watcher event with debouncing
func (m *Manager) handleWatcherEvent(event notify.EventInfo) {
	path := event.Path()

	// Only process .ndp package files
	if !strings.HasSuffix(path, PackageExtension) {
		return
	}

	pluginName := strings.TrimSuffix(filepath.Base(path), PackageExtension)

	log.Trace(m.ctx, "Plugin file event", "plugin", pluginName, "event", event.Event(), "path", path)

	// Debounce: cancel any pending timer for this plugin and start a new one
	m.debounceMu.Lock()
	if timer, exists := m.debounceTimers[pluginName]; exists {
		timer.Stop()
	}

	// Note: We don't capture the event type here. Instead, processPluginEvent
	// checks if the file exists when the timer fires. This handles sequences like
	// Remove+Create+Rename correctly by checking actual file state after debounce.
	m.debounceTimers[pluginName] = time.AfterFunc(debounceDuration, func() {
		m.processPluginEvent(pluginName)
	})
	m.debounceMu.Unlock()
}

// pluginAction represents the action to take on a plugin based on file state
type pluginAction int

const (
	actionNone   pluginAction = iota // No action needed
	actionUpdate                     // File exists: add new or update existing plugin in DB
	actionRemove                     // File gone: remove plugin from DB (unload if enabled)
)

// determinePluginAction decides what action to take based on file existence.
// We check file existence rather than relying on event type because:
// 1. Events can be coalesced on some systems (macOS FSEvents)
// 2. Rename events can mean either "renamed away" (remove) or "renamed to" (add)
// 3. Build tools often do atomic writes (write temp file, rename to target)
// By checking existence, we handle all these cases correctly.
func determinePluginAction(path string) pluginAction {
	if _, err := os.Stat(path); err == nil {
		// File exists - treat as add/update
		return actionUpdate
	}
	// File doesn't exist - it was removed
	return actionRemove
}

// processPluginEvent handles the actual plugin load/unload/reload after debouncing.
// - If file exists: extract manifest, add or update plugin in DB
// - If file gone: unload if enabled, delete from DB
func (m *Manager) processPluginEvent(pluginName string) {
	// Don't process if manager is stopping/stopped (atomic check to avoid race with Stop())
	if m.stopped.Load() {
		return
	}

	// Clean up debounce timer entry
	m.debounceMu.Lock()
	delete(m.debounceTimers, pluginName)
	m.debounceMu.Unlock()

	folder := conf.Server.Plugins.Folder
	ndpPath := filepath.Join(folder, pluginName+PackageExtension)

	action := determinePluginAction(ndpPath)
	log.Debug(m.ctx, "Plugin event action", "plugin", pluginName, "action", action, "path", ndpPath)

	ctx := adminContext(m.ctx)
	repo := m.ds.Plugin(ctx)

	switch action {
	case actionUpdate:
		// File changed - check SHA256 first, then extract manifest if needed
		sha256Hash, err := computeFileSHA256(ndpPath)
		if err != nil {
			log.Error(m.ctx, "Failed to compute SHA256 for changed plugin", "plugin", pluginName, err)
			return
		}

		dbPlugin, err := repo.Get(pluginName)
		if err != nil {
			// Plugin not in DB yet, need full manifest extraction to add it
			metadata, extractErr := m.extractManifest(ndpPath)
			if extractErr != nil {
				log.Error(m.ctx, "Failed to extract manifest from new plugin", "plugin", pluginName, extractErr)
				return
			}
			if addErr := m.addPluginToDB(m.ctx, repo, pluginName, ndpPath, metadata); addErr != nil {
				log.Error(m.ctx, "Failed to add plugin to DB", "plugin", pluginName, addErr)
			}
			return
		}

		// Check if actually changed using lightweight SHA256 comparison
		if dbPlugin.SHA256 == sha256Hash {
			return // No actual change
		}

		// Plugin changed - now extract full manifest
		metadata, err := m.extractManifest(ndpPath)
		if err != nil {
			log.Error(m.ctx, "Failed to extract manifest from changed plugin", "plugin", pluginName, err)
			// Update error in DB
			dbPlugin.LastError = err.Error()
			dbPlugin.UpdatedAt = time.Now()
			if dbPlugin.Enabled {
				_ = m.unloadPlugin(pluginName)
				dbPlugin.Enabled = false
			}
			_ = repo.Put(dbPlugin)
			return
		}

		if err := m.updatePluginInDB(m.ctx, repo, dbPlugin, ndpPath, metadata); err != nil {
			log.Error(m.ctx, "Failed to update plugin in DB", "plugin", pluginName, err)
		}

	case actionRemove:
		// File removed - unload if enabled, delete from DB
		dbPlugin, err := repo.Get(pluginName)
		if err != nil {
			log.Debug(m.ctx, "Removed plugin not in DB", "plugin", pluginName)
			return
		}

		if err := m.removePluginFromDB(m.ctx, repo, dbPlugin); err != nil {
			log.Error(m.ctx, "Failed to delete plugin from DB", "plugin", pluginName, err)
		}
	}
}
