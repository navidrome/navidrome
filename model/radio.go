package model

import (
	"time"

	"github.com/deluan/rest"

	"github.com/navidrome/navidrome/consts"
)

type Radio struct {
	ItemImage `structs:"-"`

	ID            string    `structs:"id"              json:"id"`
	StreamUrl     string    `structs:"stream_url"      json:"streamUrl"`
	Name          string    `structs:"name"            json:"name"`
	HomePageUrl   string    `structs:"home_page_url"   json:"homePageUrl"`
	UploadedImage string    `structs:"uploaded_image"   json:"uploadedImage,omitempty"`
	CreatedAt     time.Time `structs:"created_at"      json:"createdAt"`
	UpdatedAt     time.Time `structs:"updated_at"      json:"updatedAt"`
}

func (r Radio) CoverArtID() ArtworkID {
	return artworkIDFromRadio(r)
}

func (r Radio) UploadedImagePath() string {
	return UploadedImagePath(consts.EntityRadio, r.UploadedImage)
}

type Radios []Radio

type RadioRepository interface {
	rest.Repository[Radio]
	rest.Persistable[Radio]
	CountAll(options ...QueryOptions) (int64, error)
	Exists(id string) (bool, error)
	Get(id string) (*Radio, error)
	GetAll(options ...QueryOptions) (Radios, error)
	Put(u *Radio, colsToUpdate ...string) error
}
