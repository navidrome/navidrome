package scanner

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/singleton"
)

type Watcher interface {
	Run(ctx context.Context) error
	Watch(ctx context.Context, lib *model.Library) error
	StopWatching(ctx context.Context, libraryID int) error
}

type watcher struct {
	mainCtx         context.Context
	ds              model.DataStore
	scanner         Scanner
	triggerWait     time.Duration
	watcherNotify   chan model.Library
	libraryWatchers map[int]*libraryWatcherInstance
	mu              sync.RWMutex
}

type libraryWatcherInstance struct {
	library *model.Library
	cancel  context.CancelFunc
}

// GetWatcher returns the watcher singleton
func GetWatcher(ds model.DataStore, s Scanner) Watcher {
	return singleton.GetInstance(func() *watcher {
		return &watcher{
			ds:              ds,
			scanner:         s,
			triggerWait:     conf.Server.Scanner.WatcherWait,
			watcherNotify:   make(chan model.Library, 1),
			libraryWatchers: make(map[int]*libraryWatcherInstance),
		}
	})
}

func (w *watcher) Run(ctx context.Context) error {
	// Keep the main context to be used in all watchers added later
	w.mainCtx = ctx

	// Start watchers for all existing libraries
	libs, err := w.ds.Library(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("getting libraries: %w", err)
	}

	for _, lib := range libs {
		if err := w.Watch(ctx, &lib); err != nil {
			log.Warn(ctx, "Failed to start watcher for existing library", "libraryID", lib.ID, "name", lib.Name, "path", lib.Path, err)
		}
	}

	// Main scan triggering loop
	trigger := time.NewTimer(w.triggerWait)
	trigger.Stop()
	waiting := false
	for {
		select {
		case <-trigger.C:
			log.Info("Watcher: Triggering scan")
			status, err := w.scanner.Status(ctx)
			if err != nil {
				log.Error(ctx, "Watcher: Error retrieving Scanner status", err)
				break
			}
			if status.Scanning {
				log.Debug(ctx, "Watcher: Already scanning, will retry later", "waitTime", w.triggerWait*3)
				trigger.Reset(w.triggerWait * 3)
				continue
			}
			waiting = false
			go func() {
				_, err := w.scanner.ScanAll(ctx, false)
				if err != nil {
					log.Error(ctx, "Watcher: Error scanning", err)
				} else {
					log.Info(ctx, "Watcher: Scan completed")
				}
			}()
		case <-ctx.Done():
			// Stop all library watchers
			w.mu.Lock()
			for libraryID, instance := range w.libraryWatchers {
				log.Debug(ctx, "Stopping library watcher due to context cancellation", "libraryID", libraryID)
				instance.cancel()
			}
			w.libraryWatchers = make(map[int]*libraryWatcherInstance)
			w.mu.Unlock()
			return nil
		case lib := <-w.watcherNotify:
			if !waiting {
				log.Debug(ctx, "Watcher: Detected changes. Waiting for more changes before triggering scan",
					"libraryID", lib.ID, "name", lib.Name, "path", lib.Path)
				waiting = true
			}
			trigger.Reset(w.triggerWait)
		}
	}
}

func (w *watcher) Watch(ctx context.Context, lib *model.Library) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Stop existing watcher if any
	if existingInstance, exists := w.libraryWatchers[lib.ID]; exists {
		log.Debug(ctx, "Stopping existing watcher before starting new one", "libraryID", lib.ID, "name", lib.Name)
		existingInstance.cancel()
	}

	// Start new watcher
	watcherCtx, cancel := context.WithCancel(w.mainCtx)
	instance := &libraryWatcherInstance{
		library: lib,
		cancel:  cancel,
	}

	w.libraryWatchers[lib.ID] = instance

	// Start watching in a goroutine
	go func() {
		defer func() {
			w.mu.Lock()
			if currentInstance, exists := w.libraryWatchers[lib.ID]; exists && currentInstance == instance {
				delete(w.libraryWatchers, lib.ID)
			}
			w.mu.Unlock()
		}()

		err := w.watchLibrary(watcherCtx, lib)
		if err != nil && watcherCtx.Err() == nil { // Only log error if not due to cancellation
			log.Error(ctx, "Watcher error", "libraryID", lib.ID, "name", lib.Name, "path", lib.Path, err)
		}
	}()

	log.Info(ctx, "Started watcher for library", "libraryID", lib.ID, "name", lib.Name, "path", lib.Path)
	return nil
}

func (w *watcher) StopWatching(ctx context.Context, libraryID int) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	instance, exists := w.libraryWatchers[libraryID]
	if !exists {
		log.Debug(ctx, "No watcher found to stop", "libraryID", libraryID)
		return nil
	}

	instance.cancel()
	delete(w.libraryWatchers, libraryID)

	log.Info(ctx, "Stopped watcher for library", "libraryID", libraryID, "name", instance.library.Name)
	return nil
}

// watchLibrary implements the core watching logic for a single library (extracted from old watchLib function)
func (w *watcher) watchLibrary(ctx context.Context, lib *model.Library) error {
	s, err := storage.For(lib.Path)
	if err != nil {
		return fmt.Errorf("creating storage: %w", err)
	}

	fsys, err := s.FS()
	if err != nil {
		return fmt.Errorf("getting FS: %w", err)
	}

	watcher, ok := s.(storage.Watcher)
	if !ok {
		log.Info(ctx, "Watcher not supported for storage type", "libraryID", lib.ID, "path", lib.Path)
		return nil
	}

	c, err := watcher.Start(ctx)
	if err != nil {
		return fmt.Errorf("starting watcher: %w", err)
	}

	absLibPath, err := filepath.Abs(lib.Path)
	if err != nil {
		return fmt.Errorf("converting to absolute path: %w", err)
	}

	log.Info(ctx, "Watcher started for library", "libraryID", lib.ID, "name", lib.Name, "path", lib.Path, "absoluteLibPath", absLibPath)

	for {
		select {
		case <-ctx.Done():
			log.Debug(ctx, "Watcher stopped due to context cancellation", "libraryID", lib.ID, "name", lib.Name)
			return nil
		case path := <-c:
			path, err = filepath.Rel(absLibPath, path)
			if err != nil {
				log.Error(ctx, "Error getting relative path", "libraryID", lib.ID, "absolutePath", absLibPath, "path", path, err)
				continue
			}

			if isIgnoredPath(ctx, fsys, path) {
				log.Trace(ctx, "Ignoring change", "libraryID", lib.ID, "path", path)
				continue
			}

			log.Trace(ctx, "Detected change", "libraryID", lib.ID, "path", path, "absoluteLibPath", absLibPath)

			// Notify the main watcher of changes
			select {
			case w.watcherNotify <- *lib:
			default:
				// Channel is full, notification already pending
			}
		}
	}
}

func isIgnoredPath(_ context.Context, _ fs.FS, path string) bool {
	baseDir, name := filepath.Split(path)
	switch {
	case model.IsAudioFile(path):
		return false
	case model.IsValidPlaylist(path):
		return false
	case model.IsImageFile(path):
		return false
	case name == ".DS_Store":
		return true
	}
	// As it can be a deletion and not a change, we cannot reliably know if the path is a file or directory.
	// But at this point, we can assume it's a directory. If it's a file, it would be ignored anyway
	return isDirIgnored(baseDir)
}
