package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/SunilSherkhane/navidrome/model"
	"github.com/Masterminds/squirrel"
)

type listeningStatsRepository struct {
	*sql.DB
	sqlBuilder squirrel.StatementBuilderType
}

func newListeningStatsRepository(db *sql.DB) *listeningStatsRepository {
	return &listeningStatsRepository{
		DB:         db,
		sqlBuilder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (r *listeningStatsRepository) GetAll(ctx context.Context, filter model.ListeningStatsFilter) ([]model.ListeningStats, error) {
	q := r.sqlBuilder.
		Select("id", "user_id", "entity_type", "entity_id", "play_count", "total_ms_played", "first_played", "last_played", "updated_at").
		From("listening_stats").
		Where("user_id = ?", filter.UserID).
		OrderBy("total_ms_played DESC")

	if filter.EntityType != "" {
		q = q.Where("entity_type = ?", filter.EntityType)
	}
	if filter.TimeRange != "" && filter.TimeRange != model.StatsRangeAll {
		since := time.Now().Add(-parseRange(filter.TimeRange))
		q = q.Where("last_played >= ?", since)
	}
	if filter.Limit > 0 {
		q = q.Limit(uint64(filter.Limit))
	}
	if filter.Offset > 0 {
		q = q.Offset(uint64(filter.Offset))
	}

	query, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("building listening stats query: %w", err)
	}

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("executing listening stats query: %w", err)
	}
	defer rows.Close()

	var stats []model.ListeningStats
	for rows.Next() {
		var s model.ListeningStats
		err := rows.Scan(&s.ID, &s.UserID, &s.EntityType, &s.EntityID,
			&s.PlayCount, &s.TotalMsPlayed, &s.FirstPlayed, &s.LastPlayed, &s.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scanning listening stats row: %w", err)
		}
		stats = append(stats, s)
	}
	return stats, nil
}

func (r *listeningStatsRepository) Increment(ctx context.Context, userID, entityID uuid.UUID, entityType string, msPlayed int64) error {
	now := time.Now()
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO listening_stats (user_id, entity_type, entity_id, play_count, total_ms_played, first_played, last_played, updated_at)
		VALUES ($1, $2, $3, 1, $4, $5, $5, $5)
		ON CONFLICT (user_id, entity_type, entity_id)
		DO UPDATE SET
			play_count = listening_stats.play_count + 1,
			total_ms_played = listening_stats.total_ms_played + $4,
			last_played = excluded.last_played,
			updated_at = excluded.updated_at
	`, userID, entityType, entityID, msPlayed, now)
	return err
}

func (r *listeningStatsRepository) GetTopArtists(ctx context.Context, userID uuid.UUID, timeRange string, limit int) ([]model.TopItem, error) {
	since := time.Now().Add(-parseRange(timeRange))
	query := `
		SELECT s.entity_id, COALESCE(a.name, 'Unknown') as name, a.album_cover_path as image_path,
		       s.play_count, s.total_ms_played, ROW_NUMBER() OVER (ORDER BY s.total_ms_played DESC) as rank
		FROM listening_stats s
		LEFT JOIN artists a ON a.id = s.entity_id
		WHERE s.user_id = $1 AND s.entity_type = 'artist' AND s.last_played >= $2
		ORDER BY s.total_ms_played DESC
		LIMIT $3
	`
	return r.queryTopItems(ctx, query, userID, since, limit)
}

func (r *listeningStatsRepository) GetTopAlbums(ctx context.Context, userID uuid.UUID, timeRange string, limit int) ([]model.TopItem, error) {
	since := time.Now().Add(-parseRange(timeRange))
	query := `
		SELECT s.entity_id, COALESCE(a.name, 'Unknown') as name, a.cover_art_path as image_path,
		       s.play_count, s.total_ms_played, ROW_NUMBER() OVER (ORDER BY s.total_ms_played DESC) as rank
		FROM listening_stats s
		LEFT JOIN albums a ON a.id = s.entity_id
		WHERE s.user_id = $1 AND s.entity_type = 'album' AND s.last_played >= $2
		ORDER BY s.total_ms_played DESC
		LIMIT $3
	`
	return r.queryTopItems(ctx, query, userID, since, limit)
}

func (r *listeningStatsRepository) GetTopSongs(ctx context.Context, userID uuid.UUID, timeRange string, limit int) ([]model.TopItem, error) {
	since := time.Now().Add(-parseRange(timeRange))
	query := `
		SELECT s.entity_id, COALESCE(m.title, 'Unknown') as name, m.cover_art_path as image_path,
		       s.play_count, s.total_ms_played, ROW_NUMBER() OVER (ORDER BY s.total_ms_played DESC) as rank
		FROM listening_stats s
		LEFT JOIN media_files m ON m.id = s.entity_id
		WHERE s.user_id = $1 AND s.entity_type = 'song' AND s.last_played >= $2
		ORDER BY s.total_ms_played DESC
		LIMIT $3
	`
	return r.queryTopItems(ctx, query, userID, since, limit)
}

func (r *listeningStatsRepository) queryTopItems(ctx context.Context, query string, args ...interface{}) ([]model.TopItem, error) {
	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.TopItem
	for rows.Next() {
		var item model.TopItem
		err := rows.Scan(&item.ID, &item.Name, &item.ImagePath, &item.PlayCount, &item.TotalMsPlayed, &item.Rank)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func parseRange(r string) time.Duration {
	switch r {
	case model.StatsRange7d:
		return -7 * 24 * time.Hour
	case model.StatsRange30d:
		return -30 * 24 * time.Hour
	case model.StatsRange90d:
		return -90 * 24 * time.Hour
	case model.StatsRange1y:
		return -365 * 24 * time.Hour
	default:
		return -time.Hour
	}
}
