package model

import "time"

type Annotations struct {
	PlayCount int64      `structs:"play_count" json:"playCount"`
	PlayDate  *time.Time `structs:"play_date"  json:"playDate" `
	Rating    int        `structs:"rating"     json:"rating"   `
	Starred   bool       `structs:"starred"    json:"starred"  `
	StarredAt *time.Time `structs:"starred_at" json:"starredAt"`
}

type AnnotatedRepository interface {
	IncPlayCount(itemID string, ts time.Time) error
	SetStar(starred bool, itemIDs ...string) error
	SetRating(rating int, itemID string) error
}
