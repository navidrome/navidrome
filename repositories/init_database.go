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

func createCollection(name string, ix ...interface{}) *db.Col {
	log := false
	if dbInstance().Use(name) == nil {
		if err := dbInstance().Create(name); err != nil {
			beego.Error(err)
		}
		log = true
	}

	col := dbInstance().Use(name)

	createIndex(col, []string{"Id"}, log)
	for _, i := range ix {
		switch i := i.(type) {
		case string:
			createIndex(col, []string{i}, log)
		case []string:
			createIndex(col, i, log)
		default:
			beego.Error("Trying to create an Index with an invalid type: ", i)
		}
	}
	return col
}

func createIndex(col *db.Col, path []string, log bool) (err error) {
	if err := col.Index(path); err != nil && log {
		beego.Error(path, err)
	}
	return err
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