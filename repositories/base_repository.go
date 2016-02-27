package repositories

import (
	"encoding/json"
	"github.com/HouzuoGuo/tiedot/db"
	"github.com/astaxie/beego"
	"fmt"
)

type BaseRepository struct {
	col *db.Col
}

func (r *BaseRepository) marshal(rec interface{}) (map[string]interface{}, error) {
	// Convert to JSON...
	b, err := json.Marshal(rec);
	if err != nil {
		return nil, err
	}

	// ... then convert to map
	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	return m, err
}

func (r*BaseRepository) query(q string, a ...interface{}) (map[int]struct{}, error) {
	q = fmt.Sprintf(q, a)

	var query interface{}
	json.Unmarshal([]byte(q), &query)

	queryResult := make(map[int]struct{})

	err := db.EvalQuery(query, r.col, &queryResult)
	if err != nil {
		beego.Warn("Error '%s' - query='%s'", q, err)
	}
	return queryResult, err
}

func (r*BaseRepository) queryFirstKey(q string, a ...interface{}) (int, error) {
	result, err := r.query(q, a)
	if err != nil {
		return 0, err
	}
	for key, _ := range result {
		return key, nil
	}

	return 0, nil
}

func (r *BaseRepository) saveOrUpdate(rec interface{}) error {
	m, err := r.marshal(rec)
	if err != nil {
		return err
	}
	docId, err := r.queryFirstKey(`{"in": ["Id"], "eq": "%s", "limit": 1}`, m["Id"])
	if docId == 0 {
		_, err = r.col.Insert(m)
		return err
	}
	err = r.col.Update(docId, m)
	if err != nil {
		beego.Warn("Error updating %s[%d]: %s", r.col, docId, err)
	}
	return err
}

func (r *BaseRepository) Dump() {
	r.col.ForEachDoc(func(id int, docContent []byte) (willMoveOn bool) {
		beego.Debug("Document", id, "=", string(docContent))
		return true
	})
}


