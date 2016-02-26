package models

type Album struct {
	Id string
	Name string
	Artist *Artist
	CoverArtPath string
	Year int
	Compilation bool
	Rating int

}
