package persistence

import (
	"context"
	"reflect"
	"strings"
	"sync"

	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
)

var (
	once   sync.Once
	driver = "sqlite3"
)

type SQLStore struct {
	orm orm.Ormer
}

func New() model.DataStore {
	once.Do(func() {
		dbPath := conf.Server.DbPath
		if dbPath == ":memory:" {
			dbPath = "file::memory:?cache=shared"
		}
		log.Debug("Opening DataBase", "dbPath", dbPath, "driver", driver)

		err := initORM(dbPath)
		if err != nil {
			panic(err)
		}
	})
	return &SQLStore{}
}

func (db *SQLStore) Album(ctx context.Context) model.AlbumRepository {
	return NewAlbumRepository(ctx, db.getOrmer())
}

func (db *SQLStore) Artist(ctx context.Context) model.ArtistRepository {
	return NewArtistRepository(ctx, db.getOrmer())
}

func (db *SQLStore) MediaFile(ctx context.Context) model.MediaFileRepository {
	return NewMediaFileRepository(ctx, db.getOrmer())
}

func (db *SQLStore) MediaFolder(ctx context.Context) model.MediaFolderRepository {
	return NewMediaFolderRepository(ctx, db.getOrmer())
}

func (db *SQLStore) Genre(ctx context.Context) model.GenreRepository {
	return NewGenreRepository(ctx, db.getOrmer())
}

func (db *SQLStore) Playlist(ctx context.Context) model.PlaylistRepository {
	return NewPlaylistRepository(ctx, db.getOrmer())
}

func (db *SQLStore) Property(ctx context.Context) model.PropertyRepository {
	return NewPropertyRepository(ctx, db.getOrmer())
}

func (db *SQLStore) User(ctx context.Context) model.UserRepository {
	return NewUserRepository(ctx, db.getOrmer())
}

func (db *SQLStore) Resource(ctx context.Context, m interface{}) model.ResourceRepository {
	switch m.(type) {
	case model.User:
		return db.User(ctx).(model.ResourceRepository)
	case model.Artist:
		return db.Artist(ctx).(model.ResourceRepository)
	case model.Album:
		return db.Album(ctx).(model.ResourceRepository)
	case model.MediaFile:
		return db.MediaFile(ctx).(model.ResourceRepository)
	}
	log.Error("Resource no implemented", "model", reflect.TypeOf(m).Name())
	return nil
}

func (db *SQLStore) WithTx(block func(tx model.DataStore) error) error {
	o := orm.NewOrm()
	err := o.Begin()
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

func (db *SQLStore) GC(ctx context.Context) error {
	err := db.Album(ctx).PurgeEmpty()
	if err != nil {
		return err
	}
	err = db.Artist(ctx).PurgeEmpty()
	if err != nil {
		return err
	}
	err = db.MediaFile(ctx).(*mediaFileRepository).cleanSearchIndex()
	if err != nil {
		return err
	}
	err = db.Album(ctx).(*albumRepository).cleanSearchIndex()
	if err != nil {
		return err
	}
	err = db.Artist(ctx).(*artistRepository).cleanSearchIndex()
	if err != nil {
		return err
	}
	err = db.MediaFile(ctx).(*mediaFileRepository).cleanAnnotations()
	if err != nil {
		return err
	}
	err = db.Album(ctx).(*albumRepository).cleanAnnotations()
	if err != nil {
		return err
	}
	return db.Artist(ctx).(*artistRepository).cleanAnnotations()
}

func (db *SQLStore) getOrmer() orm.Ormer {
	if db.orm == nil {
		return orm.NewOrm()
	}
	return db.orm
}

func initORM(dbPath string) error {
	if strings.Contains(dbPath, "postgres") {
		driver = "postgres"
	}
	return orm.RegisterDataBase("default", driver, dbPath)
}
