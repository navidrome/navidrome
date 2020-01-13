package db_sql

import (
	"encoding/json"

	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/scanner"
)

type checkSumRepository struct {
	data map[string]string
}

const checkSumId = "1"

type CheckSums struct {
	ID   string `orm:"pk;column(id)"`
	Data string `orm:"type(text)"`
}

func NewCheckSumRepository() scanner.CheckSumRepository {
	r := &checkSumRepository{}
	return r
}

func (r *checkSumRepository) loadData() error {
	loadedData := make(map[string]string)
	r.data = loadedData

	cks := CheckSums{ID: checkSumId}
	err := Db().Read(&cks)
	if err == orm.ErrNoRows {
		_, err = Db().Insert(&cks)
		return err
	}
	if err != nil {
		return err
	}
	_ = json.Unmarshal([]byte(cks.Data), &loadedData)
	log.Debug("Loaded checksums", "total", len(loadedData))
	return err
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
	data, _ := json.Marshal(&newSums)
	cks := CheckSums{ID: checkSumId, Data: string(data)}
	var err error
	if Db().QueryTable(&CheckSums{}).Filter("id", checkSumId).Exist() {
		_, err = Db().Update(&cks)
	} else {
		_, err = Db().Insert(&cks)
	}
	if err != nil {
		return err
	}
	r.data = newSums
	return nil
}

var _ scanner.CheckSumRepository = (*checkSumRepository)(nil)
