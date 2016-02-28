package models

import (
	"time"
)

type MediaFile struct {
	Id          string
	Path        string
	Title       string
	Album       string
	Artist      string
	AlbumArtist string
	Compilation bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}