package scanner

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/deluan/navidrome/core"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
)

type Scanner interface {
	Start(interval time.Duration) error
	Stop()
	RescanAll(fullRescan bool)
	Status(mediaFolder string) (*StatusInfo, error)
	Scanning() bool
}

type StatusInfo struct {
	MediaFolder string
	Scanning    bool
	LastScan    time.Time
	Count       uint32
}

type FolderScanner interface {
	Scan(ctx context.Context, lastModifiedSince time.Time, progress chan uint32) error
}

type scanner struct {
	folders     map[string]FolderScanner
	status      map[string]*scanStatus
	lock        *sync.RWMutex
	ds          model.DataStore
	cacheWarmer core.CacheWarmer
	done        chan bool
	scan        chan bool
}

type scanStatus struct {
	active     bool
	count      uint32
	lastUpdate time.Time
}

func New(ds model.DataStore, cacheWarmer core.CacheWarmer) Scanner {
	s := &scanner{
		ds:          ds,
		cacheWarmer: cacheWarmer,
		folders:     map[string]FolderScanner{},
		status:      map[string]*scanStatus{},
		lock:        &sync.RWMutex{},
		done:        make(chan bool),
		scan:        make(chan bool),
	}
	s.loadFolders()
	return s
}

func (s *scanner) Start(interval time.Duration) error {
	var ticker *time.Ticker
	if interval == 0 {
		log.Warn("Periodic scan is DISABLED", "interval", interval)
		ticker = time.NewTicker(1 * time.Hour)
	} else {
		ticker = time.NewTicker(interval)
	}
	defer ticker.Stop()
	for {
		select {
		case full := <-s.scan:
			s.rescanAll(full)
		case <-ticker.C:
			if interval != 0 {
				s.rescanAll(false)
			}
		case <-s.done:
			return nil
		}
	}
}

func (s *scanner) Stop() {
	s.done <- true
}

func (s *scanner) rescan(mediaFolder string, fullRescan bool) error {
	folderScanner := s.folders[mediaFolder]
	start := time.Now()

	lastModifiedSince := time.Time{}
	if !fullRescan {
		lastModifiedSince = s.getLastModifiedSince(mediaFolder)
		log.Debug("Scanning folder", "folder", mediaFolder, "lastModifiedSince", lastModifiedSince)
	} else {
		log.Debug("Scanning folder (full scan)", "folder", mediaFolder)
	}

	s.setStatusActive(mediaFolder, true)
	defer s.setStatus(mediaFolder, false, 0, start)

	progress := make(chan uint32, 10)
	go func() {
		for {
			count, more := <-progress
			if !more {
				break
			}
			atomic.AddUint32(&s.status[mediaFolder].count, count)
		}
	}()

	err := folderScanner.Scan(log.NewContext(context.TODO()), lastModifiedSince, progress)
	close(progress)
	if err != nil {
		log.Error("Error importing MediaFolder", "folder", mediaFolder, err)
	}

	s.updateLastModifiedSince(mediaFolder, start)
	return err
}

func (s *scanner) RescanAll(fullRescan bool) {
	if s.Scanning() {
		log.Debug("Scanner already running, ignoring request for rescan.")
		return
	}
	s.scan <- fullRescan
}

func (s *scanner) rescanAll(fullRescan bool) {
	defer s.cacheWarmer.Flush(context.Background())
	var hasError bool
	for folder := range s.folders {
		err := s.rescan(folder, fullRescan)
		hasError = hasError || err != nil
	}
	if hasError {
		log.Error("Errors while scanning media. Please check the logs")
	}
}

func (s *scanner) getStatus(folder string) *scanStatus {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if status, ok := s.status[folder]; ok {
		return status
	}
	return nil
}

func (s *scanner) setStatus(folder string, active bool, count uint32, lastUpdate time.Time) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if status, ok := s.status[folder]; ok {
		status.active = active
		status.count = count
		status.lastUpdate = lastUpdate
	}
}

func (s *scanner) setStatusActive(folder string, active bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if status, ok := s.status[folder]; ok {
		status.active = active
	}
}

func (s *scanner) Scanning() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, status := range s.status {
		if status.active {
			return true
		}
	}
	return false
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
		Count:       status.count,
	}, nil
}

func (s *scanner) getLastModifiedSince(folder string) time.Time {
	ms, err := s.ds.Property(context.TODO()).Get(model.PropLastScan + "-" + folder)
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
	fs, _ := s.ds.MediaFolder(context.TODO()).GetAll()
	for _, f := range fs {
		log.Info("Configuring Media Folder", "name", f.Name, "path", f.Path)
		s.folders[f.Path] = s.newScanner(f)
		s.status[f.Path] = &scanStatus{
			active:     false,
			count:      0,
			lastUpdate: s.getLastModifiedSince(f.Path),
		}
	}
}

func (s *scanner) newScanner(f model.MediaFolder) FolderScanner {
	return NewTagScanner(f.Path, s.ds, s.cacheWarmer)
}
