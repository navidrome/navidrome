package model

import "time"

type Annotations struct {
	PlayCount int64      `structs:"-" json:"playCount"`
	PlayDate  *time.Time `structs:"-" json:"playDate" `
	Rating    int        `structs:"-" json:"rating"   `
	Starred   bool       `structs:"-" json:"starred"  `
	StarredAt *time.Time `structs:"-" json:"starredAt"`
}

type AnnotatedRepository interface {
	IncPlayCount(itemID string, ts time.Time) error
	SetStar(starred bool, itemIDs ...string) error
	SetRating(rating int, itemID string) error
}
