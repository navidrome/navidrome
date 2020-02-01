package persistence

import (
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/navidrome/log"
	"github.com/kennygrant/sanitize"
)

const searchTable = "search"

func (r sqlRepository) index(id string, text string) error {
	sanitizedText := strings.TrimSpace(sanitize.Accents(strings.ToLower(text)))

	values := map[string]interface{}{
		"id":        id,
		"item_type": r.tableName,
		"full_text": sanitizedText,
	}
	update := Update(searchTable).Where(Eq{"id": id}).SetMap(values)
	count, err := r.executeSQL(update)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	insert := Insert(searchTable).SetMap(values)
	_, err = r.executeSQL(insert)
	return err
}

func (r sqlRepository) doSearch(q string, offset, size int, results interface{}, orderBys ...string) error {
	q = strings.TrimSpace(sanitize.Accents(strings.ToLower(strings.TrimSuffix(q, "*"))))
	if len(q) <= 2 {
		return nil
	}
	sq := Select("*").From(r.tableName)
	sq = sq.Limit(uint64(size)).Offset(uint64(offset))
	if len(orderBys) > 0 {
		sq = sq.OrderBy(orderBys...)
	}
	sq = sq.Join("search").Where("search.id = " + r.tableName + ".id")
	parts := strings.Split(q, " ")
	for _, part := range parts {
		sq = sq.Where(Or{
			Like{"full_text": part + "%"},
			Like{"full_text": "%" + part + "%"},
		})
	}
	sql, args, err := r.toSql(sq)
	if err != nil {
		return err
	}
	_, err = r.ormer.Raw(sql, args...).QueryRows(results)
	return err
}

func (r sqlRepository) cleanSearchIndex() error {
	del := Delete(searchTable).Where(Eq{"item_type": r.tableName}).Where("id not in (select id from " + r.tableName + ")")
	c, err := r.executeSQL(del)
	if err != nil {
		return err
	}
	if c > 0 {
		log.Debug(r.ctx, "Clean-up search index", "table", r.tableName, "totalDeleted", c)
	}
	return nil
}
