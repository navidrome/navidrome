package persistence

import (
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/kennygrant/sanitize"
)

type search struct {
	ID       string `orm:"pk;column(id)"`
	Table    string `orm:"index"`
	FullText string `orm:"index"`
}

type searchableRepository struct {
	sqlRepository
}

func (r *searchableRepository) DeleteAll() error {
	_, err := r.newQuery().Filter("id__isnull", false).Delete()
	if err != nil {
		return err
	}
	return r.removeAllFromIndex(r.ormer, r.tableName)
}

func (r *searchableRepository) put(id string, textToIndex string, a interface{}, fields ...string) error {
	c, err := r.newQuery().Filter("id", id).Count()
	if err != nil {
		return err
	}
	if c == 0 {
		err = r.insert(a)
		if err != nil && err.Error() == "LastInsertId is not supported by this driver" {
			err = nil
		}
	} else {
		_, err = r.ormer.Update(a, fields...)
	}
	if err != nil {
		return err
	}
	return r.addToIndex(r.tableName, id, textToIndex)
}

func (r *searchableRepository) addToIndex(table, id, text string) error {
	item := search{ID: id, Table: table}
	err := r.ormer.Read(&item)
	if err != nil && err != orm.ErrNoRows {
		return err
	}
	sanitizedText := strings.TrimSpace(sanitize.Accents(strings.ToLower(text)))
	item = search{ID: id, Table: table, FullText: sanitizedText}
	if err == orm.ErrNoRows {
		err = r.insert(&item)
	} else {
		_, err = r.ormer.Update(&item)
	}
	return err
}

func (r *searchableRepository) removeFromIndex(table string, ids []string) error {
	var offset int
	for {
		var subset = paginateSlice(ids, offset, batchSize)
		if len(subset) == 0 {
			break
		}
		log.Trace("Deleting searchable items", "table", table, "num", len(subset), "from", offset)
		offset += len(subset)
		_, err := r.ormer.QueryTable(&search{}).Filter("table", table).Filter("id__in", subset).Delete()
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *searchableRepository) removeAllFromIndex(o orm.Ormer, table string) error {
	_, err := o.QueryTable(&search{}).Filter("table", table).Delete()
	return err
}

func (r *searchableRepository) doSearch(table string, q string, offset, size int, results interface{}, orderBys ...string) error {
	q = strings.TrimSpace(sanitize.Accents(strings.ToLower(strings.TrimSuffix(q, "*"))))
	if len(q) <= 2 {
		return nil
	}
	sq := squirrel.Select("*").From(table)
	sq = sq.Limit(uint64(size)).Offset(uint64(offset))
	if len(orderBys) > 0 {
		sq = sq.OrderBy(orderBys...)
	}
	sq = sq.Join("search").Where("search.id = " + table + ".id")
	parts := strings.Split(q, " ")
	for _, part := range parts {
		sq = sq.Where(squirrel.Or{
			squirrel.Like{"full_text": part + "%"},
			squirrel.Like{"full_text": "%" + part + "%"},
		})
	}
	sql, args, err := sq.ToSql()
	if err != nil {
		return err
	}
	_, err = r.ormer.Raw(sql, args...).QueryRows(results)
	return err
}
