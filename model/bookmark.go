package model

import "time"

type Bookmarkable struct {
	BookmarkPosition int64 `json:"bookmarkPosition"`
}

type BookmarkableRepository interface {
	AddBookmark(id, comment string, position int64) error
	DeleteBookmark(id string) error
	GetBookmarks() (Bookmarks, error)
}

type Bookmark struct {
	Item      MediaFile `json:"item"`
	Comment   string    `json:"comment"`
	Position  int64     `json:"position"`
	ChangedBy string    `json:"changed_by"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Bookmarks []Bookmark

// While I can't find a better way to make these fields optional in the models, I keep this list here
// to be used in other packages
var BookmarkFields = []string{"bookmarkPosition"}
