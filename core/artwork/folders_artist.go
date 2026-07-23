package artwork

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/str"
)

const (
	// maxArtistFolderTraversalDepth defines how many directory levels to search
	// when looking for artist images (artist folder + parent directories)
	maxArtistFolderTraversalDepth = 3
)

func fromArtistFolder(ctx context.Context, libFS fs.FS, libPath, artistFolder, pattern string) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		if libFS == nil {
			return nil, "", fmt.Errorf("artist folder lookup unavailable")
		}
		rel, err := filepath.Rel(libPath, artistFolder)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, "", fmt.Errorf(`artist folder '%s' is outside library '%s'`, artistFolder, libPath)
		}
		// fs.Glob / path.Join below expect forward-slash paths; filepath.Rel may
		// return backslash separators on Windows.
		rel = filepath.ToSlash(rel)
		current := artistFolder
		for range maxArtistFolderTraversalDepth {
			reader, hit, err := findImageInFolder(ctx, libFS, rel, current, pattern)
			if err == nil {
				return reader, hit, nil
			}
			if rel == "." {
				break // reached library root; don't traverse above it
			}
			rel = path.Dir(rel)
			current = filepath.Dir(current)
		}
		return nil, "", fmt.Errorf(`no matches for '%s' in '%s' or its parent directories (within library)`, pattern, artistFolder)
	}
}

// findImageInFolder globs libFS at relFolder for pattern and returns the first
// matching image. absFolder is used only for the returned display path and log
// messages so callers see absolute-looking paths consistent with the rest of
// the artwork pipeline.
func findImageInFolder(ctx context.Context, libFS fs.FS, relFolder, absFolder, pattern string) (io.ReadCloser, string, error) {
	log.Trace(ctx, "looking for artist image", "pattern", pattern, "folder", absFolder)
	globPattern := pattern
	if relFolder != "." {
		globPattern = path.Join(escapeGlobLiteral(relFolder), pattern)
	}
	matches, err := fs.Glob(libFS, globPattern)
	if err != nil {
		log.Warn(ctx, "Error matching artist image pattern", "pattern", pattern, "folder", absFolder, err)
		return nil, "", err
	}

	// Filter to valid image files
	var imagePaths []string
	for _, m := range matches {
		if !model.IsImageFile(m) {
			continue
		}
		imagePaths = append(imagePaths, m)
	}

	// Sort image files by prioritizing base filenames without numeric
	// suffixes (e.g., artist.jpg before artist.1.jpg)
	slices.SortFunc(imagePaths, compareImageFiles)

	for _, p := range imagePaths {
		f, err := libFS.Open(p)
		if err != nil {
			log.Warn(ctx, "Could not open cover art file", "file", p, err)
			continue
		}
		_, name := path.Split(p)
		return f, filepath.Join(absFolder, name), nil
	}

	return nil, "", fmt.Errorf(`no matches for '%s' in '%s'`, pattern, absFolder)
}

func escapeGlobLiteral(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\\', '*', '?', '[', ']':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func loadArtistFolder(ctx context.Context, ds model.DataStore, albums model.Albums, paths []string) (string, time.Time, error) {
	if len(albums) == 0 {
		return "", time.Time{}, nil
	}
	libID := albums[0].LibraryID // Just need one of the albums, as they should all be in the same Library - for now! TODO: Support multiple libraries

	folderPath := str.LongestCommonPrefix(paths)
	if !strings.HasSuffix(folderPath, string(filepath.Separator)) {
		folderPath, _ = filepath.Split(folderPath)
	}
	folderPath = filepath.Dir(folderPath)

	// Manipulate the path to get the folder ID
	// TODO: This is a bit hacky, but it's the easiest way to get the folder ID, ATM
	libPath := core.AbsolutePath(ctx, ds, libID, "")
	folderID := model.FolderID(model.Library{ID: libID, Path: libPath}, folderPath)

	log.Trace(ctx, "Calculating artist folder details", "folderPath", folderPath, "folderID", folderID,
		"libPath", libPath, "libID", libID, "albumPaths", paths)

	// Get the last update time for the folder
	folders, err := ds.Folder(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"folder.id": folderID, "missing": false}})
	if err != nil || len(folders) == 0 {
		log.Warn(ctx, "Could not find folder for artist", "folderPath", folderPath, "id", folderID,
			"libPath", libPath, "libID", libID, err)
		return "", time.Time{}, err
	}
	return folderPath, folders[0].ImagesUpdatedAt, nil
}

// findImageInArtistFolder scans a folder for an image file matching the artist's MBID or name
// (case-insensitive). Returns the full path, or empty string if not found.
func findImageInArtistFolder(folder, mbzArtistID, artistName string) string {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return ""
	}
	for _, candidate := range []string{mbzArtistID, artistName} {
		if candidate == "" {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			base := strings.TrimSuffix(name, filepath.Ext(name))
			if strings.EqualFold(base, candidate) && model.IsImageFile(name) {
				return filepath.Join(folder, name)
			}
		}
	}
	return ""
}
