package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Radio struct {
	ID          string     `structs:"id"            json:"id" orm:"pk;column(id)"`
	StreamUrl   string     `structs:"stream_url"    json:"streamUrl"`
	Name        string     `structs:"name"          json:"name"`
	HomePageUrl string     `structs:"home_page_url" json:"homePageUrl" orm:"column(home_page_url)"`
	CreatedAt   time.Time  `structs:"created_at" json:"createdAt"`
	UpdatedAt   time.Time  `structs:"updated_at" json:"updatedAt"`
	IsPlaylist  bool       `structs:"is_playlist" json:"isPlaylist"`
	Links       RadioLinks `structs:"-" json:"links,omitempty"`
}

type Radios []Radio

type RadioLink struct {
	ID      string `structs:"id"   json:"id" orm:"column(id)"`
	RadioId string `structs:"id"   json:"radioId,omitempty" orm:"column(radio_id)"`
	Name    string `structs:"name" json:"name"`
	Url     string `structs:"url"  json:"url"`
}

func NewRadioLink(radioId string, name string, url string) RadioLink {
	return RadioLink{
		ID:      strings.ReplaceAll(uuid.NewString(), "-", ""),
		Name:    name,
		RadioId: radioId,
		Url:     url,
	}
}

type RadioLinks []RadioLink

type RadioRepository interface {
	ResourceRepository
	CountAll(options ...QueryOptions) (int64, error)
	Delete(id string) error
	Get(id string) (*Radio, error)
	GetAll(options ...QueryOptions) (Radios, error)
	Put(u *Radio) error
}
