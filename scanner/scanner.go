package scanner

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"sync"
	"time"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/utils"
)

type Scanner interface {
	Run(ctx context.Context, interval time.Duration)
	RescanAll(ctx context.Context, fullRescan bool) error
	Status(mediaFolder string) (*StatusInfo, error)
	Scanning() bool
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
	Scan(ctx context.Context, lastModifiedSince time.Time, progress chan uint32) error
}

var isScanning utils.AtomicBool

type scanner struct {
	folders     map[string]FolderScanner
	status      map[string]*scanStatus
	lock        *sync.RWMutex
	ds          model.DataStore
	fsys        fs.FS
	cacheWarmer core.CacheWarmer
	broker      events.Broker
	scan        chan bool
}

type scanStatus struct {
	active      bool
	fileCount   uint32
	folderCount uint32
	lastUpdate  time.Time
}

func New(fsys fs.FS, ds model.DataStore, cacheWarmer core.CacheWarmer, broker events.Broker) Scanner {
	s := &scanner{
		ds:          ds,
		fsys:        fsys,
		cacheWarmer: cacheWarmer,
		broker:      broker,
		folders:     map[string]FolderScanner{},
		status:      map[string]*scanStatus{},
		lock:        &sync.RWMutex{},
		scan:        make(chan bool),
	}
	s.loadFolders()
	return s
}

func (s *scanner) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		err := s.RescanAll(ctx, false)
		if err != nil {
			log.Error(err)
		}
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return
		}
	}
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

	err := folderScanner.Scan(ctx, lastModifiedSince, progress)
	if err != nil {
		log.Error("Error importing MediaFolder", "folder", mediaFolder, err)
	}

	s.updateLastModifiedSince(mediaFolder, start)
	return err
}

func (s *scanner) startProgressTracker(mediaFolder string) (chan uint32, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	progress := make(chan uint32, 100)
	go func() {
		s.broker.SendMessage(&events.ScanStatus{Scanning: true, Count: 0, FolderCount: 0})
		defer func() {
			s.broker.SendMessage(&events.ScanStatus{
				Scanning:    false,
				Count:       int64(s.status[mediaFolder].fileCount),
				FolderCount: int64(s.status[mediaFolder].folderCount),
			})
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
				s.broker.SendMessage(&events.ScanStatus{
					Scanning:    true,
					Count:       int64(totalFiles),
					FolderCount: int64(totalFolders),
				})
			}
		}
	}()
	return progress, cancel
}

func (s *scanner) RescanAll(ctx context.Context, fullRescan bool) error {
	if s.Scanning() {
		log.Debug("Scanner already running, ignoring request for rescan.")
		return ErrAlreadyScanning
	}
	isScanning.Set(true)
	defer isScanning.Set(false)

	defer s.cacheWarmer.Flush(ctx)
	var hasError bool
	for folder := range s.folders {
		err := s.rescan(ctx, folder, fullRescan)
		hasError = hasError || err != nil
	}
	if hasError {
		log.Error("Errors while scanning media. Please check the logs")
		return ErrScanError
	}
	return nil
}

func (s *scanner) getStatus(folder string) *scanStatus {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if status, ok := s.status[folder]; ok {
		return status
	}
	return nil
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

func (s *scanner) Scanning() bool {
	return isScanning.Get()
}

func (s *scanner) Status(mediaFolder string) (*StatusInfo, error) {
	status := s.getStatus(mediaFolder)
	if status == nil {
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
		s.folders[f.Path] = s.newScanner(s.fsys, f)
		s.status[f.Path] = &scanStatus{
			active:      false,
			fileCount:   0,
			folderCount: 0,
			lastUpdate:  s.getLastModifiedSince(ctx, f.Path),
		}
	}
}

func (s *scanner) newScanner(fsys fs.FS, f model.MediaFolder) FolderScanner {
	return NewTagScanner(fsys, f.Path, s.ds, s.cacheWarmer)
}
