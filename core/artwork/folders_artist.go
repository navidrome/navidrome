package artwork

import (
	"context"
	"errors"
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
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/navidrome/navidrome/utils/str"
)

const (
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
		// fs.Glob needs forward slashes; filepath.Rel returns backslashes on Windows.
		rel = filepath.ToSlash(rel)
		current := artistFolder
		var unreadable error
		for range maxArtistFolderTraversalDepth {
			reader, hit, err := findImageInFolder(ctx, libFS, rel, current, pattern)
			if err == nil {
				return reader, hit, nil
			}
			if errors.Is(err, errSourceUnreadable) {
				unreadable = err
			}
			if rel == "." {
				break // reached library root
			}
			rel = path.Dir(rel)
			current = filepath.Dir(current)
		}
		if unreadable != nil {
			return nil, "", unreadable
		}
		return nil, "", fmt.Errorf(`no matches for '%s' in '%s' or its parent directories (within library)`, pattern, artistFolder)
	}
}

// findImageInFolder returns the first image matching pattern; absFolder is only used for
// the returned display path and log messages.
func findImageInFolder(ctx context.Context, libFS fs.FS, relFolder, absFolder, pattern string) (io.ReadCloser, string, error) {
	log.Trace(ctx, "Artwork: Looking for artist image", "pattern", pattern, "folder", absFolder)
	globPattern := pattern
	if relFolder != "." {
		globPattern = path.Join(escapeGlobLiteral(relFolder), pattern)
	}
	matches, err := fs.Glob(libFS, globPattern)
	if err != nil {
		log.Warn(ctx, "Artwork: Error matching artist image pattern", "pattern", pattern, "folder", absFolder, err)
		return nil, "", err
	}

	imagePaths := slice.Filter(matches, model.IsImageFile)

	// Prefer base filenames over numeric-suffixed ones (artist.jpg before artist.1.jpg)
	slices.SortFunc(imagePaths, compareImageFiles)

	var openErr error
	for _, p := range imagePaths {
		f, err := libFS.Open(p)
		if err != nil {
			log.Warn(ctx, "Artwork: Could not open cover art file", "file", p, err)
			openErr = fmt.Errorf("%w: %s: %w", errSourceUnreadable, p, err)
			continue
		}
		_, name := path.Split(p)
		return f, filepath.Join(absFolder, name), nil
	}
	if openErr != nil {
		return nil, "", openErr
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
	libID := albums[0].LibraryID // TODO: Support albums spanning multiple libraries

	folderPath := str.LongestCommonPrefix(paths)
	if !strings.HasSuffix(folderPath, string(filepath.Separator)) {
		folderPath, _ = filepath.Split(folderPath)
	}
	folderPath = filepath.Dir(folderPath)

	// TODO: Hacky, but the easiest way to get the folder ID ATM
	libPath := core.AbsolutePath(ctx, ds, libID, "")
	folderID := model.FolderID(model.Library{ID: libID, Path: libPath}, folderPath)

	log.Trace(ctx, "Artwork: Calculating artist folder details", "folderPath", folderPath, "folderID", folderID,
		"libPath", libPath, "libID", libID, "albumPaths", paths)

	folders, err := ds.Folder(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"folder.id": folderID, "missing": false}})
	if err != nil || len(folders) == 0 {
		log.Warn(ctx, "Artwork: Could not find folder for artist", "folderPath", folderPath, "id", folderID,
			"libPath", libPath, "libID", libID, err)
		return "", time.Time{}, err
	}
	return folderPath, folders[0].ImagesUpdatedAt, nil
}

// findImageInArtistFolder matches an image by MBID or artist name (case-insensitive), "" if none.
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
			base := utils.BaseName(name)
			if strings.EqualFold(base, candidate) && model.IsImageFile(name) {
				return filepath.Join(folder, name)
			}
		}
	}
	return ""
}
