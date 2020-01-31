package model

import "time"

const (
	ArtistItemType = "artist"
	AlbumItemType  = "album"
	MediaItemType  = "media_file"
)

type Annotation struct {
	AnnID     string    `json:"annID"        orm:"pk;column(ann_id)"`
	UserID    string    `json:"userID"       orm:"pk;column(user_id)"`
	ItemID    string    `json:"itemID"       orm:"pk;column(item_id)"`
	ItemType  string    `json:"itemType"`
	PlayCount int       `json:"playCount"`
	PlayDate  time.Time `json:"playDate"`
	Rating    int       `json:"rating"`
	Starred   bool      `json:"starred"`
	StarredAt time.Time `json:"starredAt"`
}

type AnnotationMap map[string]Annotation

type AnnotationRepository interface {
	Delete(itemType string, itemID ...string) error
	IncPlayCount(itemType, itemID string, ts time.Time) error
	SetStar(starred bool, itemType string, ids ...string) error
	SetRating(rating int, itemType, itemID string) error
}
