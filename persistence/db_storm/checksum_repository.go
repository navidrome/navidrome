package db_storm

import (
	"github.com/asdine/storm"
	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/scanner"
)

var (
	checkSumBucket = "_Checksums"
)

type checkSumRepository struct {
	data map[string]string
}

func NewCheckSumRepository() scanner.CheckSumRepository {
	r := &checkSumRepository{}
	return r
}

func (r *checkSumRepository) loadData() error {
	loadedData := make(map[string]string)
	err := Db().Get(checkSumBucket, checkSumBucket, &loadedData)
	if err == storm.ErrNotFound {
		return domain.ErrNotFound
	}
	log.Debug("Loaded checksums", "total", len(loadedData))
	r.data = loadedData
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
	err := Db().Set(checkSumBucket, checkSumBucket, newSums)
	if err != nil {
		return err
	}
	r.data = newSums
	return nil
}

var _ scanner.CheckSumRepository = (*checkSumRepository)(nil)
