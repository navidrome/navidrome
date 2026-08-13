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

// loadArtistAlbumRoots returns one path per album — the deepest folder holding
// all of that album's tracks — so an album split into disc subfolders can't
// pull the artist folder's common prefix below the artist level.
func loadArtistAlbumRoots(ctx context.Context, ds model.DataStore, albums model.Albums) ([]string, []string, *time.Time, error) {
	var folderIDs []string
	for _, album := range albums {
		folderIDs = append(folderIDs, album.FolderIDs...)
	}
	folders, err := loadFolders(ctx, ds, folderIDs)
	if err != nil {
		return nil, nil, nil, err
	}

	pathByID := slice.ToMap(folders, func(f model.Folder) (string, string) {
		return f.ID, f.AbsolutePath()
	})
	var roots []string
	for _, album := range albums {
		var albumPaths []string
		for _, fid := range album.FolderIDs {
			if p, ok := pathByID[fid]; ok {
				albumPaths = append(albumPaths, p)
			}
		}
		if len(albumPaths) > 0 {
			roots = append(roots, commonDir(albumPaths))
		}
	}

	imgFiles, updatedAt := folderImages(folders)
	return roots, imgFiles, &updatedAt, nil
}

// commonDir returns the deepest directory containing all paths. Trailing
// separators keep the comparison on segment boundaries, so a shared name
// fragment (".../Album" and ".../Album2") is never read as a shared directory.
func commonDir(paths []string) string {
	sep := string(filepath.Separator)
	common := str.LongestCommonPrefix(slice.Map(paths, func(p string) string { return p + sep }))
	if !strings.HasSuffix(common, sep) {
		common, _ = filepath.Split(common)
	}
	return filepath.Clean(common)
}

func loadArtistFolder(ctx context.Context, ds model.DataStore, albums model.Albums, paths []string) (string, time.Time, error) {
	if len(albums) == 0 {
		return "", time.Time{}, nil
	}
	libID := albums[0].LibraryID // TODO: Support albums spanning multiple libraries

	// paths holds one root per album: two or more distinct roots already meet at
	// the artist folder, while a single root is an album folder needing a climb.
	roots := slices.Compact(slices.Sorted(slices.Values(paths)))
	folderPath := commonDir(roots)
	if len(roots) < 2 {
		folderPath = filepath.Dir(folderPath)
	}

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
