package criteria

import (
	"strings"

	"github.com/navidrome/navidrome/log"
)

type SortField struct {
	Field string
	Desc  bool
}

func (c Criteria) OrderByFields() []SortField {
	sortValue := c.Sort
	if sortValue == "" {
		sortValue = "title"
	}

	order := strings.ToLower(strings.TrimSpace(c.Order))
	if order != "" && order != "asc" && order != "desc" {
		log.Error("Invalid value in 'order' field. Valid values: 'asc', 'desc'", "order", c.Order)
		order = ""
	}

	parts := strings.Split(sortValue, ",")
	fields := make([]SortField, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		desc := false
		if strings.HasPrefix(part, "+") || strings.HasPrefix(part, "-") {
			desc = strings.HasPrefix(part, "-")
			part = strings.TrimSpace(part[1:])
		}
		info, ok := LookupField(part)
		if !ok {
			log.Error("Invalid field in 'sort' field", "sort", part)
			continue
		}
		if order == "desc" {
			desc = !desc
		}
		fields = append(fields, SortField{Field: info.Name(), Desc: desc})
	}
	if len(fields) == 0 {
		log.Warn("No valid sort fields found in 'sort', falling back to 'title'", "sort", sortValue)
		return []SortField{{Field: "title", Desc: false}}
	}
	return fields
}

func (c Criteria) SortFieldNames() []string {
	sortFields := c.OrderByFields()
	names := make([]string, len(sortFields))
	for i, sf := range sortFields {
		names[i] = sf.Field
	}
	return names
}
