package model

type Publisher struct {
	ID         string `structs:"id" json:"id"               orm:"column(id)"`
	Name       string `structs:"name" json:"name"`
	SongCount  int    `structs:"-" json:"-"`
	AlbumCount int    `structs:"-" json:"-"`
}

type Publishers []Publisher

type PublisherRepository interface {
	GetAll(...QueryOptions) (Publishers, error)
	Put(*Publisher) error
}
