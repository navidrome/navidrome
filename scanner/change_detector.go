package scanner

import (
	"os"
	"path"
	"strings"
	"time"

	"github.com/cloudsonic/sonic-server/log"
)

type ChangeDetector struct {
	rootFolder string
	dirMap     map[string]time.Time
}

func NewChangeDetector(rootFolder string) *ChangeDetector {
	return &ChangeDetector{
		rootFolder: rootFolder,
		dirMap:     map[string]time.Time{},
	}
}

func (s *ChangeDetector) Scan(lastModifiedSince time.Time) (changed []string, deleted []string, err error) {
	start := time.Now()
	newMap := make(map[string]time.Time)
	err = s.loadMap(s.rootFolder, newMap)
	if err != nil {
		return
	}
	changed, deleted, err = s.checkForUpdates(lastModifiedSince, newMap)
	if err != nil {
		return
	}
	elapsed := time.Since(start)

	log.Trace("Folder analysis complete\n", "total", len(newMap), "changed", len(changed), "deleted", len(deleted), "elapsed", elapsed)
	s.dirMap = newMap
	return
}

func (s *ChangeDetector) loadDir(dirPath string) (children []string, lastUpdated time.Time, err error) {
	dir, err := os.Open(dirPath)
	defer dir.Close()
	if err != nil {
		return
	}
	dirInfo, err := os.Stat(dirPath)
	if err != nil {
		return
	}
	lastUpdated = dirInfo.ModTime()

	files, err := dir.Readdir(-1)
	if err != nil {
		return
	}
	for _, f := range files {
		if f.IsDir() {
			children = append(children, path.Join(dirPath, f.Name()))
		} else {
			if f.ModTime().After(lastUpdated) {
				lastUpdated = f.ModTime()
			}
		}
	}
	return
}

func (s *ChangeDetector) loadMap(rootPath string, dirMap map[string]time.Time) error {
	children, lastUpdated, err := s.loadDir(rootPath)
	if err != nil {
		return err
	}
	for _, c := range children {
		err := s.loadMap(c, dirMap)
		if err != nil {
			return err
		}
	}

	dir := s.getRelativePath(rootPath)
	dirMap[dir] = lastUpdated

	return nil
}

func (s *ChangeDetector) getRelativePath(subfolder string) string {
	dir := strings.TrimPrefix(subfolder, s.rootFolder)
	if dir == "" {
		dir = "."
	}
	return dir
}

func (s *ChangeDetector) checkForUpdates(lastModifiedSince time.Time, newMap map[string]time.Time) (changed []string, deleted []string, err error) {
	for dir, lastUpdated := range newMap {
		oldLastUpdated, ok := s.dirMap[dir]
		if !ok {
			oldLastUpdated = lastModifiedSince
		}
		if lastUpdated.After(oldLastUpdated) {
			changed = append(changed, dir)
		}
	}
	for dir := range s.dirMap {
		if _, ok := newMap[dir]; !ok {
			deleted = append(deleted, dir)
		}
	}
	return
}
