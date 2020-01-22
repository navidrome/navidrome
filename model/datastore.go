package model

import (
	"github.com/deluan/rest"
)

// Filters use the same operators as Beego ORM: See https://beego.me/docs/mvc/model/query.md#operators
// Ex: var q = QueryOptions{Filters: Filters{"name__istartswith": "Deluan","age__gt": 25}}
// All conditions will be ANDed together
// TODO Implement filter in repositories' methods
type QueryOptions struct {
	Sort    string
	Order   string
	Max     int
	Offset  int
	Filters map[string]interface{}
}

type ResourceRepository interface {
	rest.Repository
	rest.Persistable
}

type DataStore interface {
	Album() AlbumRepository
	Artist() ArtistRepository
	MediaFile() MediaFileRepository
	MediaFolder() MediaFolderRepository
	Genre() GenreRepository
	Playlist() PlaylistRepository
	Property() PropertyRepository
	User() UserRepository
	Annotation() AnnotationRepository

	Resource(model interface{}) ResourceRepository

	WithTx(func(tx DataStore) error) error
}
