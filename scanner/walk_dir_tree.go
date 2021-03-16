package scanner

import (
	"context"
	"io/fs"
	"os"
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

func walkDirTree(ctx context.Context, fsys fs.FS, rootFolder string, results walkResults) error {
	err := walkFolder(ctx, fsys, rootFolder, rootFolder, results)
	if err != nil {
		log.Error(ctx, "Error loading directory tree", err)
	}
	close(results)
	return err
}

func walkFolder(ctx context.Context, fsys fs.FS, rootPath string, currentFolder string, results walkResults) error {
	children, stats, err := loadDir(ctx, fsys, currentFolder)
	if err != nil {
		return err
	}
	for _, c := range children {
		err := walkFolder(ctx, fsys, rootPath, c, results)
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

func loadDir(ctx context.Context, fsys fs.FS, dirPath string) (children []string, stats dirStats, err error) {
	dirInfo, err := fs.Stat(fsys, dirPath)
	if err != nil {
		log.Error(ctx, "Error stating dir", "path", dirPath, err)
		return
	}
	stats.ModTime = dirInfo.ModTime()

	files, err := fs.ReadDir(fsys, dirPath)
	if err != nil {
		log.Error(ctx, "Error reading dir", "path", dirPath, err)
		return
	}
	for _, f := range files {
		info, err := f.Info()
		if err != nil {
			log.Error(ctx, "Error reading dir entry", "dir", f.Name())
			continue
		}
		isDir, err := isDirOrSymlinkToDir(fsys, dirPath, info)
		// Skip invalid symlinks
		if err != nil {
			log.Error(ctx, "Invalid symlink", "dir", dirPath)
			continue
		}
		if isDir && !isDirIgnored(fsys, dirPath, info) && isDirReadable(fsys, dirPath, info) {
			children = append(children, filepath.Join(dirPath, f.Name()))
		} else {
			info, err := f.Info()
			if err != nil {
				log.Error(ctx, "Error dirEntry", "entry", filepath.Join(dirPath, f.Name()))
				continue
			}
			if info.ModTime().After(stats.ModTime) {
				stats.ModTime = info.ModTime()
			}
			if utils.IsAudioFile(f.Name()) {
				stats.AudioFilesCount++
			} else {
				stats.HasPlaylist = stats.HasPlaylist || utils.IsPlaylist(f.Name())
				stats.HasImages = stats.HasImages || utils.IsImageFile(f.Name())
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
func isDirOrSymlinkToDir(fsys fs.FS, baseDir string, dirEntry fs.FileInfo) (bool, error) {
	if dirEntry.IsDir() {
		return true, nil
	}
	if dirEntry.Mode().Type()&os.ModeSymlink == 0 {
		return false, nil
	}
	// Does this symlink point to a directory?
	fileInfo, err := fs.Stat(fsys, filepath.Join(baseDir, dirEntry.Name()))
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), nil
}

// isDirIgnored returns true if the directory represented by dirInfo contains an
// `ignore` file (named after consts.SkipScanFile)
func isDirIgnored(fsys fs.FS, baseDir string, dirInfo fs.FileInfo) bool {
	if strings.HasPrefix(dirInfo.Name(), ".") {
		return true
	}
	_, err := fs.Stat(fsys, filepath.Join(baseDir, dirInfo.Name(), consts.SkipScanFile))
	return err == nil
}

// isDirReadable returns true if the directory represented by dirInfo is readable
func isDirReadable(fsys fs.FS, baseDir string, dirInfo fs.FileInfo) bool {
	path := filepath.Join(baseDir, dirInfo.Name())
	res, err := utils.IsDirReadable(fsys, path)
	if !res {
		log.Debug("Warning: Skipping unreadable directory", "path", path, err)
	}
	return res
}
