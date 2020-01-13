package db_sql

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

type sqlSearcher struct{}

func (s *sqlSearcher) Index(o orm.Ormer, table, id, text string) error {
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

func (s *sqlSearcher) Remove(o orm.Ormer, table string, ids []string) error {
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

func (s *sqlSearcher) DeleteAll(o orm.Ormer, table string) error {
	_, err := o.QueryTable(&Search{}).Filter("table", table).Delete()
	return err
}

func (s *sqlSearcher) Search(table string, q string, offset, size int, results interface{}, orderBys ...string) error {
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
