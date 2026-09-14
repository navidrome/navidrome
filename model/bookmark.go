package model

import (
	"context"
	"time"
)

type Bookmarkable struct {
	BookmarkPosition int64 `structs:"-" json:"bookmarkPosition"`
}

type BookmarkableRepository interface {
	AddBookmark(ctx context.Context, id, comment string, position int64) error
	DeleteBookmark(ctx context.Context, id string) error
	GetBookmarks(ctx context.Context) (Bookmarks, error)
}

type Bookmark struct {
	Item      MediaFile `structs:"item" json:"item"`
	Comment   string    `structs:"comment" json:"comment"`
	Position  int64     `structs:"position" json:"position"`
	ChangedBy string    `structs:"changed_by" json:"changed_by"`
	CreatedAt time.Time `structs:"created_at" json:"createdAt"`
	UpdatedAt time.Time `structs:"updated_at" json:"updatedAt"`
}

type Bookmarks []Bookmark
