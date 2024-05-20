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
			// Save all tags to DB
			err := slice.RangeByChunks(entry.tags, 100, func(chunk []model.Tag) error {
				err := tx.Tag(ctx).Add(chunk...)
				if err != nil {
					log.Error(ctx, "Scanner: Error adding tags to DB", "folder", entry.path, err)
					return err
				}
				return nil
			})
			if err != nil {
				return err
			}

			// Save all tracks to DB
			err = slice.RangeByChunks(entry.tracks, 100, func(chunk []model.MediaFile) error {
				for i := range chunk {
					track := chunk[i]
					err = tx.MediaFile(ctx).Put(&track)
					if err != nil {
						log.Error(ctx, "Scanner: Error adding/updating mediafile to DB", "folder", entry.path, "track", track, err)
						return err
					}
				}
				return nil
			})
			if err != nil {
				return err
			}

			// Save folder to DB
			err = tx.Folder(ctx).Put(entry.job.lib, entry.path)
			if err != nil {
				log.Error(ctx, "Scanner: Error adding/updating folder to DB", "folder", entry.path, err)
				return err
			}
			return err
		})
		if err != nil {
			log.Error(ctx, "Scanner: Error persisting changes to DB", "folder", entry.path, err)
		}
		return entry, err
	}
}
