package persistence

import (
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/liuzl/gocc"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
)

func getFullText(text ...string) string {
	fullText := utils.SanitizeStrings(text...)
	return " " + fullText
}

func (r sqlRepository) doSearch(q string, offset, size int, results interface{}, orderBys ...string) error {
	q = strings.TrimSpace(q)
	q = strings.TrimSuffix(q, "*")
	if len(q) < 2 {
		return nil
	}

	sq := r.newSelectWithAnnotation(r.tableName + ".id").Columns(r.tableName + ".*")
	filter := fullTextExpr(q)
	if filter != nil {
		sq = sq.Where(filter)
		if len(orderBys) > 0 {
			sq = sq.OrderBy(orderBys...)
		}
	} else {
		// If the filter is empty, we sort by id.
		// This is to speed up the results of `search3?query=""`, for OpenSubsonic
		sq = sq.OrderBy("id")
	}
	sq = sq.Limit(uint64(size)).Offset(uint64(offset))
	err := r.queryAll(sq, results, model.QueryOptions{Offset: offset})
	return err
}

var s2t, _ = gocc.New("s2t")

func getHantToHansFilter(q string, sep string) Sqlizer {
	tq, err := s2t.Convert(q)
	if err != nil {
		log.Warn("Error convert to traditional Chinese. ", err)
		return nil
	}
	if tq == q {
		return nil
	}
	filters := And{}
	parts := strings.Split(tq, " ")
	for _, part := range parts {
		filters = append(filters, Like{"full_text": "%" + sep + part + "%"})
	}
	return filters
}

func fullTextExpr(value string) Sqlizer {
	q := utils.SanitizeStrings(value)
	if q == "" {
		return nil
	}
	var sep string
	if !conf.Server.SearchFullString {
		sep = " "
	}

	all_filters := Or{}

	if conf.Server.SearchHantWithHans {
		filters := getHantToHansFilter(q, sep)
		if filters != nil {
			all_filters = append(all_filters, filters)
		}
	}

	parts := strings.Split(q, " ")
	filters := And{}
	for _, part := range parts {
		filters = append(filters, Like{"full_text": "%" + sep + part + "%"})
	}
	all_filters = append(all_filters, filters)

	return all_filters
}
