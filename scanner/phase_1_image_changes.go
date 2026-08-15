package scanner

import (
	"maps"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// imageChangedFolder records a folder whose image files changed during the scan, so the
// affected albums/artists can be re-enqueued for artwork resolution at the end of phase 1.
type imageChangedFolder struct {
	lib         model.Library
	id          string
	path        string
	artistImage bool
}

// imagesChanged reports whether the folder's image files differ from the previously persisted
// state, and whether an artist image is involved (present in the old or the new list).
func imagesChanged(prev *model.Folder, entry *folderEntry) (changed, artistImage bool) {
	var prevNames []string
	var prevAt time.Time
	if prev != nil {
		prevNames = prev.ImageFiles
		prevAt = prev.ImagesUpdatedAt
	}
	newNames := slices.Sorted(maps.Keys(entry.imageFiles))
	if len(prevNames) == 0 && len(newNames) == 0 {
		return false, false
	}
	sortedPrev := slices.Sorted(slices.Values(prevNames))
	if slices.Equal(sortedPrev, newNames) && prevAt.Equal(entry.imagesUpdatedAt) {
		return false, false
	}
	return true, matchesArtistPatterns(sortedPrev) || matchesArtistPatterns(newNames)
}

func artistImagePatterns() []string {
	var patterns []string
	for pat := range strings.SplitSeq(conf.Server.ArtistArtPriority, ",") {
		pat = strings.ToLower(strings.TrimSpace(pat))
		if pat == "" || pat == "external" || pat == "embedded" {
			continue
		}
		patterns = append(patterns, path.Base(pat))
	}
	return patterns
}

func matchesArtistPatterns(names []string) bool {
	patterns := artistImagePatterns()
	for _, name := range names {
		for _, pat := range patterns {
			if ok, _ := path.Match(pat, strings.ToLower(name)); ok {
				return true
			}
		}
	}
	return false
}

// enqueueArtworkForImageChanges re-enqueues artwork for entities affected by image-only folder
// changes. Best-effort: failures are logged and never fail the scan.
func (p *phaseFolders) enqueueArtworkForImageChanges() {
	if len(p.imageChangedFolders) == 0 {
		return
	}
	byLib := map[int][]imageChangedFolder{}
	for _, f := range p.imageChangedFolders {
		byLib[f.lib.ID] = append(byLib[f.lib.ID], f)
	}
	for libID, folders := range byLib {
		items, err := p.collectImageChangeQueueItems(libID, folders)
		if err != nil {
			log.Warn(p.ctx, "Scanner: could not map image changes to artwork items", "lib", libID, err)
			continue
		}
		if len(items) == 0 {
			continue
		}
		if err := p.ds.ArtworkQueue(p.ctx).Enqueue(items...); err != nil {
			log.Warn(p.ctx, "Scanner: could not enqueue artwork for image changes", "lib", libID, err)
			continue
		}
		log.Debug(p.ctx, "Scanner: Enqueued artwork resolution for image changes", "lib", libID,
			"changedFolders", len(folders), "items", len(items))
	}
}

func (p *phaseFolders) collectImageChangeQueueItems(libID int, folders []imageChangedFolder) ([]model.ArtworkQueueItem, error) {
	folderIDs := make([]string, len(folders))
	var artistFolders []imageChangedFolder
	for i, f := range folders {
		folderIDs[i] = f.id
		if f.artistImage {
			artistFolders = append(artistFolders, f)
		}
	}

	var items []model.ArtworkQueueItem

	// An album's cover search space is its track folders plus their common parent, so a changed
	// folder affects albums with tracks in it or in its direct children.
	albumFolderIDs, err := p.ds.Folder(p.ctx).GetAllIDs(model.QueryOptions{Filters: squirrel.And{
		squirrel.Eq{"library_id": libID},
		squirrel.Eq{"missing": false},
		squirrel.Or{squirrel.Eq{"id": folderIDs}, squirrel.Eq{"parent_id": folderIDs}},
	}})
	if err != nil {
		return nil, err
	}
	albumIDs, err := p.ds.MediaFile(p.ctx).GetAlbumIDsByFolder(albumFolderIDs...)
	if err != nil {
		return nil, err
	}
	for _, id := range albumIDs {
		items = append(items, model.ArtworkQueueItem{
			ItemKind: model.KindAlbumArtwork.Prefix(), ItemID: id, ImageType: model.ImageTypePrimary,
			Priority: model.ArtworkPriorityScan,
		})
	}

	// The artist resolver climbs from the artist folder up to the library root, so a changed
	// artist image affects every artist with albums anywhere under the folder's subtree.
	artistIDs, err := p.artistIDsUnderFolders(libID, artistFolders)
	if err != nil {
		return nil, err
	}
	for _, id := range artistIDs {
		if id == "" || id == consts.UnknownArtistID || id == consts.VariousArtistsID {
			continue
		}
		items = append(items, model.ArtworkQueueItem{
			ItemKind: model.KindArtistArtwork.Prefix(), ItemID: id, ImageType: model.ImageTypePrimary,
			Priority: model.ArtworkPriorityScan,
		})
	}
	return items, nil
}

func (p *phaseFolders) artistIDsUnderFolders(libID int, folders []imageChangedFolder) ([]string, error) {
	if len(folders) == 0 {
		return nil, nil
	}
	filters := squirrel.And{squirrel.Eq{"library_id": libID}, squirrel.Eq{"missing": false}}
	var conds squirrel.Or
	for _, f := range folders {
		rel := path.Clean(f.path)
		if rel == "." || rel == "" {
			conds = nil // library root: the subtree is the whole library
			break
		}
		conds = append(conds, squirrel.Eq{"id": f.id}, squirrel.Eq{"path": rel}, squirrel.Like{"path": rel + "/%"})
	}
	if conds != nil {
		filters = append(filters, conds)
	}
	subtreeIDs, err := p.ds.Folder(p.ctx).GetAllIDs(model.QueryOptions{Filters: filters})
	if err != nil {
		return nil, err
	}
	albumIDs, err := p.ds.MediaFile(p.ctx).GetAlbumIDsByFolder(subtreeIDs...)
	if err != nil {
		return nil, err
	}
	return p.ds.Album(p.ctx).GetSoleAlbumArtistIDs(albumIDs...)
}
