package model

import "time"

const (
	ArtistItemType = "artist"
	AlbumItemType  = "album"
	MediaItemType  = "media_file"
)

type Annotation struct {
	AnnotationID string
	UserID       string
	ItemID       string
	ItemType     string
	PlayCount    int
	PlayDate     time.Time
	Rating       int
	Starred      bool
	StarredAt    time.Time
}

type AnnotationMap map[string]Annotation

type AnnotationRepository interface {
	Get(userID, itemType string, itemID string) (*Annotation, error)
	GetAll(userID, itemType string, options ...QueryOptions) ([]Annotation, error)
	GetMap(userID, itemType string, itemID []string) (AnnotationMap, error)
	Delete(userID, itemType string, itemID ...string) error
	IncPlayCount(userID, itemType string, itemID string, ts time.Time) error
	SetStar(starred bool, userID, itemType string, ids ...string) error
	SetRating(rating int, userID, itemType string, itemID string) error
}
