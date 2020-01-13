package db_sql

import (
	"os"
	"path"
	"sync"

	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/log"
	_ "github.com/mattn/go-sqlite3"
)

var once sync.Once

func Db() orm.Ormer {
	once.Do(func() {
		err := os.MkdirAll(conf.Sonic.DbPath, 0700)
		if err != nil {
			panic(err)
		}
		dbPath := path.Join(conf.Sonic.DbPath, "sqlite.db")
		err = initORM(dbPath)
		if err != nil {
			panic(err)
		}
		log.Debug("Opening SQLite DB from: " + dbPath)
	})
	return orm.NewOrm()
}

func WithTx(block func(orm.Ormer) error) error {
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
	orm.Debug = true
	orm.RegisterModel(new(Artist))
	orm.RegisterModel(new(Album))
	orm.RegisterModel(new(MediaFile))
	orm.RegisterModel(new(ArtistInfo))
	orm.RegisterModel(new(CheckSums))
	orm.RegisterModel(new(Property))
	err := orm.RegisterDataBase("default", "sqlite3", dbPath)
	if err != nil {
		panic(err)
	}
	return orm.RunSyncdb("default", false, true)
}
