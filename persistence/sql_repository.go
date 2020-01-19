package persistence

import (
	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/model"
)

type sqlRepository struct {
	tableName string
	ormer     orm.Ormer
}

func (r *sqlRepository) newQuery(options ...model.QueryOptions) orm.QuerySeter {
	q := r.ormer.QueryTable(r.tableName)
	if len(options) > 0 {
		opts := options[0]
		q = q.Offset(opts.Offset)
		if opts.Size > 0 {
			q = q.Limit(opts.Size)
		}
		if opts.SortBy != "" {
			if opts.Desc {
				q = q.OrderBy("-" + opts.SortBy)
			} else {
				q = q.OrderBy(opts.SortBy)
			}
		}
	}
	return q
}

func (r *sqlRepository) CountAll() (int64, error) {
	return r.newQuery().Count()
}

func (r *sqlRepository) Exists(id string) (bool, error) {
	c, err := r.newQuery().Filter("id", id).Count()
	return c == 1, err
}

// TODO This is used to generate random lists. Can be optimized in SQL: https://stackoverflow.com/a/19419
func (r *sqlRepository) GetAllIds() ([]string, error) {
	qs := r.newQuery()
	var values []orm.Params
	num, err := qs.Values(&values, "id")
	if num == 0 {
		return nil, err
	}

	result := collectField(values, func(item interface{}) string {
		return item.(orm.Params)["ID"].(string)
	})

	return result, nil
}

// "Hack" to bypass Postgres driver limitation
func (r *sqlRepository) insert(record interface{}) error {
	_, err := r.ormer.Insert(record)
	if err != nil && err.Error() != "LastInsertId is not supported by this driver" {
		return err
	}
	return nil
}

func (r *sqlRepository) put(id string, a interface{}) error {
	c, err := r.newQuery().Filter("id", id).Count()
	if err != nil {
		return err
	}
	if c == 0 {
		err = r.insert(a)
		if err != nil && err.Error() == "LastInsertId is not supported by this driver" {
			err = nil
		}
		return err
	}
	_, err = r.ormer.Update(a)
	return err
}

func paginateSlice(slice []string, skip int, size int) []string {
	if skip > len(slice) {
		skip = len(slice)
	}

	end := skip + size
	if end > len(slice) {
		end = len(slice)
	}

	return slice[skip:end]
}

func difference(slice1 []string, slice2 []string) []string {
	var diffStr []string
	m := map[string]int{}

	for _, s1Val := range slice1 {
		m[s1Val] = 1
	}
	for _, s2Val := range slice2 {
		m[s2Val] = m[s2Val] + 1
	}

	for mKey, mVal := range m {
		if mVal == 1 {
			diffStr = append(diffStr, mKey)
		}
	}

	return diffStr
}

func (r *sqlRepository) Delete(id string) error {
	_, err := r.newQuery().Filter("id", id).Delete()
	return err
}

func (r *sqlRepository) DeleteAll() error {
	_, err := r.newQuery().Filter("id__isnull", false).Delete()
	return err
}

func (r *sqlRepository) purgeInactive(activeList interface{}, getId func(item interface{}) string) ([]string, error) {
	allIds, err := r.GetAllIds()
	if err != nil {
		return nil, err
	}
	activeIds := collectField(activeList, getId)
	idsToDelete := difference(allIds, activeIds)
	if len(idsToDelete) == 0 {
		return nil, nil
	}
	log.Debug("Purging inactive records", "table", r.tableName, "total", len(idsToDelete))

	var offset int
	for {
		var subset = paginateSlice(idsToDelete, offset, batchSize)
		if len(subset) == 0 {
			break
		}
		log.Trace("-- Purging inactive records", "table", r.tableName, "num", len(subset), "from", offset)
		offset += len(subset)
		_, err := r.newQuery().Filter("id__in", subset).Delete()
		if err != nil {
			return nil, err
		}
	}
	return idsToDelete, nil
}
