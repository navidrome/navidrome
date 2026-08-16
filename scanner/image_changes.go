package scanner

import (
	"context"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// imageChangedFolder records a folder whose image files changed during the scan, so the
// affected albums/artists can be re-enqueued for artwork resolution at the end of phase 1.
type imageChangedFolder struct {
	id          string
	path        string
	artistImage bool
}

// imageChangeCollector gathers those folders per library while phase 1 persists them, then turns
// them into artwork queue items once. Its zero value is ready to use, and record must be called
// from a single goroutine.
type imageChangeCollector struct {
	libs    map[int]model.Library
	folders map[int][]imageChangedFolder
}

func (c *imageChangeCollector) record(lib model.Library, folder imageChangedFolder) {
	if c.folders == nil {
		c.folders = map[int][]imageChangedFolder{}
		c.libs = map[int]model.Library{}
	}
	c.libs[lib.ID] = lib
	c.folders[lib.ID] = append(c.folders[lib.ID], folder)
}

// enqueue is best-effort: failures are logged and never fail the scan.
func (c *imageChangeCollector) enqueue(ctx context.Context, ds model.DataStore) {
	for libID, folders := range c.folders {
		lib := c.libs[libID]
		items, err := c.queueItems(ctx, ds, lib, folders)
		if err != nil {
			log.Warn(ctx, "Scanner: could not map image changes to artwork items", "lib", lib.Name, err)
			continue
		}
		if len(items) == 0 {
			continue
		}
		if err := ds.ArtworkQueue(ctx).Enqueue(items...); err != nil {
			log.Warn(ctx, "Scanner: could not enqueue artwork for image changes", "lib", lib.Name, err)
			continue
		}
		log.Debug(ctx, "Scanner: Enqueued artwork resolution for image changes", "lib", lib.Name,
			"changedFolders", len(folders), "items", len(items))
	}
}

func (c *imageChangeCollector) queueItems(ctx context.Context, ds model.DataStore, lib model.Library,
	folders []imageChangedFolder,
) ([]model.ArtworkQueueItem, error) {
	folderIDs := make([]string, len(folders))
	var artistFolderPaths []string
	for i, f := range folders {
		folderIDs[i] = f.id
		if f.artistImage {
			artistFolderPaths = append(artistFolderPaths, f.path)
		}
	}

	var items []model.ArtworkQueueItem

	albumIDs, err := ds.MediaFile(ctx).GetAlbumIDsByFolder(lib, folderIDs...)
	if err != nil {
		return nil, err
	}
	for _, id := range albumIDs {
		items = append(items, scanArtworkItem(model.KindAlbumArtwork, id))
	}

	if len(artistFolderPaths) == 0 {
		return items, nil
	}
	// The resolver climbs to the library root, so the subtree below the folder is the affected set.
	// A failure here must not discard the album items already collected.
	artistIDs, err := ds.Album(ctx).GetSoleAlbumArtistIDsInSubtrees(lib, artistFolderPaths...)
	if err != nil {
		log.Warn(ctx, "Scanner: could not map image changes to artists", "lib", lib.Name, err)
		return items, nil
	}
	for _, id := range artistIDs {
		if id == "" || id == consts.UnknownArtistID || id == consts.VariousArtistsID {
			continue
		}
		items = append(items, scanArtworkItem(model.KindArtistArtwork, id))
	}
	return items, nil
}
