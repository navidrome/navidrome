package model

import "time"

type Annotations struct {
	PlayCount int64      `structs:"play_count" json:"playCount,omitempty"`
	PlayDate  *time.Time `structs:"play_date"  json:"playDate,omitempty" `
	Rating    int        `structs:"rating"     json:"rating,omitempty"   `
	Starred   bool       `structs:"starred"    json:"starred,omitempty"  `
	StarredAt *time.Time `structs:"starred_at" json:"starredAt,omitempty"`
}

type AnnotatedRepository interface {
	IncPlayCount(itemID string, ts time.Time) error
	SetStar(starred bool, itemIDs ...string) error
	SetRating(rating int, itemID string) error
	ReassignAnnotation(prevID string, newID string) error
}
