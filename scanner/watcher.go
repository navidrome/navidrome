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
	scanner         model.Scanner
	triggerWait     time.Duration
	watcherNotify   chan scanNotification
	libraryWatchers map[int]*libraryWatcherInstance
	mu              sync.RWMutex
}

type libraryWatcherInstance struct {
	library *model.Library
	cancel  context.CancelFunc
}

type scanNotification struct {
	Library    *model.Library
	FolderPath string
}

// GetWatcher returns the watcher singleton
func GetWatcher(ds model.DataStore, s model.Scanner) Watcher {
	return singleton.GetInstance(func() *watcher {
		return &watcher{
			ds:              ds,
			scanner:         s,
			triggerWait:     conf.Server.Scanner.WatcherWait,
			watcherNotify:   make(chan scanNotification, 1),
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
	targets := make(map[model.ScanTarget]struct{})
	for {
		select {
		case <-trigger.C:
			log.Info("Watcher: Triggering scan for changed folders", "numTargets", len(targets))
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

			// Convert targets map to slice
			targetSlice := make([]model.ScanTarget, 0, len(targets))
			for target := range targets {
				targetSlice = append(targetSlice, target)
			}

			// Clear targets for next batch
			targets = make(map[model.ScanTarget]struct{})

			go func() {
				var err error
				if conf.Server.DevSelectiveWatcher {
					_, err = w.scanner.ScanFolders(ctx, false, targetSlice)
				} else {
					_, err = w.scanner.ScanAll(ctx, false)
				}
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
		case notification := <-w.watcherNotify:
			// Reset the trigger timer for debounce
			trigger.Reset(w.triggerWait)

			lib := notification.Library
			folderPath := notification.FolderPath

			// If already scheduled for scan, skip
			target := model.ScanTarget{LibraryID: lib.ID, FolderPath: folderPath}
			if _, exists := targets[target]; exists {
				continue
			}
			targets[target] = struct{}{}

			log.Debug(ctx, "Watcher: Detected changes. Waiting for more changes before triggering scan",
				"libraryID", lib.ID, "name", lib.Name, "path", lib.Path, "folderPath", folderPath)
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

	return w.processLibraryEvents(ctx, lib, fsys, c, absLibPath)
}

// processLibraryEvents processes filesystem events for a library.
func (w *watcher) processLibraryEvents(ctx context.Context, lib *model.Library, fsys storage.MusicFS, events <-chan string, absLibPath string) error {
	for {
		select {
		case <-ctx.Done():
			log.Debug(ctx, "Watcher stopped due to context cancellation", "libraryID", lib.ID, "name", lib.Name)
			return nil
		case path := <-events:
			path, err := filepath.Rel(absLibPath, path)
			if err != nil {
				log.Error(ctx, "Error getting relative path", "libraryID", lib.ID, "absolutePath", absLibPath, "path", path, err)
				continue
			}

			if isIgnoredPath(ctx, fsys, path) {
				log.Trace(ctx, "Ignoring change", "libraryID", lib.ID, "path", path)
				continue
			}
			log.Trace(ctx, "Detected change", "libraryID", lib.ID, "path", path, "absoluteLibPath", absLibPath)

			// Check if the original path (before resolution) matches .ndignore patterns
			// This is crucial for deleted folders - if a deleted folder matches .ndignore,
			// we should ignore it BEFORE resolveFolderPath walks up to the parent
			if w.shouldIgnoreFolderPath(ctx, fsys, path) {
				log.Debug(ctx, "Ignoring change matching .ndignore pattern", "libraryID", lib.ID, "path", path)
				continue
			}

			// Find the folder to scan - validate path exists as directory, walk up if needed
			folderPath := resolveFolderPath(fsys, path)
			// Double-check after resolution in case the resolved path is different and also matches patterns
			if folderPath != path && w.shouldIgnoreFolderPath(ctx, fsys, folderPath) {
				log.Trace(ctx, "Ignoring change in folder matching .ndignore pattern", "libraryID", lib.ID, "folderPath", folderPath)
				continue
			}

			// Notify the main watcher of changes
			select {
			case w.watcherNotify <- scanNotification{Library: lib, FolderPath: folderPath}:
			default:
				// Channel is full, notification already pending
			}
		}
	}
}

// resolveFolderPath takes a path (which may be a file or directory) and returns
// the folder path to scan. If the path is a file, it walks up to find the parent
// directory. Returns empty string if the path should scan the library root.
func resolveFolderPath(fsys fs.FS, path string) string {
	// Handle root paths immediately
	if path == "." || path == "" {
		return ""
	}

	folderPath := path
	for {
		info, err := fs.Stat(fsys, folderPath)
		if err == nil && info.IsDir() {
			// Found a valid directory
			return folderPath
		}
		if folderPath == "." || folderPath == "" {
			// Reached root, scan entire library
			return ""
		}
		// Walk up the tree
		dir, _ := filepath.Split(folderPath)
		if dir == "" || dir == "." {
			return ""
		}
		// Remove trailing slash
		folderPath = filepath.Clean(dir)
	}
}

// shouldIgnoreFolderPath checks if the given folderPath should be ignored based on .ndignore patterns
// in the library. It pushes all parent folders onto the IgnoreChecker stack before checking.
func (w *watcher) shouldIgnoreFolderPath(ctx context.Context, fsys storage.MusicFS, folderPath string) bool {
	checker := newIgnoreChecker(fsys)
	err := checker.PushAllParents(ctx, folderPath)
	if err != nil {
		log.Warn(ctx, "Watcher: Error pushing ignore patterns for folder", "path", folderPath, err)
	}
	return checker.ShouldIgnore(ctx, folderPath)
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
