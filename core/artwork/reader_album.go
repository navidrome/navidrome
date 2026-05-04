package artwork

import (
	"cmp"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/natural"
)

type albumArtworkReader struct {
	cacheKey
	a         *artwork
	provider  external.Provider
	album     model.Album
	updatedAt *time.Time
	imgFiles  []string // library-relative, forward-slash, no leading slash
	lib       libraryView
}

func newAlbumArtworkReader(ctx context.Context, artwork *artwork, artID model.ArtworkID, provider external.Provider) (*albumArtworkReader, error) {
	al, err := artwork.ds.Album(ctx).Get(artID.ID)
	if err != nil {
		return nil, err
	}
	_, imgFiles, imagesUpdateAt, err := loadAlbumFoldersPaths(ctx, artwork.ds, *al)
	if err != nil {
		return nil, err
	}
	lib, err := loadLibraryView(ctx, artwork.ds, al.LibraryID)
	if err != nil {
		return nil, err
	}
	a := &albumArtworkReader{
		a:         artwork,
		provider:  provider,
		album:     *al,
		updatedAt: imagesUpdateAt,
		imgFiles:  imgFiles,
		lib:       lib,
	}
	a.cacheKey.artID = artID
	if a.updatedAt != nil && a.updatedAt.After(al.UpdatedAt) {
		a.cacheKey.lastUpdate = *a.updatedAt
	} else {
		a.cacheKey.lastUpdate = al.UpdatedAt
	}
	return a, nil
}

func (a *albumArtworkReader) Key() string {
	hashInput := conf.Server.CoverArtPriority
	if conf.Server.EnableExternalServices {
		hashInput = conf.Server.Agents + hashInput
	}
	hash := md5.Sum([]byte(hashInput))
	return fmt.Sprintf(
		"%s.%x.%t",
		a.cacheKey.Key(),
		hash,
		conf.Server.EnableExternalServices,
	)
}
func (a *albumArtworkReader) LastUpdated() time.Time {
	return a.lastUpdate
}

func (a *albumArtworkReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	var ff = a.fromCoverArtPriority(ctx, a.a.ffmpeg, conf.Server.CoverArtPriority)
	return selectImageReader(ctx, a.artID, ff...)
}

func (a *albumArtworkReader) fromCoverArtPriority(ctx context.Context, ffmpeg ffmpeg.FFmpeg, priority string) []sourceFunc {
	var ff []sourceFunc
	for pattern := range strings.SplitSeq(strings.ToLower(priority), ",") {
		pattern = strings.TrimSpace(pattern)
		switch {
		case pattern == "embedded":
			embedRel := a.album.EmbedArtPath
			ff = append(ff,
				fromTag(ctx, a.lib.FS, embedRel),
				fromFFmpegTag(ctx, ffmpeg, a.lib.Abs(embedRel)),
			)
		case pattern == "external":
			ff = append(ff, fromAlbumExternalSource(ctx, a.album, a.provider))
		case len(a.imgFiles) > 0:
			ff = append(ff, fromExternalFile(ctx, a.lib.FS, a.imgFiles, pattern))
		}
	}
	return ff
}

func loadAlbumFoldersPaths(ctx context.Context, ds model.DataStore, albums ...model.Album) ([]string, []string, *time.Time, error) {
	var folderIDs []string
	for _, album := range albums {
		folderIDs = append(folderIDs, album.FolderIDs...)
	}
	folders, err := ds.Folder(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"folder.id": folderIDs, "missing": false}})
	if err != nil {
		return nil, nil, nil, err
	}

	folderIDSet := make(map[string]bool, len(folderIDs))
	for _, id := range folderIDs {
		folderIDSet[id] = true
	}

	// Check if all folders share a common parent that is not already included.
	// This finds cover art in the album root folder (e.g., "Artist/Album/cover.jpg"
	// when tracks are in disc subfolders like "Artist/Album/CD1/" and "Artist/Album/CD2/").
	// For single-folder albums, the parent is only included when the folder has no
	// images of its own (indicating a disc subfolder needing parent artwork).
	if commonParentID := commonParentFolder(folders, folderIDSet); commonParentID != "" {
		if len(folders) >= 2 || !anyFolderHasImages(folders) {
			parentFolder, err := ds.Folder(ctx).Get(commonParentID)
			if errors.Is(err, model.ErrNotFound) {
				log.Warn(ctx, "Parent folder not found for album cover art lookup", "parentID", commonParentID)
			} else if err != nil {
				return nil, nil, nil, err
			}
			if parentFolder != nil && parentFolder.Path != "." {
				folders = append(folders, *parentFolder)
			}
		}
	}

	var paths []string
	var imgFiles []string
	var updatedAt time.Time
	for _, f := range folders {
		paths = append(paths, f.AbsolutePath())
		if f.ImagesUpdatedAt.After(updatedAt) {
			updatedAt = f.ImagesUpdatedAt
		}
		rel := strings.TrimPrefix(path.Join(f.Path, f.Name), "/")
		for _, img := range f.ImageFiles {
			imgFiles = append(imgFiles, path.Join(rel, img))
		}
	}

	// Sort image files to ensure consistent selection of cover art
	// This prioritizes files without numeric suffixes (e.g., cover.jpg over cover.1.jpg)
	// by comparing base filenames without extensions
	slices.SortFunc(imgFiles, compareImageFiles)

	return paths, imgFiles, &updatedAt, nil
}

func anyFolderHasImages(folders []model.Folder) bool {
	for _, f := range folders {
		if len(f.ImageFiles) > 0 {
			return true
		}
	}
	return false
}

// commonParentFolder returns the shared parent folder ID when all folders have the
// same parent and that parent is not already in folderIDSet. Returns "" otherwise.
func commonParentFolder(folders []model.Folder, folderIDSet map[string]bool) string {
	if len(folders) == 0 {
		return ""
	}
	parentID := folders[0].ParentID
	if parentID == "" || folderIDSet[parentID] {
		return ""
	}
	for _, f := range folders[1:] {
		if f.ParentID != parentID {
			return ""
		}
	}
	return parentID
}

// compareImageFiles sorts image paths by: base filename (natural order),
// then path depth (shallower first), then full path (stable tiebreaker).
func compareImageFiles(a, b string) int {
	// Case-insensitive comparison
	a = strings.ToLower(a)
	b = strings.ToLower(b)

	// Extract base filenames without extensions
	baseA := strings.TrimSuffix(path.Base(a), path.Ext(a))
	baseB := strings.TrimSuffix(path.Base(b), path.Ext(b))

	// Compare base names first, then prefer shallower paths, then full path as tiebreaker
	return cmp.Or(
		natural.Compare(baseA, baseB),
		cmp.Compare(strings.Count(a, "/"), strings.Count(b, "/")),
		natural.Compare(a, b),
	)
}
