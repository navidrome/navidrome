package playlists

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

func (s *playlists) SyncPhysicalFolderPlaylists(ctx context.Context) (int, error) {
	plsList, err := s.ds.Playlist(ctx).GetSyncPlaylists()
	if err != nil {
		return 0, fmt.Errorf("fetching sync playlists: %w", err)
	}

	count := 0
	for _, pls := range plsList {
		if pls.PhysicalFolderID == "" {
			continue
		}

		log.Debug(ctx, "Syncing folder-based playlist", "name", pls.Name, "folderId", pls.PhysicalFolderID)

		// Get all tracks in the folder hierarchy
		tracks, err := s.ds.MediaFile(ctx).GetAll(model.QueryOptions{
			Filters: squirrel.Eq{"folder_id_recursive": pls.PhysicalFolderID, "media_file.missing": false},
		})
		if err != nil {
			log.Error(ctx, "Error fetching tracks for folder sync", "playlist", pls.Name, "folderId", pls.PhysicalFolderID, err)
			continue
		}

		mediaFileIds := make([]string, len(tracks))
		for i, t := range tracks {
			mediaFileIds[i] = t.ID
		}

		err = s.ds.WithTxImmediate(func(tx model.DataStore) error {
			tracksRepo := tx.Playlist(ctx).Tracks(pls.ID, false)
			if err := tracksRepo.DeleteAll(); err != nil {
				return err
			}
			_, err := tracksRepo.Add(mediaFileIds)
			return err
		})

		if err != nil {
			log.Error(ctx, "Error updating tracks for folder sync", "playlist", pls.Name, err)
			continue
		}

		count++
	}

	return count, nil
}
