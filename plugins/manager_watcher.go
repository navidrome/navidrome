package plugins

import (
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

	// Only process .wasm files
	if !strings.HasSuffix(path, ".wasm") {
		return
	}

	pluginName := strings.TrimSuffix(filepath.Base(path), ".wasm")

	log.Debug(m.ctx, "Plugin file event", "plugin", pluginName, "event", event.Event(), "path", path)

	// Debounce: cancel any pending timer for this plugin and start a new one
	m.debounceMu.Lock()
	if timer, exists := m.debounceTimers[pluginName]; exists {
		timer.Stop()
	}

	eventType := event.Event()
	m.debounceTimers[pluginName] = time.AfterFunc(debounceDuration, func() {
		m.processPluginEvent(pluginName, eventType)
	})
	m.debounceMu.Unlock()
}

// pluginAction represents the action to take on a plugin based on a file event
type pluginAction int

const (
	actionNone   pluginAction = iota // No action needed
	actionAdd                        // Add new plugin to DB (disabled)
	actionUpdate                     // Update existing plugin in DB (disable if enabled)
	actionRemove                     // Remove plugin from DB (unload if enabled)
)

// determinePluginAction decides what action to take based on the file event type.
func determinePluginAction(eventType notify.Event) pluginAction {
	switch {
	case eventType&notify.Remove != 0 || eventType&notify.Rename != 0:
		return actionRemove
	case eventType&notify.Create != 0:
		return actionAdd
	case eventType&notify.Write != 0:
		return actionUpdate
	}
	return actionNone
}

// processPluginEvent handles the actual plugin load/unload/reload after debouncing.
// - On file add: extract manifest, create DB record as disabled
// - On file change: extract manifest, update DB, disable if was enabled
// - On file remove: unload if enabled, delete DB record
func (m *Manager) processPluginEvent(pluginName string, eventType notify.Event) {
	// Don't process if manager is stopping/stopped (atomic check to avoid race with Stop())
	if m.stopped.Load() {
		return
	}

	// Clean up debounce timer entry
	m.debounceMu.Lock()
	delete(m.debounceTimers, pluginName)
	m.debounceMu.Unlock()

	action := determinePluginAction(eventType)
	log.Debug(m.ctx, "Plugin event action", "plugin", pluginName, "action", action)

	ctx := adminContext(m.ctx)
	repo := m.ds.Plugin(ctx)
	folder := conf.Server.Plugins.Folder
	wasmPath := filepath.Join(folder, pluginName+".wasm")

	switch action {
	case actionAdd:
		// New file - extract manifest and add to DB as disabled
		metadata, err := m.extractManifest(wasmPath)
		if err != nil {
			log.Error(m.ctx, "Failed to extract manifest from new plugin", "plugin", pluginName, err)
			return
		}
		if err := m.addPluginToDB(m.ctx, repo, pluginName, wasmPath, metadata); err != nil {
			log.Error(m.ctx, "Failed to add plugin to DB", "plugin", pluginName, err)
		}

	case actionUpdate:
		// File changed - check SHA256 first, then extract manifest if needed
		sha256Hash, err := computeFileSHA256(wasmPath)
		if err != nil {
			log.Error(m.ctx, "Failed to compute SHA256 for changed plugin", "plugin", pluginName, err)
			return
		}

		dbPlugin, err := repo.Get(pluginName)
		if err != nil {
			// Plugin not in DB yet, need full manifest extraction to add it
			metadata, extractErr := m.extractManifest(wasmPath)
			if extractErr != nil {
				log.Error(m.ctx, "Failed to extract manifest from new plugin", "plugin", pluginName, extractErr)
				return
			}
			if addErr := m.addPluginToDB(m.ctx, repo, pluginName, wasmPath, metadata); addErr != nil {
				log.Error(m.ctx, "Failed to add plugin to DB", "plugin", pluginName, addErr)
			}
			return
		}

		// Check if actually changed using lightweight SHA256 comparison
		if dbPlugin.SHA256 == sha256Hash {
			return // No actual change
		}

		// Plugin changed - now extract full manifest
		metadata, err := m.extractManifest(wasmPath)
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

		if err := m.updatePluginInDB(m.ctx, repo, dbPlugin, wasmPath, metadata); err != nil {
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
