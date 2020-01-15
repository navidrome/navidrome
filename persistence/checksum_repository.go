package persistence

import (
	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/log"
)

type checkSumRepository struct {
	data map[string]string
}

const checkSumId = "1"

type Checksum struct {
	ID  string `orm:"pk;column(id)"`
	Sum string
}

func NewCheckSumRepository() domain.CheckSumRepository {
	r := &checkSumRepository{}
	return r
}

func (r *checkSumRepository) loadData() error {
	loadedData := make(map[string]string)

	var all []Checksum
	_, err := Db().QueryTable(&Checksum{}).All(&all)
	if err != nil {
		return err
	}

	for _, cks := range all {
		loadedData[cks.ID] = cks.Sum
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
	err := withTx(func(o orm.Ormer) error {
		_, err := Db().Raw("delete from checksum").Exec()
		if err != nil {
			return err
		}

		var checksums []Checksum
		for k, v := range newSums {
			cks := Checksum{ID: k, Sum: v}
			checksums = append(checksums, cks)
		}
		_, err = Db().InsertMulti(batchSize, &checksums)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}
	r.data = newSums
	return nil
}

var _ domain.CheckSumRepository = (*checkSumRepository)(nil)
