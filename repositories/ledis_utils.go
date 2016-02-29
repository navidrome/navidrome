package repositories

import (
	"sync"
	"github.com/astaxie/beego"
	"github.com/siddontang/ledisdb/ledis"
	"github.com/siddontang/ledisdb/config"
)

var (
	_dbInstance *ledis.DB
	once sync.Once
)

func db() *ledis.DB {
	once.Do(func() {
		config := config.NewConfigDefault()
		config.DataDir = beego.AppConfig.String("dbPath")
		l, _ := ledis.Open(config)
		instance, err := l.Select(0)
		if err != nil {
			panic(err)
		}
		_dbInstance = instance
	})
	return _dbInstance
}