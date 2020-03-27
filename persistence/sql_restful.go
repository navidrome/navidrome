package persistence

import (
	"fmt"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/rest"
	"github.com/kennygrant/sanitize"
)

type filterFunc = func(field string, value interface{}) Sqlizer

type sqlRestful struct {
	filterMappings map[string]filterFunc
}

func (r sqlRestful) parseRestFilters(options rest.QueryOptions) Sqlizer {
	if len(options.Filters) == 0 {
		return nil
	}
	filters := And{}
	for f, v := range options.Filters {
		if ff, ok := r.filterMappings[f]; ok {
			filters = append(filters, ff(f, v))
		} else if f == "id" {
			filters = append(filters, eqFilter(f, v))
		} else {
			filters = append(filters, startsWithFilter(f, v))
		}
	}
	return filters
}

func (r sqlRestful) parseRestOptions(options ...rest.QueryOptions) model.QueryOptions {
	qo := model.QueryOptions{}
	if len(options) > 0 {
		qo.Sort = options[0].Sort
		qo.Order = strings.ToLower(options[0].Order)
		qo.Max = options[0].Max
		qo.Offset = options[0].Offset
		qo.Filters = r.parseRestFilters(options[0])
	}
	return qo
}

func eqFilter(field string, value interface{}) Sqlizer {
	return Eq{field: value}
}

func startsWithFilter(field string, value interface{}) Sqlizer {
	return Like{field: fmt.Sprintf("%s%%", value)}
}

func booleanFilter(field string, value interface{}) Sqlizer {
	v := strings.ToLower(value.(string))
	return Eq{field: strings.ToLower(v) == "true"}
}

func fullTextFilter(field string, value interface{}) Sqlizer {
	q := value.(string)
	q = strings.TrimSpace(sanitize.Accents(strings.ToLower(strings.TrimSuffix(q, "*"))))
	parts := strings.Split(q, " ")
	filters := And{}
	for _, part := range parts {
		filters = append(filters, Or{
			Like{"full_text": part + "%"},
			Like{"full_text": "%" + part + "%"},
		})
	}
	return filters
}
