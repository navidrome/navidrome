package model

type Genre struct {
	Name       string `json:"name"`
	SongCount  int    `json:"song_count"`
	AlbumCount int    `json:"album_count"`
}

type Genres []Genre

type GenreRepository interface {
	ResourceRepository
	GetAll() (Genres, error)
	Refresh(ids ...string) error
}

type GenreType struct {
	GenreID  string `json:"genreId"  orm:"column(genre_id)"`
	ItemID   string `json:"itemId"   orm:"column(item_id)"`
	ItemType string `json:"itemType"`
}

type GenreTypes []GenreType

type GenreTypeRepository interface {
	EntityName() string
	NewInstance() interface{}
	GetGenres(itemID string, itemType string) ([]string, error)
	Refresh(ids ...string) error
}
