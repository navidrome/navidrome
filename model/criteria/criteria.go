// Package criteria implements a Criteria API based on Masterminds/squirrel
package criteria

import (
	"encoding/json"

	"github.com/Masterminds/squirrel"
)

type Expression = squirrel.Sqlizer

type Criteria struct {
	Expression
	Sort   string
	Order  string
	Max    int
	Offset int
}

func (c Criteria) ToSql() (sql string, args []interface{}, err error) {
	return c.Expression.ToSql()
}

func (c Criteria) MarshalJSON() ([]byte, error) {
	aux := struct {
		All    []Expression `json:"all,omitempty"`
		Any    []Expression `json:"any,omitempty"`
		Sort   string       `json:"sort"`
		Order  string       `json:"order,omitempty"`
		Max    int          `json:"max,omitempty"`
		Offset int          `json:"offset"`
	}{
		Sort:   c.Sort,
		Order:  c.Order,
		Max:    c.Max,
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
		All    unmarshalConjunctionType `json:"all,omitempty"`
		Any    unmarshalConjunctionType `json:"any,omitempty"`
		Sort   string                   `json:"sort"`
		Order  string                   `json:"order,omitempty"`
		Max    int                      `json:"max,omitempty"`
		Offset int                      `json:"offset"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.Any) > 0 {
		c.Expression = Any(aux.Any)
	} else if len(aux.All) > 0 {
		c.Expression = All(aux.All)
	}
	c.Sort = aux.Sort
	c.Order = aux.Order
	c.Max = aux.Max
	c.Offset = aux.Offset
	return nil
}
