package persistence

import (
	"strings"
	"sync"

	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/log"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

const batchSize = 100

var (
	once   sync.Once
	driver = "sqlite3"
)

func Db() orm.Ormer {
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
	return orm.NewOrm()
}

func withTx(block func(orm.Ormer) error) error {
	o := orm.NewOrm()
	err := o.Begin()
	if err != nil {
		return err
	}

	err = block(o)
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

func initORM(dbPath string) error {
	verbose := conf.Sonic.LogLevel == "debug"
	orm.Debug = verbose
	orm.RegisterModel(new(Artist))
	orm.RegisterModel(new(Album))
	orm.RegisterModel(new(MediaFile))
	orm.RegisterModel(new(ArtistInfo))
	orm.RegisterModel(new(Checksum))
	orm.RegisterModel(new(Property))
	orm.RegisterModel(new(Playlist))
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
