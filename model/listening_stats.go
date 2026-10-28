package model

import (
	"time"

	"github.com/google/uuid"
)

// ListeningStats represents aggregated listening data for a user
type ListeningStats struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	EntityType   string // "artist", "album", "song"
	EntityID     uuid.UUID
	PlayCount    int
	TotalMsPlayed int64
	FirstPlayed  time.Time
	LastPlayed   time.Time
	UpdatedAt    time.Time
}

// ListeningStatsFilter defines filters for querying stats
type ListeningStatsFilter struct {
	UserID     uuid.UUID
	EntityType string // "" for all
	TimeRange  string // "7d", "30d", "90d", "1y", "all"
	Limit      int
	Offset     int
}

// TopItem represents a ranked item in stats results
type TopItem struct {
	ID          uuid.UUID
	Name        string
	ImagePath   string
	PlayCount   int
	TotalMsPlayed int64
	Rank        int
}

const (
	StatsRange7d  = "7d"
	StatsRange30d = "30d"
	StatsRange90d = "90d"
	StatsRange1y  = "1y"
	StatsRangeAll = "all"
)
