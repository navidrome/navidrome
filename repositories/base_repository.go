package repositories

import (
	"fmt"
	"crypto/md5"
	"strings"
	"github.com/deluan/gosonic/utils"
	"encoding/json"
	"reflect"
)


type BaseRepository struct {
	table string
}

func (r *BaseRepository) NewId(fields ...string) string {
	s := fmt.Sprintf("%s\\%s", strings.ToUpper(r.table), strings.Join(fields, ""))
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

func (r *BaseRepository) CountAll() (int, error) {
	ids, err := db().SMembers([]byte(r.table + "s:all"))
	return len(ids), err
}

func (r *BaseRepository) saveOrUpdate(id string, rec interface{}) error {
	return r.saveEntity(id, rec)
}

func (r *BaseRepository) Dump() {
}

func (r *BaseRepository) saveEntity(id string, entity interface{}) error {
	recordPrefix := fmt.Sprintf("%s:%s:", r.table, id)
	allKey := r.table + "s:all"

	h, err := utils.ToMap(entity)
	if err != nil {
		return err
	}
	for f, v := range h {
		key := recordPrefix + f
		value, _ := json.Marshal(v)
		if err := db().Set([]byte(key), value); err != nil {
			return err
		}

	}

	if _, err = db().SAdd([]byte(allKey), []byte(id)); err != nil {
		return err
	}

	if parentTable, parentId := r.getParent(entity); parentTable != "" {
		parentCollectionKey := fmt.Sprintf("%s:%s:%ss", parentTable, parentId, r.table)
		_, err = db().SAdd([]byte(parentCollectionKey), []byte(id))
	}
	return nil
}

// TODO Optimize
func (r *BaseRepository) getParent(entity interface{}) (table string, id string) {
	dt := reflect.TypeOf(entity).Elem()
	for i := 0; i < dt.NumField(); i++ {
		f := dt.Field(i)
		table := f.Tag.Get("parent")
		if table != "" {
			dv := reflect.ValueOf(entity).Elem()
			return table, dv.FieldByName(f.Name).String()
		}
	}
	return "", ""
}

func (r *BaseRepository) loadEntity(id string, entity interface{}) error {
	recordPrefix := fmt.Sprintf("%s:%s:", r.table, id)

	h, _ := utils.ToMap(entity)
	var fieldKeys = make([][]byte, len(h))
	var fieldNames = make([]string, len(h))
	i := 0
	for k, _ := range h {
		fieldNames[i] = k
		fieldKeys[i] = []byte(recordPrefix + k)
		i++
	}

	res, err := db().MGet(fieldKeys...)
	if err != nil {
		return err
	}
	var record = make(map[string]interface{}, len(res))
	for i, v := range res {
		var value interface{}
		if err := json.Unmarshal(v, &value); err != nil {
			return err
		}
		record[string(fieldNames[i])] = value
	}

	return utils.ToStruct(record, entity)
}
