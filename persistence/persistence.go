package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/run"
	"github.com/pocketbase/dbx"
)

type SQLStore struct {
	db           dbx.Builder
	library      model.LibraryRepository
	folder       model.FolderRepository
	album        model.AlbumRepository
	artist       model.ArtistRepository
	mediaFile    model.MediaFileRepository
	genre        model.GenreRepository
	tag          model.TagRepository
	playlist     model.PlaylistRepository
	playQueue    model.PlayQueueRepository
	transcoding  model.TranscodingRepository
	player       model.PlayerRepository
	radio        model.RadioRepository
	share        model.ShareRepository
	property     model.PropertyRepository
	user         model.UserRepository
	userProps    model.UserPropsRepository
	scrobbleBuf  model.ScrobbleBufferRepository
	scrobble     model.ScrobbleRepository
	plugin       model.PluginRepository
	artwork      model.ArtworkRepository
	artworkQueue model.ArtworkQueueRepository
}

func newSQLStore(db dbx.Builder) *SQLStore {
	return &SQLStore{
		db:           db,
		library:      NewLibraryRepository(db),
		folder:       newFolderRepository(db),
		album:        NewAlbumRepository(db),
		artist:       NewArtistRepository(db),
		mediaFile:    NewMediaFileRepository(db),
		genre:        NewGenreRepository(db),
		tag:          NewTagRepository(db),
		playlist:     NewPlaylistRepository(db),
		playQueue:    NewPlayQueueRepository(db),
		transcoding:  NewTranscodingRepository(db),
		player:       NewPlayerRepository(db),
		radio:        NewRadioRepository(db),
		share:        NewShareRepository(db),
		property:     NewPropertyRepository(db),
		user:         NewUserRepository(db),
		userProps:    NewUserPropsRepository(db),
		scrobbleBuf:  NewScrobbleBufferRepository(db),
		scrobble:     NewScrobbleRepository(db),
		plugin:       NewPluginRepository(db),
		artwork:      NewArtworkRepository(db),
		artworkQueue: NewArtworkQueueRepository(db),
	}
}

func New(conn *sql.DB) model.DataStore {
	return newSQLStore(dbx.NewFromDB(conn, db.Driver))
}

func (s *SQLStore) Album() model.AlbumRepository {
	return s.album
}

func (s *SQLStore) Artist() model.ArtistRepository {
	return s.artist
}

func (s *SQLStore) MediaFile() model.MediaFileRepository {
	return s.mediaFile
}

func (s *SQLStore) Library() model.LibraryRepository {
	return s.library
}

func (s *SQLStore) Folder() model.FolderRepository {
	return s.folder
}

func (s *SQLStore) Genre() model.GenreRepository {
	return s.genre
}

func (s *SQLStore) Tag() model.TagRepository {
	return s.tag
}

func (s *SQLStore) PlayQueue() model.PlayQueueRepository {
	return s.playQueue
}

func (s *SQLStore) Playlist() model.PlaylistRepository {
	return s.playlist
}

func (s *SQLStore) Property() model.PropertyRepository {
	return s.property
}

func (s *SQLStore) Radio() model.RadioRepository {
	return s.radio
}

func (s *SQLStore) UserProps() model.UserPropsRepository {
	return s.userProps
}

func (s *SQLStore) Share() model.ShareRepository {
	return s.share
}

func (s *SQLStore) User() model.UserRepository {
	return s.user
}

func (s *SQLStore) Transcoding() model.TranscodingRepository {
	return s.transcoding
}

func (s *SQLStore) Player() model.PlayerRepository {
	return s.player
}

func (s *SQLStore) ScrobbleBuffer() model.ScrobbleBufferRepository {
	return s.scrobbleBuf
}

func (s *SQLStore) Scrobble() model.ScrobbleRepository {
	return s.scrobble
}

func (s *SQLStore) Plugin() model.PluginRepository {
	return s.plugin
}

func (s *SQLStore) Artwork() model.ArtworkRepository {
	return s.artwork
}

func (s *SQLStore) ArtworkQueue() model.ArtworkQueueRepository {
	return s.artworkQueue
}

func (s *SQLStore) WithTx(block func(tx model.DataStore) error, scope ...string) error {
	var msg string
	if len(scope) > 0 {
		msg = scope[0]
	}
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
		trace(ctx, "purge empty albums", func() error { return s.album.(*albumRepository).purgeEmpty(ctx, libraryIDs...) }),
		trace(ctx, "purge empty artists", func() error { return s.artist.(*artistRepository).purgeEmpty(ctx) }),
		trace(ctx, "mark missing artists", func() error { return s.artist.(*artistRepository).markMissing(ctx) }),
		trace(ctx, "purge empty folders", func() error { return s.folder.(*folderRepository).purgeEmpty(ctx, libraryIDs...) }),
		trace(ctx, "clean album annotations", func() error { return s.album.(*albumRepository).cleanAnnotations(ctx) }),
		trace(ctx, "clean artist annotations", func() error { return s.artist.(*artistRepository).cleanAnnotations(ctx) }),
		trace(ctx, "clean media file annotations", func() error { return s.mediaFile.(*mediaFileRepository).cleanAnnotations(ctx) }),
		trace(ctx, "clean playlist annotations", func() error { return s.playlist.(*playlistRepository).cleanAnnotations(ctx) }),
		trace(ctx, "clean media file bookmarks", func() error { return s.mediaFile.(*mediaFileRepository).cleanBookmarks(ctx) }),
		trace(ctx, "purge non used tags", func() error { return s.tag.(*tagRepository).purgeUnused(ctx) }),
		trace(ctx, "remove orphan playlist tracks", func() error { return s.playlist.(*playlistRepository).removeOrphans(ctx) }),
	)
	if err != nil {
		log.Error(ctx, "Error tidying up database", err)
	}
	return err
}
