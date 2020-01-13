package db_sql

import (
	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/scanner"
)

type checkSumRepository struct {
	data map[string]string
}

const checkSumId = "1"

type CheckSums struct {
	ID    string `orm:"pk;column(id)"`
	Value string
}

func NewCheckSumRepository() scanner.CheckSumRepository {
	r := &checkSumRepository{}
	return r
}

func (r *checkSumRepository) loadData() error {
	loadedData := make(map[string]string)

	var all []CheckSums
	_, err := Db().QueryTable(&CheckSums{}).All(&all)
	if err != nil {
		return err
	}

	for _, cks := range all {
		loadedData[cks.ID] = cks.Value
	}

	r.data = loadedData
	log.Debug("Loaded checksums", "total", len(loadedData))
	return nil
}

func (r *checkSumRepository) Get(id string) (string, error) {
	if r.data == nil {
		err := r.loadData()
		if err != nil {
			return "", err
		}
	}
	return r.data[id], nil
}

func (r *checkSumRepository) SetData(newSums map[string]string) error {
	err := WithTx(func(o orm.Ormer) error {
		_, err := Db().Raw("delete from check_sums").Exec()
		if err != nil {
			return err
		}

		for k, v := range newSums {
			cks := CheckSums{ID: k, Value: v}
			// TODO Use InsertMulti
			_, err := Db().Insert(&cks)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	r.data = newSums
	return nil
}

var _ scanner.CheckSumRepository = (*checkSumRepository)(nil)
