package scanner2

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type scanJob struct {
	lib         model.Library
	fs          storage.MusicFS
	ds          model.DataStore
	startTime   time.Time
	lastUpdates map[string]time.Time
	folderLock  sync.RWMutex
	albumPIDs   map[string]string
	albumLock   sync.RWMutex
	fullRescan  bool
	numFolders  atomic.Int64
}

func newScanJob(ctx context.Context, ds model.DataStore, lib model.Library, fullRescan bool) (*scanJob, error) {
	lastUpdates, err := ds.Folder(ctx).GetLastUpdates(lib)
	if err != nil {
		return nil, fmt.Errorf("error getting last updates: %w", err)
	}
	fileStore, err := storage.For(lib.Path)
	if err != nil {
		log.Error(ctx, "Error getting storage for library", "library", lib.Name, "path", lib.Path, err)
		return nil, fmt.Errorf("error getting storage for library: %w", err)
	}
	fsys, err := fileStore.FS()
	if err != nil {
		log.Error(ctx, "Error getting fs for library", "library", lib.Name, "path", lib.Path, err)
		return nil, fmt.Errorf("error getting fs for library: %w", err)
	}
	albumPIDs, err := ds.Album(ctx).GetPIDs(lib.ID)
	if err != nil {
		log.Error(ctx, "Error getting album PIDs for library", "library", lib.Name, "path", lib.Path, err)
		return nil, fmt.Errorf("error getting album PIDs for library: %w", err)
	}
	return &scanJob{
		lib:         lib,
		fs:          fsys,
		ds:          ds,
		startTime:   time.Now(),
		lastUpdates: lastUpdates,
		albumPIDs:   albumPIDs,
		fullRescan:  fullRescan,
	}, nil
}

func (s *scanJob) getLastUpdatedInDB(folderId string) time.Time {
	s.folderLock.RLock()
	defer s.folderLock.RUnlock()

	t, ok := s.lastUpdates[folderId]
	if !ok {
		return time.Time{}
	}
	return t
}

func (s *scanJob) getAlbumID(albumPID string) string {
	// TODO: Use RLock or sync.Map
	s.albumLock.Lock()
	defer s.albumLock.Unlock()
	if id, ok := s.albumPIDs[albumPID]; ok {
		return id
	}
	id := uuid.NewString()
	s.albumPIDs[albumPID] = id
	return id
}
