package core

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/slice"
)

var (
	// ErrNotMissing is returned when a remap is attempted from a file not marked as missing.
	ErrNotMissing = errors.New("file is not marked as missing")
	// ErrTargetMissing is returned when the remap target is itself a missing file.
	ErrTargetMissing = errors.New("target file is missing")
	// ErrSameFile is returned when the remap source and target are the same file.
	ErrSameFile = errors.New("missing and target are the same file")
)

type Maintenance interface {
	// DeleteMissingFiles deletes specific missing files by their IDs
	DeleteMissingFiles(ctx context.Context, ids []string) error
	// DeleteAllMissingFiles deletes all files marked as missing
	DeleteAllMissingFiles(ctx context.Context) error
	// RemapMissingFile relocates a missing file's identity onto an existing, non-missing
	// media file. It is the manual counterpart to the scanner's automatic move detection
	// (see scanner.phaseMissingTracks.moveMatched).
	RemapMissingFile(ctx context.Context, missingID, targetID string) error
}

type maintenanceService struct {
	ds model.DataStore
	wg sync.WaitGroup
}

func NewMaintenance(ds model.DataStore) Maintenance {
	return &maintenanceService{
		ds: ds,
	}
}

func (s *maintenanceService) DeleteMissingFiles(ctx context.Context, ids []string) error {
	return s.deleteMissing(ctx, ids)
}

func (s *maintenanceService) DeleteAllMissingFiles(ctx context.Context) error {
	return s.deleteMissing(ctx, nil)
}

func (s *maintenanceService) RemapMissingFile(ctx context.Context, missingID, targetID string) error {
	if missingID == targetID {
		return fmt.Errorf("%w: %q", ErrSameFile, missingID)
	}

	missing, err := s.ds.MediaFile(ctx).Get(missingID)
	if err != nil {
		return fmt.Errorf("loading missing file %q: %w", missingID, err)
	}
	if !missing.Missing {
		return fmt.Errorf("%w: %q", ErrNotMissing, missingID)
	}

	target, err := s.ds.MediaFile(ctx).GetWithParticipants(targetID)
	if err != nil {
		return fmt.Errorf("loading target file %q: %w", targetID, err)
	}
	if target.Missing {
		return fmt.Errorf("%w: %q", ErrTargetMissing, targetID)
	}

	oldAlbumID, newAlbumID := missing.AlbumID, target.AlbumID

	// Mirrors scanner.phaseMissingTracks.moveMatched: move the missing track's identity
	// (ID, annotations and references) onto the file found at the new location.
	err = s.ds.WithTx(func(tx model.DataStore) error {
		discardedID := target.ID

		// Preserve the original created_at so the remapped track doesn't resurface in "Recently Added"
		target.CreatedAt = missing.CreatedAt
		target.ID = missing.ID
		if err := tx.MediaFile(ctx).Put(target); err != nil {
			return fmt.Errorf("update matched track: %w", err)
		}
		// Discard the target's original row
		if err := tx.MediaFile(ctx).Delete(discardedID); err != nil {
			return fmt.Errorf("delete discarded track: %w", err)
		}

		// Reassign album annotations (starred, rating) and preserve album created_at on album change
		if oldAlbumID != newAlbumID {
			if err := tx.Album(ctx).ReassignAnnotation(oldAlbumID, newAlbumID); err != nil {
				log.Warn(ctx, "Could not reassign album annotations", "from", oldAlbumID, "to", newAlbumID, err)
			}
			if err := tx.Album(ctx).CopyAttributes(oldAlbumID, newAlbumID, "created_at"); err != nil && !errors.Is(err, model.ErrNotFound) {
				log.Warn(ctx, "Could not copy album created_at", "from", oldAlbumID, "to", newAlbumID, err)
			}
		}
		return nil
	})
	if err != nil {
		log.Error(ctx, "Error remapping missing file", "missing", missing.Path, "target", target.Path, err)
		return err
	}

	// Clean up now-orphaned records and refresh affected statistics, mirroring deleteMissing
	if err := s.ds.GC(ctx); err != nil {
		log.Error(ctx, "Error running GC after remapping missing file", err)
		return err
	}

	// Refresh artist stats
	if _, err := s.ds.Artist(ctx).RefreshStats(true); err != nil {
		log.Error(ctx, "Error refreshing artist stats after deleting missing files", err)
	} else {
		log.Debug(ctx, "Successfully refreshed artist stats after deleting missing files")
	}

	// Refresh album stats if we have affected albums
	affectedAlbumIDs := slice.Unique(slice.Filter([]string{oldAlbumID, newAlbumID}, func(id string) bool { return id != "" }))
	if len(affectedAlbumIDs) > 0 {
		if err := s.refreshAlbums(ctx, affectedAlbumIDs); err != nil {
			log.Error(ctx, "Error refreshing album stats after deleting missing files", err)
		} else {
			log.Debug(ctx, "Successfully refreshed album stats after deleting missing files", "count", len(affectedAlbumIDs))
		}
	}

	return nil
}

// deleteMissing handles the deletion of missing files and triggers necessary cleanup operations
func (s *maintenanceService) deleteMissing(ctx context.Context, ids []string) error {
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

	// Refresh statistics in background. album/artist play count aggregates are not recalculated
	// here; they are refreshed by the next scan.
	s.refreshStatsAsync(ctx, affectedAlbumIDs)

	return nil
}

// refreshAlbums recalculates album attributes (size, duration, song count, etc.) from media files.
// It uses batch queries to minimize database round-trips for efficiency.
func (s *maintenanceService) refreshAlbums(ctx context.Context, albumIDs []string) error {
	if len(albumIDs) == 0 {
		return nil
	}

	log.Debug(ctx, "Refreshing albums", "count", len(albumIDs))

	// Process in chunks to avoid query size limits
	const chunkSize = 100
	for chunk := range slice.CollectChunks(slices.Values(albumIDs), chunkSize) {
		if err := s.refreshAlbumChunk(ctx, chunk); err != nil {
			return fmt.Errorf("refreshing album chunk: %w", err)
		}
	}

	log.Debug(ctx, "Successfully refreshed albums", "count", len(albumIDs))
	return nil
}

// refreshAlbumChunk processes a single chunk of album IDs
func (s *maintenanceService) refreshAlbumChunk(ctx context.Context, albumIDs []string) error {
	albumRepo := s.ds.Album(ctx)
	mfRepo := s.ds.MediaFile(ctx)

	// Batch load existing albums
	albums, err := albumRepo.GetAll(model.QueryOptions{
		Filters: squirrel.Eq{"album.id": albumIDs},
	})
	if err != nil {
		return fmt.Errorf("loading albums: %w", err)
	}

	// Create a map for quick lookup
	albumMap := make(map[string]*model.Album, len(albums))
	for i := range albums {
		albumMap[albums[i].ID] = &albums[i]
	}

	// Batch load all media files for these albums
	mediaFiles, err := mfRepo.GetAll(model.QueryOptions{
		Filters: squirrel.Eq{"album_id": albumIDs},
		Sort:    "album_id, path",
	})
	if err != nil {
		return fmt.Errorf("loading media files: %w", err)
	}

	// Group media files by album ID
	filesByAlbum := make(map[string]model.MediaFiles)
	for i := range mediaFiles {
		albumID := mediaFiles[i].AlbumID
		filesByAlbum[albumID] = append(filesByAlbum[albumID], mediaFiles[i])
	}

	// Recalculate each album from its media files
	for albumID, oldAlbum := range albumMap {
		mfs, hasTracks := filesByAlbum[albumID]
		if !hasTracks {
			// Album has no tracks anymore, skip (will be cleaned up by GC)
			log.Debug(ctx, "Skipping album with no tracks", "albumID", albumID)
			continue
		}

		// Recalculate album from media files
		newAlbum := mfs.ToAlbum()

		// Only update if something changed (avoid unnecessary writes)
		if !oldAlbum.Equals(newAlbum) {
			// Preserve original timestamps
			newAlbum.UpdatedAt = time.Now()
			newAlbum.CreatedAt = oldAlbum.CreatedAt

			if err := albumRepo.Put(&newAlbum); err != nil {
				log.Error(ctx, "Error updating album during refresh", "albumID", albumID, err)
				// Continue with other albums instead of failing entirely
				continue
			}
			log.Trace(ctx, "Refreshed album", "albumID", albumID, "name", newAlbum.Name)
		}
	}

	return nil
}

// getAffectedAlbumIDs returns distinct album IDs from missing media files
func (s *maintenanceService) getAffectedAlbumIDs(ctx context.Context, ids []string) ([]string, error) {
	var filters squirrel.Sqlizer = squirrel.Eq{"missing": true}
	if len(ids) > 0 {
		filters = squirrel.And{
			squirrel.Eq{"missing": true},
			squirrel.Eq{"media_file.id": ids},
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
func (s *maintenanceService) refreshStatsAsync(ctx context.Context, affectedAlbumIDs []string) {
	// Refresh artist stats in background
	s.wg.Go(func() {
		bgCtx := request.AddValues(context.Background(), ctx)
		if _, err := s.ds.Artist(bgCtx).RefreshStats(true); err != nil {
			log.Error(bgCtx, "Error refreshing artist stats after deleting missing files", err)
		} else {
			log.Debug(bgCtx, "Successfully refreshed artist stats after deleting missing files")
		}

		// Refresh album stats in background if we have affected albums
		if len(affectedAlbumIDs) > 0 {
			if err := s.refreshAlbums(bgCtx, affectedAlbumIDs); err != nil {
				log.Error(bgCtx, "Error refreshing album stats after deleting missing files", err)
			} else {
				log.Debug(bgCtx, "Successfully refreshed album stats after deleting missing files", "count", len(affectedAlbumIDs))
			}
		}
	})
}

// Wait waits for all background goroutines to complete.
// WARNING: This method is ONLY for testing. Never call this in production code.
// Calling Wait() in production will block until ALL background operations complete
// and may cause race conditions with new operations starting.
func (s *maintenanceService) wait() {
	s.wg.Wait()
}
