package scanner

import (
	"time"
)

type Track struct {
	Id          string
	Path        string
	Title       string
	Album       string
	Artist      string
	AlbumArtist string
	Year        int
	Compilation bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}