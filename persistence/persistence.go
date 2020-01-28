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

type NewSQLStore struct {
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
	return &NewSQLStore{}
}

func (db *NewSQLStore) Album(ctx context.Context) model.AlbumRepository {
	return NewAlbumRepository(ctx, db.getOrmer())
}

func (db *NewSQLStore) Artist(ctx context.Context) model.ArtistRepository {
	return NewArtistRepository(ctx, db.getOrmer())
}

func (db *NewSQLStore) MediaFile(ctx context.Context) model.MediaFileRepository {
	return NewMediaFileRepository(ctx, db.getOrmer())
}

func (db *NewSQLStore) MediaFolder(ctx context.Context) model.MediaFolderRepository {
	return NewMediaFolderRepository(ctx, db.getOrmer())
}

func (db *NewSQLStore) Genre(ctx context.Context) model.GenreRepository {
	return NewGenreRepository(ctx, db.getOrmer())
}

func (db *NewSQLStore) Playlist(ctx context.Context) model.PlaylistRepository {
	return NewPlaylistRepository(ctx, db.getOrmer())
}

func (db *NewSQLStore) Property(ctx context.Context) model.PropertyRepository {
	return NewPropertyRepository(ctx, db.getOrmer())
}

func (db *NewSQLStore) User(ctx context.Context) model.UserRepository {
	return NewUserRepository(ctx, db.getOrmer())
}

func (db *NewSQLStore) Annotation(ctx context.Context) model.AnnotationRepository {
	return NewAnnotationRepository(ctx, db.getOrmer())
}

func getTypeName(model interface{}) string {
	return reflect.TypeOf(model).Name()
}

func (db *NewSQLStore) Resource(ctx context.Context, m interface{}) model.ResourceRepository {
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
	log.Error("Resource no implemented", "model", getTypeName(m))
	return nil
}

func (db *NewSQLStore) WithTx(block func(tx model.DataStore) error) error {
	o := orm.NewOrm()
	err := o.Begin()
	if err != nil {
		return err
	}

	newDb := &NewSQLStore{orm: o}
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

func (db *NewSQLStore) getOrmer() orm.Ormer {
	if db.orm == nil {
		return orm.NewOrm()
	}
	return db.orm
}

func initORM(dbPath string) error {
	//verbose := conf.Server.LogLevel == "trace"
	//orm.Debug = verbose
	if strings.Contains(dbPath, "postgres") {
		driver = "postgres"
	}
	err := orm.RegisterDataBase("default", driver, dbPath)
	if err != nil {
		return err
	}
	// TODO Remove all RegisterModels (i.e. don't use orm.Insert/Update)
	orm.RegisterModel(new(annotation))

	return nil
}
