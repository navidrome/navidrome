package scanner

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/deluan/navidrome/log"
)

type dirInfo struct {
	mdate time.Time
	maybe bool
}
type dirInfoMap map[string]dirInfo

type changeDetector struct {
	rootFolder string
	dirMap     dirInfoMap
}

func newChangeDetector(rootFolder string) *changeDetector {
	return &changeDetector{
		rootFolder: rootFolder,
		dirMap:     dirInfoMap{},
	}
}

func (s *changeDetector) Scan(ctx context.Context, lastModifiedSince time.Time) (changed []string, deleted []string, err error) {
	start := time.Now()
	newMap := make(dirInfoMap)
	err = s.loadMap(ctx, newMap, s.rootFolder, lastModifiedSince, false)
	if err != nil {
		return
	}
	changed, deleted, err = s.checkForUpdates(lastModifiedSince, newMap)
	if err != nil {
		return
	}
	elapsed := time.Since(start)

	log.Trace(ctx, "Folder analysis complete", "total", len(newMap), "changed", len(changed), "deleted", len(deleted), "elapsed", elapsed)
	s.dirMap = newMap
	return
}

func (s *changeDetector) loadDir(ctx context.Context, dirPath string) (children []string, lastUpdated time.Time, err error) {
	dirInfo, err := os.Stat(dirPath)
	if err != nil {
		log.Error(ctx, "Error stating dir", "path", dirPath, err)
		return
	}
	lastUpdated = dirInfo.ModTime()

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		log.Error(ctx, "Error reading dir", "path", dirPath, err)
		return
	}
	for _, f := range files {
		isDir, err := isDirOrSymlinkToDir(dirPath, f)
		// Skip invalid symlinks
		if err != nil {
			continue
		}
		if isDir && !isDirIgnored(dirPath, f) && isDirReadable(dirPath, f) {
			children = append(children, filepath.Join(dirPath, f.Name()))
		} else {
			if f.ModTime().After(lastUpdated) {
				lastUpdated = f.ModTime()
			}
		}
	}
	return
}

func (s *changeDetector) loadMap(ctx context.Context, dirMap dirInfoMap, path string, since time.Time, maybe bool) error {
	children, lastUpdated, err := s.loadDir(ctx, path)
	if err != nil {
		return err
	}
	maybe = maybe || lastUpdated.After(since)
	for _, c := range children {
		err := s.loadMap(ctx, dirMap, c, since, maybe)
		if err != nil {
			return err
		}
	}

	dir := s.getRelativePath(path)
	dirMap[dir] = dirInfo{mdate: lastUpdated, maybe: maybe}

	return nil
}

func (s *changeDetector) getRelativePath(subFolder string) string {
	dir, _ := filepath.Rel(s.rootFolder, subFolder)
	if dir == "" {
		dir = "."
	}
	return dir
}

func (s *changeDetector) checkForUpdates(lastModifiedSince time.Time, newMap dirInfoMap) (changed []string, deleted []string, err error) {
	for dir, newEntry := range newMap {
		lastUpdated := newEntry.mdate
		oldLastUpdated := lastModifiedSince
		if oldEntry, ok := s.dirMap[dir]; ok {
			oldLastUpdated = oldEntry.mdate
		} else {
			if newEntry.maybe {
				oldLastUpdated = time.Time{}
			}
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
