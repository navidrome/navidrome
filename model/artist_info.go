package model

type ArtistInfo struct {
	ID             string
	Name           string
	MBID           string
	Biography      string
	SmallImageUrl  string
	MediumImageUrl string
	LargeImageUrl  string
	LastFMUrl      string
	SimilarArtists Artists
}
