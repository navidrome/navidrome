// Package criteria implements a Criteria API based on Masterminds/squirrel
package criteria

import (
	"encoding/json"
	"errors"
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
	f := fieldMap[strings.ToLower(c.Sort)]
	var mapped string
	if f == nil {
		log.Error("Invalid field in 'sort' field. Using 'title'", "sort", c.Sort)
		mapped = fieldMap["title"].field
	} else {
		if f.order == "" {
			mapped = f.field
		} else {
			mapped = f.order
		}
	}
	if c.Order != "" {
		if strings.EqualFold(c.Order, "asc") || strings.EqualFold(c.Order, "desc") {
			mapped = mapped + " " + c.Order
		} else {
			log.Error("Invalid value in 'order' field. Valid values: 'asc', 'desc'", "order", c.Order)
		}
	}
	return mapped
}

func (c Criteria) ToSql() (sql string, args []interface{}, err error) {
	return c.Expression.ToSql()
}

func (c Criteria) ChildPlaylistIds() (ids []string) {
	if c.Expression == nil {
		return ids
	}

	switch rules := c.Expression.(type) {
	case Any:
		ids = rules.ChildPlaylistIds()
	case All:
		ids = rules.ChildPlaylistIds()
	}

	return ids
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
