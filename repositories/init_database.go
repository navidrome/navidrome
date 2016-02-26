package repositories

import (
	"github.com/HouzuoGuo/tiedot/db"
	"github.com/astaxie/beego"
	"sync"
)

var (
	_dbInstance *db.DB
	once sync.Once
)

func createCollection(name string) *db.Col {
	col := dbInstance().Use(name)
	if col != nil {
		return col
	}
	if err := dbInstance().Create(name); err != nil {
		beego.Error(err)
	}
	if err := col.Index([]string{"Id"}); err != nil {
		beego.Error(name, err)
	}
	return col
}

func dbInstance() *db.DB {
	once.Do(func() {
		instance, err := db.OpenDB(beego.AppConfig.String("dbPath"))
		if err != nil {
			panic(err)
		}
		_dbInstance = instance
	})
	return _dbInstance
}