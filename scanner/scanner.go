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
	"github.com/deluan/navidrome/server/events"
	"github.com/deluan/navidrome/utils"
)

type Scanner interface {
	Start(interval time.Duration)
	Stop()
	RescanAll(fullRescan bool) error
	Status(mediaFolder string) (*StatusInfo, error)
	Scanning() bool
}

type StatusInfo struct {
	MediaFolder string
	Scanning    bool
	LastScan    time.Time
	Count       uint32
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
	cacheWarmer core.CacheWarmer
	broker      events.Broker
	done        chan bool
	scan        chan bool
}

type scanStatus struct {
	active     bool
	count      uint32
	lastUpdate time.Time
}

func New(ds model.DataStore, cacheWarmer core.CacheWarmer, broker events.Broker) Scanner {
	s := &scanner{
		ds:          ds,
		cacheWarmer: cacheWarmer,
		broker:      broker,
		folders:     map[string]FolderScanner{},
		status:      map[string]*scanStatus{},
		lock:        &sync.RWMutex{},
		done:        make(chan bool),
		scan:        make(chan bool),
	}
	s.loadFolders()
	return s
}

func (s *scanner) Start(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		err := s.RescanAll(false)
		if err != nil {
			log.Error(err)
		}
		select {
		case <-ticker.C:
			continue
		case <-s.done:
			return
		}
	}
}

func (s *scanner) Stop() {
	s.done <- true
}

func (s *scanner) rescan(mediaFolder string, fullRescan bool) error {
	folderScanner := s.folders[mediaFolder]
	start := time.Now()

	s.setStatusStart(mediaFolder)
	defer s.setStatusEnd(mediaFolder, start)

	lastModifiedSince := time.Time{}
	if !fullRescan {
		lastModifiedSince = s.getLastModifiedSince(mediaFolder)
		log.Debug("Scanning folder", "folder", mediaFolder, "lastModifiedSince", lastModifiedSince)
	} else {
		log.Debug("Scanning folder (full scan)", "folder", mediaFolder)
	}

	progress := make(chan uint32, 100)
	go func() {
		defer func() {
			s.broker.SendMessage(&events.ScanStatus{Scanning: false, Count: int64(s.status[mediaFolder].count)})
		}()
		for {
			count, more := <-progress
			if !more {
				break
			}
			if count == 0 {
				continue
			}
			total := atomic.AddUint32(&s.status[mediaFolder].count, count)
			s.broker.SendMessage(&events.ScanStatus{Scanning: true, Count: int64(total)})
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

func (s *scanner) RescanAll(fullRescan bool) error {
	if s.Scanning() {
		log.Debug("Scanner already running, ignoring request for rescan.")
		return ErrAlreadyScanning
	}
	isScanning.Set(true)
	defer func() { isScanning.Set(false) }()

	defer s.cacheWarmer.Flush(context.Background())
	var hasError bool
	for folder := range s.folders {
		err := s.rescan(folder, fullRescan)
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

func (s *scanner) setStatusStart(folder string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if status, ok := s.status[folder]; ok {
		status.active = true
		status.count = 0
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
