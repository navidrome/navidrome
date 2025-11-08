package core

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

type MissingFiles interface {
	// DeleteMissingFiles deletes specific missing files by their IDs
	DeleteMissingFiles(ctx context.Context, ids []string) error
	// DeleteAllMissingFiles deletes all files marked as missing
	DeleteAllMissingFiles(ctx context.Context) error
}

type missingFilesService struct {
	ds model.DataStore
}

func NewMissingFiles(ds model.DataStore) MissingFiles {
	return &missingFilesService{
		ds: ds,
	}
}

func (s *missingFilesService) DeleteMissingFiles(ctx context.Context, ids []string) error {
	return s.deleteMissing(ctx, ids)
}

func (s *missingFilesService) DeleteAllMissingFiles(ctx context.Context) error {
	return s.deleteMissing(ctx, nil)
}

// deleteMissing handles the deletion of missing files and triggers necessary cleanup operations
func (s *missingFilesService) deleteMissing(ctx context.Context, ids []string) error {
	// Track affected album IDs before deletion for refresh
	affectedAlbumIDs, err := s.getAffectedAlbumIDs(ctx, ids)
	if err != nil {
		log.Warn(ctx, "Error tracking affected albums for refresh", err)
		// Don't fail the operation, just log the warning
	}

	// Delete missing files within a transaction
	err = s.ds.WithTx(func(tx model.DataStore) error {
		if len(ids) == 0 {
			_, err := tx.MediaFile(ctx).DeleteAllMissing()
			return err
		}
		return tx.MediaFile(ctx).DeleteMissing(ids)
	})
	if err != nil {
		log.Error(ctx, "Error deleting missing tracks from DB", "ids", ids, err)
		return err
	}

	// Run garbage collection to clean up orphaned records
	if err := s.ds.GC(ctx); err != nil {
		log.Error(ctx, "Error running GC after deleting missing tracks", err)
		return err
	}

	// Refresh statistics in background
	s.refreshStatsAsync(ctx, affectedAlbumIDs)

	return nil
}

// getAffectedAlbumIDs returns distinct album IDs from missing media files
func (s *missingFilesService) getAffectedAlbumIDs(ctx context.Context, ids []string) ([]string, error) {
	var filters squirrel.Sqlizer = squirrel.Eq{"missing": true}
	if len(ids) > 0 {
		filters = squirrel.And{
			squirrel.Eq{"missing": true},
			squirrel.Eq{"id": ids},
		}
	}

	mfs, err := s.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	// Extract unique album IDs
	albumIDMap := make(map[string]struct{}, len(mfs))
	for _, mf := range mfs {
		if mf.AlbumID != "" {
			albumIDMap[mf.AlbumID] = struct{}{}
		}
	}

	albumIDs := make([]string, 0, len(albumIDMap))
	for id := range albumIDMap {
		albumIDs = append(albumIDs, id)
	}

	return albumIDs, nil
}

// refreshStatsAsync refreshes artist and album statistics in background goroutines
func (s *missingFilesService) refreshStatsAsync(ctx context.Context, affectedAlbumIDs []string) {
	// Refresh artist stats in background
	go func() {
		bgCtx := request.AddValues(context.Background(), ctx)
		if _, err := s.ds.Artist(bgCtx).RefreshStats(true); err != nil {
			log.Error(bgCtx, "Error refreshing artist stats after deleting missing files", err)
		} else {
			log.Debug(bgCtx, "Successfully refreshed artist stats after deleting missing files")
		}

		// Refresh album stats in background if we have affected albums
		if len(affectedAlbumIDs) > 0 {
			if err := s.ds.Album(bgCtx).RefreshAlbums(affectedAlbumIDs); err != nil {
				log.Error(bgCtx, "Error refreshing album stats after deleting missing files", err)
			} else {
				log.Debug(bgCtx, "Successfully refreshed album stats after deleting missing files", "count", len(affectedAlbumIDs))
			}
		}
	}()
}
