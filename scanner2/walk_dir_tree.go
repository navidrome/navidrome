package scanner2

import (
	"context"
	"io/fs"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"golang.org/x/exp/maps"
)

type folderEntry struct {
	job             *scanJob
	startTime       time.Time
	path            string    // Full path
	id              string    // DB ID
	modTime         time.Time // From FS
	updTime         time.Time // from DB
	audioFiles      map[string]fs.DirEntry
	imageFiles      map[string]fs.DirEntry
	playlists       []fs.DirEntry
	imagesUpdatedAt time.Time
	tracks          model.MediaFiles
	albums          model.Albums
	artists         model.Artists
	tags            model.TagList
	missingTracks   model.MediaFiles
}

func newFolderEntry(job *scanJob, path string) *folderEntry {
	id := model.FolderID(job.lib, path)
	return &folderEntry{
		id:         id,
		job:        job,
		path:       path,
		startTime:  time.Now(),
		audioFiles: make(map[string]fs.DirEntry),
		imageFiles: make(map[string]fs.DirEntry),
		updTime:    job.popLastUpdate(id),
	}
}

func (f *folderEntry) isOutdated() bool {
	return f.updTime.Before(f.modTime)
}

func walkDirTree(ctx context.Context, job *scanJob) (<-chan *folderEntry, error) {
	results := make(chan *folderEntry)
	go func() {
		defer close(results)
		rootFolder := job.lib.Path
		err := walkFolder(ctx, job, ".", results)
		if err != nil {
			log.Error(ctx, "Scanner: There were errors reading directories from filesystem", "path", rootFolder, err)
			return
		}
		log.Debug(ctx, "Scanner: Finished reading folders", "lib", job.lib.Name, "path", rootFolder, "numFolders", job.numFolders.Load())
	}()
	return results, nil
}

func walkFolder(ctx context.Context, job *scanJob, currentFolder string, results chan<- *folderEntry) error {
	folder, children, err := loadDir(ctx, job, currentFolder)
	if err != nil {
		log.Warn(ctx, "Scanner: Error loading dir. Skipping", "path", currentFolder, err)
		return nil
	}
	for _, c := range children {
		err := walkFolder(ctx, job, c, results)
		if err != nil {
			return err
		}
	}

	//if !folder.isOutdated() && !job.fullRescan {
	//	return nil
	//}
	dir := filepath.Clean(currentFolder)
	log.Trace(ctx, "Scanner: Found directory", " path", dir, "audioFiles", maps.Keys(folder.audioFiles),
		"images", maps.Keys(folder.imageFiles), "playlists", folder.playlists, "imagesUpdatedAt", folder.imagesUpdatedAt,
		"updTime", folder.updTime, "modTime", folder.modTime, "numChildren", len(children))
	folder.path = dir
	results <- folder

	return nil
}

func loadDir(ctx context.Context, job *scanJob, dirPath string) (folder *folderEntry, children []string, err error) {
	folder = newFolderEntry(job, dirPath)

	dirInfo, err := fs.Stat(job.fs, dirPath)
	if err != nil {
		log.Warn(ctx, "Scanner: Error stating dir", "path", dirPath, err)
		return nil, nil, err
	}
	folder.modTime = dirInfo.ModTime()

	dir, err := job.fs.Open(dirPath)
	if err != nil {
		log.Warn(ctx, "Scanner: Error in Opening directory", "path", dirPath, err)
		return folder, children, err
	}
	defer dir.Close()
	dirFile, ok := dir.(fs.ReadDirFile)
	if !ok {
		log.Error(ctx, "Not a directory", "path", dirPath)
		return folder, children, err
	}

	for _, entry := range fullReadDir(ctx, dirFile) {
		if ctx.Err() != nil {
			return folder, children, ctx.Err()
		}
		isDir, err := isDirOrSymlinkToDir(job.fs, dirPath, entry)
		// Skip invalid symlinks
		if err != nil {
			log.Warn(ctx, "Scanner: Invalid symlink", "dir", filepath.Join(dirPath, entry.Name()), err)
			continue
		}
		if isDir && !isDirIgnored(job.fs, dirPath, entry) && isDirReadable(ctx, job.fs, dirPath, entry) {
			children = append(children, filepath.Join(dirPath, entry.Name()))
		} else {
			fileInfo, err := entry.Info()
			if err != nil {
				log.Warn(ctx, "Scanner: Error getting fileInfo", "name", entry.Name(), err)
				return folder, children, err
			}
			if fileInfo.ModTime().After(folder.modTime) {
				folder.modTime = fileInfo.ModTime()
			}
			switch {
			case model.IsAudioFile(entry.Name()):
				folder.audioFiles[entry.Name()] = entry
			case model.IsValidPlaylist(entry.Name()):
				folder.playlists = append(folder.playlists, entry)
			case model.IsImageFile(entry.Name()):
				folder.imageFiles[entry.Name()] = entry
				if fileInfo.ModTime().After(folder.imagesUpdatedAt) {
					folder.imagesUpdatedAt = fileInfo.ModTime()
				}
			}
		}
	}
	return folder, children, nil
}

// fullReadDir reads all files in the folder, skipping the ones with errors.
// It also detects when it is "stuck" with an error in the same directory over and over.
// In this case, it stops and returns whatever it was able to read until it got stuck.
// See discussion here: https://github.com/navidrome/navidrome/issues/1164#issuecomment-881922850
func fullReadDir(ctx context.Context, dir fs.ReadDirFile) []fs.DirEntry {
	var allEntries []fs.DirEntry
	var prevErrStr = ""
	for {
		if ctx.Err() != nil {
			return []fs.DirEntry{}
		}
		entries, err := dir.ReadDir(-1)
		allEntries = append(allEntries, entries...)
		if err == nil {
			break
		}
		log.Warn(ctx, "Skipping DirEntry", err)
		if prevErrStr == err.Error() {
			log.Error(ctx, "Scanner: Duplicate DirEntry failure, bailing", err)
			break
		}
		prevErrStr = err.Error()
	}
	sort.Slice(allEntries, func(i, j int) bool { return allEntries[i].Name() < allEntries[j].Name() })
	return allEntries
}

// isDirOrSymlinkToDir returns true if and only if the dirEnt represents a file
// system directory, or a symbolic link to a directory. Note that if the dirEnt
// is not a directory but is a symbolic link, this method will resolve by
// sending a request to the operating system to follow the symbolic link.
// originally copied from github.com/karrick/godirwalk, modified to use dirEntry for
// efficiency for go 1.16 and beyond
func isDirOrSymlinkToDir(fsys fs.FS, baseDir string, dirEnt fs.DirEntry) (bool, error) {
	if dirEnt.IsDir() {
		return true, nil
	}
	if dirEnt.Type()&fs.ModeSymlink == 0 {
		return false, nil
	}
	// Does this symlink point to a directory?
	fileInfo, err := fs.Stat(fsys, filepath.Join(baseDir, dirEnt.Name()))
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), nil
}

// isDirReadable returns true if the directory represented by dirEnt is readable
func isDirReadable(ctx context.Context, fsys fs.FS, baseDir string, dirEnt fs.DirEntry) bool {
	path := filepath.Join(baseDir, dirEnt.Name())

	dir, err := fsys.Open(path)
	if err != nil {
		log.Warn("Scanner: Skipping unreadable directory", "path", path, err)
		return false
	}

	err = dir.Close()
	if err != nil {
		log.Warn(ctx, "Scanner: Error closing directory", "path", path, err)
	}

	return true
}

// isDirIgnored returns true if the directory represented by dirEnt contains an
// `ignore` file (named after skipScanFile)
func isDirIgnored(fsys fs.FS, baseDir string, dirEnt fs.DirEntry) bool {
	// allows Album folders for albums which eg start with ellipses
	name := dirEnt.Name()
	if strings.HasPrefix(name, ".") && !strings.HasPrefix(name, "..") {
		return true
	}

	if runtime.GOOS == "windows" && strings.EqualFold(name, "$RECYCLE.BIN") {
		return true
	}
	_, err := fs.Stat(fsys, filepath.Join(baseDir, name, consts.SkipScanFile))
	return err == nil
}
