package persistence

import (
	"context"
	"reflect"
	"time"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/chain"
	"github.com/pocketbase/dbx"
)

type SQLStore struct {
	db dbx.Builder
}

func New(d db.DB) model.DataStore {
	return &SQLStore{db: NewDBXBuilder(d)}
}

func (s *SQLStore) Album(ctx context.Context) model.AlbumRepository {
	return NewAlbumRepository(ctx, s.getDBXBuilder())
}

func (s *SQLStore) Artist(ctx context.Context) model.ArtistRepository {
	return NewArtistRepository(ctx, s.getDBXBuilder())
}

func (s *SQLStore) MediaFile(ctx context.Context) model.MediaFileRepository {
	return NewMediaFileRepository(ctx, s.getDBXBuilder())
}

func (s *SQLStore) Library(ctx context.Context) model.LibraryRepository {
	return NewLibraryRepository(ctx, s.getDBXBuilder())
}

func (s *SQLStore) Folder(ctx context.Context) model.FolderRepository {
	return newFolderRepository(ctx, s.getDBXBuilder())
}

func (s *SQLStore) Genre(ctx context.Context) model.GenreRepository {
	return NewGenreRepository(ctx, s.getDBXBuilder())
}

func (s *SQLStore) Tag(ctx context.Context) model.TagRepository {
	return NewTagRepository(ctx, s.getDBXBuilder())
}

func (s *SQLStore) PlayQueue(ctx context.Context) model.PlayQueueRepository {
	return NewPlayQueueRepository(ctx, s.getDBXBuilder())
}

func (s *SQLStore) Playlist(ctx context.Context) model.PlaylistRepository {
	return NewPlaylistRepository(ctx, s.getDBXBuilder())
}

func (s *SQLStore) Property(ctx context.Context) model.PropertyRepository {
	return NewPropertyRepository(ctx, s.getDBXBuilder())
}

func (s *SQLStore) Radio(ctx context.Context) model.RadioRepository {
	return NewRadioRepository(ctx, s.getDBXBuilder())
}

func (s *SQLStore) UserProps(ctx context.Context) model.UserPropsRepository {
	return NewUserPropsRepository(ctx, s.getDBXBuilder())
}

func (s *SQLStore) Share(ctx context.Context) model.ShareRepository {
	return NewShareRepository(ctx, s.getDBXBuilder())
}

func (s *SQLStore) User(ctx context.Context) model.UserRepository {
	return NewUserRepository(ctx, s.getDBXBuilder())
}

func (s *SQLStore) Transcoding(ctx context.Context) model.TranscodingRepository {
	return NewTranscodingRepository(ctx, s.getDBXBuilder())
}

func (s *SQLStore) Player(ctx context.Context) model.PlayerRepository {
	return NewPlayerRepository(ctx, s.getDBXBuilder())
}

func (s *SQLStore) ScrobbleBuffer(ctx context.Context) model.ScrobbleBufferRepository {
	return NewScrobbleBufferRepository(ctx, s.getDBXBuilder())
}

func (s *SQLStore) Resource(ctx context.Context, m interface{}) model.ResourceRepository {
	switch m.(type) {
	case model.User:
		return s.User(ctx).(model.ResourceRepository)
	case model.Transcoding:
		return s.Transcoding(ctx).(model.ResourceRepository)
	case model.Player:
		return s.Player(ctx).(model.ResourceRepository)
	case model.Artist:
		return s.Artist(ctx).(model.ResourceRepository)
	case model.Album:
		return s.Album(ctx).(model.ResourceRepository)
	case model.MediaFile:
		return s.MediaFile(ctx).(model.ResourceRepository)
	case model.Genre:
		return s.Genre(ctx).(model.ResourceRepository)
	case model.Playlist:
		return s.Playlist(ctx).(model.ResourceRepository)
	case model.Radio:
		return s.Radio(ctx).(model.ResourceRepository)
	case model.Share:
		return s.Share(ctx).(model.ResourceRepository)
	}
	log.Error("Resource not implemented", "model", reflect.TypeOf(m).Name())
	return nil
}

type transactional interface {
	Transactional(f func(*dbx.Tx) error) (err error)
}

func (s *SQLStore) WithTx(block func(tx model.DataStore) error) error {
	// If we are already in a transaction, just pass it down
	if conn, ok := s.db.(*dbx.Tx); ok {
		return block(&SQLStore{db: conn})
	}

	return s.db.(transactional).Transactional(func(tx *dbx.Tx) error {
		return block(&SQLStore{db: tx})
	})
}

func trace(ctx context.Context, msg string, f func() error) func() error {
	return func() error {
		start := time.Now()
		err := f()
		log.Debug(ctx, "GC: "+msg, "elapsed", time.Since(start), err)
		return err
	}
}

func (s *SQLStore) GC(ctx context.Context) error {
	err := chain.RunSequentially(
		trace(ctx, "purge empty albums", func() error { return s.Album(ctx).(*albumRepository).purgeEmpty() }),
		trace(ctx, "purge empty artists", func() error { return s.Artist(ctx).(*artistRepository).purgeEmpty() }),
		trace(ctx, "purge empty folders", func() error { return s.Folder(ctx).(*folderRepository).purgeEmpty() }),
		trace(ctx, "clean album annotations", func() error { return s.Album(ctx).(*albumRepository).cleanAnnotations() }),
		trace(ctx, "clean artist annotations", func() error { return s.Artist(ctx).(*artistRepository).cleanAnnotations() }),
		trace(ctx, "clean media file annotations", func() error { return s.MediaFile(ctx).(*mediaFileRepository).cleanAnnotations() }),
		trace(ctx, "clean media file bookmarks", func() error { return s.MediaFile(ctx).(*mediaFileRepository).cleanBookmarks() }),
		trace(ctx, "purge non used tags", func() error { return s.Tag(ctx).(*tagRepository).purgeNonUsed() }),
		trace(ctx, "remove orphan playlists", func() error { return s.Playlist(ctx).(*playlistRepository).removeOrphans() }),
	)
	if err != nil {
		log.Error(ctx, "Error tidying up database", err)
	}
	return err
}

func (s *SQLStore) getDBXBuilder() dbx.Builder {
	if s.db == nil {
		return NewDBXBuilder(db.Db())
	}
	return s.db
}
