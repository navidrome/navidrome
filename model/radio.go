package model

import "time"

type Radio struct {
	ID          string    `structs:"id"            json:"id"`
	StreamUrl   string    `structs:"stream_url"    json:"streamUrl"`
	Name        string    `structs:"name"          json:"name"`
	HomePageUrl string    `structs:"home_page_url" json:"homePageUrl"`
	CreatedAt   time.Time `structs:"created_at"    json:"createdAt"`
	UpdatedAt   time.Time `structs:"updated_at"    json:"updatedAt"`
}

type Radios []Radio

type RadioRepository interface {
	ResourceRepository
	CountAll(options ...QueryOptions) (int64, error)
	Delete(id string) error
	Get(id string) (*Radio, error)
	GetAll(options ...QueryOptions) (Radios, error)
	Put(u *Radio) error
}
