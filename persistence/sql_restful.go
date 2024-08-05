package persistence

import (
	"fmt"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
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
		if v == "" {
			continue
		}
		if ff, ok := r.filterMappings[f]; ok {
			filters = append(filters, ff(f, v))
		} else if strings.HasSuffix(strings.ToLower(f), "id") {
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
		if seed, ok := options[0].Filters["seed"].(string); ok {
			qo.Seed = seed
			delete(options[0].Filters, "seed")
		}
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

func containsFilter(field string) func(string, any) Sqlizer {
	return func(_ string, value any) Sqlizer {
		return Like{field: fmt.Sprintf("%%%s%%", value)}
	}
}

func booleanFilter(field string, value interface{}) Sqlizer {
	v := strings.ToLower(value.(string))
	return Eq{field: strings.ToLower(v) == "true"}
}

func fullTextFilter(field string, value interface{}) Sqlizer {
	return fullTextExpr(value.(string))
}

func substringFilter(field string, value interface{}) Sqlizer {
	parts := strings.Split(value.(string), " ")
	filters := And{}
	for _, part := range parts {
		filters = append(filters, Like{field: "%" + part + "%"})
	}
	return filters
}

func idFilter(tableName string) func(string, interface{}) Sqlizer {
	return func(field string, value interface{}) Sqlizer {
		return Eq{tableName + ".id": value}
	}
}
