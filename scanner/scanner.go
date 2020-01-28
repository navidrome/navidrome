package scanner

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
)

type Scanner struct {
	folders map[string]FolderScanner
	ds      model.DataStore
}

func New(ds model.DataStore) *Scanner {
	s := &Scanner{ds: ds, folders: map[string]FolderScanner{}}
	s.loadFolders()
	return s
}

func (s *Scanner) Rescan(mediaFolder string, fullRescan bool) error {
	folderScanner := s.folders[mediaFolder]
	start := time.Now()

	lastModifiedSince := time.Time{}
	if !fullRescan {
		lastModifiedSince = s.getLastModifiedSince(mediaFolder)
		log.Debug("Scanning folder", "folder", mediaFolder, "lastModifiedSince", lastModifiedSince)
	} else {
		log.Debug("Scanning folder (full scan)", "folder", mediaFolder)
	}

	err := folderScanner.Scan(log.NewContext(nil), lastModifiedSince)
	if err != nil {
		log.Error("Error importing MediaFolder", "folder", mediaFolder, err)
	}

	s.updateLastModifiedSince(mediaFolder, start)
	log.Debug("Finished scanning folder", "folder", mediaFolder, "elapsed", time.Since(start))
	return err
}

func (s *Scanner) RescanAll(fullRescan bool) error {
	var hasError bool
	for folder := range s.folders {
		err := s.Rescan(folder, fullRescan)
		hasError = hasError || err != nil
	}
	if hasError {
		log.Error("Errors while scanning media. Please check the logs")
		return errors.New("errors while scanning media")
	}
	return nil
}

func (s *Scanner) Status() []StatusInfo { return nil }

func (s *Scanner) getLastModifiedSince(folder string) time.Time {
	ms, err := s.ds.Property(nil).Get(model.PropLastScan + "-" + folder)
	if err != nil {
		return time.Time{}
	}
	if ms == "" {
		return time.Time{}
	}
	i, _ := strconv.ParseInt(ms, 10, 64)
	return time.Unix(0, i*int64(time.Millisecond))
}

func (s *Scanner) updateLastModifiedSince(folder string, t time.Time) {
	millis := t.UnixNano() / int64(time.Millisecond)
	s.ds.Property(nil).Put(model.PropLastScan+"-"+folder, fmt.Sprint(millis))
}

func (s *Scanner) loadFolders() {
	fs, _ := s.ds.MediaFolder(nil).GetAll()
	for _, f := range fs {
		log.Info("Configuring Media Folder", "name", f.Name, "path", f.Path)
		s.folders[f.Path] = NewTagScanner(f.Path, s.ds)
	}
}

type Status int

const (
	StatusComplete Status = iota
	StatusInProgress
	StatusError
)

type StatusInfo struct {
	MediaFolder string
	Status      Status
}

type FolderScanner interface {
	Scan(ctx context.Context, lastModifiedSince time.Time) error
}
