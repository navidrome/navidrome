package persistence

import (
	"context"
	"io/fs"
	"reflect"

	"github.com/astaxie/beego/orm"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type SQLStore struct {
	orm  orm.Ormer
	fsys fs.FS
}

func New() model.DataStore {
	return &SQLStore{}
}

func (s *SQLStore) Album(ctx context.Context) model.AlbumRepository {
	return NewAlbumRepository(ctx, s.fsys, s.getOrmer())
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

func (s *SQLStore) PlayQueue(ctx context.Context) model.PlayQueueRepository {
	return NewPlayQueueRepository(ctx, s.getOrmer())
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
	case model.Playlist:
		return s.Playlist(ctx).(model.ResourceRepository)
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

func (s *SQLStore) GC(ctx context.Context, rootFolder string) error {
	err := s.MediaFile(ctx).(*mediaFileRepository).deleteNotInPath(rootFolder)
	if err != nil {
		log.Error(ctx, "Error removing dangling tracks", err)
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
	return err
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
