package repositories

import (
	"fmt"
	"crypto/md5"
	"strings"
)

type BaseRepository struct {
	key string // TODO Rename to 'table'
}

func (r *BaseRepository) NewId(fields ...string) string {
	s := fmt.Sprintf("%s\\%s", strings.ToUpper(r.key), strings.Join(fields, ""))
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

func (r *BaseRepository) CountAll() (int, error) {
	return count(r.key)
}

func (r *BaseRepository) saveOrUpdate(id string, rec interface{}) error {
	return saveStruct(r.key, id, rec)
}

func (r *BaseRepository) Dump() {
}


