package persistence

import (
	"context"
	"database/sql"
	"reflect"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type SQLStore struct {
	db dbx.Builder
}

func New(conn *sql.DB) model.DataStore {
	return &SQLStore{db: dbx.NewFromDB(conn, db.Driver)}
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

func (s *SQLStore) MediaFolder(ctx context.Context) model.MediaFolderRepository {
	return NewMediaFolderRepository(ctx, s.getDBXBuilder())
}

func (s *SQLStore) Genre(ctx context.Context) model.GenreRepository {
	return NewGenreRepository(ctx, s.getDBXBuilder())
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

func (s *SQLStore) WithTx(block func(tx model.DataStore) error) error {
	conn, ok := s.db.(*dbx.DB)
	if !ok {
		conn = dbx.NewFromDB(db.Db(), db.Driver)
	}
	return conn.Transactional(func(tx *dbx.Tx) error {
		newDb := &SQLStore{db: tx}
		return block(newDb)
	})
}

func (s *SQLStore) GC(ctx context.Context, rootFolder string) error {
	err := s.MediaFile(ctx).(*mediaFileRepository).deleteNotInPath(rootFolder)
	if err != nil {
		log.Error(ctx, "Error removing dangling tracks", err)
		return err
	}
	err = s.MediaFile(ctx).(*mediaFileRepository).removeNonAlbumArtistIds()
	if err != nil {
		log.Error(ctx, "Error removing non-album artist_ids", err)
		return err
	}
	err = s.Album(ctx).(*albumRepository).purgeEmpty()
	if err != nil {
		log.Error(ctx, "Error removing empty albums", err)
		return err
	}
	err = s.Artist(ctx).(*artistRepository).purgeEmpty()
	if err != nil {
		log.Error(ctx, "Error removing empty artists", err)
		return err
	}
	err = s.MediaFile(ctx).(*mediaFileRepository).cleanAnnotations()
	if err != nil {
		log.Error(ctx, "Error removing orphan mediafile annotations", err)
		return err
	}
	err = s.Album(ctx).(*albumRepository).cleanAnnotations()
	if err != nil {
		log.Error(ctx, "Error removing orphan album annotations", err)
		return err
	}
	err = s.Artist(ctx).(*artistRepository).cleanAnnotations()
	if err != nil {
		log.Error(ctx, "Error removing orphan artist annotations", err)
		return err
	}
	err = s.MediaFile(ctx).(*mediaFileRepository).cleanBookmarks()
	if err != nil {
		log.Error(ctx, "Error removing orphan bookmarks", err)
		return err
	}
	err = s.Playlist(ctx).(*playlistRepository).removeOrphans()
	if err != nil {
		log.Error(ctx, "Error tidying up playlists", err)
	}
	err = s.Genre(ctx).(*genreRepository).purgeEmpty()
	if err != nil {
		log.Error(ctx, "Error removing unused genres", err)
		return err
	}
	return err
}

func (s *SQLStore) getDBXBuilder() dbx.Builder {
	if s.db == nil {
		return dbx.NewFromDB(db.Db(), db.Driver)
	}
	return s.db
}
