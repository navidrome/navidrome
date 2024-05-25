package scanner2

import (
	"context"

	"github.com/google/go-pipeline/pkg/pipeline"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
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

			// Save all albums to DB
			// TODO Collect all album PIDs that were affected by the changes in the folder:
			//  - Albums from modified media_files, previous (from DB) and new PIDs (from FS)
			//  - Albums from deleted media_files
			//  Then get get albums from DB, merge modified/missed tracks, refresh and save them back
			albumIds := map[string]string{}
			for i := range entry.albums {
				err = tx.Album(ctx).Put(&entry.albums[i])
				if err != nil {
					log.Error(ctx, "Scanner: Error persisting album to DB", "folder", entry.path, "album", entry.albums[i], err)
					return err
				}
				albumIds[entry.albums[i].PID] = entry.albums[i].ID
			}

			// Save all tags to DB
			err = slice.RangeByChunks(entry.tags, 100, func(chunk []model.Tag) error {
				err := tx.Tag(ctx).Add(chunk...)
				if err != nil {
					log.Error(ctx, "Scanner: Error persisting tags to DB", "folder", entry.path, err)
					return err
				}
				return nil
			})
			if err != nil {
				return err
			}

			// Save all tracks to DB
			for i := range entry.tracks {
				entry.tracks[i].AlbumID = albumIds[entry.tracks[i].AlbumPID]
				err = tx.MediaFile(ctx).Put(&entry.tracks[i])
				if err != nil {
					log.Error(ctx, "Scanner: Error persisting mediafile to DB", "folder", entry.path, "track", entry.tracks[i], err)
					return err
				}
			}

			// Mark all missing tracks as not available
			err = tx.MediaFile(ctx).MarkMissing(entry.missingTracks, true)
			if err != nil {
				log.Error(ctx, "Scanner: Error marking missing tracks", "folder", entry.path, err)
				return err
			}
			return nil
		})
		if err != nil {
			log.Error(ctx, "Scanner: Error persisting changes to DB", "folder", entry.path, err)
		}
		return entry, err
	}
}
