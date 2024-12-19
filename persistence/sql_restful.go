package persistence

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/fatih/structs"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type filterFunc = func(field string, value any) Sqlizer

func (r *sqlRepository) parseRestFilters(ctx context.Context, options rest.QueryOptions) Sqlizer {
	if len(options.Filters) == 0 {
		return nil
	}
	filters := And{}
	for f, v := range options.Filters {
		// Ignore filters with empty values
		if v == "" {
			continue
		}
		// Look for a custom filter function
		f = strings.ToLower(f)
		if ff, ok := r.filterMappings[f]; ok {
			if filter := ff(f, v); filter != nil {
				filters = append(filters, filter)
			}
			continue
		}
		// Ignore invalid filters (not based on a field or filter function)
		if r.isFieldWhiteListed != nil && !r.isFieldWhiteListed(f) {
			log.Warn(ctx, "Ignoring filter not whitelisted", "filter", f, "table", r.tableName)
			continue
		}
		// For fields ending in "id", use an exact match
		if strings.HasSuffix(f, "id") {
			filters = append(filters, eqFilter(f, v))
			continue
		}
		// Default to a "starts with" filter
		filters = append(filters, startsWithFilter(f, v))
	}
	return filters
}

func (r *sqlRepository) parseRestOptions(ctx context.Context, options ...rest.QueryOptions) model.QueryOptions {
	qo := model.QueryOptions{}
	if len(options) > 0 {
		qo.Sort, qo.Order = r.sanitizeSort(options[0].Sort, options[0].Order)
		qo.Max = options[0].Max
		qo.Offset = options[0].Offset
		if seed, ok := options[0].Filters["seed"].(string); ok {
			qo.Seed = seed
			delete(options[0].Filters, "seed")
		}
		qo.Filters = r.parseRestFilters(ctx, options[0])
	}
	return qo
}

func (r sqlRepository) sanitizeSort(sort, order string) (string, string) {
	if sort != "" {
		sort = toSnakeCase(sort)
		if mapped, ok := r.sortMappings[sort]; ok {
			sort = mapped
		} else {
			if !r.isFieldWhiteListed(sort) {
				log.Warn(r.ctx, "Ignoring sort not whitelisted", "sort", sort, "table", r.tableName)
				sort = ""
			}
		}
	}
	if order != "" {
		order = strings.ToLower(order)
		if order != "desc" {
			order = "asc"
		}
	}
	return sort, order
}

func eqFilter(field string, value any) Sqlizer {
	return Eq{field: value}
}

func startsWithFilter(field string, value any) Sqlizer {
	return Like{field: fmt.Sprintf("%s%%", value)}
}

func containsFilter(field string) func(string, any) Sqlizer {
	return func(_ string, value any) Sqlizer {
		return Like{field: fmt.Sprintf("%%%s%%", value)}
	}
}

func booleanFilter(field string, value any) Sqlizer {
	v := strings.ToLower(value.(string))
	return Eq{field: v == "true"}
}

func fullTextFilter(tableName string) func(string, any) Sqlizer {
	return func(field string, value any) Sqlizer { return fullTextExpr(tableName, value.(string)) }
}

func substringFilter(field string, value any) Sqlizer {
	parts := strings.Fields(value.(string))
	filters := And{}
	for _, part := range parts {
		filters = append(filters, Like{field: "%" + part + "%"})
	}
	return filters
}

func idFilter(tableName string) func(string, any) Sqlizer {
	return func(field string, value any) Sqlizer { return Eq{tableName + ".id": value} }
}

func invalidFilter(ctx context.Context) func(string, any) Sqlizer {
	return func(field string, value any) Sqlizer {
		log.Warn(ctx, "Invalid filter", "fieldName", field, "value", value)
		return Eq{"1": "0"}
	}
}

var (
	whiteList = map[string]map[string]struct{}{}
	mutex     sync.RWMutex
)

func registerModelWhiteList(instance any) fieldWhiteListedFunc {
	name := reflect.TypeOf(instance).String()
	registerFieldWhiteList(name, instance)
	return getFieldWhiteListedFunc(name)
}

func registerFieldWhiteList(name string, instance any) {
	mutex.Lock()
	defer mutex.Unlock()
	if whiteList[name] != nil {
		return
	}
	m := structs.Map(instance)
	whiteList[name] = map[string]struct{}{}
	for k := range m {
		whiteList[name][toSnakeCase(k)] = struct{}{}
	}
	ma := structs.Map(model.Annotations{})
	for k := range ma {
		whiteList[name][toSnakeCase(k)] = struct{}{}
	}
}

type fieldWhiteListedFunc func(field string) bool

func getFieldWhiteListedFunc(tableName string) fieldWhiteListedFunc {
	return func(field string) bool {
		mutex.RLock()
		defer mutex.RUnlock()
		if _, ok := whiteList[tableName]; !ok {
			return false
		}
		_, ok := whiteList[tableName][field]
		return ok
	}
}
