package scanner

import (
	"bufio"
	"context"
	"io/fs"
	"maps"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
	ignore "github.com/sabhiram/go-gitignore"
)

func walkDirTree(ctx context.Context, job *scanJob) (<-chan *folderEntry, error) {
	results := make(chan *folderEntry)
	go func() {
		defer close(results)
		err := walkFolder(ctx, job, ".", nil, results)
		if err != nil {
			log.Error(ctx, "Scanner: There were errors reading directories from filesystem", "path", job.lib.Path, err)
			return
		}
		log.Debug(ctx, "Scanner: Finished reading folders", "lib", job.lib.Name, "path", job.lib.Path, "numFolders", job.numFolders.Load())
	}()
	return results, nil
}

func walkFolder(ctx context.Context, job *scanJob, currentFolder string, ignorePatterns []string, results chan<- *folderEntry) error {
	ignorePatterns = loadIgnoredPatterns(ctx, job.fs, currentFolder, ignorePatterns)

	folder, children, err := loadDir(ctx, job, currentFolder, ignorePatterns)
	if err != nil {
		log.Warn(ctx, "Scanner: Error loading dir. Skipping", "path", currentFolder, err)
		return nil
	}
	for _, c := range children {
		err := walkFolder(ctx, job, c, ignorePatterns, results)
		if err != nil {
			return err
		}
	}

	dir := path.Clean(currentFolder)
	log.Trace(ctx, "Scanner: Found directory", " path", dir, "audioFiles", maps.Keys(folder.audioFiles),
		"images", maps.Keys(folder.imageFiles), "playlists", folder.numPlaylists, "imagesUpdatedAt", folder.imagesUpdatedAt,
		"updTime", folder.updTime, "modTime", folder.modTime, "numChildren", len(children))
	folder.path = dir
	folder.elapsed.Start()

	results <- folder

	return nil
}

func loadIgnoredPatterns(ctx context.Context, fsys fs.FS, currentFolder string, currentPatterns []string) []string {
	ignoreFilePath := path.Join(currentFolder, consts.ScanIgnoreFile)
	var newPatterns []string
	if _, err := fs.Stat(fsys, ignoreFilePath); err == nil {
		// Read and parse the .ndignore file
		ignoreFile, err := fsys.Open(ignoreFilePath)
		if err != nil {
			log.Warn(ctx, "Scanner: Error opening .ndignore file", "path", ignoreFilePath, err)
			// Continue with previous patterns
		} else {
			defer ignoreFile.Close()
			scanner := bufio.NewScanner(ignoreFile)
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" || strings.HasPrefix(line, "#") {
					continue // Skip empty lines and comments
				}
				newPatterns = append(newPatterns, line)
			}
			if err := scanner.Err(); err != nil {
				log.Warn(ctx, "Scanner: Error reading .ignore file", "path", ignoreFilePath, err)
			}
		}
		// If the .ndignore file is empty, mimic the current behavior and ignore everything
		if len(newPatterns) == 0 {
			log.Trace(ctx, "Scanner: .ndignore file is empty, ignoring everything", "path", currentFolder)
			newPatterns = []string{"**/*"}
		} else {
			log.Trace(ctx, "Scanner: .ndignore file found ", "path", ignoreFilePath, "patterns", newPatterns)
		}
	}
	// Combine the patterns from the .ndignore file with the ones passed as argument
	combinedPatterns := append([]string{}, currentPatterns...)
	return append(combinedPatterns, newPatterns...)
}

func loadDir(ctx context.Context, job *scanJob, dirPath string, ignorePatterns []string) (folder *folderEntry, children []string, err error) {
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

	ignoreMatcher := ignore.CompileIgnoreLines(ignorePatterns...)
	entries := fullReadDir(ctx, dirFile)
	children = make([]string, 0, len(entries))
	for _, entry := range entries {
		entryPath := path.Join(dirPath, entry.Name())
		if len(ignorePatterns) > 0 && isScanIgnored(ctx, ignoreMatcher, entryPath) {
			log.Trace(ctx, "Scanner: Ignoring entry", "path", entryPath)
			continue
		}
		if isEntryIgnored(entry.Name()) {
			continue
		}
		if ctx.Err() != nil {
			return folder, children, ctx.Err()
		}
		isDir, err := isDirOrSymlinkToDir(job.fs, dirPath, entry)
		// Skip invalid symlinks
		if err != nil {
			log.Warn(ctx, "Scanner: Invalid symlink", "dir", entryPath, err)
			continue
		}
		if isDir && !isDirIgnored(entry.Name()) && isDirReadable(ctx, job.fs, entryPath) {
			children = append(children, entryPath)
			folder.numSubFolders++
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
				folder.numPlaylists++
			case model.IsImageFile(entry.Name()):
				folder.imageFiles[entry.Name()] = entry
				folder.imagesUpdatedAt = utils.TimeNewest(folder.imagesUpdatedAt, fileInfo.ModTime(), folder.modTime)
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
			return nil
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
	// If symlinks are disabled, return false for symlinks
	if !conf.Server.Scanner.FollowSymlinks {
		return false, nil
	}
	// Does this symlink point to a directory?
	fileInfo, err := fs.Stat(fsys, path.Join(baseDir, dirEnt.Name()))
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), nil
}

// isDirReadable returns true if the directory represented by dirEnt is readable
func isDirReadable(ctx context.Context, fsys fs.FS, dirPath string) bool {
	dir, err := fsys.Open(dirPath)
	if err != nil {
		log.Warn("Scanner: Skipping unreadable directory", "path", dirPath, err)
		return false
	}
	err = dir.Close()
	if err != nil {
		log.Warn(ctx, "Scanner: Error closing directory", "path", dirPath, err)
	}
	return true
}

// List of special directories to ignore
var ignoredDirs = []string{
	"$RECYCLE.BIN",
	"#snapshot",
	"@Recently-Snapshot",
	".streams",
	"lost+found",
}

// isDirIgnored returns true if the directory represented by dirEnt should be ignored
func isDirIgnored(name string) bool {
	// allows Album folders for albums which eg start with ellipses
	if strings.HasPrefix(name, ".") && !strings.HasPrefix(name, "..") {
		return true
	}
	if slices.ContainsFunc(ignoredDirs, func(s string) bool { return strings.EqualFold(s, name) }) {
		return true
	}
	return false
}

func isEntryIgnored(name string) bool {
	return strings.HasPrefix(name, ".") && !strings.HasPrefix(name, "..")
}

func isScanIgnored(ctx context.Context, matcher *ignore.GitIgnore, entryPath string) bool {
	matches := matcher.MatchesPath(entryPath)
	if matches {
		log.Trace(ctx, "Scanner: Ignoring entry matching .ndignore: ", "path", entryPath)
	}
	return matches
}
