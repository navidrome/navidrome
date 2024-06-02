package scanner2

import (
	"context"

	"github.com/google/go-pipeline/pkg/pipeline"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	"golang.org/x/exp/maps"
)

func persistChanges(ctx context.Context) pipeline.StageFn[*folderEntry] {
	return func(entry *folderEntry) (*folderEntry, error) {
		err := entry.job.ds.WithTx(func(tx model.DataStore) error {
			// Save folder to DB
			err := tx.Folder(ctx).Put(entry.job.lib, entry.path)
			if err != nil {
				log.Error(ctx, "Scanner: Error persisting folder to DB", "folder", entry.path, err)
				return err
			}

			// Save all new/modified albums to DB. Their information will be incomplete, but they will be refreshed
			// in phase 3
			for i := range entry.albums {
				err := tx.Album(ctx).Put(&entry.albums[i])
				if err != nil {
					log.Error(ctx, "Scanner: Error persisting album to DB", "folder", entry.path, "album", entry.albums[i], err)
					return err
				}
			}

			// Save all tags to DB
			err = tx.Tag(ctx).Add(entry.tags...)
			if err != nil {
				log.Error(ctx, "Scanner: Error persisting tags to DB", "folder", entry.path, err)
				return err
			}

			// Save all tracks to DB
			for i := range entry.tracks {
				err = tx.MediaFile(ctx).Put(&entry.tracks[i])
				if err != nil {
					log.Error(ctx, "Scanner: Error persisting mediafile to DB", "folder", entry.path, "track", entry.tracks[i], err)
					return err
				}
			}

			// Mark all missing tracks as not available
			if len(entry.missingTracks) > 0 {
				err = tx.MediaFile(ctx).MarkMissing(entry.missingTracks, true)
				if err != nil {
					log.Error(ctx, "Scanner: Error marking missing tracks", "folder", entry.path, err)
					return err
				}

				// Touch all albums that have missing tracks, so they get refreshed in phase 3
				groupedMissingTracks := slice.ToMap(entry.missingTracks, func(mf model.MediaFile) (string, struct{}) { return mf.AlbumID, struct{}{} })
				albumsToUpdate := maps.Keys(groupedMissingTracks)
				err = tx.Album(ctx).Touch(albumsToUpdate...)
				if err != nil {
					log.Error(ctx, "Scanner: Error touching album", "folder", entry.path, "albums", albumsToUpdate, err)
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Error(ctx, "Scanner: Error persisting changes to DB", "folder", entry.path, err)
		}
		return entry, err
	}
}
