package persistence

import (
	"reflect"
	"strings"
	"sync"

	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/model"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

const batchSize = 100

var (
	once   sync.Once
	driver = "sqlite3"
)

type SQLStore struct {
	orm orm.Ormer
}

func New() model.DataStore {
	once.Do(func() {
		dbPath := conf.Sonic.DbPath
		if dbPath == ":memory:" {
			dbPath = "file::memory:?cache=shared"
		}
		err := initORM(dbPath)
		if err != nil {
			panic(err)
		}
		log.Debug("Opening DB from: "+dbPath, "driver", driver)
	})
	return &SQLStore{}
}

func (db *SQLStore) Album() model.AlbumRepository {
	return NewAlbumRepository(db.getOrmer())
}

func (db *SQLStore) Artist() model.ArtistRepository {
	return NewArtistRepository(db.getOrmer())
}

func (db *SQLStore) MediaFile() model.MediaFileRepository {
	return NewMediaFileRepository(db.getOrmer())
}

func (db *SQLStore) MediaFolder() model.MediaFolderRepository {
	return NewMediaFolderRepository(db.getOrmer())
}

func (db *SQLStore) Genre() model.GenreRepository {
	return NewGenreRepository(db.getOrmer())
}

func (db *SQLStore) Playlist() model.PlaylistRepository {
	return NewPlaylistRepository(db.getOrmer())
}

func (db *SQLStore) Property() model.PropertyRepository {
	return NewPropertyRepository(db.getOrmer())
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
	verbose := conf.Sonic.LogLevel == "trace"
	orm.Debug = verbose
	orm.RegisterModel(new(artist))
	orm.RegisterModel(new(album))
	orm.RegisterModel(new(mediaFile))
	orm.RegisterModel(new(checksum))
	orm.RegisterModel(new(property))
	orm.RegisterModel(new(playlist))
	orm.RegisterModel(new(Search))
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
