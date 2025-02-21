package scanner

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type Watcher interface {
	Run(ctx context.Context) error
}

type watcher struct {
	ds          model.DataStore
	scanner     Scanner
	triggerWait time.Duration
}

func NewWatcher(ds model.DataStore, s Scanner) Watcher {
	return &watcher{ds: ds, scanner: s, triggerWait: conf.Server.Scanner.WatcherWait}
}

func (w *watcher) Run(ctx context.Context) error {
	libs, err := w.ds.Library(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("getting libraries: %w", err)
	}

	watcherChan := make(chan struct{})
	defer close(watcherChan)

	// Start a watcher for each library
	for _, lib := range libs {
		go watchLib(ctx, lib, watcherChan)
	}

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
			return nil
		case <-watcherChan:
			if !waiting {
				log.Debug(ctx, "Watcher: Detected changes. Waiting for more changes before triggering scan")
				waiting = true
			}

			trigger.Reset(w.triggerWait)
		}
	}
}

func watchLib(ctx context.Context, lib model.Library, watchChan chan struct{}) {
	s, err := storage.For(lib.Path)
	if err != nil {
		log.Error(ctx, "Watcher: Error creating storage", "library", lib.ID, "path", lib.Path, err)
		return
	}
	fsys, err := s.FS()
	if err != nil {
		log.Error(ctx, "Watcher: Error getting FS", "library", lib.ID, "path", lib.Path, err)
		return
	}
	watcher, ok := s.(storage.Watcher)
	if !ok {
		log.Info(ctx, "Watcher not supported", "library", lib.ID, "path", lib.Path)
		return
	}
	c, err := watcher.Start(ctx)
	if err != nil {
		log.Error(ctx, "Watcher: Error watching library", "library", lib.ID, "path", lib.Path, err)
		return
	}
	log.Info(ctx, "Watcher started", "library", lib.ID, "path", lib.Path)
	for {
		select {
		case <-ctx.Done():
			return
		case path := <-c:
			path, err = filepath.Rel(lib.Path, path)
			if err != nil {
				log.Error(ctx, "Watcher: Error getting relative path", "library", lib.ID, "path", path, err)
				continue
			}
			if isIgnoredPath(ctx, fsys, path) {
				log.Trace(ctx, "Watcher: Ignoring change", "library", lib.ID, "path", path)
				continue
			}
			log.Trace(ctx, "Watcher: Detected change", "library", lib.ID, "path", path)
			watchChan <- struct{}{}
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
