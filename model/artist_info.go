package model

type ArtistInfo struct {
	ID             string
	Name           string
	MBID           string
	Biography      string
	Similar        []Artist
	SmallImageUrl  string
	MediumImageUrl string
	LargeImageUrl  string
	LastFMUrl      string
}
