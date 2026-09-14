package model

import (
	"context"

	"github.com/Masterminds/squirrel"
)

type QueryOptions struct {
	Sort    string
	Order   string
	Max     int
	Offset  int
	Filters squirrel.Sqlizer
	Seed    string // for random sorting
}

type DataStore interface {
	Library() LibraryRepository
	Folder(ctx context.Context) FolderRepository
	Album(ctx context.Context) AlbumRepository
	Artist(ctx context.Context) ArtistRepository
	MediaFile(ctx context.Context) MediaFileRepository
	Genre() GenreRepository
	Tag() TagRepository
	Playlist(ctx context.Context) PlaylistRepository
	PlayQueue() PlayQueueRepository
	Transcoding() TranscodingRepository
	Player() PlayerRepository
	Radio() RadioRepository
	Share() ShareRepository
	Property() PropertyRepository
	User(ctx context.Context) UserRepository
	UserProps() UserPropsRepository
	ScrobbleBuffer(ctx context.Context) ScrobbleBufferRepository
	Scrobble(ctx context.Context) ScrobbleRepository
	Plugin(ctx context.Context) PluginRepository
	Artwork(ctx context.Context) ArtworkRepository
	ArtworkQueue(ctx context.Context) ArtworkQueueRepository

	WithTx(block func(tx DataStore) error, scope ...string) error
	WithTxImmediate(block func(tx DataStore) error, scope ...string) error
	GC(ctx context.Context, libraryIDs ...int) error
}
