package repositories

import (
	"sync"
	"encoding/json"
	"github.com/astaxie/beego"
	"github.com/siddontang/ledisdb/ledis"
	"github.com/siddontang/ledisdb/config"
	"github.com/deluan/gosonic/utils"
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

func hmset(key string, data interface{}) error {
	h, err := utils.Flatten(data)
	if err != nil {
		return err
	}
	var fvList = make([]ledis.FVPair, len(h))
	i := 0
	for f, v := range h {
		fvList[i].Field = []byte(f)
		fvList[i].Value, _ = json.Marshal(v)
		i++
	}
	return db().HMset([]byte(key), fvList...)
}

func hset(key, field, value string) error {
	_, err := db().HSet([]byte(key), []byte(field), []byte(value))
	return err
}