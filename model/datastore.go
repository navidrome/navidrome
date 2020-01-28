package model

import (
	"context"

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
}

type DataStore interface {
	Album(ctx context.Context) AlbumRepository
	Artist(ctx context.Context) ArtistRepository
	MediaFile(ctx context.Context) MediaFileRepository
	MediaFolder(ctx context.Context) MediaFolderRepository
	Genre(ctx context.Context) GenreRepository
	Playlist(ctx context.Context) PlaylistRepository
	Property(ctx context.Context) PropertyRepository
	User(ctx context.Context) UserRepository
	Annotation(ctx context.Context) AnnotationRepository

	Resource(ctx context.Context, model interface{}) ResourceRepository

	WithTx(func(tx DataStore) error) error
}
