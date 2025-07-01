// Package criteria implements a Criteria API based on Masterminds/squirrel
package criteria

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
)

type Expression = squirrel.Sqlizer

type Criteria struct {
	Expression
	Sort   string
	Order  string
	Limit  int
	Offset int
}

func (c Criteria) OrderBy() string {
	if c.Sort == "" {
		c.Sort = "title"
	}

	order := strings.ToLower(strings.TrimSpace(c.Order))
	if order != "" && order != "asc" && order != "desc" {
		log.Error("Invalid value in 'order' field. Valid values: 'asc', 'desc'", "order", c.Order)
		order = ""
	}

	parts := strings.Split(c.Sort, ",")
	fields := make([]string, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		dir := "asc"
		if strings.HasPrefix(p, "+") || strings.HasPrefix(p, "-") {
			if strings.HasPrefix(p, "-") {
				dir = "desc"
			}
			p = strings.TrimSpace(p[1:])
		}

		sortField := strings.ToLower(p)
		f := fieldMap[sortField]
		if f == nil {
			log.Error("Invalid field in 'sort' field", "sort", sortField)
			continue
		}

		var mapped string

		if f.order != "" {
			mapped = f.order
		} else if f.isTag {
			mapped = "COALESCE(json_extract(media_file.tags, '$." + sortField + "[0].value'), '')"
		} else if f.isRole {
			mapped = "COALESCE(json_extract(media_file.participants, '$." + sortField + "[0].name'), '')"
		} else {
			mapped = f.field
		}
		if f.numeric {
			mapped = fmt.Sprintf("CAST(%s AS REAL)", mapped)
		}
		// If the global 'order' field is set to 'desc', reverse the default or field-specific sort direction.
		// This ensures that the global order applies consistently across all fields.
		if order == "desc" {
			if dir == "asc" {
				dir = "desc"
			} else {
				dir = "asc"
			}
		}

		fields = append(fields, mapped+" "+dir)
	}

	return strings.Join(fields, ", ")
}

func (c Criteria) ToSql() (sql string, args []any, err error) {
	return c.Expression.ToSql()
}

func (c Criteria) ChildPlaylistIds() []string {
	if c.Expression == nil {
		return nil
	}

	if parent := c.Expression.(interface{ ChildPlaylistIds() (ids []string) }); parent != nil {
		return parent.ChildPlaylistIds()
	}

	return nil
}

func (c Criteria) MarshalJSON() ([]byte, error) {
	aux := struct {
		All    []Expression `json:"all,omitempty"`
		Any    []Expression `json:"any,omitempty"`
		Sort   string       `json:"sort,omitempty"`
		Order  string       `json:"order,omitempty"`
		Limit  int          `json:"limit,omitempty"`
		Offset int          `json:"offset,omitempty"`
	}{
		Sort:   c.Sort,
		Order:  c.Order,
		Limit:  c.Limit,
		Offset: c.Offset,
	}
	switch rules := c.Expression.(type) {
	case Any:
		aux.Any = rules
	case All:
		aux.All = rules
	default:
		aux.All = All{rules}
	}
	return json.Marshal(aux)
}

func (c *Criteria) UnmarshalJSON(data []byte) error {
	var aux struct {
		All    unmarshalConjunctionType `json:"all"`
		Any    unmarshalConjunctionType `json:"any"`
		Sort   string                   `json:"sort"`
		Order  string                   `json:"order"`
		Limit  int                      `json:"limit"`
		Offset int                      `json:"offset"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.Any) > 0 {
		c.Expression = Any(aux.Any)
	} else if len(aux.All) > 0 {
		c.Expression = All(aux.All)
	} else {
		return errors.New("invalid criteria json. missing rules (key 'all' or 'any')")
	}
	c.Sort = aux.Sort
	c.Order = aux.Order
	c.Limit = aux.Limit
	c.Offset = aux.Offset
	return nil
}
