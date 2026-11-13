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
	// SortOrders is one "asc"/"desc" per Sort key, kept on every column its mapping expands into.
	// Empty means Order applies to every key; a key past the end reuses the first entry.
	SortOrders []string
}

type DataStore interface {
	Library() LibraryRepository
	Folder() FolderRepository
	Album() AlbumRepository
	Artist() ArtistRepository
	MediaFile() MediaFileRepository
	Genre() GenreRepository
	Tag() TagRepository
	Playlist() PlaylistRepository
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
	Grant() GrantRepository

	WithTx(block func(tx DataStore) error, scope ...string) error
	WithTxImmediate(block func(tx DataStore) error, scope ...string) error
	// WithTxRetry runs block in a transaction, rerunning it while SQLite reports the database busy.
	// For background work only (it can take minutes), and block must be safe to rerun after a rollback.
	WithTxRetry(ctx context.Context, block func(ctx context.Context, tx DataStore) error, scope ...string) error
	GC(ctx context.Context, libraryIDs ...int) error
}
