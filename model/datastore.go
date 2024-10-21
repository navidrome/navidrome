package model

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
)

type QueryOptions struct {
	Sort    string
	Order   string
	Max     int
	Offset  int
	Filters squirrel.Sqlizer
	Seed    string // for random sorting
}

type ResourceRepository interface {
	rest.Repository
}

type DataStore interface {
	Library(ctx context.Context) LibraryRepository
	Folder(ctx context.Context) FolderRepository
	Album(ctx context.Context) AlbumRepository
	Artist(ctx context.Context) ArtistRepository
	MediaFile(ctx context.Context) MediaFileRepository
	Genre(ctx context.Context) GenreRepository
	Tag(ctx context.Context) TagRepository
	Playlist(ctx context.Context) PlaylistRepository
	PlayQueue(ctx context.Context) PlayQueueRepository
	Transcoding(ctx context.Context) TranscodingRepository
	Player(ctx context.Context) PlayerRepository
	Radio(ctx context.Context) RadioRepository
	Share(ctx context.Context) ShareRepository
	Property(ctx context.Context) PropertyRepository
	User(ctx context.Context) UserRepository
	UserProps(ctx context.Context) UserPropsRepository
	ScrobbleBuffer(ctx context.Context) ScrobbleBufferRepository

	Resource(ctx context.Context, model interface{}) ResourceRepository

	WithTx(func(tx DataStore) error) error
	GC(ctx context.Context, rootFolder string) error
}
