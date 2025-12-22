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
// This is a var (not const) to allow tests to use a shorter duration.
var debounceDuration = 500 * time.Millisecond

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
	actionLoad                       // Load the plugin
	actionUnload                     // Unload the plugin
	actionReload                     // Reload the plugin
)

// determinePluginAction decides what action to take based on the file event type
// and whether the plugin is currently loaded. This is a pure function with no side effects.
func determinePluginAction(eventType notify.Event, isLoaded bool) pluginAction {
	switch {
	case eventType&notify.Remove != 0 || eventType&notify.Rename != 0:
		// File removed or renamed away - unload if loaded
		return actionUnload

	case eventType&notify.Create != 0:
		// New file - load it
		return actionLoad

	case eventType&notify.Write != 0:
		// File modified - reload if loaded, otherwise load
		if isLoaded {
			return actionReload
		}
		return actionLoad
	}
	return actionNone
}

// processPluginEvent handles the actual plugin load/unload/reload after debouncing
func (m *Manager) processPluginEvent(pluginName string, eventType notify.Event) {
	// Don't process if manager is stopping/stopped (atomic check to avoid race with Stop())
	if m.stopped.Load() {
		return
	}

	// Clean up debounce timer entry
	m.debounceMu.Lock()
	delete(m.debounceTimers, pluginName)
	m.debounceMu.Unlock()

	// Check if plugin is currently loaded
	m.mu.RLock()
	_, isLoaded := m.plugins[pluginName]
	m.mu.RUnlock()

	// Determine and execute the appropriate action
	action := determinePluginAction(eventType, isLoaded)
	switch action {
	case actionLoad:
		if err := m.LoadPlugin(pluginName); err != nil {
			log.Error(m.ctx, "Failed to load plugin", "plugin", pluginName, err)
		}
	case actionUnload:
		if err := m.UnloadPlugin(pluginName); err != nil {
			log.Debug(m.ctx, "Plugin not loaded, skipping unload", "plugin", pluginName, err)
		}
	case actionReload:
		if err := m.ReloadPlugin(pluginName); err != nil {
			log.Error(m.ctx, "Failed to reload plugin", "plugin", pluginName, err)
		}
	}
}
