package model

import "time"

type Annotations struct {
	PlayCount int64      `structs:"-" json:"playCount,omitempty"`
	PlayDate  *time.Time `structs:"-" json:"playDate,omitempty"`
	Rating    int        `structs:"-" json:"rating,omitempty"`
	Starred   bool       `structs:"-" json:"starred,omitempty"`
	StarredAt *time.Time `structs:"-" json:"starredAt,omitempty"`
}

type AnnotatedRepository interface {
	IncPlayCount(itemID string, ts time.Time) error
	SetStar(starred bool, itemIDs ...string) error
	SetRating(rating int, itemID string) error
}
