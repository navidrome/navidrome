package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/run"
	"github.com/pocketbase/dbx"
)

type SQLStore struct {
	db           dbx.Builder
	library      func() model.LibraryRepository
	folder       func() model.FolderRepository
	album        func() model.AlbumRepository
	artist       func() model.ArtistRepository
	mediaFile    func() model.MediaFileRepository
	genre        func() model.GenreRepository
	tag          func() model.TagRepository
	playlist     func() model.PlaylistRepository
	playQueue    func() model.PlayQueueRepository
	transcoding  func() model.TranscodingRepository
	player       func() model.PlayerRepository
	radio        func() model.RadioRepository
	share        func() model.ShareRepository
	property     func() model.PropertyRepository
	user         func() model.UserRepository
	userProps    func() model.UserPropsRepository
	scrobbleBuf  func() model.ScrobbleBufferRepository
	scrobble     func() model.ScrobbleRepository
	plugin       func() model.PluginRepository
	artwork      func() model.ArtworkRepository
	artworkQueue func() model.ArtworkQueueRepository
}

// Repositories are built on first use, so a transaction store only pays for the ones its block touches.
func newSQLStore(db dbx.Builder) *SQLStore {
	return &SQLStore{
		db:           db,
		library:      sync.OnceValue(func() model.LibraryRepository { return NewLibraryRepository(db) }),
		folder:       sync.OnceValue(func() model.FolderRepository { return newFolderRepository(db) }),
		album:        sync.OnceValue(func() model.AlbumRepository { return NewAlbumRepository(db) }),
		artist:       sync.OnceValue(func() model.ArtistRepository { return NewArtistRepository(db) }),
		mediaFile:    sync.OnceValue(func() model.MediaFileRepository { return NewMediaFileRepository(db) }),
		genre:        sync.OnceValue(func() model.GenreRepository { return NewGenreRepository(db) }),
		tag:          sync.OnceValue(func() model.TagRepository { return NewTagRepository(db) }),
		playlist:     sync.OnceValue(func() model.PlaylistRepository { return NewPlaylistRepository(db) }),
		playQueue:    sync.OnceValue(func() model.PlayQueueRepository { return NewPlayQueueRepository(db) }),
		transcoding:  sync.OnceValue(func() model.TranscodingRepository { return NewTranscodingRepository(db) }),
		player:       sync.OnceValue(func() model.PlayerRepository { return NewPlayerRepository(db) }),
		radio:        sync.OnceValue(func() model.RadioRepository { return NewRadioRepository(db) }),
		share:        sync.OnceValue(func() model.ShareRepository { return NewShareRepository(db) }),
		property:     sync.OnceValue(func() model.PropertyRepository { return NewPropertyRepository(db) }),
		user:         sync.OnceValue(func() model.UserRepository { return NewUserRepository(db) }),
		userProps:    sync.OnceValue(func() model.UserPropsRepository { return NewUserPropsRepository(db) }),
		scrobbleBuf:  sync.OnceValue(func() model.ScrobbleBufferRepository { return NewScrobbleBufferRepository(db) }),
		scrobble:     sync.OnceValue(func() model.ScrobbleRepository { return NewScrobbleRepository(db) }),
		plugin:       sync.OnceValue(func() model.PluginRepository { return NewPluginRepository(db) }),
		artwork:      sync.OnceValue(func() model.ArtworkRepository { return NewArtworkRepository(db) }),
		artworkQueue: sync.OnceValue(func() model.ArtworkQueueRepository { return NewArtworkQueueRepository(db) }),
	}
}

func New(conn *sql.DB) model.DataStore {
	return newSQLStore(dbx.NewFromDB(conn, db.Driver))
}

func (s *SQLStore) Album() model.AlbumRepository {
	return s.album()
}

func (s *SQLStore) Artist() model.ArtistRepository {
	return s.artist()
}

func (s *SQLStore) MediaFile() model.MediaFileRepository {
	return s.mediaFile()
}

func (s *SQLStore) Library() model.LibraryRepository {
	return s.library()
}

func (s *SQLStore) Folder() model.FolderRepository {
	return s.folder()
}

func (s *SQLStore) Genre() model.GenreRepository {
	return s.genre()
}

func (s *SQLStore) Tag() model.TagRepository {
	return s.tag()
}

func (s *SQLStore) PlayQueue() model.PlayQueueRepository {
	return s.playQueue()
}

func (s *SQLStore) Playlist() model.PlaylistRepository {
	return s.playlist()
}

func (s *SQLStore) Property() model.PropertyRepository {
	return s.property()
}

func (s *SQLStore) Radio() model.RadioRepository {
	return s.radio()
}

func (s *SQLStore) UserProps() model.UserPropsRepository {
	return s.userProps()
}

func (s *SQLStore) Share() model.ShareRepository {
	return s.share()
}

func (s *SQLStore) User() model.UserRepository {
	return s.user()
}

func (s *SQLStore) Transcoding() model.TranscodingRepository {
	return s.transcoding()
}

func (s *SQLStore) Player() model.PlayerRepository {
	return s.player()
}

func (s *SQLStore) ScrobbleBuffer() model.ScrobbleBufferRepository {
	return s.scrobbleBuf()
}

func (s *SQLStore) Scrobble() model.ScrobbleRepository {
	return s.scrobble()
}

func (s *SQLStore) Plugin() model.PluginRepository {
	return s.plugin()
}

func (s *SQLStore) Artwork() model.ArtworkRepository {
	return s.artwork()
}

func (s *SQLStore) ArtworkQueue() model.ArtworkQueueRepository {
	return s.artworkQueue()
}

func scopeLabel(scope []string) string {
	if len(scope) > 0 {
		return scope[0]
	}
	return ""
}

func (s *SQLStore) WithTx(block func(tx model.DataStore) error, scope ...string) error {
	msg := scopeLabel(scope)
	start := time.Now()
	conn, inTx := s.db.(*dbx.DB)
	if !inTx {
		log.Trace("Nested Transaction started", "scope", msg)
		conn = dbx.NewFromDB(db.Db(), db.Driver)
	} else {
		log.Trace("Transaction started", "scope", msg)
	}
	return conn.Transactional(func(tx *dbx.Tx) error {
		newDb := newSQLStore(tx)
		err := block(newDb)
		if !inTx {
			log.Trace("Nested Transaction finished", "scope", msg, "elapsed", time.Since(start), err)
		} else {
			log.Trace("Transaction finished", "scope", msg, "elapsed", time.Since(start), err)
		}
		return err
	})
}

func (s *SQLStore) WithTxImmediate(block func(tx model.DataStore) error, scope ...string) error {
	ctx := context.Background()
	return s.WithTx(func(tx model.DataStore) error {
		// Workaround to force the transaction to be upgraded to immediate mode to avoid deadlocks
		// See https://berthub.eu/articles/posts/a-brief-post-on-sqlite3-database-locked-despite-timeout/
		_ = tx.Property().Put(ctx, "tmp_lock_flag", "")
		defer func() {
			_ = tx.Property().Delete(ctx, "tmp_lock_flag")
		}()

		return block(tx)
	}, scope...)
}

// txRetryDelay spaces out reruns of a busy transaction. Each attempt has already waited out the
// busy timeout, so WithTxRetry gives up only after a sustained lock.
var txRetryDelay = 5 * time.Second

const txMaxRetries = 3

func (s *SQLStore) WithTxRetry(ctx context.Context, block func(ctx context.Context, tx model.DataStore) error, scope ...string) error {
	// Inside a transaction, join it: the outer one holds the lock and owns commit and rollback
	if _, ok := s.db.(*dbx.DB); !ok {
		return block(ctx, s)
	}
	for attempt := 0; ; attempt++ {
		attemptCtx := ctx
		if attempt < txMaxRetries {
			attemptCtx = withBusyRetry(ctx)
		}
		err := s.WithTx(func(tx model.DataStore) error { return block(attemptCtx, tx) }, scope...)
		if attempt == txMaxRetries || !db.IsBusy(err) {
			return err
		}
		log.Warn(ctx, "Database busy, retrying transaction", "scope", scopeLabel(scope), "attempt", attempt+1, err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * txRetryDelay):
		}
	}
}

type busyRetryKey struct{}

// withBusyRetry marks a transaction attempt that WithTxRetry will rerun, so a busy statement in it
// is logged as a warning rather than an error.
func withBusyRetry(ctx context.Context) context.Context {
	return context.WithValue(ctx, busyRetryKey{}, true)
}

func hasBusyRetry(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	retry, _ := ctx.Value(busyRetryKey{}).(bool)
	return retry
}

func (s *SQLStore) GC(ctx context.Context, libraryIDs ...int) error {
	trace := func(ctx context.Context, msg string, f func() error) func() error {
		return func() error {
			start := time.Now()
			err := f()
			log.Debug(ctx, "GC: "+msg, "elapsed", time.Since(start), err)
			return err
		}
	}

	// If libraryIDs are provided, scope operations to those libraries where possible
	scoped := len(libraryIDs) > 0
	if scoped {
		log.Debug(ctx, "GC: Running selective garbage collection", "libraryIDs", libraryIDs)
	}

	err := run.Sequentially(
		trace(ctx, "purge empty albums", func() error { return s.album().(*albumRepository).purgeEmpty(ctx, libraryIDs...) }),
		trace(ctx, "purge empty artists", func() error { return s.artist().(*artistRepository).purgeEmpty(ctx) }),
		trace(ctx, "mark missing artists", func() error { return s.artist().(*artistRepository).markMissing(ctx) }),
		trace(ctx, "purge empty folders", func() error { return s.folder().(*folderRepository).purgeEmpty(ctx, libraryIDs...) }),
		trace(ctx, "clean album annotations", func() error { return s.album().(*albumRepository).cleanAnnotations(ctx) }),
		trace(ctx, "clean artist annotations", func() error { return s.artist().(*artistRepository).cleanAnnotations(ctx) }),
		trace(ctx, "clean media file annotations", func() error { return s.mediaFile().(*mediaFileRepository).cleanAnnotations(ctx) }),
		trace(ctx, "clean playlist annotations", func() error { return s.playlist().(*playlistRepository).cleanAnnotations(ctx) }),
		trace(ctx, "clean media file bookmarks", func() error { return s.mediaFile().(*mediaFileRepository).cleanBookmarks(ctx) }),
		trace(ctx, "purge non used tags", func() error { return s.tag().(*tagRepository).purgeUnused(ctx) }),
		trace(ctx, "remove orphan playlist tracks", func() error { return s.playlist().(*playlistRepository).removeOrphans(ctx) }),
	)
	if err != nil {
		return fmt.Errorf("tidying up database: %w", err)
	}
	return nil
}
