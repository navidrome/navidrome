// Package criteria implements the smart playlist criteria DSL.
package criteria

import (
	"encoding/json"
	"errors"

	"github.com/navidrome/navidrome/log"
)

type Expression interface {
	criteriaExpression()
}

type Criteria struct {
	Expression
	Sort         string
	Order        string
	Limit        int
	LimitPercent int
	Offset       int
}

// EffectiveLimit resolves the effective limit for a query. If a fixed Limit is
// set it takes precedence. Otherwise, if LimitPercent is set, the limit is
// computed as a percentage of totalCount (minimum 1 when totalCount > 0).
// Returns 0 when no limit applies.
func (c Criteria) EffectiveLimit(totalCount int64) int {
	if c.Limit > 0 {
		return c.Limit
	}
	if c.LimitPercent > 0 && c.LimitPercent <= 100 {
		if totalCount <= 0 {
			return 0
		}
		result := int(totalCount) * c.LimitPercent / 100
		if result < 1 {
			return 1
		}
		return result
	}
	return 0
}

// IsPercentageLimit returns true when the criteria uses a valid percentage-based
// limit (i.e. LimitPercent is in [1, 100] and no fixed Limit overrides it).
func (c Criteria) IsPercentageLimit() bool {
	return c.Limit == 0 && c.LimitPercent > 0 && c.LimitPercent <= 100
}

func (c Criteria) ChildPlaylistIds() []string {
	if c.Expression == nil {
		return nil
	}

	if parent, ok := c.Expression.(interface{ ChildPlaylistIds() (ids []string) }); ok {
		return parent.ChildPlaylistIds()
	}

	return nil
}

func (c Criteria) MarshalJSON() ([]byte, error) {
	aux := struct {
		All          []Expression `json:"all,omitempty"`
		Any          []Expression `json:"any,omitempty"`
		Sort         string       `json:"sort,omitempty"`
		Order        string       `json:"order,omitempty"`
		Limit        int          `json:"limit,omitempty"`
		LimitPercent int          `json:"limitPercent,omitempty"`
		Offset       int          `json:"offset,omitempty"`
	}{
		Sort:         c.Sort,
		Order:        c.Order,
		Limit:        c.Limit,
		LimitPercent: c.LimitPercent,
		Offset:       c.Offset,
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
		All          unmarshalConjunctionType `json:"all"`
		Any          unmarshalConjunctionType `json:"any"`
		Sort         string                   `json:"sort"`
		Order        string                   `json:"order"`
		Limit        int                      `json:"limit"`
		LimitPercent int                      `json:"limitPercent"`
		Offset       int                      `json:"offset"`
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

	// Clamp LimitPercent to [0, 100]
	if aux.LimitPercent < 0 {
		log.Warn("limitPercent value out of range, clamping to 0", "value", aux.LimitPercent)
		aux.LimitPercent = 0
	} else if aux.LimitPercent > 100 {
		log.Warn("limitPercent value out of range, clamping to 100", "value", aux.LimitPercent)
		aux.LimitPercent = 100
	}
	c.LimitPercent = aux.LimitPercent
	return nil
}
