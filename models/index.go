package models

type ArtistInfo struct {
	ArtistId string
	Artist string
}

type ArtistIndex struct {
	Id string
	Artists []ArtistInfo
}


