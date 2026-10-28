package core

import (
	"context"
	"time"

	"github.com/SunilSherkhane/navidrome/model"
	"github.com/SunilSherkhane/navidrome/persistence"
)

// ListeningStatsService aggregates and reports listening statistics
type ListeningStatsService struct {
	repo *persistence.ListeningStatsRepository
}

func NewListeningStatsService(repo *persistence.ListeningStatsRepository) *ListeningStatsService {
	return &ListeningStatsService{repo: repo}
}

// RecordPlay increments stats for a played entity
func (s *ListeningStatsService) RecordPlay(ctx context.Context, userID, entityID uuid.UUID, entityType string, durationMs int64) error {
	return s.repo.Increment(ctx, userID, entityID, entityType, durationMs)
}

// GetTopArtists returns the most-played artists for a user
func (s *ListeningStatsService) GetTopArtists(ctx context.Context, userID uuid.UUID, timeRange string, limit int) ([]model.TopItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.GetTopArtists(ctx, userID, timeRange, limit)
}

// GetTopAlbums returns the most-played albums for a user
func (s *ListeningStatsService) GetTopAlbums(ctx context.Context, userID uuid.UUID, timeRange string, limit int) ([]model.TopItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.GetTopAlbums(ctx, userID, timeRange, limit)
}

// GetTopSongs returns the most-played songs for a user
func (s *ListeningStatsService) GetTopSongs(ctx context.Context, userID uuid.UUID, timeRange string, limit int) ([]model.TopItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.GetTopSongs(ctx, userID, timeRange, limit)
}

// Wire up the repository
func init() {
	// Repository is initialized in persistence.New()
}

var _ = time.Now // ensure time import is used
