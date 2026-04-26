package artwork

import (
	"context"
	"crypto/md5"
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
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/str"
)

const (
	// maxArtistFolderTraversalDepth defines how many directory levels to search
	// when looking for artist images (artist folder + parent directories)
	maxArtistFolderTraversalDepth = 3
)

type artistReader struct {
	cacheKey
	a                *artwork
	provider         external.Provider
	artist           model.Artist
	artistFolder     string
	imgFiles         []string
	imgFolderImgPath string // cached path from ArtistImageFolder lookup
	lib              libraryView
}

func newArtistArtworkReader(ctx context.Context, artwork *artwork, artID model.ArtworkID, provider external.Provider) (*artistReader, error) {
	ar, err := artwork.ds.Artist(ctx).Get(artID.ID)
	if err != nil {
		return nil, err
	}
	// Only consider albums where the artist is the sole album artist.
	als, err := artwork.ds.Album(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.And{
			squirrel.Eq{"album_artist_id": artID.ID},
			squirrel.Eq{"json_array_length(participants, '$.albumartist')": 1},
		},
	})
	if err != nil {
		return nil, err
	}
	albumPaths, imgFiles, imagesUpdatedAt, err := loadAlbumFoldersPaths(ctx, artwork.ds, als...)
	if err != nil {
		return nil, err
	}
	artistFolder, artistFolderLastUpdate, err := loadArtistFolder(ctx, artwork.ds, als, albumPaths)
	if err != nil {
		return nil, err
	}
	var lib libraryView
	if len(als) > 0 {
		lib, err = loadLibraryView(ctx, artwork.ds, als[0].LibraryID)
		if err != nil {
			return nil, err
		}
	}
	a := &artistReader{
		a:            artwork,
		provider:     provider,
		artist:       *ar,
		artistFolder: artistFolder,
		imgFiles:     imgFiles,
		lib:          lib,
	}
	// TODO Find a way to factor in the ExternalUpdateInfoAt in the cache key. Problem is that it can
	// change _after_ retrieving from external sources, making the key invalid
	//a.cacheKey.lastUpdate = ar.ExternalInfoUpdatedAt

	a.cacheKey.lastUpdate = *imagesUpdatedAt
	if ar.UpdatedAt != nil && ar.UpdatedAt.After(a.cacheKey.lastUpdate) {
		a.cacheKey.lastUpdate = *ar.UpdatedAt
	}
	if artistFolderLastUpdate.After(a.cacheKey.lastUpdate) {
		a.cacheKey.lastUpdate = artistFolderLastUpdate
	}
	if conf.Server.ArtistImageFolder != "" && strings.Contains(strings.ToLower(conf.Server.ArtistArtPriority), "image-folder") {
		a.imgFolderImgPath = findImageInArtistFolder(conf.Server.ArtistImageFolder, ar.MbzArtistID, ar.Name)
		if a.imgFolderImgPath != "" {
			if info, err := os.Stat(a.imgFolderImgPath); err == nil && info.ModTime().After(a.cacheKey.lastUpdate) {
				a.cacheKey.lastUpdate = info.ModTime()
			}
		}
	}
	a.cacheKey.artID = artID
	return a, nil
}

func (a *artistReader) Key() string {
	hash := md5.Sum([]byte(conf.Server.Agents))
	return fmt.Sprintf(
		"%s.%t.%x",
		a.cacheKey.Key(),
		conf.Server.EnableExternalServices,
		hash,
	)
}

func (a *artistReader) LastUpdated() time.Time {
	return a.lastUpdate
}

func (a *artistReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	ff := []sourceFunc{a.fromArtistUploadedImage()}
	ff = append(ff, a.fromArtistArtPriority(ctx, conf.Server.ArtistArtPriority)...)
	return selectImageReader(ctx, a.artID, ff...)
}

func (a *artistReader) fromArtistUploadedImage() sourceFunc {
	return fromLocalFile(a.artist.UploadedImagePath())
}

func (a *artistReader) fromArtistArtPriority(ctx context.Context, priority string) []sourceFunc {
	var ff []sourceFunc
	for pattern := range strings.SplitSeq(strings.ToLower(priority), ",") {
		pattern = strings.TrimSpace(pattern)
		switch {
		case pattern == "external":
			ff = append(ff, fromArtistExternalSource(ctx, a.artist, a.provider))
		case pattern == "image-folder":
			ff = append(ff, a.fromArtistImageFolder(ctx))
		case strings.HasPrefix(pattern, "album/"):
			if a.lib.FS != nil {
				ff = append(ff, fromExternalFile(ctx, a.lib.FS, a.imgFiles, strings.TrimPrefix(pattern, "album/")))
			}
		default:
			ff = append(ff, fromArtistFolder(ctx, a.lib.FS, a.lib.absRoot, a.artistFolder, pattern))
		}
	}
	return ff
}

// fromArtistFolder walks up from artistFolder toward libPath looking for a
// file matching pattern. Traversal is bounded by both maxArtistFolderTraversalDepth
// and the library root: once we reach libPath (or if artistFolder is outside
// libPath), the walk stops. All reads go through libFS, which keeps artwork
// resolution scoped to the configured library.
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

func (a *artistReader) fromArtistImageFolder(ctx context.Context) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		folder := conf.Server.ArtistImageFolder
		if folder == "" {
			return nil, "", nil
		}
		// Use cached path from newArtistArtworkReader if available,
		// avoiding a second directory scan.
		path := a.imgFolderImgPath
		if path == "" {
			path = findImageInArtistFolder(folder, a.artist.MbzArtistID, a.artist.Name)
		}
		if path == "" {
			return nil, "", fmt.Errorf("no image found for artist %q in %s", a.artist.Name, folder)
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, "", err
		}
		return f, path, nil
	}
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
