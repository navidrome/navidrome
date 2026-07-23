package artwork

import (
	"cmp"
	"context"
	"errors"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/natural"
)

func loadAlbumFoldersPaths(ctx context.Context, ds model.DataStore, albums ...model.Album) ([]string, []string, *time.Time, error) {
	var folderIDs []string
	for _, album := range albums {
		folderIDs = append(folderIDs, album.FolderIDs...)
	}
	folders, err := ds.Folder(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"folder.id": folderIDs, "missing": false}})
	if err != nil {
		return nil, nil, nil, err
	}

	parent, err := albumRootParent(ctx, ds, folders, folderIDs)
	if err != nil {
		return nil, nil, nil, err
	}
	if parent != nil {
		folders = append(folders, *parent)
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

// albumRootParent returns the common parent of the album's folders when it
// qualifies as the album's root folder (e.g. "Artist/Album" above disc
// subfolders), or nil when there is no such parent. This finds cover art in
// the album root folder when tracks live in disc subfolders, like
// "Artist/Album/cover.jpg" with tracks in "Artist/Album/CD1/" and
// "Artist/Album/CD2/". The parent must look like an album root, not an
// artist-level folder — it qualifies only when it holds no audio belonging to
// other albums — so artist images are never served as album art.
func albumRootParent(ctx context.Context, ds model.DataStore, folders []model.Folder, folderIDs []string) (*model.Folder, error) {
	folderIDSet := make(map[string]bool, len(folderIDs))
	for _, id := range folderIDs {
		folderIDSet[id] = true
	}
	commonParentID := commonParentFolder(folders, folderIDSet)
	if commonParentID == "" {
		return nil, nil
	}
	// Single-folder albums only use the parent when the folder has no images
	// of its own (indicating a disc subfolder needing parent artwork).
	if len(folders) < 2 && anyFolderHasImages(folders) {
		return nil, nil
	}
	parent, err := ds.Folder(ctx).Get(commonParentID)
	if errors.Is(err, model.ErrNotFound) {
		log.Warn(ctx, "Parent folder not found for album cover art lookup", "parentID", commonParentID)
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if parent.ParentID == "" {
		// The library root can never be an album root
		return nil, nil
	}
	hasOtherAudio, err := ds.Folder(ctx).HasAudioOutsideFolders(*parent, folderIDs)
	if err != nil {
		return nil, err
	}
	if hasOtherAudio {
		return nil, nil
	}
	return parent, nil
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
