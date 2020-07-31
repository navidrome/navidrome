package model

import (
	"time"
)

type PlayQueue struct {
	ID        string     `json:"id"          orm:"column(id)"`
	UserID    string     `json:"userId"      orm:"column(user_id)"`
	Comment   string     `json:"comment"`
	Current   string     `json:"current"`
	Position  int64      `json:"position"`
	ChangedBy string     `json:"changedBy"`
	Items     MediaFiles `json:"items,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type PlayQueues []PlayQueue

type PlayQueueRepository interface {
	Store(queue *PlayQueue) error
	Retrieve(userId string) (*PlayQueue, error)
	AddBookmark(userId, id, comment string, position int64) error
	GetBookmarks(userId string) (Bookmarks, error)
	DeleteBookmark(userId, id string) error
}

type Bookmark struct {
	Item      MediaFile `json:"item"`
	Comment   string    `json:"comment"`
	Position  int64     `json:"position"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Bookmarks []Bookmark
