package scanner

import (
	"context"
	"os"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/utils"
)

type (
	dirStats struct {
		Path            string
		ModTime         time.Time
		HasImages       bool
		HasPlaylist     bool
		AudioFilesCount uint32
	}
	walkResults = chan dirStats
)

func unsortedfullReadDir(name string) ([]os.DirEntry, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	allDirs := []os.DirEntry{}
	for {
		dirs, err := f.ReadDir(-1)
		allDirs = append(allDirs, dirs...)
		if err == nil {
			break
		}
		if err != nil {
			log.Warn("Skipping DirEntry", err)
		}
	}
	return allDirs, nil
}

func walkDirTree(ctx context.Context, rootFolder string, results walkResults) error {
	err := walkFolder(ctx, rootFolder, rootFolder, results)
	if err != nil {
		log.Error(ctx, "Error loading directory tree", err)
	}
	close(results)
	return err
}

func walkFolder(ctx context.Context, rootPath string, currentFolder string, results walkResults) error {
	children, stats, err := loadDir(ctx, currentFolder)
	if err != nil {
		return err
	}
	for _, c := range children {
		err := walkFolder(ctx, rootPath, c, results)
		if err != nil {
			return err
		}
	}

	dir := filepath.Clean(currentFolder)
	log.Trace(ctx, "Found directory", "dir", dir, "audioCount", stats.AudioFilesCount,
		"hasImages", stats.HasImages, "hasPlaylist", stats.HasPlaylist)
	stats.Path = dir
	results <- stats

	return nil
}

func loadDir(ctx context.Context, dirPath string) (children []string, stats dirStats, err error) {
	dirInfo, err := os.Stat(dirPath)
	if err != nil {
		log.Error(ctx, "Error stating dir", "path", dirPath, err)
		return
	}
	stats.ModTime = dirInfo.ModTime()

	dirents, err := unsortedfullReadDir(dirPath)
	if err != nil {
		log.Error(ctx, "Error in ReadDir", "path", dirPath, err)
		return
	}
	for _, d := range dirents {
		isDir, err2 := isDirOrSymlinkToDir(dirPath, d)
		// Skip invalid symlinks
		if err2 != nil {
			log.Error(ctx, "Invalid symlink", "dir", filepath.Join(dirPath, d.Name()), err2)
			continue
		}
		if isDir && !isDirIgnored(dirPath, d) && isDirReadable(dirPath, d) {
			children = append(children, filepath.Join(dirPath, d.Name()))
		} else {
			fileInfo, err2 := d.Info()
			if err2 != nil {
				log.Error(ctx, "Error getting fileInfo", "name", d.Name(), err2)
				err = err2
				return
			}
			if fileInfo.ModTime().After(stats.ModTime) {
				stats.ModTime = fileInfo.ModTime()
			}
			if utils.IsAudioFile(d.Name()) {
				stats.AudioFilesCount++
			} else {
				stats.HasPlaylist = stats.HasPlaylist || utils.IsPlaylist(d.Name())
				stats.HasImages = stats.HasImages || utils.IsImageFile(d.Name())
			}
		}
	}
	return
}

// isDirOrSymlinkToDir returns true if and only if the dirEnt represents a file
// system directory, or a symbolic link to a directory. Note that if the dirEnt
// is not a directory but is a symbolic link, this method will resolve by
// sending a request to the operating system to follow the symbolic link.
// originally copied from github.com/karrick/godirwalk, modified to use dirEntry for efficiency
//      for go 1.16 and beyond
func isDirOrSymlinkToDir(baseDir string, dirEnt fs.DirEntry) (bool, error) {
	if dirEnt.IsDir() {
		return true, nil
	}
	if dirEnt.Type()&os.ModeSymlink == 0 {
		return false, nil
	}
	// Does this symlink point to a directory?
	fileInfo, err := os.Stat(filepath.Join(baseDir, dirEnt.Name()))
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), nil
}

// isDirIgnored returns true if the directory represented by dirEnt contains an
// `ignore` file (named after consts.SkipScanFile)
func isDirIgnored(baseDir string, dirEnt fs.DirEntry) bool {
        // allows Album folders for albums which eg start with ellipses
	if strings.HasPrefix(dirEnt.Name(), ".") && !strings.HasPrefix(dirEnt.Name(), "..") {
		return true
	}
	_, err := os.Stat(filepath.Join(baseDir, dirEnt.Name(), consts.SkipScanFile))
	return err == nil
}

// isDirReadable returns true if the directory represented by dirEnt is readable
func isDirReadable(baseDir string, dirEnt fs.DirEntry) bool {
	path := filepath.Join(baseDir, dirEnt.Name())
	res, err := utils.IsDirReadable(path)
	if !res {
		log.Warn("Skipping unreadable directory", "path", path, err)
	}
	return res
}
