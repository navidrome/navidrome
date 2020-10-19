package model

type ArtistInfo struct {
	ID             string
	Name           string
	MbzID          string
	Bio            string
	Similar        []Artist
	SmallImageUrl  string
	MediumImageUrl string
	LargeImageUrl  string
	LastFMUrl      string
}
