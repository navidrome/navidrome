package scanner

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/events"
)

type Scanner interface {
	RescanAll(ctx context.Context, fullRescan bool) error
	Status(mediaFolder string) (*StatusInfo, error)
}

type StatusInfo struct {
	MediaFolder string
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
	Scan(ctx context.Context, lastModifiedSince time.Time, progress chan uint32) (int64, error)
}

var isScanning sync.Mutex

type scanner struct {
	folders     map[string]FolderScanner
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

func New(ds model.DataStore, playlists core.Playlists, cacheWarmer artwork.CacheWarmer, broker events.Broker) Scanner {
	s := &scanner{
		ds:          ds,
		pls:         playlists,
		broker:      broker,
		folders:     map[string]FolderScanner{},
		status:      map[string]*scanStatus{},
		lock:        &sync.RWMutex{},
		cacheWarmer: cacheWarmer,
	}
	s.loadFolders()
	return s
}

func (s *scanner) rescan(ctx context.Context, mediaFolder string, fullRescan bool) error {
	folderScanner := s.folders[mediaFolder]
	start := time.Now()

	s.setStatusStart(mediaFolder)
	defer s.setStatusEnd(mediaFolder, start)

	lastModifiedSince := time.Time{}
	if !fullRescan {
		lastModifiedSince = s.getLastModifiedSince(ctx, mediaFolder)
		log.Debug("Scanning folder", "folder", mediaFolder, "lastModifiedSince", lastModifiedSince)
	} else {
		log.Debug("Scanning folder (full scan)", "folder", mediaFolder)
	}

	progress, cancel := s.startProgressTracker(mediaFolder)
	defer cancel()

	changeCount, err := folderScanner.Scan(ctx, lastModifiedSince, progress)
	if err != nil {
		log.Error("Error importing MediaFolder", "folder", mediaFolder, err)
	}

	if changeCount > 0 {
		log.Debug(ctx, "Detected changes in the music folder. Sending refresh event",
			"folder", mediaFolder, "changeCount", changeCount)
		// Don't use real context, forcing a refresh in all open windows, including the one that triggered the scan
		s.broker.SendMessage(context.Background(), &events.RefreshResource{})
	}

	s.updateLastModifiedSince(mediaFolder, start)
	return err
}

func (s *scanner) startProgressTracker(mediaFolder string) (chan uint32, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	progress := make(chan uint32, 100)
	go func() {
		s.broker.SendMessage(ctx, &events.ScanStatus{Scanning: true, Count: 0, FolderCount: 0})
		defer func() {
			if status, ok := s.getStatus(mediaFolder); ok {
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
				totalFolders, totalFiles := s.incStatusCounter(mediaFolder, count)
				s.broker.SendMessage(ctx, &events.ScanStatus{
					Scanning:    true,
					Count:       int64(totalFiles),
					FolderCount: int64(totalFolders),
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
	ctx = contextWithoutCancel(ctx)
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
func (s *scanner) Status(mediaFolder string) (*StatusInfo, error) {
	status, ok := s.getStatus(mediaFolder)
	if !ok {
		return nil, errors.New("mediaFolder not found")
	}
	return &StatusInfo{
		MediaFolder: mediaFolder,
		Scanning:    status.active,
		LastScan:    status.lastUpdate,
		Count:       status.fileCount,
		FolderCount: status.folderCount,
	}, nil
}

func (s *scanner) getLastModifiedSince(ctx context.Context, folder string) time.Time {
	ms, err := s.ds.Property(ctx).Get(model.PropLastScan + "-" + folder)
	if err != nil {
		return time.Time{}
	}
	if ms == "" {
		return time.Time{}
	}
	i, _ := strconv.ParseInt(ms, 10, 64)
	return time.Unix(0, i*int64(time.Millisecond))
}

func (s *scanner) updateLastModifiedSince(folder string, t time.Time) {
	millis := t.UnixNano() / int64(time.Millisecond)
	if err := s.ds.Property(context.TODO()).Put(model.PropLastScan+"-"+folder, fmt.Sprint(millis)); err != nil {
		log.Error("Error updating DB after scan", err)
	}
}

func (s *scanner) loadFolders() {
	ctx := context.TODO()
	fs, _ := s.ds.MediaFolder(ctx).GetAll()
	for _, f := range fs {
		log.Info("Configuring Media Folder", "name", f.Name, "path", f.Path)
		s.folders[f.Path] = s.newScanner(f)
		s.status[f.Path] = &scanStatus{
			active:      false,
			fileCount:   0,
			folderCount: 0,
			lastUpdate:  s.getLastModifiedSince(ctx, f.Path),
		}
	}
}

func (s *scanner) newScanner(f model.MediaFolder) FolderScanner {
	return NewTagScanner(f.Path, s.ds, s.pls, s.cacheWarmer)
}
