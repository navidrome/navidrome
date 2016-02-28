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

func saveStruct(key, id string, data interface{}) error {
	h, err := utils.ToMap(data)
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
	kh := key + "_id_" + id
	ks := key + "_ids"
	db().SAdd([]byte(ks), []byte(id))
	return db().HMset([]byte(kh), fvList...)
}

func readStruct(key string) (interface{}, error) {
	fvs, _ := db().HGetAll([]byte(key))
	var m = make(map[string]interface{}, len(fvs))
	for _, fv := range fvs {
		var v interface{}
		json.Unmarshal(fv.Value, &v)
		m[string(fv.Field)] = v
	}

	return utils.ToStruct(m)
}

func count(key string) (int, error) {
	ids, err := db().SMembers([]byte(key + "_ids"))
	return len(ids), err
}

func hset(key, field, value string) error {
	_, err := db().HSet([]byte(key), []byte(field), []byte(value))
	return err
}