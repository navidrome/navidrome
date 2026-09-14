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
	Folder() FolderRepository
	Album() AlbumRepository
	Artist() ArtistRepository
	MediaFile() MediaFileRepository
	Genre() GenreRepository
	Tag() TagRepository
	Playlist(ctx context.Context) PlaylistRepository
	PlayQueue() PlayQueueRepository
	Transcoding() TranscodingRepository
	Player() PlayerRepository
	Radio() RadioRepository
	Share() ShareRepository
	Property() PropertyRepository
	User() UserRepository
	UserProps() UserPropsRepository
	ScrobbleBuffer() ScrobbleBufferRepository
	Scrobble() ScrobbleRepository
	Plugin() PluginRepository
	Artwork() ArtworkRepository
	ArtworkQueue() ArtworkQueueRepository

	WithTx(block func(tx DataStore) error, scope ...string) error
	WithTxImmediate(block func(tx DataStore) error, scope ...string) error
	GC(ctx context.Context, libraryIDs ...int) error
}
