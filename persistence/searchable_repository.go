package persistence

import (
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/kennygrant/sanitize"
)

type Search struct {
	ID       string `orm:"pk;column(id)"`
	Table    string `orm:"index"`
	FullText string `orm:"type(text)"`
}

type searchableRepository struct {
	sqlRepository
}

func (r *searchableRepository) DeleteAll() error {
	return withTx(func(o orm.Ormer) error {
		_, err := r.newQuery(Db()).Filter("id__isnull", false).Delete()
		if err != nil {
			return err
		}
		return r.removeAllFromIndex(o, r.tableName)
	})
}

func (r *searchableRepository) put(o orm.Ormer, id string, textToIndex string, a interface{}) error {
	c, err := r.newQuery(o).Filter("id", id).Count()
	if err != nil {
		return err
	}
	if c == 0 {
		_, err = o.Insert(a)
	} else {
		_, err = o.Update(a)
	}
	if err != nil {
		return err
	}
	return r.addToIndex(o, r.tableName, id, textToIndex)
}

func (r *searchableRepository) purgeInactive(o orm.Ormer, activeList interface{}, getId func(item interface{}) string) ([]string, error) {
	idsToDelete, err := r.sqlRepository.purgeInactive(o, activeList, getId)
	if err != nil {
		return nil, err
	}
	return idsToDelete, r.removeFromIndex(o, r.tableName, idsToDelete)
}

func (r *searchableRepository) addToIndex(o orm.Ormer, table, id, text string) error {
	item := Search{ID: id, Table: table}
	err := o.Read(&item)
	if err != nil && err != orm.ErrNoRows {
		return err
	}
	sanitizedText := strings.TrimSpace(sanitize.Accents(strings.ToLower(text)))
	item = Search{ID: id, Table: table, FullText: sanitizedText}
	if err == orm.ErrNoRows {
		_, err = o.Insert(&item)
	} else {
		_, err = o.Update(&item)
	}
	return err
}

func (r *searchableRepository) removeFromIndex(o orm.Ormer, table string, ids []string) error {
	var offset int
	for {
		var subset = paginateSlice(ids, offset, batchSize)
		if len(subset) == 0 {
			break
		}
		log.Trace("Deleting searchable items", "table", table, "num", len(subset), "from", offset)
		offset += len(subset)
		_, err := o.QueryTable(&Search{}).Filter("table", table).Filter("id__in", subset).Delete()
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *searchableRepository) removeAllFromIndex(o orm.Ormer, table string) error {
	_, err := o.QueryTable(&Search{}).Filter("table", table).Delete()
	return err
}

func (r *searchableRepository) doSearch(table string, q string, offset, size int, results interface{}, orderBys ...string) error {
	q = strings.TrimSpace(sanitize.Accents(strings.ToLower(strings.TrimSuffix(q, "*"))))
	if len(q) <= 2 {
		return nil
	}
	sq := squirrel.Select("*").From(table).OrderBy()
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
	_, err = Db().Raw(sql, args...).QueryRows(results)
	return err
}
