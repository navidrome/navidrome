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
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

const batchSize = 100

var (
	once         sync.Once
	driver       = "sqlite3"
	mappedModels map[interface{}]interface{}
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

func (db *SQLStore) Album(context.Context) model.AlbumRepository {
	return NewAlbumRepository(db.getOrmer())
}

func (db *SQLStore) Artist(context.Context) model.ArtistRepository {
	return NewArtistRepository(db.getOrmer())
}

func (db *SQLStore) MediaFile(context.Context) model.MediaFileRepository {
	return NewMediaFileRepository(db.getOrmer())
}

func (db *SQLStore) MediaFolder(context.Context) model.MediaFolderRepository {
	return NewMediaFolderRepository(db.getOrmer())
}

func (db *SQLStore) Genre(context.Context) model.GenreRepository {
	return NewGenreRepository(db.getOrmer())
}

func (db *SQLStore) Playlist(context.Context) model.PlaylistRepository {
	return NewPlaylistRepository(db.getOrmer())
}

func (db *SQLStore) Property(context.Context) model.PropertyRepository {
	return NewPropertyRepository(db.getOrmer())
}

func (db *SQLStore) User(context.Context) model.UserRepository {
	return NewUserRepository(db.getOrmer())
}

func (db *SQLStore) Annotation(context.Context) model.AnnotationRepository {
	return NewAnnotationRepository(db.getOrmer())
}

func (db *SQLStore) Resource(ctx context.Context, model interface{}) model.ResourceRepository {
	return NewResource(db.getOrmer(), model, getMappedModel(model))
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

func (db *SQLStore) getOrmer() orm.Ormer {
	if db.orm == nil {
		return orm.NewOrm()
	}
	return db.orm
}

func initORM(dbPath string) error {
	verbose := conf.Server.LogLevel == "trace"
	orm.Debug = verbose
	if strings.Contains(dbPath, "postgres") {
		driver = "postgres"
	}
	err := orm.RegisterDataBase("default", driver, dbPath)
	if err != nil {
		panic(err)
	}
	return orm.RunSyncdb("default", false, verbose)
}

func collectField(collection interface{}, getValue func(item interface{}) string) []string {
	s := reflect.ValueOf(collection)
	result := make([]string, s.Len())

	for i := 0; i < s.Len(); i++ {
		result[i] = getValue(s.Index(i).Interface())
	}

	return result
}

func getType(myvar interface{}) string {
	if t := reflect.TypeOf(myvar); t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	} else {
		return t.Name()
	}
}

func registerModel(model interface{}, mappedModel interface{}) {
	mappedModels[getType(model)] = mappedModel
	orm.RegisterModel(mappedModel)
}

func getMappedModel(model interface{}) interface{} {
	return mappedModels[getType(model)]
}

func init() {
	mappedModels = map[interface{}]interface{}{}

	registerModel(model.Artist{}, new(artist))
	registerModel(model.Album{}, new(album))
	registerModel(model.MediaFile{}, new(mediaFile))
	registerModel(model.Property{}, new(property))
	registerModel(model.Playlist{}, new(playlist))
	registerModel(model.User{}, new(user))
	registerModel(model.Annotation{}, new(annotation))

	orm.RegisterModel(new(search))
}
