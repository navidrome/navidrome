package models

type Album struct {
	Id           string
	Name         string
	ArtistId     string `parent:"artist"`
	CoverArtPath string
	Year         int
	Compilation  bool
	Rating       int
}