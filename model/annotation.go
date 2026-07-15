package model

import "time"

type Annotations struct {
	PlayCount     int64      `structs:"play_count"     json:"playCount,omitempty"`
	PlayDate      *time.Time `structs:"play_date"      json:"playDate,omitempty" `
	Rating        int        `structs:"rating"         json:"rating,omitempty"   `
	RatedAt       *time.Time `structs:"rated_at"       json:"ratedAt,omitempty"  `
	Starred       bool       `structs:"starred"        json:"starred,omitempty"  `
	StarredAt     *time.Time `structs:"starred_at"     json:"starredAt,omitempty"`
	AverageRating float64    `structs:"average_rating" json:"averageRating,omitempty"`
	Skipped       bool       `structs:"skipped"        json:"skipped,omitempty"  `
	SkippedAt     *time.Time `structs:"skipped_at"     json:"skippedAt,omitempty"`
}

type AnnotatedRepository interface {
	IncPlayCount(itemID string, ts time.Time) error
	SetStar(starred bool, itemIDs ...string) error
	SetRating(rating int, itemID string) error
	SetSkip(skip bool, itemIDs ...string) error
	ReassignAnnotation(prevID string, newID string) error
}
