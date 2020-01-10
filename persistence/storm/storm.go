package storm

import (
	"sync"

	"github.com/asdine/storm"
)

var (
	_dbInstance *storm.DB
	once        sync.Once
)

func Db() *storm.DB {
	once.Do(func() {
		instance, err := storm.Open("./storm.db")
		if err != nil {
			panic(err)
		}
		_dbInstance = instance
	})
	return _dbInstance
}
