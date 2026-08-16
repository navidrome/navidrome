package scanner

import (
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

func scanArtworkItem(kind model.Kind, id string) model.ArtworkQueueItem {
	return model.ArtworkQueueItem{
		ItemKind: kind.Prefix(), ItemID: id, ImageType: model.ImageTypePrimary,
		Priority: model.ArtworkPriorityScan,
	}
}

// enqueueArtworkForImageChanges re-enqueues artwork for entities affected by image-only folder
// changes. Best-effort: failures are logged and never fail the scan.
func (p *phaseFolders) enqueueArtworkForImageChanges() {
	for _, job := range p.jobs {
		folders := p.imageChangedFolders[job.lib.ID]
		if len(folders) == 0 {
			continue
		}
		items, err := p.collectImageChangeQueueItems(job.lib, folders)
		if err != nil {
			log.Warn(p.ctx, "Scanner: could not map image changes to artwork items", "lib", job.lib.Name, err)
			continue
		}
		if len(items) == 0 {
			continue
		}
		if err := p.ds.ArtworkQueue(p.ctx).Enqueue(items...); err != nil {
			log.Warn(p.ctx, "Scanner: could not enqueue artwork for image changes", "lib", job.lib.Name, err)
			continue
		}
		log.Debug(p.ctx, "Scanner: Enqueued artwork resolution for image changes", "lib", job.lib.Name,
			"changedFolders", len(folders), "items", len(items))
	}
}

func (p *phaseFolders) collectImageChangeQueueItems(lib model.Library, folders []imageChangedFolder) ([]model.ArtworkQueueItem, error) {
	folderIDs := make([]string, len(folders))
	var artistFolderPaths []string
	for i, f := range folders {
		folderIDs[i] = f.id
		if f.artistImage {
			artistFolderPaths = append(artistFolderPaths, f.path)
		}
	}

	var items []model.ArtworkQueueItem

	albumIDs, err := p.ds.MediaFile(p.ctx).GetAlbumIDsByFolder(lib, folderIDs...)
	if err != nil {
		return nil, err
	}
	for _, id := range albumIDs {
		items = append(items, scanArtworkItem(model.KindAlbumArtwork, id))
	}

	// The artist resolver climbs from the artist folder up to the library root, so a changed
	// artist image affects every artist with albums anywhere under the folder's subtree. A failure
	// here must not discard the album items already collected.
	artistIDs, err := p.artistIDsUnderPaths(lib, artistFolderPaths)
	if err != nil {
		log.Warn(p.ctx, "Scanner: could not map image changes to artists", "lib", lib.Name, err)
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

func (p *phaseFolders) artistIDsUnderPaths(lib model.Library, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	return p.ds.Album(p.ctx).GetSoleAlbumArtistIDsInSubtrees(lib, paths...)
}
