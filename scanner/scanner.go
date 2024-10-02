package scanner

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/utils/singleton"
	"golang.org/x/time/rate"
)

type Scanner interface {
	RescanAll(ctx context.Context, fullRescan bool) error
	Status(library string) (*StatusInfo, error)
}

type StatusInfo struct {
	Library     string
	Scanning    bool
	LastScan    time.Time
	Count       uint32
	FolderCount uint32
}

var (
	ErrAlreadyScanning = errors.New("already scanning")
	ErrScanError       = errors.New("scan error")
)

type FolderScanner interface {
	// Scan process finds any changes after `lastModifiedSince` and returns the number of changes found
	Scan(ctx context.Context, lib model.Library, fullRescan bool, progress chan uint32) (int64, error)
}

var isScanning sync.Mutex

type scanner struct {
	once        sync.Once
	folders     map[string]FolderScanner
	libs        map[string]model.Library
	status      map[string]*scanStatus
	lock        *sync.RWMutex
	ds          model.DataStore
	pls         core.Playlists
	broker      events.Broker
	cacheWarmer artwork.CacheWarmer
}

type scanStatus struct {
	active      bool
	fileCount   uint32
	folderCount uint32
	lastUpdate  time.Time
}

func GetInstance(ds model.DataStore, playlists core.Playlists, cacheWarmer artwork.CacheWarmer, broker events.Broker) Scanner {
	return singleton.GetInstance(func() *scanner {
		s := &scanner{
			ds:          ds,
			pls:         playlists,
			broker:      broker,
			folders:     map[string]FolderScanner{},
			libs:        map[string]model.Library{},
			status:      map[string]*scanStatus{},
			lock:        &sync.RWMutex{},
			cacheWarmer: cacheWarmer,
		}
		s.loadFolders()
		return s
	})
}

func (s *scanner) rescan(ctx context.Context, library string, fullRescan bool) error {
	folderScanner := s.folders[library]
	start := time.Now()

	lib, ok := s.libs[library]
	if !ok {
		log.Error(ctx, "Folder not a valid library path", "folder", library)
		return fmt.Errorf("folder %s not a valid library path", library)
	}

	s.setStatusStart(library)
	defer s.setStatusEnd(library, start)

	if fullRescan {
		log.Debug("Scanning folder (full scan)", "folder", library)
	} else {
		log.Debug("Scanning folder", "folder", library, "lastScan", lib.LastScanAt)
	}

	progress, cancel := s.startProgressTracker(library)
	defer cancel()

	changeCount, err := folderScanner.Scan(ctx, lib, fullRescan, progress)
	if err != nil {
		log.Error("Error scanning Library", "folder", library, err)
	}

	if changeCount > 0 {
		log.Debug(ctx, "Detected changes in the music folder. Sending refresh event",
			"folder", library, "changeCount", changeCount)
		// Don't use real context, forcing a refresh in all open windows, including the one that triggered the scan
		s.broker.SendMessage(context.Background(), &events.RefreshResource{})
	}

	s.updateLastModifiedSince(ctx, library, start)
	return err
}

func (s *scanner) startProgressTracker(library string) (chan uint32, context.CancelFunc) {
	// Must be a new context (not the one passed to the scan method) to allow broadcasting the scan status to all clients
	ctx, cancel := context.WithCancel(context.Background())
	progress := make(chan uint32, 1000)
	limiter := rate.Sometimes{Interval: conf.Server.DevActivityPanelUpdateRate}
	go func() {
		s.broker.SendMessage(ctx, &events.ScanStatus{Scanning: true, Count: 0, FolderCount: 0})
		defer func() {
			if status, ok := s.getStatus(library); ok {
				s.broker.SendMessage(ctx, &events.ScanStatus{
					Scanning:    false,
					Count:       int64(status.fileCount),
					FolderCount: int64(status.folderCount),
				})
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case count := <-progress:
				if count == 0 {
					continue
				}
				totalFolders, totalFiles := s.incStatusCounter(library, count)
				limiter.Do(func() {
					s.broker.SendMessage(ctx, &events.ScanStatus{
						Scanning:    true,
						Count:       int64(totalFiles),
						FolderCount: int64(totalFolders),
					})
				})
			}
		}
	}()
	return progress, cancel
}

func (s *scanner) getStatus(folder string) (scanStatus, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	status, ok := s.status[folder]
	return *status, ok
}

func (s *scanner) incStatusCounter(folder string, numFiles uint32) (totalFolders uint32, totalFiles uint32) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if status, ok := s.status[folder]; ok {
		status.fileCount += numFiles
		status.folderCount++
		totalFolders = status.folderCount
		totalFiles = status.fileCount
	}
	return
}

func (s *scanner) setStatusStart(folder string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if status, ok := s.status[folder]; ok {
		status.active = true
		status.fileCount = 0
		status.folderCount = 0
	}
}

func (s *scanner) setStatusEnd(folder string, lastUpdate time.Time) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if status, ok := s.status[folder]; ok {
		status.active = false
		status.lastUpdate = lastUpdate
	}
}

func (s *scanner) RescanAll(ctx context.Context, fullRescan bool) error {
	ctx = context.WithoutCancel(ctx)
	s.once.Do(s.loadFolders)

	if !isScanning.TryLock() {
		log.Debug(ctx, "Scanner already running, ignoring request for rescan.")
		return ErrAlreadyScanning
	}
	defer isScanning.Unlock()

	var hasError bool
	for folder := range s.folders {
		err := s.rescan(ctx, folder, fullRescan)
		hasError = hasError || err != nil
	}
	if hasError {
		log.Error(ctx, "Errors while scanning media. Please check the logs")
		core.WriteAfterScanMetrics(ctx, s.ds, false)
		return ErrScanError
	}
	core.WriteAfterScanMetrics(ctx, s.ds, true)
	return nil
}

func (s *scanner) Status(library string) (*StatusInfo, error) {
	s.once.Do(s.loadFolders)
	status, ok := s.getStatus(library)
	if !ok {
		return nil, errors.New("library not found")
	}
	return &StatusInfo{
		Library:     library,
		Scanning:    status.active,
		LastScan:    status.lastUpdate,
		Count:       status.fileCount,
		FolderCount: status.folderCount,
	}, nil
}

func (s *scanner) updateLastModifiedSince(ctx context.Context, folder string, t time.Time) {
	lib := s.libs[folder]
	id := lib.ID
	if err := s.ds.Library(ctx).UpdateLastScan(id, t); err != nil {
		log.Error("Error updating DB after scan", err)
	}
	lib.LastScanAt = t
	s.libs[folder] = lib
}

func (s *scanner) loadFolders() {
	ctx := context.TODO()
	libs, _ := s.ds.Library(ctx).GetAll()
	for _, lib := range libs {
		log.Info("Configuring Media Folder", "name", lib.Name, "path", lib.Path)
		s.folders[lib.Path] = s.newScanner()
		s.libs[lib.Path] = lib
		s.status[lib.Path] = &scanStatus{
			active:      false,
			fileCount:   0,
			folderCount: 0,
			lastUpdate:  lib.LastScanAt,
		}
	}
}

func (s *scanner) newScanner() FolderScanner {
	return NewTagScanner(s.ds, s.pls, s.cacheWarmer)
}
