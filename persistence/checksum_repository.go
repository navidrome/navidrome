package persistence

import (
	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/model"
)

type checkSumRepository struct {
}

const checkSumId = "1"

type Checksum struct {
	ID  string `orm:"pk;column(id)"`
	Sum string
}

func NewCheckSumRepository() model.ChecksumRepository {
	r := &checkSumRepository{}
	return r
}

func (r *checkSumRepository) GetData() (model.ChecksumMap, error) {
	loadedData := make(map[string]string)

	var all []Checksum
	_, err := Db().QueryTable(&Checksum{}).Limit(-1).All(&all)
	if err != nil {
		return nil, err
	}

	for _, cks := range all {
		loadedData[cks.ID] = cks.Sum
	}

	return loadedData, nil
}

func (r *checkSumRepository) SetData(newSums model.ChecksumMap) error {
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
	return nil
}

var _ model.ChecksumRepository = (*checkSumRepository)(nil)
