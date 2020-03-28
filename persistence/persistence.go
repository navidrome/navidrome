package persistence

import (
	"context"
	"reflect"

	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/db"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
)

type SQLStore struct {
	orm orm.Ormer
}

func New() model.DataStore {
	return &SQLStore{}
}

func (s *SQLStore) Album(ctx context.Context) model.AlbumRepository {
	return NewAlbumRepository(ctx, s.getOrmer())
}

func (s *SQLStore) Artist(ctx context.Context) model.ArtistRepository {
	return NewArtistRepository(ctx, s.getOrmer())
}

func (s *SQLStore) MediaFile(ctx context.Context) model.MediaFileRepository {
	return NewMediaFileRepository(ctx, s.getOrmer())
}

func (s *SQLStore) MediaFolder(ctx context.Context) model.MediaFolderRepository {
	return NewMediaFolderRepository(ctx, s.getOrmer())
}

func (s *SQLStore) Genre(ctx context.Context) model.GenreRepository {
	return NewGenreRepository(ctx, s.getOrmer())
}

func (s *SQLStore) Playlist(ctx context.Context) model.PlaylistRepository {
	return NewPlaylistRepository(ctx, s.getOrmer())
}

func (s *SQLStore) Property(ctx context.Context) model.PropertyRepository {
	return NewPropertyRepository(ctx, s.getOrmer())
}

func (s *SQLStore) User(ctx context.Context) model.UserRepository {
	return NewUserRepository(ctx, s.getOrmer())
}

func (s *SQLStore) Transcoding(ctx context.Context) model.TranscodingRepository {
	return NewTranscodingRepository(ctx, s.getOrmer())
}

func (s *SQLStore) Player(ctx context.Context) model.PlayerRepository {
	return NewPlayerRepository(ctx, s.getOrmer())
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
	}
	log.Error("Resource not implemented", "model", reflect.TypeOf(m).Name())
	return nil
}

func (s *SQLStore) WithTx(block func(tx model.DataStore) error) error {
	o, err := orm.NewOrmWithDB(db.Driver, "default", db.Db())
	if err != nil {
		return err
	}
	err = o.Begin()
	if err != nil {
		return err
	}

	newDb := &SQLStore{orm: o}
	err = block(newDb)

	if err != nil {
		err2 := o.Rollback()
		if err2 != nil {
			return err2
		}
		return err
	}

	err2 := o.Commit()
	if err2 != nil {
		return err2
	}
	return nil
}

func (s *SQLStore) GC(ctx context.Context) error {
	err := s.Album(ctx).PurgeEmpty()
	if err != nil {
		return err
	}
	err = s.Artist(ctx).PurgeEmpty()
	if err != nil {
		return err
	}
	err = s.MediaFile(ctx).(*mediaFileRepository).cleanAnnotations()
	if err != nil {
		return err
	}
	err = s.Album(ctx).(*albumRepository).cleanAnnotations()
	if err != nil {
		return err
	}
	return s.Artist(ctx).(*artistRepository).cleanAnnotations()
}

func (s *SQLStore) getOrmer() orm.Ormer {
	if s.orm == nil {
		o, err := orm.NewOrmWithDB(db.Driver, "default", db.Db())
		if err != nil {
			log.Error("Error obtaining new orm instance", err)
		}
		return o
	}
	return s.orm
}
