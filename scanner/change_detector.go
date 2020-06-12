package scanner

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/log"
)

type dirInfo struct {
	mdate time.Time
	maybe bool
}
type dirInfoMap map[string]dirInfo

type ChangeDetector struct {
	rootFolder string
	dirMap     dirInfoMap
}

func NewChangeDetector(rootFolder string) *ChangeDetector {
	return &ChangeDetector{
		rootFolder: rootFolder,
		dirMap:     dirInfoMap{},
	}
}

func (s *ChangeDetector) Scan(lastModifiedSince time.Time) (changed []string, deleted []string, err error) {
	start := time.Now()
	newMap := make(dirInfoMap)
	err = s.loadMap(newMap, s.rootFolder, lastModifiedSince, false)
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
	dirInfo, err := os.Stat(dirPath)
	if err != nil {
		return
	}
	lastUpdated = dirInfo.ModTime()

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
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

// isDirOrSymlinkToDir returns true if and only if the dirInfo represents a file
// system directory, or a symbolic link to a directory. Note that if the dirInfo
// is not a directory but is a symbolic link, this method will resolve by
// sending a request to the operating system to follow the symbolic link.
// Copied from github.com/karrick/godirwalk
func isDirOrSymlinkToDir(baseDir string, dirInfo os.FileInfo) (bool, error) {
	if dirInfo.IsDir() {
		return true, nil
	}
	if dirInfo.Mode()&os.ModeSymlink == 0 {
		return false, nil
	}
	// Does this symlink point to a directory?
	dirInfo, err := os.Stat(filepath.Join(baseDir, dirInfo.Name()))
	if err != nil {
		return false, err
	}
	return dirInfo.IsDir(), nil
}

// isDirIgnored returns true if the directory represented by dirInfo contains an
// `ignore` file (named after consts.SkipScanFile)
func isDirIgnored(baseDir string, dirInfo os.FileInfo) bool {
	_, err := os.Stat(filepath.Join(baseDir, dirInfo.Name(), consts.SkipScanFile))
	return err == nil
}

// isDirReadable returns true if the directory represented by dirInfo is readable
func isDirReadable(baseDir string, dirInfo os.FileInfo) bool {
	path := filepath.Join(baseDir, dirInfo.Name())
	dir, err := os.Open(path)
	if err != nil {
		log.Debug("Warning: Skipping unreadable directory", "path", path, err)
		return false
	}
	if err := dir.Close(); err != nil {
		log.Error("Error closing directory", "path", path, err)
	}
	return true
}

func (s *ChangeDetector) loadMap(dirMap dirInfoMap, path string, since time.Time, maybe bool) error {
	children, lastUpdated, err := s.loadDir(path)
	if err != nil {
		return err
	}
	maybe = maybe || lastUpdated.After(since)
	for _, c := range children {
		err := s.loadMap(dirMap, c, since, maybe)
		if err != nil {
			return err
		}
	}

	dir := s.getRelativePath(path)
	dirMap[dir] = dirInfo{mdate: lastUpdated, maybe: maybe}

	return nil
}

func (s *ChangeDetector) getRelativePath(subFolder string) string {
	dir, _ := filepath.Rel(s.rootFolder, subFolder)
	if dir == "" {
		dir = "."
	}
	return dir
}

func (s *ChangeDetector) checkForUpdates(lastModifiedSince time.Time, newMap dirInfoMap) (changed []string, deleted []string, err error) {
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
