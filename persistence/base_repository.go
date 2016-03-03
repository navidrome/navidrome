package persistence

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/deluan/gosonic/utils"
	"reflect"
	"strings"
)

type baseRepository struct {
	table      string
	entityType reflect.Type
	fieldNames []string
}

func (r *baseRepository) init(table string, entity interface{}) {
	r.table = table
	r.entityType = reflect.TypeOf(entity).Elem()

	h, _ := utils.ToMap(entity)
	r.fieldNames = make([]string, len(h))
	i := 0
	for k := range h {
		r.fieldNames[i] = k
		i++
	}
}

// TODO Use annotations to specify fields to be used
func (r *baseRepository) NewId(fields ...string) string {
	s := fmt.Sprintf("%s\\%s", strings.ToUpper(r.table), strings.Join(fields, ""))
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

func (r *baseRepository) CountAll() (int, error) {
	ids, err := db().SMembers([]byte(r.table + "s:all"))
	return len(ids), err
}

func (r *baseRepository) Exists(id string) (bool, error) {
	res, err := db().SIsMember([]byte(r.table + "s:all"), []byte(id))
	return res != 0, err
}

func (r *baseRepository) saveOrUpdate(id string, entity interface{}) error {
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
func (r *baseRepository) getParent(entity interface{}) (table string, id string) {
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

func (r *baseRepository) getFieldKeys(id string) [][]byte {
	recordPrefix := fmt.Sprintf("%s:%s:", r.table, id)
	var fieldKeys = make([][]byte, len(r.fieldNames))
	for i, n := range r.fieldNames {
		fieldKeys[i] = []byte(recordPrefix + n)
	}
	return fieldKeys
}

func (r *baseRepository) newInstance() interface{} {
	return reflect.New(r.entityType).Interface()
}

func (r *baseRepository) readEntity(id string) (interface{}, error) {
	entity := r.newInstance()

	fieldKeys := r.getFieldKeys(id)

	res, err := db().MGet(fieldKeys...)
	if err != nil {
		return nil, err
	}
	err = r.toEntity(res, entity)
	return entity, err
}

func (r *baseRepository) toEntity(response [][]byte, entity interface{}) error {
	var record = make(map[string]interface{}, len(response))
	for i, v := range response {
		if len(v) > 0 {
			var value interface{}
			if err := json.Unmarshal(v, &value); err != nil {
				return err
			}
			record[string(r.fieldNames[i])] = value
		}
	}

	return utils.ToStruct(record, entity)
}

func (r *baseRepository) loadAll(entities interface{}, sortBy string, alpha bool) error {
	setName := r.table + "s:all"
	return r.loadFromSet(setName, entities, sortBy, alpha)
}

func (r* baseRepository) loadChildren(parentTable string, parentId string, entities interface{}, sortBy string, alpha bool) error {
	setName := fmt.Sprintf("%s:%s:%ss", parentTable, parentId, r.table)
	return r.loadFromSet(setName, entities, sortBy, alpha)
}

// TODO Optimize it! Probably very slow (and confusing!)
func (r *baseRepository) loadFromSet(setName string, entities interface{}, sortBy string, alpha bool) error {
	reflected := reflect.ValueOf(entities).Elem()
	var sortKey []byte = nil
	if sortBy != "" {
		sortKey = []byte(fmt.Sprintf("%s:*:%s", r.table, sortBy))
	}
	response, err := db().XSSort([]byte(setName), 0, 0, alpha, false, sortKey, r.getFieldKeys("*"))
	if err != nil {
		return err
	}
	numFields := len(r.fieldNames)
	for i := 0; i < (len(response) / numFields); i++ {
		start := i * numFields
		entity := reflect.New(r.entityType).Interface()

		if err := r.toEntity(response[start:start+numFields], entity); err != nil {
			return err
		}
		reflected.Set(reflect.Append(reflected, reflect.ValueOf(entity).Elem()))
	}

	return nil

}