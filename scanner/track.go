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

func (m *Track) RealArtist() string {
	if (m.Compilation) {
		return "Various Artists"
	}
	if (m.AlbumArtist != "") {
		return m.AlbumArtist
	}
	return m.Artist
}