package model

import "time"

type Annotations struct {
	PlayCount int64     `json:"playCount"`
	PlayDate  time.Time `json:"playDate"`
	Rating    int       `json:"rating"`
	Starred   bool      `json:"starred"`
	StarredAt time.Time `json:"starredAt"`
}

type AnnotatedModel interface {
	GetAnnotations() Annotations
}

type AnnotatedRepository interface {
	IncPlayCount(itemID string, ts time.Time) error
	SetStar(starred bool, itemIDs ...string) error
	SetRating(rating int, itemID string) error
}

// While I can't find a better way to make these fields optional in the models, I keep this list here
// to be used in other packages
var AnnotationFields = []string{"playCount", "playDate", "rating", "starred", "starredAt"}
