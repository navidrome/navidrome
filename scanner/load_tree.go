package scanner

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/utils"
)

type (
	dirMapValue struct {
		modTime     time.Time
		hasPlaylist bool
	}
	dirMap = map[string]dirMapValue
)

func loadDirTree(ctx context.Context, rootFolder string) (dirMap, error) {
	newMap := make(dirMap)
	err := loadMap(ctx, rootFolder, rootFolder, newMap)
	if err != nil {
		log.Error(ctx, "Error loading directory tree", err)
	}
	return newMap, err
}

func loadMap(ctx context.Context, rootPath string, currentFolder string, dirMap dirMap) error {
	children, dirMapValue, err := loadDir(ctx, currentFolder)
	if err != nil {
		return err
	}
	for _, c := range children {
		err := loadMap(ctx, rootPath, c, dirMap)
		if err != nil {
			return err
		}
	}

	dir := filepath.Clean(currentFolder)
	dirMap[dir] = dirMapValue

	return nil
}

func loadDir(ctx context.Context, dirPath string) (children []string, info dirMapValue, err error) {
	dirInfo, err := os.Stat(dirPath)
	if err != nil {
		log.Error(ctx, "Error stating dir", "path", dirPath, err)
		return
	}
	info.modTime = dirInfo.ModTime()

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
			if f.ModTime().After(info.modTime) {
				info.modTime = f.ModTime()
			}
			info.hasPlaylist = info.hasPlaylist || utils.IsPlaylist(f.Name())
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
	res, err := utils.IsDirReadable(path)
	if !res {
		log.Debug("Warning: Skipping unreadable directory", "path", path, err)
	}
	return res
}
