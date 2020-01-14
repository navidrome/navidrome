package persistence

import (
	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/log"
)

type sqlRepository struct {
	tableName string
}

func (r *sqlRepository) newQuery(o orm.Ormer, options ...domain.QueryOptions) orm.QuerySeter {
	q := o.QueryTable(r.tableName)
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
	return r.newQuery(Db()).Count()
}

func (r *sqlRepository) Exists(id string) (bool, error) {
	c, err := r.newQuery(Db()).Filter("id", id).Count()
	return c == 1, err
}

// TODO This is used to generate random lists. Can be optimized in SQL: https://stackoverflow.com/a/19419
func (r *sqlRepository) GetAllIds() ([]string, error) {
	qs := r.newQuery(Db())
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

func (r *sqlRepository) put(o orm.Ormer, id string, a interface{}) error {
	c, err := r.newQuery(o).Filter("id", id).Count()
	if err != nil {
		return err
	}
	if c == 0 {
		_, err = o.Insert(a)
		return err
	}
	_, err = o.Update(a)
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

func (r *sqlRepository) DeleteAll() error {
	return withTx(func(o orm.Ormer) error {
		_, err := r.newQuery(Db()).Filter("id__isnull", false).Delete()
		return err
	})
}

func (r *sqlRepository) purgeInactive(o orm.Ormer, activeList interface{}, getId func(item interface{}) string) ([]string, error) {
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
		_, err := r.newQuery(o).Filter("id__in", subset).Delete()
		if err != nil {
			return nil, err
		}
	}
	return idsToDelete, nil
}
